package server

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"visor/internal/agent"
	"visor/internal/config"
	"visor/internal/forgejo"
	"visor/internal/memory"
	"visor/internal/observability"
	"visor/internal/platform/telegram"
	"visor/internal/scheduler"
	"visor/internal/selfevolve"
	"visor/internal/setup"
	"visor/internal/skills"
	"visor/internal/voice"
)

type Server struct {
	cfg          *config.Config
	mux          *http.ServeMux
	tg           *telegram.Client
	dedup        *telegram.Dedup
	agent        *agent.QueuedAgent
	voice        *voice.Handler
	scheduler    *scheduler.Scheduler
	quickActions *scheduler.QuickActionHandler
	skills       *skills.Manager
	selfevolver  *selfevolve.Manager
	setupState   setup.State
	log          *observability.Logger
}

func New(cfg *config.Config, a agent.Agent) *Server {
	tg := telegram.NewClient(cfg.TelegramBotToken)

	s := &Server{
		cfg:   cfg,
		mux:   http.NewServeMux(),
		tg:    tg,
		dedup: telegram.NewDedup(5 * time.Minute),
		log:   observability.Component("server"),
	}

	if cfg.OpenAIAPIKey != "" {
		s.voice = voice.NewHandler(tg, cfg.OpenAIAPIKey)
		if cfg.ElevenLabsAPIKey != "" && cfg.ElevenLabsVoiceID != "" {
			s.voice.SetTTS(cfg.ElevenLabsAPIKey, cfg.ElevenLabsVoiceID)
		}
	}

	// skill manager
	sm := skills.NewManager(cfg.DataDir + "/skills")
	if loadErr := sm.Reload(); loadErr != nil {
		s.log.Warn(context.Background(), "skills load failed", "error", loadErr.Error())
	}
	s.skills = sm

	s.selfevolver = selfevolve.New(selfevolve.Config{
		Enabled: cfg.SelfEvolutionEnabled,
		RepoDir: cfg.SelfEvolutionRepoDir,
		Push:    cfg.SelfEvolutionPush,
	})

	// wire up backend switch notification for multi-backend registry
	if reg, ok := a.(*agent.Registry); ok {
		reg.OnSwitch = func(from, to string) {
			note := fmt.Sprintf("⚡ backend switched: %s → %s (rate limit / quota)", from, to)
			s.log.Info(context.Background(), "backend failover", "from", from, "to", to)
			if cfg.UserChatID != "" {
				_ = s.tg.SendMessage(mustParseChatID(cfg.UserChatID), note)
			}
		}
	}

	s.agent = agent.NewQueuedAgent(a, cfg.AgentBackend, func(ctx context.Context, chatID int64, response string, err error, duration time.Duration) {
		if err != nil {
			s.log.Error(ctx, "agent processing failed", "chat_id", chatID, "backend", cfg.AgentBackend, "error", err.Error())
			response = fmt.Sprintf("error: %v", err)
		}

		// skill actions from agent response
		if s.skills != nil {
			clean, skillActions, parseErr := skills.ExtractActions(response)
			if parseErr != nil {
				s.log.Error(ctx, "skill action parse failed", "chat_id", chatID, "error", parseErr.Error())
			} else if skillActions != nil {
				response = clean
				s.executeSkillActions(ctx, chatID, skillActions)
			}
		}

		// scheduler actions from agent response
		if s.scheduler != nil {
			clean, scheduleActions, parseErr := scheduler.ExtractActions(response)
			if parseErr != nil {
				s.log.Error(ctx, "schedule action parse failed", "chat_id", chatID, "error", parseErr.Error())
			} else if scheduleActions != nil {
				response = clean
				note := s.executeScheduleActions(ctx, scheduleActions)
				if note != "" {
					response = strings.TrimSpace(response + "\n\n" + note)
				}
			}
		}

		clean, setupActions, parseErr := setup.ExtractActions(response)
		if parseErr != nil {
			s.log.Error(ctx, "setup action parse failed", "chat_id", chatID, "error", parseErr.Error())
		} else if setupActions != nil {
			response = clean
			note := s.executeSetupActions(ctx, setupActions)
			if note != "" {
				response = strings.TrimSpace(response + "\n\n" + note)
			}
		}

		if strings.TrimSpace(response) == "" {
			response = "ok"
		}

		s.log.Info(ctx, "webhook message processed", "chat_id", chatID, "backend", cfg.AgentBackend)
		text, meta := parseResponse(response)
		text = sanitizeUserReply(text)
		text = strings.TrimSpace(text + "\n\n⏱ " + formatDuration(duration) + " · " + s.agent.CurrentBackend())

		if meta.SendVoice && s.voice != nil && s.voice.TTSEnabled() {
			if err := s.voice.SynthesizeAndSend(chatID, text); err != nil {
				s.log.Error(ctx, "voice synth failed, fallback to text", "chat_id", chatID, "error", err.Error())
				if sendErr := s.tg.SendMessage(chatID, text); sendErr != nil {
					s.log.Error(ctx, "send reply failed", "chat_id", chatID, "error", sendErr.Error())
				} else {
					s.log.Info(ctx, "webhook reply sent", "chat_id", chatID, "mode", "text-fallback")
				}
			} else {
				s.log.Info(ctx, "webhook reply sent", "chat_id", chatID, "mode", "voice")
			}
		} else {
			if sendErr := s.tg.SendMessage(chatID, text); sendErr != nil {
				s.log.Error(ctx, "send reply failed", "chat_id", chatID, "error", sendErr.Error())
			} else {
				s.log.Info(ctx, "webhook reply sent", "chat_id", chatID, "mode", "text")
			}
		}

		if meta.CodeChanges && s.selfevolver != nil && s.selfevolver.Enabled() {
			go s.runSelfEvolution(chatID, meta.CommitMessage)
		}

		if meta.GitPush {
			pushDir := meta.GitPushDir
			if pushDir == "" {
				pushDir = s.cfg.SelfEvolutionRepoDir
			}
			if pushDir != "" {
				forgejo.PushBackground(ctx, pushDir, s.log)
			}
		}
	})
	s.agent.SetLongRunningHandler(func(ctx context.Context, chatID int64, elapsed time.Duration, preview string) {
		note := fmt.Sprintf("⏳ request läuft seit %s\n\naktueller rpc output:\n`%s`", formatDuration(elapsed), escapeTelegramCode(preview))
		if err := s.tg.SendMessage(chatID, note); err != nil {
			s.log.Warn(ctx, "long-running notification failed", "chat_id", chatID, "error", err.Error())
			return
		}
		s.log.Info(ctx, "long-running notification sent", "chat_id", chatID, "elapsed_ms", elapsed.Milliseconds())
	})

	schedulerInstance, err := scheduler.New(cfg.DataDir+"/scheduler", func(ctx context.Context, task scheduler.Task) {
		if s.quickActions != nil {
			s.quickActions.RecordTrigger(task)
		}
		content := fmt.Sprintf("[scheduled task]\nid: %s\nrecurring: %t\nprompt: %s", task.ID, task.Recurring, task.Prompt)
		s.agent.Enqueue(ctx, agent.Message{
			ChatID:  mustParseChatID(cfg.UserChatID),
			Content: content,
			Type:    "scheduled",
		})
	})
	if err != nil {
		panic(fmt.Sprintf("scheduler init failed: %v", err))
	}
	s.scheduler = schedulerInstance

	loc, locErr := time.LoadLocation(cfg.Timezone)
	if locErr != nil {
		s.log.Warn(context.Background(), "invalid timezone, defaulting to UTC", "tz", cfg.Timezone, "error", locErr.Error())
		loc = time.UTC
	}
	s.quickActions = scheduler.NewQuickActionHandler(schedulerInstance, loc, s.log)

	projectRoot := cfg.SelfEvolutionRepoDir
	if strings.TrimSpace(projectRoot) == "" {
		projectRoot = "."
	}
	setupState, setupErr := setup.Detect(projectRoot, cfg.DataDir)
	if setupErr != nil {
		s.log.Warn(context.Background(), "setup detect failed", "error", setupErr.Error())
	} else {
		s.setupState = setupState
		if setupState.FirstRun {
			s.log.Info(context.Background(), "first-run setup mode active", "missing", setupState.Missing)
		}
	}

	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("POST /webhook", s.handleWebhook)
	s.mux.HandleFunc("POST /forgejo/webhook", s.handleForgejoWebhook)
	return s
}

func (s *Server) ListenAndServe() error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	s.log.Info(context.Background(), "server starting", "addr", addr, "log_level", s.cfg.LogLevel, "log_verbose", s.cfg.LogVerbose)

	if s.scheduler != nil {
		go s.scheduler.Start(context.Background())
		s.log.Info(context.Background(), "scheduler started")
	}

	s.notifyStartup(context.Background())

	handler := observability.RequestIDMiddleware(observability.RecoverMiddleware("http", s.mux))
	return http.ListenAndServe(addr, handler)
}

func (s *Server) notifyStartup(ctx context.Context) {
	chatID := mustParseChatID(s.cfg.UserChatID)
	rev := currentShortRevision(s.cfg.SelfEvolutionRepoDir)
	msg := fmt.Sprintf("🎺 visor restarted — rev `%s`", rev)
	if err := s.tg.SendMessage(chatID, msg); err != nil {
		s.log.Warn(ctx, "startup notification failed", "chat_id", chatID, "error", err.Error())
		return
	}
	s.log.Info(ctx, "startup notification sent", "chat_id", chatID, "rev", rev)
}

func currentShortRevision(repoDir string) string {
	dir := strings.TrimSpace(repoDir)
	if dir == "" {
		dir = "."
	}
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	rev := strings.TrimSpace(string(out))
	if rev == "" {
		return "unknown"
	}
	return rev
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx, span := observability.StartSpan(r.Context(), "webhook.handle")
	defer span.End()
	r = r.WithContext(ctx)

	s.log.Debug(r.Context(), "webhook lifecycle", "stage", "received", "method", r.Method, "path", r.URL.Path)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.log.Warn(r.Context(), "webhook read body failed", "error", err.Error())
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}

	if s.cfg.TelegramWebhookSecret != "" {
		sig := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
		if !verifySignature(sig, s.cfg.TelegramWebhookSecret) {
			s.log.Warn(r.Context(), "webhook signature invalid", "remote_addr", r.RemoteAddr)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	var update telegram.Update
	if err := json.Unmarshal(body, &update); err != nil {
		s.log.Warn(r.Context(), "webhook bad json", "error", err.Error())
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	s.log.Debug(r.Context(), "webhook lifecycle", "stage", "parsed", "update_id", update.UpdateID)

	if s.dedup.IsDuplicate(update.UpdateID) {
		s.log.Debug(r.Context(), "webhook lifecycle", "stage", "deduped", "result", "duplicate", "update_id", update.UpdateID)
		w.WriteHeader(http.StatusOK)
		return
	}
	s.log.Debug(r.Context(), "webhook lifecycle", "stage", "deduped", "result", "accepted", "update_id", update.UpdateID)

	msg := update.Message
	if msg == nil {
		s.log.Debug(r.Context(), "webhook has no message payload", "update_id", update.UpdateID)
		w.WriteHeader(http.StatusOK)
		return
	}

	chatID := strconv.FormatInt(msg.Chat.ID, 10)
	if chatID != s.cfg.UserChatID {
		s.log.Warn(r.Context(), "webhook unauthorized chat", "chat_id", chatID)
		w.WriteHeader(http.StatusOK)
		return
	}
	s.log.Debug(r.Context(), "webhook lifecycle", "stage", "authorized", "chat_id", chatID)

	var content string
	var msgType string
	switch {
	case msg.Voice != nil:
		msgType = "voice"
		if s.voice != nil {
			text, err := s.voice.Transcribe(msg.Voice.FileID)
			if err != nil {
				s.log.Error(r.Context(), "voice transcription failed", "chat_id", chatID, "error", err.Error())
				content = "[Voice message - transcription failed]"
			} else {
				content = fmt.Sprintf("[Voice message] %s", text)
			}
		} else {
			content = fmt.Sprintf("[voice:%s]", msg.Voice.FileID)
		}
	case len(msg.Photo) > 0:
		msgType = "photo"
		best := msg.Photo[len(msg.Photo)-1]
		content = fmt.Sprintf("[photo:%s]", best.FileID)
		if msg.Caption != "" {
			content += " " + msg.Caption
		}
	case msg.Text != "":
		msgType = "text"
		content = msg.Text
	default:
		s.log.Warn(r.Context(), "webhook unsupported message type", "chat_id", chatID)
		w.WriteHeader(http.StatusOK)
		return
	}

	s.log.Info(r.Context(), "webhook message accepted", "message_type", msgType, "chat_id", chatID, "preview", truncate(content, 80))

	// quick action intercept: check if this is a reply to a recently triggered reminder
	if msgType == "text" && s.quickActions != nil {
		if reply, handled := s.quickActions.TryHandle(r.Context(), content); handled {
			s.log.Info(r.Context(), "quick action handled", "chat_id", chatID, "reply", reply)
			if sendErr := s.tg.SendMessage(msg.Chat.ID, reply); sendErr != nil {
				s.log.Error(r.Context(), "quick action reply failed", "chat_id", chatID, "error", sendErr.Error())
			}
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	// agent switch command: /agent [name]
	if msgType == "text" {
		trimmed := strings.TrimSpace(content)
		if trimmed == "/agent" || strings.HasPrefix(trimmed, "/agent ") {
			parts := strings.Fields(trimmed)
			var reply string
			if len(parts) == 1 {
				reply = fmt.Sprintf("current agent: *%s*", s.agent.CurrentBackend())
			} else {
				name := parts[1]
				if err := s.agent.SwitchBackend(name); err != nil {
					reply = fmt.Sprintf("❌ %v", err)
				} else {
					reply = fmt.Sprintf("✅ switched to *%s*", name)
				}
			}
			s.log.Info(r.Context(), "agent switch command", "chat_id", chatID, "command", trimmed)
			if sendErr := s.tg.SendMessage(msg.Chat.ID, reply); sendErr != nil {
				s.log.Error(r.Context(), "agent switch reply failed", "chat_id", chatID, "error", sendErr.Error())
			}
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	// auto-trigger: run matching skills and prepend output to agent context
	if s.skills != nil {
		content = s.enrichWithSkills(r.Context(), content, chatID, msgType)
	}
	if s.setupState.FirstRun {
		if setupCtx := setup.BuildContext(s.setupState); setupCtx != "" {
			content = content + "\n\n" + setupCtx
		}
	}

	// detach from request context so agent processing isn't canceled as soon as webhook returns 200.
	agentCtx := context.Background()
	s.agent.Enqueue(agentCtx, agent.Message{
		ChatID:  msg.Chat.ID,
		Content: content,
		Type:    msgType,
	})
	s.log.Debug(r.Context(), "webhook lifecycle", "stage", "queued", "chat_id", chatID, "message_type", msgType, "queue_len", s.agent.QueueLen())

	w.WriteHeader(http.StatusOK)
}

func verifySignature(got, secret string) bool {
	return hmac.Equal([]byte(got), []byte(secret))
}

func mustParseChatID(chatID string) int64 {
	id, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("invalid USER_PHONE_NUMBER/chat id: %s", chatID))
	}
	return id
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	mins := int(d / time.Minute)
	secs := int((d % time.Minute) / time.Second)
	return fmt.Sprintf("%dm%02ds", mins, secs)
}

func escapeTelegramCode(s string) string {
	s = strings.ReplaceAll(s, "`", "'")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	return truncate(s, 220)
}

// enrichWithSkills checks for auto-trigger matches and injects skill context.
func (s *Server) enrichWithSkills(ctx context.Context, content, chatID, msgType string) string {
	matched := s.skills.Match(content)
	if len(matched) == 0 {
		// no trigger matches, but still inject skill discovery
		desc := s.skills.Describe()
		if desc != "" {
			return content + "\n\n[system context]\n" + desc
		}
		return content
	}

	var enrichments []string
	for _, skill := range matched {
		result, err := s.skills.Exec().Run(ctx, skill, skills.Context{
			UserMessage: content,
			ChatID:      chatID,
			MessageType: msgType,
			Platform:    "telegram",
			DataDir:     s.cfg.DataDir,
			SkillDir:    skill.Dir,
		})
		if err != nil {
			s.log.Error(ctx, "auto-trigger skill failed", "skill", skill.Manifest.Name, "error", err.Error())
			continue
		}

		if result.ExitCode != 0 {
			s.log.Warn(ctx, "auto-trigger skill non-zero exit", "skill", skill.Manifest.Name, "exit_code", result.ExitCode, "stderr", truncate(result.Stderr, 200))
			continue
		}

		output := strings.TrimSpace(result.Stdout)
		if output != "" {
			enrichments = append(enrichments, fmt.Sprintf("[skill:%s output]\n%s", skill.Manifest.Name, output))
			s.log.Info(ctx, "auto-trigger skill ran", "skill", skill.Manifest.Name, "output_len", len(output))
		}
	}

	if len(enrichments) > 0 {
		return content + "\n\n" + strings.Join(enrichments, "\n\n")
	}
	return content
}

// executeSkillActions processes create/edit/delete actions from agent response.
func (s *Server) executeSkillActions(ctx context.Context, chatID int64, actions *skills.ActionEnvelope) {
	for _, a := range actions.Create {
		if err := s.skills.Create(a); err != nil {
			s.log.Error(ctx, "skill create failed", "name", a.Name, "error", err.Error())
		} else {
			s.log.Info(ctx, "skill created by agent", "name", a.Name)
		}
	}
	for _, a := range actions.Edit {
		if err := s.skills.Edit(a); err != nil {
			s.log.Error(ctx, "skill edit failed", "name", a.Name, "error", err.Error())
		} else {
			s.log.Info(ctx, "skill edited by agent", "name", a.Name)
		}
	}
	for _, a := range actions.Delete {
		if err := s.skills.Delete(a.Name); err != nil {
			s.log.Error(ctx, "skill delete failed", "name", a.Name, "error", err.Error())
		} else {
			s.log.Info(ctx, "skill deleted by agent", "name", a.Name)
		}
	}
}

func (s *Server) executeScheduleActions(ctx context.Context, actions *scheduler.ActionEnvelope) string {
	messages := make([]string, 0)

	for _, a := range actions.Create {
		runAt, err := time.Parse(time.RFC3339, strings.TrimSpace(a.RunAt))
		if err != nil {
			msg := fmt.Sprintf("schedule create failed (%q): invalid run_at (RFC3339 required)", a.Prompt)
			messages = append(messages, msg)
			s.log.Error(ctx, "schedule create failed", "prompt", a.Prompt, "error", err.Error())
			continue
		}

		if a.IntervalSeconds > 0 {
			id, err := s.scheduler.AddRecurring(a.Prompt, runAt.UTC(), time.Duration(a.IntervalSeconds)*time.Second)
			if err != nil {
				msg := fmt.Sprintf("schedule create failed (%q): %s", a.Prompt, err.Error())
				messages = append(messages, msg)
				s.log.Error(ctx, "schedule create recurring failed", "prompt", a.Prompt, "error", err.Error())
				continue
			}
			messages = append(messages, fmt.Sprintf("scheduled recurring ✅ id=%s", id))
			continue
		}

		id, err := s.scheduler.AddOneShot(a.Prompt, runAt.UTC())
		if err != nil {
			msg := fmt.Sprintf("schedule create failed (%q): %s", a.Prompt, err.Error())
			messages = append(messages, msg)
			s.log.Error(ctx, "schedule create one-shot failed", "prompt", a.Prompt, "error", err.Error())
			continue
		}
		messages = append(messages, fmt.Sprintf("scheduled ✅ id=%s", id))
	}

	for _, a := range actions.Update {
		in := scheduler.UpdateTaskInput{}
		if strings.TrimSpace(a.Prompt) != "" {
			prompt := a.Prompt
			in.Prompt = &prompt
		}
		if strings.TrimSpace(a.RunAt) != "" {
			runAt, err := time.Parse(time.RFC3339, strings.TrimSpace(a.RunAt))
			if err != nil {
				msg := fmt.Sprintf("schedule update failed (%s): invalid run_at (RFC3339 required)", a.ID)
				messages = append(messages, msg)
				s.log.Error(ctx, "schedule update failed", "task_id", a.ID, "error", err.Error())
				continue
			}
			r := runAt.UTC()
			in.RunAt = &r
		}
		in.Recurring = a.Recurring
		in.IntervalSeconds = a.IntervalSeconds

		if err := s.scheduler.Update(a.ID, in); err != nil {
			msg := fmt.Sprintf("schedule update failed (%s): %s", a.ID, err.Error())
			messages = append(messages, msg)
			s.log.Error(ctx, "schedule update failed", "task_id", a.ID, "error", err.Error())
			continue
		}
		messages = append(messages, fmt.Sprintf("schedule updated ✅ id=%s", a.ID))
	}

	for _, a := range actions.Delete {
		if err := s.scheduler.Delete(a.ID); err != nil {
			msg := fmt.Sprintf("schedule delete failed (%s): %s", a.ID, err.Error())
			messages = append(messages, msg)
			s.log.Error(ctx, "schedule delete failed", "task_id", a.ID, "error", err.Error())
			continue
		}
		messages = append(messages, fmt.Sprintf("schedule deleted ✅ id=%s", a.ID))
	}

	if actions.List {
		list := s.scheduler.List()
		if len(list) == 0 {
			messages = append(messages, "no scheduled tasks")
		} else {
			messages = append(messages, "scheduled tasks:")
			for i, task := range list {
				if i >= 20 {
					messages = append(messages, "...truncated")
					break
				}
				when := task.NextRunAt.UTC().Format(time.RFC3339)
				if task.Recurring {
					messages = append(messages, fmt.Sprintf("- %s @ %s (every %ds) [%s]", task.Prompt, when, task.IntervalSeconds, task.ID))
				} else {
					messages = append(messages, fmt.Sprintf("- %s @ %s [%s]", task.Prompt, when, task.ID))
				}
			}
		}
	}

	return strings.TrimSpace(strings.Join(messages, "\n"))
}

func (s *Server) executeSetupActions(ctx context.Context, actions *setup.ActionEnvelope) string {
	messages := make([]string, 0)
	projectRoot := s.cfg.SelfEvolutionRepoDir
	if strings.TrimSpace(projectRoot) == "" {
		projectRoot = "."
	}

	if len(actions.EnvSet) > 0 || len(actions.EnvUnset) > 0 {
		if err := setup.UpdateDotEnv(projectRoot, actions.EnvSet, actions.EnvUnset); err != nil {
			messages = append(messages, "setup .env update failed: "+err.Error())
		} else {
			messages = append(messages, ".env updated ✅")
		}
	}

	token := strings.TrimSpace(actions.EnvSet["TELEGRAM_BOT_TOKEN"])
	if token == "" {
		token = s.cfg.TelegramBotToken
	}
	openAIKey := strings.TrimSpace(actions.EnvSet["OPENAI_API_KEY"])
	if openAIKey == "" {
		openAIKey = s.cfg.OpenAIAPIKey
	}

	if actions.ValidateTelegram {
		if token == "" {
			messages = append(messages, "telegram validation failed: TELEGRAM_BOT_TOKEN missing")
		} else {
			tg := telegram.NewClient(token)
			if err := tg.ValidateToken(); err != nil {
				messages = append(messages, "telegram validation failed: "+err.Error())
			} else {
				messages = append(messages, "telegram token valid ✅")
			}
		}
	}
	if actions.ValidateOpenAI {
		if openAIKey == "" {
			messages = append(messages, "openai validation failed: OPENAI_API_KEY missing")
		} else {
			embedder := memory.NewEmbeddingClient(openAIKey)
			if _, err := embedder.Embed("ping"); err != nil {
				messages = append(messages, "openai validation failed: "+err.Error())
			} else {
				messages = append(messages, "openai key valid ✅")
			}
		}
	}

	if strings.TrimSpace(actions.WebhookURL) != "" {
		if token == "" {
			messages = append(messages, "set webhook failed: TELEGRAM_BOT_TOKEN missing")
		} else {
			tg := telegram.NewClient(token)
			if err := tg.SetWebhook(strings.TrimSpace(actions.WebhookURL), strings.TrimSpace(actions.WebhookSecret)); err != nil {
				messages = append(messages, "set webhook failed: "+err.Error())
			} else {
				messages = append(messages, "webhook set ✅")
			}
		}
	}

	if actions.CheckHealth {
		healthURL := fmt.Sprintf("http://127.0.0.1:%d/health", s.cfg.Port)
		resp, err := http.Get(healthURL)
		if err != nil {
			messages = append(messages, "health check failed: "+err.Error())
		} else {
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				messages = append(messages, fmt.Sprintf("health check failed: status %d", resp.StatusCode))
			} else {
				messages = append(messages, "health check ok ✅")
			}
		}
	}

	if strings.TrimSpace(actions.PersonalityChoice) != "" {
		if err := setup.ApplyPersonalityOverride(projectRoot, actions.PersonalityFile, actions.PersonalityChoice, actions.PersonalityText); err != nil {
			messages = append(messages, "personality update failed: "+err.Error())
		} else {
			messages = append(messages, "personality updated ✅")
		}
	}

	if strings.TrimSpace(actions.SendTestMessage) != "" {
		chatID := mustParseChatID(s.cfg.UserChatID)
		if err := s.tg.SendMessage(chatID, actions.SendTestMessage); err != nil {
			messages = append(messages, "test message failed: "+err.Error())
		} else {
			messages = append(messages, "test message sent ✅")
		}
	}

	if actions.WriteSummary {
		summaryPath, err := setup.WriteSetupSummary(projectRoot, setup.SummaryInput{
			AgentBackend: s.cfg.AgentBackend,
			WebhookURL:   strings.TrimSpace(actions.WebhookURL),
			HealthOK:     actions.CheckHealth,
		})
		if err != nil {
			messages = append(messages, "write setup summary failed: "+err.Error())
		} else {
			messages = append(messages, "setup summary written ✅ ("+summaryPath+")")
		}
	}

	if actions.CleanupSetupHints {
		if err := setup.CleanupSetupInstructions(projectRoot, actions.PersonalityFile); err != nil {
			messages = append(messages, "cleanup setup hints failed: "+err.Error())
		} else {
			messages = append(messages, "setup hints cleaned ✅")
		}
	}

	state, err := setup.Detect(projectRoot, s.cfg.DataDir)
	if err == nil {
		s.setupState = state
	}

	return strings.TrimSpace(strings.Join(messages, "\n"))
}

type responseMeta struct {
	SendVoice     bool
	CodeChanges   bool
	CommitMessage string
	GitPush       bool
	GitPushDir    string // repo dir to push; defaults to SelfEvolutionRepoDir
}

// parseResponse extracts metadata from agent response.
// metadata block format:
//
//	---
//	send_voice: true
//	code_changes: true
//	commit_message: your message
//	git_push: true
//	git_push_dir: /path/to/repo
func parseResponse(raw string) (text string, meta responseMeta) {
	parts := strings.SplitN(raw, "\n---\n", 2)
	text = parts[0]
	if len(parts) == 2 {
		for _, line := range strings.Split(parts[1], "\n") {
			line = strings.TrimSpace(line)
			switch {
			case line == "send_voice: true" || line == "send_voice:true":
				meta.SendVoice = true
			case line == "code_changes: true" || line == "code_changes:true":
				meta.CodeChanges = true
			case strings.HasPrefix(line, "commit_message:"):
				meta.CommitMessage = strings.TrimSpace(strings.TrimPrefix(line, "commit_message:"))
			case line == "git_push: true" || line == "git_push:true":
				meta.GitPush = true
			case strings.HasPrefix(line, "git_push_dir:"):
				meta.GitPushDir = strings.TrimSpace(strings.TrimPrefix(line, "git_push_dir:"))
			}
		}
	}
	return
}

func sanitizeUserReply(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	// hide raw internal/tool event dumps if model leaks them
	internalMarkers := []string{
		"assistantMessageEvent",
		"tool_execution_start",
		"tool_execution_update",
		"tool_execution_end",
		"\"type\":\"toolcall",
		"\"role\":\"toolResult\"",
		"[system context]",
	}
	for _, marker := range internalMarkers {
		if strings.Contains(s, marker) {
			return "sorry — internal debug text leaked. try that once more 🙈"
		}
	}

	return s
}

func (s *Server) runSelfEvolution(chatID int64, commitMessage string) {
	ctx := context.Background()
	result, err := s.selfevolver.Apply(ctx, selfevolve.Request{
		CommitMessage: commitMessage,
		ChatID:        chatID,
		Backend:       s.cfg.AgentBackend,
	})
	if err != nil {
		s.log.Error(ctx, "self-evolution failed", "chat_id", chatID, "error", err.Error())
		_ = s.tg.SendMessage(chatID, "self-evolution failed: "+truncate(err.Error(), 200))
		return
	}

	if result.VetErr != "" {
		s.log.Warn(ctx, "self-evolution vet failed, commit rolled back", "chat_id", chatID, "vet_error", result.VetErr)
		_ = s.tg.SendMessage(chatID, "⚠️ go vet failed, rolled back:\n"+truncate(result.VetErr, 300))
		return
	}

	if result.BuildErr != "" {
		s.log.Warn(ctx, "self-evolution build failed, commit rolled back", "chat_id", chatID, "build_error", result.BuildErr)
		_ = s.tg.SendMessage(chatID, "⚠️ build failed, rolled back:\n"+truncate(result.BuildErr, 300))
		return
	}

	if result.Built {
		s.log.Info(ctx, "self-evolution completed, restarting", "chat_id", chatID)
		_ = s.tg.SendMessage(chatID, "self-evolution done, restarting... 🔄")
		s.selfevolver.Restart()
		return
	}

	s.log.Info(ctx, "self-evolution completed", "chat_id", chatID)
	_ = s.tg.SendMessage(chatID, "self-evolution done ✅")
}

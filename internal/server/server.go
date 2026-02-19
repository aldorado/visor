package server

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"visor/internal/agent"
	"visor/internal/config"
	emaillevelup "visor/internal/levelup/email"
	"visor/internal/observability"
	"visor/internal/platform/telegram"
	"visor/internal/voice"
)

type Server struct {
	cfg         *config.Config
	mux         *http.ServeMux
	tg          *telegram.Client
	dedup       *telegram.Dedup
	agent       *agent.QueuedAgent
	voice       *voice.Handler
	emailSender emaillevelup.Sender
	emailPoller *emaillevelup.Poller
	log         *observability.Logger
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

	if cfg.HimalayaEnabled {
		himalaya := emaillevelup.NewHimalayaClient(cfg.HimalayaAccount)
		s.emailSender = himalaya
		s.emailPoller = emaillevelup.NewPoller(himalaya, time.Duration(cfg.HimalayaPollInterval)*time.Second, func(msg emaillevelup.IncomingMessage) {
			s.agent.Enqueue(context.Background(), agent.Message{
				ChatID:  mustParseChatID(cfg.UserChatID),
				Content: emaillevelup.FormatInboundForAgent(msg),
				Type:    "email",
			})
		})
	}

	s.agent = agent.NewQueuedAgent(a, cfg.AgentBackend, func(ctx context.Context, chatID int64, response string, err error) {
		if err != nil {
			s.log.Error(ctx, "agent processing failed", "chat_id", chatID, "backend", cfg.AgentBackend, "error", err.Error())
			response = fmt.Sprintf("error: %v", err)
		}

		if s.emailSender != nil {
			clean, actions, parseErr := emaillevelup.ExtractActions(response)
			if parseErr != nil {
				s.log.Error(ctx, "email action parse failed", "chat_id", chatID, "error", parseErr.Error())
				response = response + "\n\n(email action parse failed)"
			} else {
				response = clean
				if len(actions) > 0 {
					if sendErr := emaillevelup.ExecuteActions(ctx, s.emailSender, actions); sendErr != nil {
						s.log.Error(ctx, "email action execution failed", "chat_id", chatID, "error", sendErr.Error())
						response = strings.TrimSpace(response + "\n\nemail send failed: " + sendErr.Error())
					} else {
						s.log.Info(ctx, "email action executed", "chat_id", chatID, "actions", len(actions))
						response = strings.TrimSpace(response + "\n\nemail action executed ✅")
					}
				}
			}
		}

		if strings.TrimSpace(response) == "" {
			response = "ok"
		}

		s.log.Info(ctx, "webhook message processed", "chat_id", chatID, "backend", cfg.AgentBackend)
		text, sendVoice := parseResponse(response)

		if sendVoice && s.voice != nil && s.voice.TTSEnabled() {
			if err := s.voice.SynthesizeAndSend(chatID, text); err != nil {
				s.log.Error(ctx, "voice synth failed, fallback to text", "chat_id", chatID, "error", err.Error())
				if sendErr := tg.SendMessage(chatID, text); sendErr != nil {
					s.log.Error(ctx, "send reply failed", "chat_id", chatID, "error", sendErr.Error())
				} else {
					s.log.Info(ctx, "webhook reply sent", "chat_id", chatID, "mode", "text-fallback")
				}
			} else {
				s.log.Info(ctx, "webhook reply sent", "chat_id", chatID, "mode", "voice")
			}
		} else {
			if sendErr := tg.SendMessage(chatID, text); sendErr != nil {
				s.log.Error(ctx, "send reply failed", "chat_id", chatID, "error", sendErr.Error())
			} else {
				s.log.Info(ctx, "webhook reply sent", "chat_id", chatID, "mode", "text")
			}
		}
	})

	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("POST /webhook", s.handleWebhook)
	return s
}

func (s *Server) ListenAndServe() error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	s.log.Info(context.Background(), "server starting", "addr", addr, "log_level", s.cfg.LogLevel, "log_verbose", s.cfg.LogVerbose)

	if s.emailPoller != nil {
		go s.emailPoller.Start(context.Background())
		s.log.Info(context.Background(), "email poller started", "interval_seconds", s.cfg.HimalayaPollInterval)
	}

	handler := observability.RequestIDMiddleware(observability.RecoverMiddleware("http", s.mux))
	return http.ListenAndServe(addr, handler)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
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

	s.agent.Enqueue(r.Context(), agent.Message{
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

// parseResponse extracts metadata from agent response.
// The agent can signal voice response by ending with:
//
//	---
//	send_voice: true
func parseResponse(raw string) (text string, sendVoice bool) {
	parts := strings.SplitN(raw, "\n---\n", 2)
	text = parts[0]
	if len(parts) == 2 {
		meta := parts[1]
		for _, line := range strings.Split(meta, "\n") {
			line = strings.TrimSpace(line)
			if line == "send_voice: true" || line == "send_voice:true" {
				sendVoice = true
			}
		}
	}
	return
}

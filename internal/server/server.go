package server

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"visor/internal/agent"
	"visor/internal/config"
	emaillevelup "visor/internal/levelup/email"
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
}

func New(cfg *config.Config, a agent.Agent) *Server {
	tg := telegram.NewClient(cfg.TelegramBotToken)

	s := &Server{
		cfg:   cfg,
		mux:   http.NewServeMux(),
		tg:    tg,
		dedup: telegram.NewDedup(5 * time.Minute),
	}

	if cfg.OpenAIAPIKey != "" {
		s.voice = voice.NewHandler(tg, cfg.OpenAIAPIKey)
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

	s.agent = agent.NewQueuedAgent(a, func(chatID int64, response string, err error) {
		if err != nil {
			log.Printf("agent: error: %v", err)
			response = fmt.Sprintf("error: %v", err)
		}

		if s.emailSender != nil {
			clean, actions, parseErr := emaillevelup.ExtractActions(response)
			if parseErr != nil {
				log.Printf("email action parse failed: %v", parseErr)
				response = response + "\n\n(email action parse failed)"
			} else {
				response = clean
				if len(actions) > 0 {
					if sendErr := emaillevelup.ExecuteActions(context.Background(), s.emailSender, actions); sendErr != nil {
						log.Printf("email send failed: %v", sendErr)
						response = strings.TrimSpace(response + "\n\nemail send failed: " + sendErr.Error())
					} else {
						response = strings.TrimSpace(response + "\n\nemail action executed ✅")
					}
				}
			}
		}

		if strings.TrimSpace(response) == "" {
			response = "ok"
		}
		if sendErr := tg.SendMessage(chatID, response); sendErr != nil {
			log.Printf("agent: send reply failed: %v", sendErr)
		}
	})

	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("POST /webhook", s.handleWebhook)
	return s
}

func (s *Server) ListenAndServe() error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	log.Printf("visor listening on %s", addr)

	if s.emailPoller != nil {
		go s.emailPoller.Start(context.Background())
		log.Printf("email poller started")
	}

	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}

	if s.cfg.TelegramWebhookSecret != "" {
		sig := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
		if !verifySignature(sig, s.cfg.TelegramWebhookSecret) {
			log.Printf("webhook: invalid signature")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	var update telegram.Update
	if err := json.Unmarshal(body, &update); err != nil {
		log.Printf("webhook: bad json: %v", err)
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	if s.dedup.IsDuplicate(update.UpdateID) {
		w.WriteHeader(http.StatusOK)
		return
	}

	msg := update.Message
	if msg == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	chatID := strconv.FormatInt(msg.Chat.ID, 10)
	if chatID != s.cfg.UserChatID {
		log.Printf("webhook: dropping message from unauthorized chat %s", chatID)
		w.WriteHeader(http.StatusOK)
		return
	}

	var content string
	var msgType string
	switch {
	case msg.Voice != nil:
		msgType = "voice"
		if s.voice != nil {
			text, err := s.voice.Transcribe(msg.Voice.FileID)
			if err != nil {
				log.Printf("webhook: voice transcription failed: %v", err)
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
		log.Printf("webhook: unsupported message type from chat %s", chatID)
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("webhook: %s message from %s: %s", msgType, chatID, truncate(content, 80))

	s.agent.Enqueue(context.Background(), agent.Message{
		ChatID:  msg.Chat.ID,
		Content: content,
		Type:    msgType,
	})

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

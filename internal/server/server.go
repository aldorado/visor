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
	"time"

	"visor/internal/agent"
	"visor/internal/config"
	"visor/internal/platform/telegram"
)

type Server struct {
	cfg   *config.Config
	mux   *http.ServeMux
	tg    *telegram.Client
	dedup *telegram.Dedup
	agent *agent.QueuedAgent
}

func New(cfg *config.Config, a agent.Agent) *Server {
	tg := telegram.NewClient(cfg.TelegramBotToken)

	s := &Server{
		cfg:   cfg,
		mux:   http.NewServeMux(),
		tg:    tg,
		dedup: telegram.NewDedup(5 * time.Minute),
	}

	s.agent = agent.NewQueuedAgent(a, func(chatID int64, response string, err error) {
		if err != nil {
			log.Printf("agent: error: %v", err)
			response = fmt.Sprintf("error: %v", err)
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
		content = fmt.Sprintf("[voice:%s]", msg.Voice.FileID)
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

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

package server

import (
	"crypto/hmac"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"visor/internal/config"
	"visor/internal/platform/telegram"
)

type Server struct {
	cfg    *config.Config
	mux    *http.ServeMux
	tg     *telegram.Client
	dedup  *telegram.Dedup
}

func New(cfg *config.Config) *Server {
	s := &Server{
		cfg:   cfg,
		mux:   http.NewServeMux(),
		tg:    telegram.NewClient(cfg.TelegramBotToken),
		dedup: telegram.NewDedup(5 * time.Minute),
	}
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

	// signature verification (if webhook secret is configured)
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

	// dedup
	if s.dedup.IsDuplicate(update.UpdateID) {
		w.WriteHeader(http.StatusOK)
		return
	}

	// only process messages (reactions etc. ignored for now)
	msg := update.Message
	if msg == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	// auth: only accept messages from the configured user
	chatID := strconv.FormatInt(msg.Chat.ID, 10)
	if chatID != s.cfg.UserChatID {
		log.Printf("webhook: dropping message from unauthorized chat %s", chatID)
		w.WriteHeader(http.StatusOK)
		return
	}

	// determine message type and content
	var content string
	var msgType string
	switch {
	case msg.Voice != nil:
		msgType = "voice"
		content = fmt.Sprintf("[voice:%s]", msg.Voice.FileID)
	case len(msg.Photo) > 0:
		msgType = "photo"
		best := msg.Photo[len(msg.Photo)-1] // largest resolution
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

	// echo mode — will be replaced by agent process manager in M2
	var reply string
	switch msgType {
	case "voice":
		reply = fmt.Sprintf("🎤 got your voice message (%ds)", msg.Voice.Duration)
	case "photo":
		reply = "📷 got your photo"
		if msg.Caption != "" {
			reply += fmt.Sprintf(": %s", msg.Caption)
		}
	default:
		reply = content
	}

	if err := s.tg.SendMessage(msg.Chat.ID, reply); err != nil {
		log.Printf("webhook: send reply failed: %v", err)
	}

	w.WriteHeader(http.StatusOK)
}

func verifySignature(got, secret string) bool {
	// Telegram sends the secret token as-is in the header, not HMAC.
	// We just do a constant-time compare.
	return hmac.Equal([]byte(got), []byte(secret))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

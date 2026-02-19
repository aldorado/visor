package voice

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"visor/internal/platform/telegram"
)

// Handler manages voice message processing: download from Telegram, transcribe via Whisper.
type Handler struct {
	tg      *telegram.Client
	whisper *WhisperClient
}

func NewHandler(tg *telegram.Client, openAIKey string) *Handler {
	return &Handler{
		tg:      tg,
		whisper: NewWhisperClient(openAIKey),
	}
}

// Transcribe downloads a voice message from Telegram and returns the transcribed text.
func (h *Handler) Transcribe(fileID string) (string, error) {
	fileURL, err := h.tg.GetFileURL(fileID)
	if err != nil {
		return "", fmt.Errorf("voice: get file URL: %w", err)
	}

	resp, err := http.Get(fileURL)
	if err != nil {
		return "", fmt.Errorf("voice: download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("voice: download status %d: %s", resp.StatusCode, body)
	}

	text, err := h.whisper.Transcribe(resp.Body, "voice.ogg")
	if err != nil {
		return "", fmt.Errorf("voice: transcribe: %w", err)
	}

	log.Printf("voice: transcribed %d chars from %s", len(text), fileID)
	return text, nil
}

package voice

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"

	"visor/internal/platform/telegram"
)

// Handler manages voice message processing: download from Telegram, transcribe via Whisper, synthesize via ElevenLabs.
type Handler struct {
	tg      *telegram.Client
	whisper *WhisperClient
	tts     *ElevenLabsClient
}

func NewHandler(tg *telegram.Client, openAIKey string) *Handler {
	return &Handler{
		tg:      tg,
		whisper: NewWhisperClient(openAIKey),
	}
}

func (h *Handler) SetTTS(apiKey, voiceID string) {
	h.tts = NewElevenLabsClient(apiKey, voiceID)
}

func (h *Handler) TTSEnabled() bool {
	return h.tts != nil
}

// SynthesizeAndSend converts text to speech and sends it as a voice message.
func (h *Handler) SynthesizeAndSend(chatID int64, text string) error {
	if h.tts == nil {
		return fmt.Errorf("voice: TTS not configured")
	}

	audio, err := h.tts.Synthesize(text)
	if err != nil {
		return fmt.Errorf("voice: synthesize: %w", err)
	}

	log.Printf("voice: synthesized %d bytes for chat %d", len(audio), chatID)

	if err := h.tg.SendVoice(chatID, bytes.NewReader(audio), "voice.mp3"); err != nil {
		return fmt.Errorf("voice: send voice: %w", err)
	}
	return nil
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

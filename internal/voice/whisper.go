package voice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

const whisperURL = "https://api.openai.com/v1/audio/transcriptions"

type WhisperClient struct {
	apiKey     string
	httpClient *http.Client
}

func NewWhisperClient(apiKey string) *WhisperClient {
	return &WhisperClient{
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

// Transcribe sends audio data to OpenAI Whisper and returns the transcribed text.
func (c *WhisperClient) Transcribe(audio io.Reader, filename string) (string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if err := w.WriteField("model", "whisper-1"); err != nil {
		return "", fmt.Errorf("whisper: write model field: %w", err)
	}

	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("whisper: create form file: %w", err)
	}
	if _, err := io.Copy(part, audio); err != nil {
		return "", fmt.Errorf("whisper: copy audio: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("whisper: close multipart: %w", err)
	}

	req, err := http.NewRequest("POST", whisperURL, &buf)
	if err != nil {
		return "", fmt.Errorf("whisper: create request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("whisper: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("whisper: status %d: %s", resp.StatusCode, body)
	}

	var result whisperResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("whisper: decode: %w", err)
	}
	return result.Text, nil
}

type whisperResponse struct {
	Text string `json:"text"`
}

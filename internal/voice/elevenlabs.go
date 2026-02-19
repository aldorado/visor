package voice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const elevenLabsBaseURL = "https://api.elevenlabs.io/v1"

type ElevenLabsClient struct {
	apiKey     string
	voiceID    string
	httpClient *http.Client
}

func NewElevenLabsClient(apiKey, voiceID string) *ElevenLabsClient {
	return &ElevenLabsClient{
		apiKey:     apiKey,
		voiceID:    voiceID,
		httpClient: &http.Client{},
	}
}

// Synthesize converts text to speech and returns the audio data as MP3 bytes.
func (c *ElevenLabsClient) Synthesize(text string) ([]byte, error) {
	return synthesizeWithURL(c, fmt.Sprintf("%s/text-to-speech/%s", elevenLabsBaseURL, c.voiceID), text)
}

func synthesizeWithURL(c *ElevenLabsClient, url string, text string) ([]byte, error) {

	payload := map[string]any{
		"text":     text,
		"model_id": "eleven_multilingual_v2",
		"voice_settings": map[string]any{
			"stability":        0.5,
			"similarity_boost": 0.75,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("elevenlabs: marshal: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("elevenlabs: request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("xi-api-key", c.apiKey)
	req.Header.Set("Accept", "audio/mpeg")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("elevenlabs: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("elevenlabs: status %d: %s", resp.StatusCode, respBody)
	}

	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("elevenlabs: read audio: %w", err)
	}
	return audio, nil
}

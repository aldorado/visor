package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const apiBase = "https://api.telegram.org/bot"

type Client struct {
	token      string
	httpClient *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		token:      token,
		httpClient: &http.Client{},
	}
}

func (c *Client) SendMessage(chatID int64, text string) error {
	return c.sendJSON("sendMessage", map[string]any{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	})
}

func (c *Client) SendVoice(chatID int64, audio io.Reader, filename string) error {
	// TODO: M4 — multipart upload for voice
	return fmt.Errorf("voice send not implemented yet")
}

func (c *Client) GetFileURL(fileID string) (string, error) {
	url := fmt.Sprintf("%s%s/getFile?file_id=%s", apiBase, c.token, fileID)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("getFile request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			FilePath string `json:"file_path"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("getFile decode: %w", err)
	}
	if !result.OK {
		return "", fmt.Errorf("getFile: API returned ok=false")
	}
	return fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", c.token, result.Result.FilePath), nil
}

func (c *Client) sendJSON(method string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", method, err)
	}

	url := fmt.Sprintf("%s%s/%s", apiBase, c.token, method)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%s request: %w", method, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s: status %d: %s", method, resp.StatusCode, respBody)
	}
	return nil
}

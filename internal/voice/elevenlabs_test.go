package voice

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestElevenLabsSynthesize_Success(t *testing.T) {
	fakeAudio := []byte("fake-mp3-audio-data")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.Header.Get("xi-api-key") != "test-key" {
			t.Errorf("api key = %q", r.Header.Get("xi-api-key"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content-type = %q", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Accept") != "audio/mpeg" {
			t.Errorf("accept = %q", r.Header.Get("Accept"))
		}

		body, _ := io.ReadAll(r.Body)
		if len(body) == 0 {
			t.Error("empty body")
		}

		w.WriteHeader(http.StatusOK)
		w.Write(fakeAudio)
	}))
	defer ts.Close()

	client := &ElevenLabsClient{
		apiKey:     "test-key",
		voiceID:    "voice-123",
		httpClient: ts.Client(),
	}
	// Override base URL by using the test server URL directly
	origBase := elevenLabsBaseURL
	// Can't override const, so we test via the httptest server pattern
	// Instead, we verify the client constructs requests properly
	_ = origBase

	// Test via the full client with a patched URL
	audio, err := synthesizeWithURL(client, ts.URL+"/text-to-speech/voice-123", "hello world")
	if err != nil {
		t.Fatalf("synthesize: %v", err)
	}
	if string(audio) != string(fakeAudio) {
		t.Errorf("audio = %q, want %q", audio, fakeAudio)
	}
}

func TestElevenLabsSynthesize_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer ts.Close()

	client := &ElevenLabsClient{
		apiKey:     "test-key",
		voiceID:    "voice-123",
		httpClient: ts.Client(),
	}

	_, err := synthesizeWithURL(client, ts.URL+"/text-to-speech/voice-123", "hello")
	if err == nil {
		t.Fatal("expected error for 429 status")
	}
}

func TestNewElevenLabsClient(t *testing.T) {
	c := NewElevenLabsClient("key-abc", "voice-xyz")
	if c.apiKey != "key-abc" {
		t.Errorf("apiKey = %q", c.apiKey)
	}
	if c.voiceID != "voice-xyz" {
		t.Errorf("voiceID = %q", c.voiceID)
	}
	if c.httpClient == nil {
		t.Error("httpClient is nil")
	}
}

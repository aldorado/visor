package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// forgejoEventHeader is the header Forgejo sends to identify the event type.
// Gitea uses the same header name for API compatibility.
const forgejoEventHeader = "X-Forgejo-Event"

type forgejoPushPayload struct {
	Ref    string `json:"ref"`
	Before string `json:"before"`
	After  string `json:"after"`
	Commits []struct {
		Message string `json:"message"`
		ID      string `json:"id"`
	} `json:"commits"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	Pusher struct {
		Login string `json:"login"`
	} `json:"pusher"`
}

type forgejoPRPayload struct {
	Action      string `json:"action"`
	Number      int    `json:"number"`
	PullRequest struct {
		Title  string `json:"title"`
		HTMLURL string `json:"html_url"`
	} `json:"pull_request"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	Sender struct {
		Login string `json:"login"`
	} `json:"sender"`
}

func (s *Server) handleForgejoWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.log.Warn(ctx, "forgejo webhook: read body failed", "error", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	event := r.Header.Get(forgejoEventHeader)
	if event == "" {
		event = r.Header.Get("X-Gitea-Event") // fallback for compatibility
	}

	s.log.Info(ctx, "forgejo webhook received", "event", event)

	chatID := mustParseChatID(s.cfg.UserChatID)
	var msg string

	switch event {
	case "push":
		var p forgejoPushPayload
		if err := json.Unmarshal(body, &p); err != nil {
			s.log.Warn(ctx, "forgejo webhook: parse push payload failed", "error", err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		branch := strings.TrimPrefix(p.Ref, "refs/heads/")
		n := len(p.Commits)
		firstMsg := ""
		if n > 0 {
			firstMsg = " — " + truncate(p.Commits[0].Message, 60)
		}
		msg = fmt.Sprintf("[forgejo] push to *%s* by %s\nbranch: %s · %d commit(s)%s",
			p.Repository.FullName, p.Pusher.Login, branch, n, firstMsg)

	case "pull_request":
		var p forgejoPRPayload
		if err := json.Unmarshal(body, &p); err != nil {
			s.log.Warn(ctx, "forgejo webhook: parse PR payload failed", "error", err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		msg = fmt.Sprintf("[forgejo] PR #%d *%s* — %s by %s\n%s",
			p.Number, p.Action, p.PullRequest.Title, p.Sender.Login, p.PullRequest.HTMLURL)

	default:
		// unknown event — log and acknowledge
		s.log.Debug(ctx, "forgejo webhook: unhandled event", "event", event)
		w.WriteHeader(http.StatusOK)
		return
	}

	if msg != "" && s.cfg.UserChatID != "" {
		if err := s.tg.SendMessage(chatID, msg); err != nil {
			s.log.Error(ctx, "forgejo webhook: send notification failed", "error", err.Error())
		}
	}

	w.WriteHeader(http.StatusOK)
}

package email

import (
	"context"
	"time"
)

type IncomingMessage struct {
	ID      string
	From    string
	Subject string
	Body    string
	Date    time.Time
}

type OutgoingMessage struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type Fetcher interface {
	Fetch(ctx context.Context) ([]IncomingMessage, error)
}

type Sender interface {
	Send(ctx context.Context, msg OutgoingMessage) error
}

package email

import (
	"context"
	"fmt"
	"sync"
	"time"

	"visor/internal/observability"
)

type Poller struct {
	fetcher  Fetcher
	interval time.Duration
	onEmail  func(IncomingMessage)

	mu   sync.Mutex
	seen map[string]struct{}
	log  *observability.Logger
}

func NewPoller(fetcher Fetcher, interval time.Duration, onEmail func(IncomingMessage)) *Poller {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &Poller{
		fetcher:  fetcher,
		interval: interval,
		onEmail:  onEmail,
		seen:     map[string]struct{}{},
		log:      observability.Component("levelup.email.poller"),
	}
}

func (p *Poller) Start(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	p.log.Info(ctx, "email poller started", "interval_seconds", p.interval.Seconds())
	for {
		if err := p.Tick(ctx); err != nil {
			p.log.Error(ctx, "email poller tick failed", "error", err.Error())
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (p *Poller) Tick(ctx context.Context) error {
	messages, err := p.fetcher.Fetch(ctx)
	if err != nil {
		return fmt.Errorf("fetch mail: %w", err)
	}

	newCount := 0
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, msg := range messages {
		if msg.ID == "" {
			continue
		}
		if _, ok := p.seen[msg.ID]; ok {
			continue
		}
		p.seen[msg.ID] = struct{}{}
		newCount++
		if p.onEmail != nil {
			p.onEmail(msg)
		}
	}
	p.log.Debug(ctx, "email poller tick done", "fetched", len(messages), "new", newCount)
	return nil
}

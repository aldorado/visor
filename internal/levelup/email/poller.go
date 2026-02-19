package email

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type Poller struct {
	fetcher  Fetcher
	interval time.Duration
	onEmail  func(IncomingMessage)

	mu   sync.Mutex
	seen map[string]struct{}
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
	}
}

func (p *Poller) Start(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		if err := p.Tick(ctx); err != nil {
			log.Printf("email poller: %v", err)
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
		if p.onEmail != nil {
			p.onEmail(msg)
		}
	}
	return nil
}

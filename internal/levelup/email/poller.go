package email

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"visor/internal/observability"
)

type Poller struct {
	fetcher        Fetcher
	interval       time.Duration
	onEmail        func(IncomingMessage)
	allowedSenders map[string]struct{}

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

// SetAllowedSenders configures sender allowlist filtering.
// Only messages from these addresses will be delivered. Empty = no filter.
func (p *Poller) SetAllowedSenders(addrs []string) {
	if len(addrs) == 0 {
		return
	}
	p.allowedSenders = make(map[string]struct{}, len(addrs))
	for _, a := range addrs {
		p.allowedSenders[strings.ToLower(strings.TrimSpace(a))] = struct{}{}
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

func (p *Poller) senderAllowed(from string) bool {
	if len(p.allowedSenders) == 0 {
		return true
	}
	addr := strings.ToLower(strings.TrimSpace(from))
	// handle "Name <email>" format
	if i := strings.LastIndex(addr, "<"); i != -1 {
		if j := strings.LastIndex(addr, ">"); j > i {
			addr = addr[i+1 : j]
		}
	}
	_, ok := p.allowedSenders[addr]
	return ok
}

func (p *Poller) Tick(ctx context.Context) error {
	messages, err := p.fetcher.Fetch(ctx)
	if err != nil {
		return fmt.Errorf("fetch mail: %w", err)
	}

	newCount := 0
	filteredCount := 0
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
		if !p.senderAllowed(msg.From) {
			filteredCount++
			p.log.Info(ctx, "email sender not in allowlist, dropped", "from", msg.From, "subject", msg.Subject)
			continue
		}
		newCount++
		if p.onEmail != nil {
			p.onEmail(msg)
		}
	}
	p.log.Debug(ctx, "email poller tick done", "fetched", len(messages), "new", newCount, "filtered", filteredCount)
	return nil
}

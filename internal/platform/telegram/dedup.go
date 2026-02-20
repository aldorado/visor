package telegram

import (
	"sync"
	"time"
)

type Dedup struct {
	mu   sync.Mutex
	seen map[int]time.Time
	ttl  time.Duration
}

func NewDedup(ttl time.Duration) *Dedup {
	d := &Dedup{
		seen: make(map[int]time.Time),
		ttl:  ttl,
	}
	go d.cleanup()
	return d
}

// IsDuplicate returns true if this update_id was seen recently.
func (d *Dedup) IsDuplicate(updateID int) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.seen[updateID]; ok {
		return true
	}
	d.seen[updateID] = time.Now()
	return false
}

func (d *Dedup) cleanup() {
	ticker := time.NewTicker(d.ttl)
	defer ticker.Stop()
	for range ticker.C {
		d.mu.Lock()
		cutoff := time.Now().Add(-d.ttl)
		for id, t := range d.seen {
			if t.Before(cutoff) {
				delete(d.seen, id)
			}
		}
		d.mu.Unlock()
	}
}

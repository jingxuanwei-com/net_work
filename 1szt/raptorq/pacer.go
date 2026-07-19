package raptorq

import (
	"context"
	"sync"
	"time"
)

// PacedSender is a token bucket rate limiter for UDP sends.
type PacedSender struct {
	bps       uint64 // bits per second
	lastSend  time.Time
	allowance float64 // token balance in bytes
	maxBurst  float64 // max token accumulation in bytes
	mu        sync.Mutex
}

// NewPacedSender creates a rate limiter with the given bps.
func NewPacedSender(bps uint64) *PacedSender {
	burstBytes := float64(bps) / 8 // 1 second burst
	if burstBytes < 65536 {
		burstBytes = 65536 // at least 64KB burst
	}
	return &PacedSender{
		bps:       bps,
		lastSend:  time.Now(),
		maxBurst:  burstBytes,
		allowance: burstBytes, // start with full burst
	}
}

// Wait blocks until the rate limiter allows sending the given number of bytes.
// Returns nil when ready, or ctx.Err() if context is cancelled.
func (p *PacedSender) Wait(ctx context.Context, bytes int) error {
	p.mu.Lock()

	now := time.Now()
	elapsed := now.Sub(p.lastSend).Seconds()
	p.lastSend = now

	// Accumulate tokens at bps/8 bytes per second
	p.allowance += elapsed * float64(p.bps) / 8
	if p.allowance > p.maxBurst {
		p.allowance = p.maxBurst
	}

	p.allowance -= float64(bytes)

	if p.allowance >= 0 {
		p.mu.Unlock()
		return nil
	}

	// Need to wait: calculate sleep time
	needBytes := -p.allowance
	p.mu.Unlock()

	sleepDuration := time.Duration(needBytes / (float64(p.bps) / 8) * float64(time.Second))
	if sleepDuration > time.Second {
		sleepDuration = time.Second // cap at 1 second
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(sleepDuration):
		// Deduct from allowance after sleep (use remaining time)
		p.mu.Lock()
		now := time.Now()
		elapsedExtra := now.Sub(p.lastSend).Seconds()
		p.lastSend = now
		p.allowance += elapsedExtra * float64(p.bps) / 8
		if p.allowance > p.maxBurst {
			p.allowance = p.maxBurst
		}
		p.allowance -= float64(bytes)
		p.mu.Unlock()
		return nil
	}
}

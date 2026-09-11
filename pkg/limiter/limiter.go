package limiter

import (
	"context"
	"sync"
	"time"
)

type Limiter struct {
	interval time.Duration
	mu       sync.Mutex
	next     time.Time
}

func New(requestsPerMinute int) *Limiter {
	if requestsPerMinute <= 0 {
		return &Limiter{}
	}

	return &Limiter{interval: time.Minute / time.Duration(requestsPerMinute)}
}

// TODO: implement bucket
func (limiter *Limiter) Wait(ctx context.Context) error {
	if limiter == nil || limiter.interval <= 0 {
		return nil
	}

	limiter.mu.Lock()
	now := time.Now()
	if limiter.next.Before(now) {
		limiter.next = now
	}
	wait := time.Until(limiter.next)
	limiter.next = limiter.next.Add(limiter.interval)
	limiter.mu.Unlock()

	if wait <= 0 {
		return nil
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

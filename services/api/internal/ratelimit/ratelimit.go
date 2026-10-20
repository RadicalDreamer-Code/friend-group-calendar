// Package ratelimit provides an in-memory sliding-window limiter.
//
// State lives in the API process, which is enough for the single instance
// this app runs as; limits reset when the API restarts.
package ratelimit

import (
	"sync"
	"time"
)

type Limiter struct {
	limit  int
	window time.Duration
	now    func() time.Time

	mu    sync.Mutex
	hits  map[string][]time.Time
	calls int
}

// New allows `limit` events per key within any `window`.
func New(limit int, window time.Duration) *Limiter {
	return &Limiter{limit: limit, window: window, now: time.Now, hits: map[string][]time.Time{}}
}

// Allow records an event for key and reports whether it is within the limit.
// When it is not, retryAfter says when the oldest event leaves the window.
func (l *Limiter) Allow(key string) (ok bool, retryAfter time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	cutoff := now.Add(-l.window)

	l.calls++
	if l.calls%1024 == 0 {
		l.sweep(cutoff)
	}

	recent := prune(l.hits[key], cutoff)
	if len(recent) >= l.limit {
		l.hits[key] = recent
		return false, recent[0].Add(l.window).Sub(now)
	}
	l.hits[key] = append(recent, now)
	return true, 0
}

// sweep drops keys with no events in the window so memory stays bounded.
func (l *Limiter) sweep(cutoff time.Time) {
	for k, v := range l.hits {
		if len(prune(v, cutoff)) == 0 {
			delete(l.hits, k)
		}
	}
}

// prune returns the events after cutoff; events are kept in time order.
func prune(events []time.Time, cutoff time.Time) []time.Time {
	i := 0
	for i < len(events) && !events[i].After(cutoff) {
		i++
	}
	return events[i:]
}

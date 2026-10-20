// Package ratelimit provides an in-memory sliding-window limiter.
//
// State lives in the API process, which is enough for the single instance
// this app runs as; limits reset when the API restarts.
package ratelimit

import (
	"cmp"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

type Limiter struct {
	// id orders limiters for locking in AllowAll.
	id     uint64
	limit  int
	window time.Duration
	now    func() time.Time

	mu    sync.Mutex
	hits  map[string][]time.Time
	calls int
}

var lastID atomic.Uint64

// New allows `limit` events per key within any `window`.
func New(limit int, window time.Duration) *Limiter {
	return &Limiter{id: lastID.Add(1), limit: limit, window: window, now: time.Now, hits: map[string][]time.Time{}}
}

// Allow records an event for key and reports whether it is within the limit.
// When it is not, retryAfter says when the oldest event leaves the window.
func (l *Limiter) Allow(key string) (ok bool, retryAfter time.Duration) {
	return AllowAll(Check{l, key})
}

// Check is one limiter and the key an event counts under.
type Check struct {
	Limiter *Limiter
	Key     string
}

// AllowAll records the event under every check only if all of them are within
// their limits, so a rejected event uses up no budget anywhere. retryAfter is
// the longest wait among the checks that failed. It is atomic: concurrent
// calls cannot both pass a limit that has room for only one of them.
func AllowAll(checks ...Check) (ok bool, retryAfter time.Duration) {
	// Lock every limiter involved, always in id order so calls that share
	// limiters cannot deadlock.
	var limiters []*Limiter
	for _, c := range checks {
		if !slices.Contains(limiters, c.Limiter) {
			limiters = append(limiters, c.Limiter)
		}
	}
	slices.SortFunc(limiters, func(a, b *Limiter) int { return cmp.Compare(a.id, b.id) })
	for _, l := range limiters {
		l.mu.Lock()
		defer l.mu.Unlock()
	}

	ok = true
	for _, c := range checks {
		if wait := c.Limiter.wait(c.Key); wait > 0 {
			ok, retryAfter = false, max(retryAfter, wait)
		}
	}
	if ok {
		for _, c := range checks {
			c.Limiter.record(c.Key)
		}
	}
	return ok, retryAfter
}

// wait reports how long until key has room for another event; 0 means now.
// l.mu must be held.
func (l *Limiter) wait(key string) time.Duration {
	now := l.now()
	recent := prune(l.hits[key], now.Add(-l.window))
	l.hits[key] = recent
	if len(recent) < l.limit {
		return 0
	}
	return recent[0].Add(l.window).Sub(now)
}

// record counts an event for key. l.mu must be held.
func (l *Limiter) record(key string) {
	now := l.now()
	l.calls++
	if l.calls%1024 == 0 {
		l.sweep(now.Add(-l.window))
	}
	l.hits[key] = append(l.hits[key], now)
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

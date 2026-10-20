package ratelimit

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLimiterSlidingWindow(t *testing.T) {
	now := time.Unix(0, 0)
	l := New(2, time.Minute)
	l.now = func() time.Time { return now }

	for i := range 2 {
		if ok, _ := l.Allow("a"); !ok {
			t.Fatalf("event %d should be allowed", i)
		}
	}
	ok, wait := l.Allow("a")
	if ok || wait != time.Minute {
		t.Fatalf("third event: ok=%v wait=%v, want blocked for 1m", ok, wait)
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("other keys have their own budget")
	}

	now = now.Add(time.Minute + time.Second)
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("events older than the window no longer count")
	}
}

func TestAllowAllRecordsNothingWhenOneCheckFails(t *testing.T) {
	now := time.Unix(0, 0)
	ip, email := New(2, time.Hour), New(1, time.Minute)
	ip.now = func() time.Time { return now }
	email.now = ip.now

	if ok, _ := AllowAll(Check{ip, "1.2.3.4"}, Check{email, "ada"}); !ok {
		t.Fatal("first event should be allowed")
	}
	ok, wait := AllowAll(Check{ip, "1.2.3.4"}, Check{email, "ada"})
	if ok || wait != time.Minute {
		t.Fatalf("second event: ok=%v wait=%v, want blocked for 1m by the email limit", ok, wait)
	}
	// The rejected event did not use up the IP's second slot.
	if ok, _ := AllowAll(Check{ip, "1.2.3.4"}, Check{email, "bob"}); !ok {
		t.Fatal("a rejected event must not count against the other limits")
	}
}

func TestAllowAllReportsTheLongestWait(t *testing.T) {
	now := time.Unix(0, 0)
	short, long := New(1, time.Minute), New(1, time.Hour)
	short.now = func() time.Time { return now }
	long.now = short.now
	AllowAll(Check{short, "k"}, Check{long, "k"})
	if ok, wait := AllowAll(Check{short, "k"}, Check{long, "k"}); ok || wait != time.Hour {
		t.Fatalf("ok=%v wait=%v, want blocked for 1h", ok, wait)
	}
}

func TestAllowAllIsAtomic(t *testing.T) {
	a, b := New(1, time.Hour), New(1, time.Hour)
	var (
		wg      sync.WaitGroup
		allowed atomic.Int32
	)
	for i := range 64 {
		wg.Go(func() {
			// Alternate the order to check that shared limiters cannot deadlock.
			checks := []Check{{a, "k"}, {b, "k"}}
			if i%2 == 1 {
				checks[0], checks[1] = checks[1], checks[0]
			}
			if ok, _ := AllowAll(checks...); ok {
				allowed.Add(1)
			}
		})
	}
	wg.Wait()
	if n := allowed.Load(); n != 1 {
		t.Fatalf("%d concurrent events were allowed, want 1", n)
	}
}

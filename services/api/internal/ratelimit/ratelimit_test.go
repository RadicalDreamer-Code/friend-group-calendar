package ratelimit

import (
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

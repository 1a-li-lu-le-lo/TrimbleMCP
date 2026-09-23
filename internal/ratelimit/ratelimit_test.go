package ratelimit

import (
	"testing"
	"time"
)

func TestKeyedBuckets(t *testing.T) {
	now := time.Unix(0, 0)
	k := NewKeyed(1, 2)
	k.SetClock(func() time.Time { return now })
	for i := 0; i < 2; i++ {
		if ok, _ := k.Allow("a"); !ok {
			t.Fatal("burst denied")
		}
	}
	ok, wait := k.Allow("a")
	if ok || wait <= 0 || wait > time.Second {
		t.Fatalf("ok=%v wait=%v", ok, wait)
	}
	if ok, _ := k.Allow("b"); !ok {
		t.Fatal("keys must be independent")
	}
	now = now.Add(time.Second)
	if ok, _ := k.Allow("a"); !ok {
		t.Fatal("refill failed")
	}
}

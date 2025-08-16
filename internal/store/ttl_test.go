package store_test

import (
	"testing"
	"time"

	"github.com/amir-aharon/goliath/internal/store"
)

type fakeClock struct{ t time.Time }

func newFakeClock(start time.Time) *fakeClock { return &fakeClock{t: start} }
func (f *fakeClock) Now() time.Time           { return f.t }
func (f *fakeClock) Advance(d time.Duration)  { f.t = f.t.Add(d) }

func TestTTL_MissingKey(t *testing.T) {
	fc := newFakeClock(time.Unix(1_700_000_000, 0))
	mem := store.NewMemoryWithClock(fc)

	secs, exists, hasExp := mem.TTL("nope")
	if exists || hasExp || secs != 0 {
		t.Fatalf("TTL missing: got (secs=%d, exists=%v, hasExp=%v), want (0,false,false)", secs, exists, hasExp)
	}
}

func TestTTL_NoExpiry(t *testing.T) {
	fc := newFakeClock(time.Unix(1_700_000_000, 0))
	mem := store.NewMemoryWithClock(fc)

	mem.Set("a", "1")

	secs, exists, hasExp := mem.TTL("a")
	if !exists || hasExp || secs != 0 {
		t.Fatalf("TTL no-exp: got (secs=%d, exists=%v, hasExp=%v), want (0,true,false)", secs, exists, hasExp)
	}
}

func TestSetEx_TTL_DecreasesAndExpires(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	fc := newFakeClock(start)
	mem := store.NewMemoryWithClock(fc)

	mem.SetEx("k", "v", 5*time.Second)

	if secs, exists, hasExp := mem.TTL("k"); !exists || !hasExp || secs != 5 {
		t.Fatalf("after setex: got (secs=%d, exists=%v, hasExp=%v), want (5,true,true)", secs, exists, hasExp)
	}

	fc.Advance(3 * time.Second)
	if secs, _, _ := mem.TTL("k"); secs != 2 {
		t.Fatalf("after +3s: got secs=%d, want 2", secs)
	}

	fc.Advance(3 * time.Second) // total +6s
	if _, ok := mem.Get("k"); ok {
		t.Fatalf("expected Get after expiry to return not found")
	}

	if secs, exists, hasExp := mem.TTL("k"); exists || hasExp || secs != 0 {
		t.Fatalf("after expiry TTL: got (secs=%d, exists=%v, hasExp=%v), want (0,false,false)", secs, exists, hasExp)
	}
}

func TestPersist_RemovesExpiry(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	fc := newFakeClock(start)
	mem := store.NewMemoryWithClock(fc)

	mem.SetEx("p", "v", 5*time.Second)

	if n := mem.Persist("p"); !n {
		t.Fatalf("Persist returned false; expected true for key with expiry")
	}

	// Advance past original expiry; key should still exist
	fc.Advance(10 * time.Second)

	if v, ok := mem.Get("p"); !ok || v != "v" {
		t.Fatalf("after persist+advance: got (%q,%v), want (\"v\",true)", v, ok)
	}

	// TTL should say "exists, no expiry"
	if secs, exists, hasExp := mem.TTL("p"); !exists || hasExp || secs != 0 {
		t.Fatalf("persist TTL: got (secs=%d, exists=%v, hasExp=%v), want (0,true,false)", secs, exists, hasExp)
	}
}

func TestPersist_OnExpiredKey(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	fc := newFakeClock(start)
	mem := store.NewMemoryWithClock(fc)

	mem.SetEx("gone", "v", 2*time.Second)

	fc.Advance(3 * time.Second)

	if ok := mem.Persist("gone"); ok {
		t.Fatalf("Persist on expired key: got true, want false")
	}

	if _, exists := mem.Get("gone"); exists {
		t.Fatalf("expected expired key to be removed on access")
	}

	if secs, exists, hasExp := mem.TTL("gone"); exists || hasExp || secs != 0 {
		t.Fatalf("TTL after expiry: got (secs=%d, exists=%v, hasExp=%v), want (0,false,false)", secs, exists, hasExp)
	}
}

package command_test

import (
	"testing"
	"time"
)

func TestSETEX_NonIntegerSeconds(t *testing.T) {
	d := newDispatcher()
	got, err := run(d, "SETEX", "k", "abc", "v")
	if err != nil {
		t.Fatalf("dispatch error: %v", err)
	}
	want := "-ERR seconds must be a positive integer\r\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSETEX_NonPositiveSeconds_Zero(t *testing.T) {
	d := newDispatcher()
	got, err := run(d, "SETEX", "k", "0", "v")
	if err != nil {
		t.Fatalf("dispatch error: %v", err)
	}
	want := "-ERR seconds must be a positive integer\r\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSETEX_NonPositiveSeconds_Negative(t *testing.T) {
	d := newDispatcher()
	got, err := run(d, "SETEX", "k", "-3", "v")
	if err != nil {
		t.Fatalf("dispatch error: %v", err)
	}
	want := "-ERR seconds must be a positive integer\r\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSETEX_ValidSeconds_ReturnsOK(t *testing.T) {
	d := newDispatcher()
	got, err := run(d, "SETEX", "k", "5", "v")
	if err != nil {
		t.Fatalf("dispatch error: %v", err)
	}
	if got != "+OK\r\n" {
		t.Fatalf("got %q, want %q", got, "+OK\r\n")
	}
}

func TestTTL_MissingKey_ReturnsMinus2(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	fc := newFakeClock(start)
	d := newDispatcherWithClock(fc)

	got, err := run(d, "TTL", "nope")
	if err != nil {
		t.Fatalf("dispatch error: %v", err)
	}
	if got != "-2\r\n" {
		t.Fatalf("got %q, want %q", got, "-2\r\n")
	}
}

func TestTTL_NoExpiry_ReturnsMinus1(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	fc := newFakeClock(start)
	d := newDispatcherWithClock(fc)

	if _, err := run(d, "SET", "k", "v"); err != nil {
		t.Fatalf("SET error: %v", err)
	}
	got, err := run(d, "TTL", "k")
	if err != nil {
		t.Fatalf("TTL error: %v", err)
	}
	if got != "-1\r\n" {
		t.Fatalf("got %q, want %q", got, "-1\r\n")
	}
}

func TestTTL_CountsDown_WithExpiry(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	fc := newFakeClock(start)
	d := newDispatcherWithClock(fc)

	if _, err := run(d, "SETEX", "k", "5", "v"); err != nil {
		t.Fatalf("SETEX error: %v", err)
	}

	if got, _ := run(d, "TTL", "k"); got != "5\r\n" {
		t.Fatalf("after setex: got %q, want %q", got, "5\r\n")
	}

	fc.Advance(3 * time.Second)
	if got, _ := run(d, "TTL", "k"); got != "2\r\n" {
		t.Fatalf("after +3s: got %q, want %q", got, "2\r\n")
	}
}

func TestTTL_AfterExpiry_ReturnsMinus2(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	fc := newFakeClock(start)
	d := newDispatcherWithClock(fc)

	if _, err := run(d, "SETEX", "k", "2", "v"); err != nil {
		t.Fatalf("SETEX error: %v", err)
	}

	// Advance beyond expiry
	fc.Advance(3 * time.Second)
	if got, _ := run(d, "TTL", "k"); got != "-2\r\n" {
		t.Fatalf("after expiry: got %q, want %q", got, "-2\r\n")
	}
}

func TestPERSIST_OnKeyWithExpiry_Returns1_ThenTTLMinus1(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	fc := newFakeClock(start)
	d := newDispatcherWithClock(fc)

	if _, err := run(d, "SETEX", "k", "5", "v"); err != nil {
		t.Fatalf("SETEX error: %v", err)
	}

	if got, _ := run(d, "PERSIST", "k"); got != "1\r\n" {
		t.Fatalf("PERSIST: got %q, want %q", got, "1\r\n")
	}

	if got, _ := run(d, "TTL", "k"); got != "-1\r\n" {
		t.Fatalf("TTL after persist: got %q, want %q", got, "-1\r\n")
	}

	fc.Advance(10 * time.Second)
	if got, _ := run(d, "GET", "k"); got != "v\r\n" {
		t.Fatalf("GET after persist+advance: got %q, want %q", got, "v\r\n")
	}
}

func TestPERSIST_OnKeyWithoutExpiry_Returns0(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	fc := newFakeClock(start)
	d := newDispatcherWithClock(fc)

	if _, err := run(d, "SET", "k", "v"); err != nil {
		t.Fatalf("SET error: %v", err)
	}
	if got, _ := run(d, "PERSIST", "k"); got != "0\r\n" {
		t.Fatalf("PERSIST: got %q, want %q", got, "0\r\n")
	}
}

func TestPERSIST_OnMissingOrExpiredKey_Returns0(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	fc := newFakeClock(start)
	d := newDispatcherWithClock(fc)

	if got, _ := run(d, "PERSIST", "nope"); got != "0\r\n" {
		t.Fatalf("PERSIST missing: got %q, want %q", got, "0\r\n")
	}

	if _, err := run(d, "SETEX", "gone", "2", "v"); err != nil {
		t.Fatalf("SETEX error: %v", err)
	}
	fc.Advance(3 * time.Second)
	if got, _ := run(d, "PERSIST", "gone"); got != "0\r\n" {
		t.Fatalf("PERSIST expired: got %q, want %q", got, "0\r\n")
	}
}

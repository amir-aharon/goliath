package command_test

import (
	"strings"
	"testing"
)

func TestSET_ReturnsOK(t *testing.T) {
	r := newRouter()
	got, err := run(r, "SET", "k", "v")
	if err != nil {
		t.Fatalf("dispatch error: %v", err)
	}
	if got != "+OK\r\n" {
		t.Fatalf("got %q, want %q", got, "+OK\r\n")
	}
}

func TestGET_ExistingKey(t *testing.T) {
	r := newRouter()
	if _, err := run(r, "SET", "k", "v"); err != nil {
		t.Fatalf("SET error: %v", err)
	}
	got, err := run(r, "GET", "k")
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}
	if got != "v\r\n" {
		t.Fatalf("got %q, want %q", got, "v\r\n")
	}
}

func TestGET_MissingKey(t *testing.T) {
	r := newRouter()
	got, err := run(r, "GET", "nope")
	if err != nil {
		t.Fatalf("dispatch error: %v", err)
	}
	if got != "-ERR key not found\r\n" {
		t.Fatalf("got %q, want %q", got, "-ERR key not found\r\n")
	}
}

func TestDEL_ExistingKey_RemovesKey(t *testing.T) {
	r := newRouter()
	if _, err := run(r, "SET", "k", "v"); err != nil {
		t.Fatalf("SET error: %v", err)
	}
	got, err := run(r, "DEL", "k")
	if err != nil {
		t.Fatalf("DEL error: %v", err)
	}
	if got != "+OK\r\n" {
		t.Fatalf("got %q, want %q", got, "+OK\r\n")
	}
	// follow-up GET should now error
	if got, _ := run(r, "GET", "k"); got != "-ERR key not found\r\n" {
		t.Fatalf("after DEL, GET got %q, want %q", got, "-ERR key not found\r\n")
	}
}

func TestDEL_MissingKey_ReturnsErr(t *testing.T) {
	r := newRouter()
	got, err := run(r, "DEL", "nope")
	if err != nil {
		t.Fatalf("dispatch error: %v", err)
	}
	if got != "-ERR key not found\r\n" {
		t.Fatalf("got %q, want %q", got, "-ERR key not found\r\n")
	}
}

func TestKV_WrongArity(t *testing.T) {
	r := newRouter()
	cases := []struct {
		name string
		args []string
	}{
		{"GET", nil},                  // too few
		{"GET", []string{"a", "b"}},   // too many
		{"SET", []string{"k"}},        // too few
		{"SET", []string{"k", "v", "x"}}, // too many
		{"DEL", nil},                  // too few
		{"DEL", []string{"a", "b"}},   // too many
	}
	for _, c := range cases {
		got, err := run(r, c.name, c.args...)
		if err != nil {
			t.Fatalf("%s wrong-arity: unexpected error: %v", c.name, err)
		}
		if !strings.HasPrefix(got, "-ERR wrong number of arguments") {
			t.Fatalf("%s wrong-arity: got %q", c.name, got)
		}
	}
}

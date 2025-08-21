package command_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/amir-aharon/goliath/internal/command"
)

func TestPING_WritesPONG(t *testing.T) {
	d := newDispatcher()
	got, err := run(d, "PING")
	if err != nil {
		t.Fatalf("dispatch error: %v", err)
	}
	if got != "+PONG\r\n" {
		t.Fatalf("got %q, want %q", got, "+PONG\r\n")
	}
}

func TestECHO_EchoesArgument(t *testing.T) {
	d := newDispatcher()
	got, err := run(d, "ECHO", "hello")
	if err != nil {
		t.Fatalf("dispatch error: %v", err)
	}
	if got != "hello\r\n" {
		t.Fatalf("got %q, want %q", got, "hello\r\n")
	}
}

func TestQUIT_RespondsOKAndReturnsErrQuit(t *testing.T) {
	d := newDispatcher()
	got, err := run(d, "QUIT")
	if got != "+OK\r\n" {
		t.Fatalf("reply got %q, want %q", got, "+OK\r\n")
	}
	// QUIT handler returns command.ErrQuit; Dispatch should surface it.
	if !errors.Is(err, command.ErrQuit) {
		t.Fatalf("expected ErrQuit, got %v", err)
	}
}

func TestBuiltins_WrongArity(t *testing.T) {
	type tc struct {
		name string
		args []string
	}
	cases := []tc{
		{"PING", []string{"x"}}, // too many
		{"ECHO", []string{}},    // too few
		{"QUIT", []string{"x"}}, // too many
	}

	d := newDispatcher()
	for _, c := range cases {
		got, err := run(d, c.name, c.args...)
		// Wrong-arity never calls the handler, so no ErrQuit and no other errors.
		if err != nil {
			t.Fatalf("%s wrong-arity: unexpected error: %v", c.name, err)
		}
		if !strings.HasPrefix(got, "-ERR ") {
			t.Fatalf("%s wrong-arity: expected reply to start with -ERR, got %q", c.name, got)
		}
	}
}

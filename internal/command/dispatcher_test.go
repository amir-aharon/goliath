package command_test

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/amir-aharon/goliath/internal/command"
)

func TestUnknownCommand(t *testing.T) {
	d := command.NewDispatcher()
	var buf bytes.Buffer
	if err := d.Dispatch(&buf, "NODER", nil); err != nil {
		t.Fatalf("unexpected error from Dispatch: %v", err)
	}

	got := buf.String()
	want := "-ERR unknown command\r\n"
	if got != want {
		t.Errorf("Dispatch wrote %q, want %q", got, want)
	}
}

func TestDispatcherArityErrors(t *testing.T) {
	d := command.NewDispatcher()

	d.Register("ECHO", 1, 1, false, func(w io.Writer, args []string) error {
		_, _ = fmt.Fprint(w, "ok\r\n")
		return nil
	})

	t.Run("too few", func(t *testing.T) {
		var buf bytes.Buffer
		if err := d.Dispatch(&buf, "ECHO", []string{}); err != nil {
			t.Fatalf("unexpected error from Dispatch: %v", err)
		}
		want := "-ERR wrong number of arguments for 'ECHO'\r\n"
		if got := buf.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("too many", func(t *testing.T) {
		var buf bytes.Buffer
		if err := d.Dispatch(&buf, "ECHO", []string{"a", "b"}); err != nil {
			t.Fatalf("unexpected error from Dispatch: %v", err)
		}
		want := "-ERR wrong number of arguments for 'ECHO'\r\n"
		if got := buf.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestDispatcherCaseInsensitive(t *testing.T) {
	d := command.NewDispatcher()
	d.Register("PING", 0, 0, false, func(w io.Writer, _ []string) error {
		_, _ = fmt.Fprint(w, "ran\r\n")
		return nil
	})

	var buf bytes.Buffer
	if err := d.Dispatch(&buf, "ping", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := buf.String(), "ran\r\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDispatcherHappyPath(t *testing.T) {
	d := command.NewDispatcher()
	d.Register("HI", 0, 0, false, func(w io.Writer, _ []string) error {
		_, _ = fmt.Fprint(w, "hi\r\n")
		return nil
	})

	var buf bytes.Buffer
	if err := d.Dispatch(&buf, "HI", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := buf.String(), "hi\r\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

# Chat Prompt — Full Project Snapshot

This file contains the entire repo, concatenated for context.

---
## Makefile
```Makefile
# Simple Makefile for a Go project

# Build the application
all: build test

build:
	@echo "Building..."


	@go build -o main cmd/api/main.go

# Run the application
run:
	@go run cmd/api/main.go

# Test the application
test:
	@echo "Testing..."
	@go test ./...

# Clean the binary
clean:
	@echo "Cleaning..."
	@rm -f main

# Live Reload
watch:
	@if command -v air > /dev/null; then \
            air; \
            echo "Watching...";\
        else \
            read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
            if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
                go install github.com/air-verse/air@latest; \
                air; \
                echo "Watching...";\
            else \
                echo "You chose not to install air. Exiting..."; \
                exit 1; \
            fi; \
        fi

.PHONY: all build run test clean watch
```

---
## cmd/api/main.go
```go
package main

import (
	"log"
	"net"
	"os"
	"strconv"

	"github.com/amir-aharon/goliath/internal/command"
	"github.com/amir-aharon/goliath/internal/server"
	"github.com/amir-aharon/goliath/internal/session"
	"github.com/amir-aharon/goliath/internal/store"
)

func main() {
	port := 6379
	if p, err := strconv.Atoi(os.Getenv("PORT")); err == nil && p > 0 {
		port = p
	}

	d := command.NewDispatcher()
	command.RegisterBuiltins(d)

	kv := store.NewMemory()
	command.RegisterKV(d, kv)
	command.RegisterTTL(d, kv)

	srv := server.Server{
		Addr: "0.0.0.0",
		Port: port,
		New: func(c net.Conn) interface{ Run() } {
			return session.New(c, d)
		},
	}
	log.Fatal(srv.Serve())
}
```

---
## go.mod
```mod
module github.com/amir-aharon/goliath

go 1.23.4
```

---
## internal/command/builtin.go
```go
package command

import (
	"io"
	"strconv"
	"time"

	"github.com/amir-aharon/goliath/internal/proto"
	"github.com/amir-aharon/goliath/internal/store"
)

func RegisterBuiltins(d *Dispatcher) {
	d.Register("PING", 0, 0, false, func(w io.Writer, _ []string) error {
		return proto.PONG(w)
	})
	d.Register("ECHO", 1, 1, false, func(w io.Writer, args []string) error {
		return proto.Line(w, args[0])
	})
	d.Register("QUIT", 0, 0, false, func(w io.Writer, _ []string) error {
		if err := proto.OK(w); err != nil {
			return err
		}
		return ErrQuit
	})
}

func RegisterKV(d *Dispatcher, kv store.KV) {
	d.Register("GET", 1, 1, false, func(w io.Writer, args []string) error {
		if v, ok := kv.Get(args[0]); ok {
			return proto.Line(w, v)
		}
		return proto.Err(w, "key not found")
	})

	d.Register("SET", 2, 2, true, func(w io.Writer, args []string) error {
		kv.Set(args[0], args[1])
		return proto.OK(w)
	})

	d.Register("SETEX", 3, 3, true, func(w io.Writer, args []string) error {
		secs, err := strconv.Atoi(args[1])
		if err != nil || secs <= 0 {
			return proto.Err(w, "seconds must be a positive integer")
		}
		kv.SetEx(args[0], args[2], time.Duration(secs)*time.Second)
		return proto.OK(w)
	})

	d.Register("DEL", 1, 1, true, func(w io.Writer, args []string) error {
		if kv.Del(args[0]) {
			return proto.OK(w)
		}
		return proto.Err(w, "key not found")
	})
}

func RegisterTTL(d *Dispatcher, kv store.KV) {
	d.Register("TTL", 1, 1, false, func(w io.Writer, args []string) error {
		secs, exists, hasExp := kv.TTL(args[0])
		switch {
		case !exists:
			return proto.Int(w, -2)
		case !hasExp:
			return proto.Int(w, -1)
		default:
			return proto.Int(w, int64(secs))
		}
	})

	d.Register("PERSIST", 1, 1, true, func(w io.Writer, args []string) error {
		hasExp := kv.Persist(args[0])
		if hasExp {
			return proto.Int(w, 1)
		}
		return proto.Int(w, 0)
	})
}
```

---
## internal/command/builtins_test.go
```go
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
```

---
## internal/command/helpers_test.go
```go
package command_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/amir-aharon/goliath/internal/command"
	"github.com/amir-aharon/goliath/internal/store"
)

type fakeClock struct{ t time.Time }
func newFakeClock(t time.Time) *fakeClock    { return &fakeClock{t: t} }
func (f *fakeClock) Now() time.Time          { return f.t }
func (f *fakeClock) Advance(d time.Duration) { f.t = f.t.Add(d) }

func newDispatcher() *command.Dispatcher {
	d := command.NewDispatcher()
	command.RegisterBuiltins(d)
	kv := store.NewMemory()
	command.RegisterKV(d, kv)
	command.RegisterTTL(d, kv)
	return d
}

func newDispatcherWithClock(c store.Clock) *command.Dispatcher {
	d := command.NewDispatcher()
	command.RegisterBuiltins(d)
	kv := store.NewMemoryWithClock(c)
	command.RegisterKV(d, kv)
	command.RegisterTTL(d, kv)
	return d
}

func run(d *command.Dispatcher, name string, args ...string) (string, error) {
	var buf bytes.Buffer
	err := d.Dispatch(&buf, name, args)
	return buf.String(), err
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil { t.Fatalf("unexpected error: %v", err) }
}
```

---
## internal/command/kv_test.go
```go
package command_test

import (
	"strings"
	"testing"
)

func TestSET_ReturnsOK(t *testing.T) {
	d := newDispatcher()
	got, err := run(d, "SET", "k", "v")
	if err != nil {
		t.Fatalf("dispatch error: %v", err)
	}
	if got != "+OK\r\n" {
		t.Fatalf("got %q, want %q", got, "+OK\r\n")
	}
}

func TestGET_ExistingKey(t *testing.T) {
	d := newDispatcher()
	if _, err := run(d, "SET", "k", "v"); err != nil {
		t.Fatalf("SET error: %v", err)
	}
	got, err := run(d, "GET", "k")
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}
	if got != "v\r\n" {
		t.Fatalf("got %q, want %q", got, "v\r\n")
	}
}

func TestGET_MissingKey(t *testing.T) {
	d := newDispatcher()
	got, err := run(d, "GET", "nope")
	if err != nil {
		t.Fatalf("dispatch error: %v", err)
	}
	if got != "-ERR key not found\r\n" {
		t.Fatalf("got %q, want %q", got, "-ERR key not found\r\n")
	}
}

func TestDEL_ExistingKey_RemovesKey(t *testing.T) {
	d := newDispatcher()
	if _, err := run(d, "SET", "k", "v"); err != nil {
		t.Fatalf("SET error: %v", err)
	}
	got, err := run(d, "DEL", "k")
	if err != nil {
		t.Fatalf("DEL error: %v", err)
	}
	if got != "+OK\r\n" {
		t.Fatalf("got %q, want %q", got, "+OK\r\n")
	}
	// follow-up GET should now error
	if got, _ := run(d, "GET", "k"); got != "-ERR key not found\r\n" {
		t.Fatalf("after DEL, GET got %q, want %q", got, "-ERR key not found\r\n")
	}
}

func TestDEL_MissingKey_ReturnsErr(t *testing.T) {
	d := newDispatcher()
	got, err := run(d, "DEL", "nope")
	if err != nil {
		t.Fatalf("dispatch error: %v", err)
	}
	if got != "-ERR key not found\r\n" {
		t.Fatalf("got %q, want %q", got, "-ERR key not found\r\n")
	}
}

func TestKV_WrongArity(t *testing.T) {
	d := newDispatcher()
	cases := []struct {
		name string
		args []string
	}{
		{"GET", nil},                     // too few
		{"GET", []string{"a", "b"}},      // too many
		{"SET", []string{"k"}},           // too few
		{"SET", []string{"k", "v", "x"}}, // too many
		{"DEL", nil},                     // too few
		{"DEL", []string{"a", "b"}},      // too many
	}
	for _, c := range cases {
		got, err := run(d, c.name, c.args...)
		if err != nil {
			t.Fatalf("%s wrong-arity: unexpected error: %v", c.name, err)
		}
		if !strings.HasPrefix(got, "-ERR wrong number of arguments") {
			t.Fatalf("%s wrong-arity: got %q", c.name, got)
		}
	}
}
```

---
## internal/command/router.go
```go

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
## internal/command/dispatcher.go
```go
package command

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/amir-aharon/goliath/internal/proto"
)

var ErrQuit = errors.New("quit")

type Handler func(w io.Writer, args []string) error

type Spec struct {
	MinArgs  int
	MaxArgs  int
	Mutating bool
	Handler  Handler
}

type CommandTable map[string]Spec

type Dispatcher struct {
	Table CommandTable
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{Table: make(CommandTable)}
}

func (d *Dispatcher) Register(name string, minArgs, maxArgs int, mutating bool, h Handler) {
	d.Table[strings.ToUpper(name)] = Spec{
		MinArgs:  minArgs,
		MaxArgs:  maxArgs,
		Mutating: mutating,
		Handler:  h,
	}
}

func (d *Dispatcher) Dispatch(w io.Writer, name string, args []string) error {
	spec, ok := d.Table[strings.ToUpper(name)]
	if !ok {
		return proto.Err(w, "unknown command")
	}

	n := len(args)
	if n < spec.MinArgs || (spec.MaxArgs >= 0 && n > spec.MaxArgs) {
		return proto.Err(w, fmt.Sprintf("wrong number of arguments for '%s'", strings.ToUpper(name)))
	}

	return spec.Handler(w, args)
}
```

---
## internal/command/dispatcher_test.go
```go
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
## internal/command/ttl_test.go
```go
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
```

---
## internal/command/wiring_test.go
```go
package command_test

import (
	"testing"

	"github.com/amir-aharon/goliath/internal/command"
	"github.com/amir-aharon/goliath/internal/store"
)

func TestRegisterKV_RegistersKVCommands(t *testing.T) {
	d := command.NewDispatcher()
	kv := store.NewMemory()
	command.RegisterKV(d, kv)

	for _, name := range []string{"GET", "SET", "DEL", "SETEX"} {
		if got, _ := run(d, name); got == "-ERR unknown command\r\n" {
			t.Fatalf("%s unexpectedly unknown", name)
		}
	}
}

func TestRegisterTTL_RegistersTTLAndPersist(t *testing.T) {
	d := command.NewDispatcher()
	kv := store.NewMemory()
	command.RegisterTTL(d, kv)

	for _, name := range []string{"TTL", "PERSIST"} {
		if got, _ := run(d, name); got == "-ERR unknown command\r\n" {
			t.Fatalf("%s unexpectedly unknown", name)
		}
	}
}
```

---
## internal/proto/reply.go
```go
package proto

import (
	"fmt"
	"io"
)

func OK(w io.Writer) error {
	_, err := fmt.Fprint(w, "+OK\r\n")
	return err
}

func Err(w io.Writer, msg string) error {
	_, err := fmt.Fprintf(w, "-ERR %s\r\n", msg)
	return err
}

func PONG(w io.Writer) error {
	_, err := fmt.Fprint(w, "+PONG\r\n")
	return err
}

func Line(w io.Writer, text string) error {
	_, err := fmt.Fprintf(w, "%s\r\n", text)
	return err
}

func Bulk(w io.Writer, s string) error {
	_, err := fmt.Fprintf(w, "$%d\r\n%s\r\n", len(s), s)
	return err
}

func Int(w io.Writer, n int64) error {
	_, err := fmt.Fprintf(w, "%d\r\n", n)
	return err
}
```

---
## internal/proto/reply_test.go
```go
package proto_test

import (
	"bytes"
	"testing"

	"github.com/amir-aharon/goliath/internal/proto"
)

func TestOK(t *testing.T) {
	var buf bytes.Buffer
	err := proto.OK(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	want := "+OK\r\n"

	if got != want {
		t.Errorf("proto.OK() wrote %q, want %q", got, want)
	}
}

func TestPONG(t *testing.T) {
	var buf bytes.Buffer
	if err := proto.PONG(&buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := buf.String(), "+PONG\r\n"; got != want {
		t.Errorf("proto.PONG() wrote %q, want %q", got, want)
	}
}

func TestLine(t *testing.T) {
	var buf bytes.Buffer
	if err := proto.Line(&buf, "hello"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := buf.String(), "hello\r\n"; got != want {
		t.Errorf("proto.Line() wrote %q, want %q", got, want)
	}
}

func TestErr(t *testing.T) {
	var buf bytes.Buffer
	if err := proto.Err(&buf, "key not found"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := buf.String(), "-ERR key not found\r\n"; got != want {
		t.Errorf("proto.Err() wrote %q, want %q", got, want)
	}
}

func TestInt(t *testing.T) {
	tests := []struct {
		name string
		in   int64
		want string
	}{
		{"zero", 0, "0\r\n"},
		{"positive", 42, "42\r\n"},
		{"negative", -2, "-2\r\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := proto.Int(&buf, tc.in); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := buf.String(); got != tc.want {
				t.Errorf("proto.Int(%d) wrote %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestBulkASCII(t *testing.T) {
	var buf bytes.Buffer
	if err := proto.Bulk(&buf, "hello"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := buf.String(), "$5\r\nhello\r\n"; got != want {
		t.Errorf("proto.Bulk() wrote %q, want %q", got, want)
	}
}

func TestBulkUTF8(t *testing.T) {
	var buf bytes.Buffer
	s := "שלום" // 4 runes, 8 bytes in UTF-8
	if err := proto.Bulk(&buf, s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := buf.String(), "$8\r\nשלום\r\n"; got != want {
		t.Errorf("proto.Bulk() wrote %q, want %q", got, want)
	}
}

```

---
## internal/server/server.go
```go
package server

import (
	"fmt"
	"log"
	"net"
)

type NewSession func(net.Conn) interface{ Run() }

// initialize session on new connection
type Server struct {
	Addr string
	Port int
	New  NewSession
}

func (srv *Server) Serve() error {
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", srv.Addr, srv.Port))
	if err != nil {
		return err
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("accept:", err)
			continue
		}
		go srv.New(conn).Run()
	}
}
```

---
## internal/server/server_test.go
```go
package server_test

import (
	"bufio"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/amir-aharon/goliath/internal/command"
	"github.com/amir-aharon/goliath/internal/server"
	"github.com/amir-aharon/goliath/internal/session"
	"github.com/amir-aharon/goliath/internal/store"
)

func waitReady(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return errors.New("server did not start in time")
		}
		c, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err == nil {
			_ = c.Close()
			return nil
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func TestServer_EchoSmoke(t *testing.T) {
	// Build dispatcher + store
	d := command.NewDispatcher()
	command.RegisterBuiltins(d)
	kv := store.NewMemory()
	command.RegisterKV(d, kv)
	command.RegisterTTL(d, kv)

	// Pick a free port by binding to :0, grab it, close it, then let server use it.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("prelisten: %v", err)
	}
	addr := ln.Addr().String()
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()

	srv := server.Server{
		Addr: "127.0.0.1",
		Port: port,
		New: func(c net.Conn) interface{ Run() } {
			return session.New(c, d)
		},
	}

	go func() { _ = srv.Serve() }()

	if err := waitReady(addr, 1*time.Second); err != nil {
		t.Fatal(err)
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	_ = conn.SetWriteDeadline(time.Now().Add(1 * time.Second))

	if _, err := conn.Write([]byte("ECHO hi\r\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	resp, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if resp != "hi\r\n" {
		t.Fatalf("got %q, want %q", resp, "hi\r\n")
	}

	if _, err := conn.Write([]byte("QUIT\r\n")); err == nil {
		_, _ = bufio.NewReader(conn).ReadString('\n') // "+OK\r\n"
	}
}
```

---
## internal/session/session.go
```go
package session

import (
	"bufio"
	"errors"
	"net"
	"strings"

	"github.com/amir-aharon/goliath/internal/command"
)

// per-connection handler
type Session struct {
	Conn       net.Conn
	Dispatcher *command.Dispatcher
}

func New(c net.Conn, d *command.Dispatcher) *Session {
	return &Session{
		Conn:       c,
		Dispatcher: d,
	}
}

func (sess *Session) Run() {
	defer sess.Conn.Close()

	r := bufio.NewReader(sess.Conn)

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}

		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 0 {
			continue
		}

		if err := sess.Dispatcher.Dispatch(sess.Conn, fields[0], fields[1:]); err != nil {
			if errors.Is(err, command.ErrQuit) {
				return
			}
		}
	}
}
```

---
## internal/session/session_test.go
```go
package session_test

import (
	"bufio"
	"net"
	"testing"
	"time"

	"github.com/amir-aharon/goliath/internal/command"
	"github.com/amir-aharon/goliath/internal/session"
	"github.com/amir-aharon/goliath/internal/store"
)

func newDispatcher() *command.Dispatcher {
	d := command.NewDispatcher()
	command.RegisterBuiltins(d)
	kv := store.NewMemory()
	command.RegisterKV(d, kv)
	command.RegisterTTL(d, kv)
	return d
}

func TestSession_PINGAndQUIT(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	d := newDispatcher()
	sess := session.New(serverConn, d)
	done := make(chan struct{})
	go func() { defer close(done); sess.Run() }()

	reader := bufio.NewReader(clientConn)

	// optional safety: deadlines so the test never hangs forever
	_ = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_ = clientConn.SetWriteDeadline(time.Now().Add(2 * time.Second))

	// send PING and read reply
	if _, err := clientConn.Write([]byte("PING\r\n")); err != nil {
		t.Fatalf("write PING: %v", err)
	}
	if resp, err := reader.ReadString('\n'); err != nil || resp != "+PONG\r\n" {
		t.Fatalf("PING resp: got %q, err=%v; want %q", resp, err, "+PONG\r\n")
	}

	// send QUIT, **read the +OK reply**, then wait for session to exit
	if _, err := clientConn.Write([]byte("QUIT\r\n")); err != nil {
		t.Fatalf("write QUIT: %v", err)
	}
	if resp, err := reader.ReadString('\n'); err != nil || resp != "+OK\r\n" {
		t.Fatalf("QUIT resp: got %q, err=%v; want %q", resp, err, "+OK\r\n")
	}

	<-done // session should exit after writing +OK and returning ErrQuit
}
```

---
## internal/store/clock.go
```go
package store

import "time"

type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }


```

---
## internal/store/config.go
```go
package store

import (
	"os"
	"strconv"
)

type MemoryConfig struct {
	SweepIntervalSec int
	SweepSampleSize  int
}

func LoadMemoryConfig() MemoryConfig {
	cfg := MemoryConfig{
		SweepIntervalSec: 60,
		SweepSampleSize:  20,
	}

	if v := os.Getenv("EXPIRED_SWEEP_INTERVAL"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			cfg.SweepIntervalSec = parsed
		}
	}

	if v := os.Getenv("SWEEP_SAMPLE_SIZE"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			cfg.SweepSampleSize = parsed
		}
	}

	return cfg
}
```

---
## internal/store/memory.go
```go
package store

import (
	"sync"
	"time"
)

type entry struct {
	val       string
	expiresAt time.Time
}

type memory struct {
	mu               sync.RWMutex
	m                map[string]entry
	cfg MemoryConfig
	clock Clock
}

func (mem *memory) getEntry(k string) (entry, bool) {
	now := mem.clock.Now()

	mem.mu.RLock()
	e, ok := mem.m[k]
	mem.mu.RUnlock()

	if !ok {
		return entry{}, false
	}

	if e.expired(now) {
		mem.mu.Lock()
		if e2, ok2 := mem.m[k]; ok2 && e2.expired(mem.clock.Now()) {
			delete(mem.m, k)
			mem.mu.Unlock()
			return entry{}, false
		}
		e = mem.m[k]
		mem.mu.Unlock()
	}

	return e, true
}

func (mem *memory) Get(k string) (string, bool) {
	e, ok := mem.getEntry(k)
	if !ok {
		return "", false
	}
	return e.val, true
}

func (mem *memory) Set(k, v string) {
	mem.mu.Lock()
	mem.m[k] = entry{val: v}
	mem.mu.Unlock()
}

func (mem *memory) SetEx(k, v string, ttl time.Duration) {
	mem.mu.Lock()
	mem.m[k] = entry{val: v, expiresAt: mem.clock.Now().Add(ttl)}
	mem.mu.Unlock()
}

func (mem *memory) Del(k string) bool {
	mem.mu.Lock()
	_, existed := mem.m[k]
	delete(mem.m, k)
	mem.mu.Unlock()
	return existed
}
```

---
## internal/store/memory_test.go
```go
package store_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/amir-aharon/goliath/internal/store"
)

// TestStoreSetGet ensures a value set into the store can be retrieved.
func TestStoreSetGet(t *testing.T) {
	mem := store.NewMemory()
	k := "a"
	v := "1"

	mem.Set(k, v)

	got, ok := mem.Get(k)
	if !ok {
		t.Fatalf("expected key to exist")
	}
	if got != v {
		t.Fatalf("got value %q, want %q", got, v)
	}
}

func TestStoreDel(t *testing.T) {
	mem := store.NewMemory()
	k := "a"
	v := "1"

	// deleting a missing key → should return false
	if existed := mem.Del(k); existed {
		t.Fatalf("Del(%q): got true, want false for missing key", k)
	}

	// set then delete → should return true
	mem.Set(k, v)
	if existed := mem.Del(k); !existed {
		t.Fatalf("Del(%q): got false, want true after setting", k)
	}

	// after delete, key is gone
	if _, ok := mem.Get(k); ok {
		t.Fatalf("expected key %q to be gone after Del", k)
	}
}

func TestStoreGetMissing(t *testing.T) {
	mem := store.NewMemory()
	if _, ok := mem.Get("missing"); ok {
		t.Fatalf("expected ok=false for missing key")
	}
}

func TestStoreOverwrite(t *testing.T) {
	mem := store.NewMemory()
	k := "a"

	mem.Set(k, "1")
	if v, _ := mem.Get(k); v != "1" {
		t.Fatalf("got %q, want %q after first set", v, "1")
	}

	mem.Set(k, "2")
	if v, _ := mem.Get(k); v != "2" {
		t.Fatalf("got %q, want %q after overwrite", v, "2")
	}
}

func TestStoreDelIdempotent(t *testing.T) {
	mem := store.NewMemory()
	k := "a"

	mem.Set(k, "1")
	if existed := mem.Del(k); !existed {
		t.Fatalf("first delete: got false, want true")
	}
	if existed := mem.Del(k); existed {
		t.Fatalf("second delete: got true, want false for already-deleted key")
	}
}

func TestStore_NoDataRaces(t *testing.T) {
	const (
		workers  = 8
		iters    = 500
		delEvery = 20 // delete roughly 1/20 iterations per worker
	)

	mem := store.NewMemory()
	keys := []string{"k1", "k2", "k3", "k4", "k5"}

	var wg sync.WaitGroup
	wg.Add(workers)

	for w := range workers {
		go func(id int) {
			defer wg.Done()
			for i := range iters {
				k := keys[(id+i)%len(keys)]
				mem.Set(k, fmt.Sprintf("w%d-%d", id, i))
				_, _ = mem.Get(k)

				if i%delEvery == 0 {
					_ = mem.Del(k)
				}
			}
		}(w)
	}

	wg.Wait()
}
```

---
## internal/store/store.go
```go
package store

import (
	"time"
)

type KV interface {
	Get(k string) (string, bool)
	Set(k, v string)
	SetEx(k, v string, ttl time.Duration)
	Del(k string) bool
	TTL(k string) (int64, bool, bool)
	Persist(k string) bool
}

func NewMemory() *memory {
	cfg := LoadMemoryConfig()
	return newMemory(realClock{}, cfg)
}

func NewMemoryWithClock(c Clock) *memory { // <-- for tests
	cfg := LoadMemoryConfig()
	return newMemory(c, cfg)
}

func newMemory(c Clock, cfg MemoryConfig) *memory {
	m := &memory{
		m:     make(map[string]entry),
		cfg:   cfg,
		clock: c,
	}
	go m.startSweeper()
	return m
}

```

---
## internal/store/sweeper.go
```go
package store

import (
	"math/rand/v2"
	"time"
)

func (mem *memory) startSweeper() {
	ticker := time.NewTicker(time.Duration(mem.cfg.SweepIntervalSec) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		mem.sweepExpired()
	}
}

func (mem *memory) sweepExpired() {
	mem.mu.RLock()
	keys := make([]string, 0, len(mem.m))
	for k := range mem.m {
		keys = append(keys, k)
	}
	mem.mu.RUnlock()

	if len(keys) == 0 {
		return
	}

	N := min(mem.cfg.SweepSampleSize, len(keys))
	now := mem.clock.Now()

	// random unique sample of up to N keys
	rand.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })
	sampled := keys[:N]

	mem.mu.Lock()
	for _, k := range sampled {
		if e, ok := mem.m[k]; ok && e.expired(now) {
			delete(mem.m, k)
		}
	}
	mem.mu.Unlock()
}
```

---
## internal/store/ttl.go
```go
package store

import (
	"time"
)

func (e entry) expired(now time.Time) bool {
	return !e.expiresAt.IsZero() && now.After(e.expiresAt)
}

func (mem *memory) TTL(k string) (int64, bool, bool) {
	e, ok := mem.getEntry(k)
	if !ok {
		return 0, false, false
	}

	if e.expiresAt.IsZero() {
		return 0, true, false
	}

	remaining := e.expiresAt.Sub(mem.clock.Now())
	secs := max(int64(remaining.Seconds()), 0)
	return secs, true, true
}

func (mem *memory) Persist(k string) bool {
	e, ok := mem.getEntry(k)
	if !ok {
		return false
	}

	if e.expiresAt.IsZero() {
		return false
	}

	mem.mu.Lock()
	e.expiresAt = time.Time{}
	mem.m[k] = e
	mem.mu.Unlock()
	return true
}
```

---
## internal/store/ttl_test.go
```go
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
```

---
## make_prompt.sh
```sh
#!/usr/bin/env bash
# make_paste.sh — emit a paste-ready snapshot of your repo into chat_prompt.md
# Usage:
#   ./make_paste.sh                 # core files only (lean)
#   ./make_paste.sh --all           # include all matched files
#   OUTFILE=foo.md ./make_paste.sh  # change output name

set -euo pipefail

OUTFILE="${OUTFILE:-chat_prompt.md}"
MODE="core" # or "all"
if [[ "${1:-}" == "--all" ]]; then MODE="all"; fi

# ------------------------------
# 0) Hardcoded project summary (authoritative)
# ------------------------------
read -r -d '' PROJECT_DESCRIPTION <<'DESC'
This project is a tiny, Redis-inspired in-memory key-value server written in Go.
It exposes a line-based TCP interface (telnet/nc friendly) with a layered design:
server (accepts TCP) → session (parse line, dispatch) → router (arity-checked dispatch) → store (thread-safe map with TTL) → proto (CRLF replies).
Supported commands: PING, ECHO, QUIT, SET, GET, DEL, SETEX, TTL, PERSIST.
TTLs are enforced both lazily (on access) and proactively via a randomized sweeper (configurable interval/sample size).
Replies are centralized so switching to RESP later won’t require touching handlers.
DESC

# If README has a "## paragraph_project_description" block, prefer it; otherwise use hardcoded.
if [[ -f README.md ]] && awk '/^## *paragraph_project_description/{found=1} END{exit !found}' README.md >/dev/null 2>&1; then
  DESCRIPTION="$(awk '/^## *paragraph_project_description/{flag=1; next} /^## /{flag=0} flag {print}' README.md)"
else
  DESCRIPTION="$PROJECT_DESCRIPTION"
fi

# ------------------------------
# 1) Repo metadata
# ------------------------------
in_git=false
if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then in_git=true; fi

repo_name="$(basename "$(pwd)")"
branch="(no-git)"
commit="(no-git)"
dirty=""
if $in_git; then
  branch="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)"
  commit="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
  git diff --quiet --ignore-submodules -- || dirty=" (dirty)"
fi

# Determine Go version (best-effort)
go_ver="$( (go version 2>/dev/null | awk '{print $3}') || true )"
mod_go="$( (grep -E '^go [0-9]+\.[0-9]+' go.mod | awk '{print $2}') || true )"
go_summary="${go_ver:-unknown}${mod_go:+ (module go $mod_go)}"

# Parse Makefile targets (optional, best-effort)
make_targets="$( (grep -E '^[a-zA-Z0-9_\-]+:' -o Makefile 2>/dev/null | sed 's/:$//' | sort -u | paste -sd ', ' -) || true )"

# ------------------------------
# 2) File selection
# ------------------------------
# Default include set tries to stay lean. --all broadens it.
include_globs=(
  "*.go" "go.mod" "go.sum" "*.md" "*.yaml" "*.yml" "*.json" "*.sh" "Makefile"
)
exclude_globs=(
  "vendor/**" "**/bin/**" "**/.git/**" "**/.idea/**" "**/.vscode/**" "**/node_modules/**"
  "**/*.exe" "**/*.dll" "**/*.so" "**/*.dylib" "**/*.a" "**/*.o"
  "**/*.png" "**/*.jpg" "**/*.jpeg" "**/*.gif" "**/*.svg"
  "**/*.log" "**/*.out"
  "**/coverage*"
)
# In "core" mode, further narrow files to essentials
core_dirs=( "cmd" "internal" )
core_allow=( "README.md" "Makefile" "go.mod" "go.sum" )

# Collect candidate files
if $in_git; then
  mapfile -t all_files < <(git ls-files)
else
  mapfile -t all_files < <(find . -type f | sed 's#^\./##' | sort -u)
fi

# Filter by includes
filtered=()
for f in "${all_files[@]}"; do
  keep=false
  # include filter
  for pat in "${include_globs[@]}"; do
    if [[ "$f" == $pat ]]; then keep=true; break; fi
  done
  $keep || continue
  # exclude filter
  for pat in "${exclude_globs[@]}"; do
    if [[ "$f" == $pat ]]; then keep=false; break; fi
  done
  $keep || continue
  # core mode additional narrowing
  if [[ "$MODE" == "core" ]]; then
    in_core=false
    for d in "${core_dirs[@]}"; do
      [[ "$f" == "$d/"* ]] && in_core=true
    done
    for a in "${core_allow[@]}"; do
      [[ "$f" == "$a" ]] && in_core=true
    done
    $in_core || continue
  fi
  filtered+=("$f")
done

# Deterministic sort
IFS=$'\n' filtered=($(sort <<<"${filtered[*]}")); unset IFS

# ------------------------------
# 3) Output header
# ------------------------------
cat > "$OUTFILE" <<EOF
# Chat Prompt — Project Summary

This file is meant to bootstrap ChatGPT with context when starting a new conversation.
Paste this whole file into a new chat to restore project awareness.

---

## Project Description
$DESCRIPTION

---

## Repo At A Glance
- **Repo:** $repo_name
- **Branch:** $branch
- **Commit:** $commit$dirty
- **Go:** $go_summary
- **Make targets:** ${make_targets:-n/a}
- **Mode:** $MODE
- **Files included:** ${#filtered[@]}

> Tip: Ask for “a test plan by layers” or “generate a failing test for X” right after pasting.

---
EOF

# ------------------------------
# 4) Append files with syntax fences (truncate very large files)
# ------------------------------
max_lines_per_file=800   # keep paste snappy; adjust if needed
tail_lines=100           # when truncating, show head and tail

for f in "${filtered[@]}"; do
  lang="text"
  case "$f" in
    *.go)   lang="go" ;;
    *.md)   lang="markdown" ;;
    *.yaml|*.yml) lang="yaml" ;;
    *.json) lang="json" ;;
    *.sh)   lang="bash" ;;
    Makefile) lang="make" ;;
    go.mod) lang="go" ;;
    go.sum) lang="text" ;; # often huge
  esac

  echo "## $f" >> "$OUTFILE"
  echo '```'"$lang" >> "$OUTFILE"

  line_count=$(wc -l < "$f" || echo 0)
  if [[ "$line_count" -gt "$max_lines_per_file" ]]; then
    head -n "$((max_lines_per_file - tail_lines - 5))" "$f" >> "$OUTFILE" || true
    echo -e "\n# ... [truncated: $((line_count - (max_lines_per_file - tail_lines - 5))) lines omitted] ...\n" >> "$OUTFILE"
    tail -n "$tail_lines" "$f" >> "$OUTFILE" || true
  else
    cat "$f" >> "$OUTFILE"
  fi

  echo '```' >> "$OUTFILE"
  echo >> "$OUTFILE"
done

# Footer
cat >> "$OUTFILE" <<'EOF'
---
_End of snapshot. Paste everything above into a new ChatGPT thread._
EOF

echo "Wrote $OUTFILE (${#filtered[@]} files, mode=$MODE)"
```

---
## make_test_prompt.sh
```sh
#!/usr/bin/env bash
# make_test_prompt.sh — emit a paste-ready snapshot of test files into test_prompt.md
# Usage: ./make_test_prompt.sh

set -euo pipefail

OUTFILE="${OUTFILE:-test_prompt.md}"

# collect test files (*.go ending with _test.go)
if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  mapfile -t files < <(git ls-files '*_test.go')
else
  mapfile -t files < <(find . -type f -name '*_test.go' | sed 's#^\./##' | sort)
fi

# deterministic sort
IFS=$'\n' files=($(sort <<<"${files[*]}")); unset IFS

# write header
cat > "$OUTFILE" <<EOF
# Chat Prompt — Test Suite Summary

This file is meant to bootstrap ChatGPT with context on the tests written so far.
Paste this into a new chat to restore test awareness.

---

## Project Tests Included
- Files: ${#files[@]}
EOF

echo >> "$OUTFILE"

# append files with syntax fences
for f in "${files[@]}"; do
  echo "## $f" >> "$OUTFILE"
  echo '```go' >> "$OUTFILE"
  cat "$f" >> "$OUTFILE"
  echo '```' >> "$OUTFILE"
  echo >> "$OUTFILE"
done

# footer
cat >> "$OUTFILE" <<'EOF'
---
_End of test snapshot. Paste above into a new ChatGPT thread to continue test planning or debugging._
EOF

echo "Wrote $OUTFILE (${#files[@]} test files)"
```

---
## snapshot.sh
```sh
#!/usr/bin/env bash

set -euo pipefail

OUTFILE="snapshot.md"

# Start fresh
echo "# Chat Prompt — Full Project Snapshot" > "$OUTFILE"
echo >> "$OUTFILE"
echo "This file contains the entire repo, concatenated for context." >> "$OUTFILE"
echo >> "$OUTFILE"

# Iterate over tracked files (skip .git and large binaries)
# Adjust the grep -v filters if needed
for f in $(git ls-files | grep -vE '(^\.git|\.md$|\.png$|\.jpg$|\.gif$|\.svg$|\.lock$|go\.sum$)'); do
  echo "---" >> "$OUTFILE"
  echo "## $f" >> "$OUTFILE"
  echo '```'$(basename "$f" | awk -F. '{print $NF}') >> "$OUTFILE"
  cat "$f" >> "$OUTFILE"
  echo '```' >> "$OUTFILE"
  echo >> "$OUTFILE"
done

echo "Snapshot written to $OUTFILE"

```

---
## spec
```spec
tcp server :6379
handles lines of messages using goroutines
GET, SET, DEL

persistence:
log every executed command
load snapshot from log

atomic on log file + dict

pubsub:
topics hold groups of channels
SUBSCRIBE topic
PUBLISH topic message

key expiration:
cleaner runs in background
another expiration by key dict
EXPIRE key duration
check expiration on GET
```


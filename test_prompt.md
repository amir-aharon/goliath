# Chat Prompt — Test Suite Summary

This file is meant to bootstrap ChatGPT with context on the tests written so far.
Paste this into a new chat to restore test awareness.

---

## Project Tests Included
- Files: 11

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
	r := newRouter()
	got, err := run(r, "PING")
	if err != nil {
		t.Fatalf("dispatch error: %v", err)
	}
	if got != "+PONG\r\n" {
		t.Fatalf("got %q, want %q", got, "+PONG\r\n")
	}
}

func TestECHO_EchoesArgument(t *testing.T) {
	r := newRouter()
	got, err := run(r, "ECHO", "hello")
	if err != nil {
		t.Fatalf("dispatch error: %v", err)
	}
	if got != "hello\r\n" {
		t.Fatalf("got %q, want %q", got, "hello\r\n")
	}
}

func TestQUIT_RespondsOKAndReturnsErrQuit(t *testing.T) {
	r := newRouter()
	got, err := run(r, "QUIT")
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

	r := newRouter()
	for _, c := range cases {
		got, err := run(r, c.name, c.args...)
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

func newRouter() *command.Router {
	r := command.NewRouter()
	command.RegisterBuiltins(r)
	kv := store.NewMemory()
	command.RegisterKV(r, kv)
	command.RegisterTTL(r, kv)
	return r
}

func newRouterWithClock(c store.Clock) *command.Router {
	r := command.NewRouter()
	command.RegisterBuiltins(r)
	kv := store.NewMemoryWithClock(c)
	command.RegisterKV(r, kv)
	command.RegisterTTL(r, kv)
	return r
}

func run(r *command.Router, name string, args ...string) (string, error) {
	var buf bytes.Buffer
	err := r.Dispatch(&buf, name, args)
	return buf.String(), err
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil { t.Fatalf("unexpected error: %v", err) }
}
```

## internal/command/kv_test.go
```go
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
```

## internal/command/router_test.go
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
	r := command.NewRouter()
	var buf bytes.Buffer
	if err := r.Dispatch(&buf, "NODER", nil); err != nil {
		t.Fatalf("unexpected error from Dispatch: %v", err)
	}

	got := buf.String()
	want := "-ERR unknown command\r\n"
	if got != want {
		t.Errorf("Dispatch wrote %q, want %q", got, want)
	}
}

func TestRouterArityErrors(t *testing.T) {
    r := command.NewRouter()

	r.Handle("ECHO", 1, 1, func(w io.Writer, args []string) error {
        _, _ = fmt.Fprint(w, "ok\r\n")
        return nil
    })

    t.Run("too few", func(t *testing.T) {
        var buf bytes.Buffer
        if err := r.Dispatch(&buf, "ECHO", []string{}); err != nil {
            t.Fatalf("unexpected error from Dispatch: %v", err)
        }
        want := "-ERR wrong number of arguments for 'ECHO'\r\n"
        if got := buf.String(); got != want {
            t.Errorf("got %q, want %q", got, want)
        }
    })

    t.Run("too many", func(t *testing.T) {
        var buf bytes.Buffer
        if err := r.Dispatch(&buf, "ECHO", []string{"a", "b"}); err != nil {
            t.Fatalf("unexpected error from Dispatch: %v", err)
        }
        want := "-ERR wrong number of arguments for 'ECHO'\r\n"
        if got := buf.String(); got != want {
            t.Errorf("got %q, want %q", got, want)
        }
    })
}

func TestRouterCaseInsensitive(t *testing.T) {
	r := command.NewRouter()
	r.Handle("PING", 0, 0, func(w io.Writer, _ []string) error {
		_, _ = fmt.Fprint(w, "ran\r\n")
		return nil
	})

	var buf bytes.Buffer
	if err := r.Dispatch(&buf, "ping", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := buf.String(), "ran\r\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRouterHappyPath(t *testing.T) {
	r := command.NewRouter()
	r.Handle("HI", 0, 0, func(w io.Writer, _ []string) error {
		_, _ = fmt.Fprint(w, "hi\r\n")
		return nil
	})

	var buf bytes.Buffer
	if err := r.Dispatch(&buf, "HI", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := buf.String(), "hi\r\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

```

## internal/command/ttl_test.go
```go
package command_test

import (
	"testing"
	"time"
)

func TestSETEX_NonIntegerSeconds(t *testing.T) {
	r := newRouter()
	got, err := run(r, "SETEX", "k", "abc", "v")
	if err != nil {
		t.Fatalf("dispatch error: %v", err)
	}
	want := "-ERR seconds must be a positive integer\r\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSETEX_NonPositiveSeconds_Zero(t *testing.T) {
	r := newRouter()
	got, err := run(r, "SETEX", "k", "0", "v")
	if err != nil {
		t.Fatalf("dispatch error: %v", err)
	}
	want := "-ERR seconds must be a positive integer\r\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSETEX_NonPositiveSeconds_Negative(t *testing.T) {
	r := newRouter()
	got, err := run(r, "SETEX", "k", "-3", "v")
	if err != nil {
		t.Fatalf("dispatch error: %v", err)
	}
	want := "-ERR seconds must be a positive integer\r\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSETEX_ValidSeconds_ReturnsOK(t *testing.T) {
	r := newRouter()
	got, err := run(r, "SETEX", "k", "5", "v")
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
	r := newRouterWithClock(fc)

	got, err := run(r, "TTL", "nope")
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
	r := newRouterWithClock(fc)

	if _, err := run(r, "SET", "k", "v"); err != nil {
		t.Fatalf("SET error: %v", err)
	}
	got, err := run(r, "TTL", "k")
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
	r := newRouterWithClock(fc)

	if _, err := run(r, "SETEX", "k", "5", "v"); err != nil {
		t.Fatalf("SETEX error: %v", err)
	}

	if got, _ := run(r, "TTL", "k"); got != "5\r\n" {
		t.Fatalf("after setex: got %q, want %q", got, "5\r\n")
	}

	fc.Advance(3 * time.Second)
	if got, _ := run(r, "TTL", "k"); got != "2\r\n" {
		t.Fatalf("after +3s: got %q, want %q", got, "2\r\n")
	}
}

func TestTTL_AfterExpiry_ReturnsMinus2(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	fc := newFakeClock(start)
	r := newRouterWithClock(fc)

	if _, err := run(r, "SETEX", "k", "2", "v"); err != nil {
		t.Fatalf("SETEX error: %v", err)
	}

	// Advance beyond expiry
	fc.Advance(3 * time.Second)
	// A TTL call after expiry should report -2 (missing)
	if got, _ := run(r, "TTL", "k"); got != "-2\r\n" {
		t.Fatalf("after expiry: got %q, want %q", got, "-2\r\n")
	}
}

func TestPERSIST_OnKeyWithExpiry_Returns1_ThenTTLMinus1(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	fc := newFakeClock(start)
	r := newRouterWithClock(fc)

	if _, err := run(r, "SETEX", "k", "5", "v"); err != nil {
		t.Fatalf("SETEX error: %v", err)
	}

	if got, _ := run(r, "PERSIST", "k"); got != "1\r\n" {
		t.Fatalf("PERSIST: got %q, want %q", got, "1\r\n")
	}

	if got, _ := run(r, "TTL", "k"); got != "-1\r\n" {
		t.Fatalf("TTL after persist: got %q, want %q", got, "-1\r\n")
	}

	fc.Advance(10 * time.Second)
	if got, _ := run(r, "GET", "k"); got != "v\r\n" {
		t.Fatalf("GET after persist+advance: got %q, want %q", got, "v\r\n")
	}
}

func TestPERSIST_OnKeyWithoutExpiry_Returns0(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	fc := newFakeClock(start)
	r := newRouterWithClock(fc)

	if _, err := run(r, "SET", "k", "v"); err != nil {
		t.Fatalf("SET error: %v", err)
	}
	if got, _ := run(r, "PERSIST", "k"); got != "0\r\n" {
		t.Fatalf("PERSIST: got %q, want %q", got, "0\r\n")
	}
}

func TestPERSIST_OnMissingOrExpiredKey_Returns0(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	fc := newFakeClock(start)
	r := newRouterWithClock(fc)

	if got, _ := run(r, "PERSIST", "nope"); got != "0\r\n" {
		t.Fatalf("PERSIST missing: got %q, want %q", got, "0\r\n")
	}

	if _, err := run(r, "SETEX", "gone", "2", "v"); err != nil {
		t.Fatalf("SETEX error: %v", err)
	}
	fc.Advance(3 * time.Second)
	if got, _ := run(r, "PERSIST", "gone"); got != "0\r\n" {
		t.Fatalf("PERSIST expired: got %q, want %q", got, "0\r\n")
	}
}```

## internal/command/wiring_test.go
```go
package command_test

import (
	"testing"

	"github.com/amir-aharon/goliath/internal/command"
	"github.com/amir-aharon/goliath/internal/store"
)

func TestRegisterKV_RegistersKVCommands(t *testing.T) {
	r := command.NewRouter()
	kv := store.NewMemory()
	command.RegisterKV(r, kv)

	for _, name := range []string{"GET", "SET", "DEL", "SETEX"} {
		if got, _ := run(r, name); got == "-ERR unknown command\r\n" {
			t.Fatalf("%s unexpectedly unknown", name)
		}
	}
}

func TestRegisterTTL_RegistersTTLAndPersist(t *testing.T) {
	r := command.NewRouter()
	kv := store.NewMemory()
	command.RegisterTTL(r, kv)

	for _, name := range []string{"TTL", "PERSIST"} {
		if got, _ := run(r, name); got == "-ERR unknown command\r\n" {
			t.Fatalf("%s unexpectedly unknown", name)
		}
	}
}
```

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
	// Build router + store
	r := command.NewRouter()
	command.RegisterBuiltins(r)
	kv := store.NewMemory()
	command.RegisterKV(r, kv)
	command.RegisterTTL(r, kv)

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
			return session.New(c, r)
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

func newRouter() *command.Router {
	r := command.NewRouter()
	command.RegisterBuiltins(r)
	kv := store.NewMemory()
	command.RegisterKV(r, kv)
	command.RegisterTTL(r, kv)
	return r
}

func TestSession_PINGAndQUIT(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	r := newRouter()
	sess := session.New(serverConn, r)
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
_End of test snapshot. Paste above into a new ChatGPT thread to continue test planning or debugging._

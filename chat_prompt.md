# Chat Prompt — Project Summary

This file is meant to bootstrap ChatGPT with context when starting a new conversation.
Paste this whole file into a new chat to restore project awareness.

---

## Project Description

This project is a tiny, Redis-inspired in-memory key-value server written in Go.
It exposes a line-based TCP interface you can use via telnet/nc. The design is layered: server (accepts TCP) → session (parse line, dispatch) → command/router (arity-checked dispatch) → store (thread-safe map with TTL) → proto (CRLF replies).
Supported commands: PING, ECHO, QUIT, SET, GET, DEL, SETEX, TTL, PERSIST. TTLs are enforced lazily on access and proactively via a randomized sweeper with configurable interval/sample size. Replies are centralized so switching to RESP later won’t touch handlers.

---
## README.md
```markdown
# Project goliath

One Paragraph of project description goes here

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes. See deployment for notes on how to deploy the project on a live system.

## MakeFile

Run build make command with tests
```bash
make all
```

Build the application
```bash
make build
```

Run the application
```bash
make run
```

Live reload the application:
```bash
make watch
```

Run the test suite:
```bash
make test
```

Clean up binary from the last build:
```bash
make clean
```
```

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

	r := command.NewRouter()
	command.RegisterBuiltins(r)

	kv := store.NewMemory()
	command.RegisterKV(r, kv)
	command.RegisterTTL(r, kv)

	srv := server.Server{
		Addr: "0.0.0.0",
		Port: port,
		New: func(c net.Conn) interface{ Run() } {
			return session.New(c, r)
		},
	}
	log.Fatal(srv.Serve())
}
```

## go.mod
```go
module github.com/amir-aharon/goliath

go 1.23.4

require github.com/joho/godotenv v1.5.1
```

## go.sum
```text
github.com/joho/godotenv v1.5.1 h1:7eLL/+HRGLY0ldzfGMeQkb7vMd0as4CfYvUVzLqw0N0=
github.com/joho/godotenv v1.5.1/go.mod h1:f4LDr5Voq0i2e/R5DDNOoa2zzDfwtkZa6DnEwAbqwq4=
```

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

func RegisterBuiltins(r *Router) {
	r.Handle("PING", 0, 0, func(w io.Writer, _ []string) error {
		return proto.PONG(w)
	})
	r.Handle("ECHO", 1, 1, func(w io.Writer, args []string) error {
		return proto.Line(w, args[0])
	})
	r.Handle("QUIT", 0, 0, func(w io.Writer, _ []string) error {
		if err := proto.OK(w); err != nil {
			return err
		}
		return ErrQuit
	})
}

func RegisterKV(r *Router, kv store.KV) {
	r.Handle("GET", 1, 1, func(w io.Writer, args []string) error {
		if v, ok := kv.Get(args[0]); ok {
			return proto.Line(w, v)
		}
		return proto.Err(w, "key not found")
	})

	r.Handle("SET", 2, 2, func(w io.Writer, args []string) error {
		kv.Set(args[0], args[1])
		return proto.OK(w)
	})

	r.Handle("SETEX", 3, 3, func(w io.Writer, args []string) error {
		secs, err := strconv.Atoi(args[1])
		if err != nil || secs <= 0 {
			return proto.Err(w, "seconds must be a positive integer")
		}
		kv.SetEx(args[0], args[2], time.Duration(secs)*time.Second)
		return proto.OK(w)
	})

	r.Handle("DEL", 1, 1, func(w io.Writer, args []string) error {
		if kv.Del(args[0]) {
			return proto.OK(w)
		}
		return proto.Err(w, "key not found")
	})
}

func RegisterTTL(r *Router, kv store.KV) {
	r.Handle("TTL", 1, 1, func(w io.Writer, args []string) error {
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

	r.Handle("PERSIST", 1, 1, func(w io.Writer, args []string) error {
		hasExp := kv.Persist(args[0])
		if hasExp {
			return proto.Int(w, 1)
		}
		return proto.Int(w, 0)
	})
}
```

## internal/command/router.go
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

type spec struct {
	minArgs int
	maxArgs int
	h       Handler
}

type Router struct {
	m map[string]spec
}

func NewRouter() *Router { return &Router{m: make(map[string]spec)} }

// Register handlers
func (r *Router) Handle(name string, minArgs, maxArgs int, h Handler) {
	r.m[strings.ToUpper(name)] = spec{minArgs: minArgs, maxArgs: maxArgs, h: h}
}

func (r *Router) Dispatch(w io.Writer, name string, args []string) error {
	sp, ok := r.m[strings.ToUpper(name)]
	if !ok {
		return proto.Err(w, "unknown command")
	}
	n := len(args)
	if n < sp.minArgs || (sp.maxArgs >= 0 && n > sp.maxArgs) {
		return proto.Err(w, fmt.Sprintf("wrong number of arguments for '%s'", strings.ToUpper(name)))
	}
	return sp.h(w, args)
}
```

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
	Conn   net.Conn
	Router *command.Router
}

func New(c net.Conn, r *command.Router) *Session {
	return &Session{
		Conn:   c,
		Router: r,
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

		if err := sess.Router.Dispatch(sess.Conn, fields[0], fields[1:]); err != nil {
			if errors.Is(err, command.ErrQuit) {
				return
			}
		}
	}
}
```

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
	SweepIntervalSec int
	SweepSampleSize  int
}

func (mem *memory) getEntry(k string) (entry, bool) {
	now := time.Now()

	mem.mu.RLock()
	e, ok := mem.m[k]
	mem.mu.RUnlock()

	if !ok {
		return entry{}, false
	}

	if e.expired(now) {
		mem.mu.Lock()
		if e2, ok2 := mem.m[k]; ok2 && e2.expired(time.Now()) {
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
	mem.m[k] = entry{val: v, expiresAt: time.Now().Add(ttl)}
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
	m := &memory{
		m:   make(map[string]entry),
		cfg: cfg,
	}
	go m.startSweeper()
	return m
}
```

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
	now := time.Now()

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

## internal/store/ttl.go
```go
package store

import "time"

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

	secs := max(int64(time.Until(e.expiresAt).Seconds()), 0)
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


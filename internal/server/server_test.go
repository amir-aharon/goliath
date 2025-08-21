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

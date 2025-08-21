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

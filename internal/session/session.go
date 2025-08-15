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

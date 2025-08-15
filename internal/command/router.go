package command

import (
	"errors"
	"io"
	"strings"

	"github.com/amir-aharon/goliath/internal/proto"
)

var ErrQuit = errors.New("quit")

type Handler func(w io.Writer, args []string) error

type Router struct {
	m map[string]Handler
}

func NewRouter() *Router { return &Router{m: make(map[string]Handler)} }

// Register handlers
func (r *Router) Handle(name string, h Handler) {
	r.m[strings.ToUpper(name)] = h
}

func (r *Router) Dispatch(w io.Writer, name string, args []string) error {
	h, ok := r.m[strings.ToUpper(name)]
	if !ok {
		return proto.Err(w, "unknown command")
	}
	return h(w, args)
}

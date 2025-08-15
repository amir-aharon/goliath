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

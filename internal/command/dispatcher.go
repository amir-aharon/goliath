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

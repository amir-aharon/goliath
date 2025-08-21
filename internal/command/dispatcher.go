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

func (s Spec) Validate(name string, args []string) error {
	n := len(args)
	if n < s.MinArgs || (s.MaxArgs >= 0 && n > s.MaxArgs) {
		return fmt.Errorf("wrong number of arguments for '%s'", strings.ToUpper(name))
	}
	return nil
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

	if err := spec.Validate(name, args); err != nil {
		return proto.Err(w, err.Error())
	}

	return spec.Handler(w, args)
}

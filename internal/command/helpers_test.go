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

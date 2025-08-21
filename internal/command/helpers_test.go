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

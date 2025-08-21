package command_test

import (
	"testing"

	"github.com/amir-aharon/goliath/internal/command"
	"github.com/amir-aharon/goliath/internal/store"
)

func TestRegisterKV_RegistersKVCommands(t *testing.T) {
	r := command.NewRouter()
	kv := store.NewMemory()
	command.RegisterKV(r, kv)

	for _, name := range []string{"GET", "SET", "DEL", "SETEX"} {
		if got, _ := run(r, name); got == "-ERR unknown command\r\n" {
			t.Fatalf("%s unexpectedly unknown", name)
		}
	}
}

func TestRegisterTTL_RegistersTTLAndPersist(t *testing.T) {
	r := command.NewRouter()
	kv := store.NewMemory()
	command.RegisterTTL(r, kv)

	for _, name := range []string{"TTL", "PERSIST"} {
		if got, _ := run(r, name); got == "-ERR unknown command\r\n" {
			t.Fatalf("%s unexpectedly unknown", name)
		}
	}
}

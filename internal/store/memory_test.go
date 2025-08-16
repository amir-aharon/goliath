package store_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/amir-aharon/goliath/internal/store"
)

// TestStoreSetGet ensures a value set into the store can be retrieved.
func TestStoreSetGet(t *testing.T) {
	mem := store.NewMemory()
	k := "a"
	v := "1"

	mem.Set(k, v)

	got, ok := mem.Get(k)
	if !ok {
		t.Fatalf("expected key to exist")
	}
	if got != v {
		t.Fatalf("got value %q, want %q", got, v)
	}
}

func TestStoreDel(t *testing.T) {
	mem := store.NewMemory()
	k := "a"
	v := "1"

	// deleting a missing key → should return false
	if existed := mem.Del(k); existed {
		t.Fatalf("Del(%q): got true, want false for missing key", k)
	}

	// set then delete → should return true
	mem.Set(k, v)
	if existed := mem.Del(k); !existed {
		t.Fatalf("Del(%q): got false, want true after setting", k)
	}

	// after delete, key is gone
	if _, ok := mem.Get(k); ok {
		t.Fatalf("expected key %q to be gone after Del", k)
	}
}

func TestStoreGetMissing(t *testing.T) {
	mem := store.NewMemory()
	if _, ok := mem.Get("missing"); ok {
		t.Fatalf("expected ok=false for missing key")
	}
}

func TestStoreOverwrite(t *testing.T) {
	mem := store.NewMemory()
	k := "a"

	mem.Set(k, "1")
	if v, _ := mem.Get(k); v != "1" {
		t.Fatalf("got %q, want %q after first set", v, "1")
	}

	mem.Set(k, "2")
	if v, _ := mem.Get(k); v != "2" {
		t.Fatalf("got %q, want %q after overwrite", v, "2")
	}
}

func TestStoreDelIdempotent(t *testing.T) {
	mem := store.NewMemory()
	k := "a"

	mem.Set(k, "1")
	if existed := mem.Del(k); !existed {
		t.Fatalf("first delete: got false, want true")
	}
	if existed := mem.Del(k); existed {
		t.Fatalf("second delete: got true, want false for already-deleted key")
	}
}

func TestStore_NoDataRaces(t *testing.T) {
	const (
		workers  = 8
		iters    = 500
		delEvery = 20 // delete roughly 1/20 iterations per worker
	)

	mem := store.NewMemory()
	keys := []string{"k1", "k2", "k3", "k4", "k5"}

	var wg sync.WaitGroup
	wg.Add(workers)

	for w := range workers {
		go func(id int) {
			defer wg.Done()
			for i := range iters {
				k := keys[(id+i)%len(keys)]
				mem.Set(k, fmt.Sprintf("w%d-%d", id, i))
				_, _ = mem.Get(k)

				if i%delEvery == 0 {
					_ = mem.Del(k)
				}
			}
		}(w)
	}

	wg.Wait()
}

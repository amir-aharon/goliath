package store

import (
	"math/rand/v2"
	"time"
)

func (mem *memory) startSweeper() {
	ticker := time.NewTicker(time.Duration(mem.cfg.SweepIntervalSec) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		mem.sweepExpired()
	}
}

func (mem *memory) sweepExpired() {
	mem.mu.RLock()
	keys := make([]string, 0, len(mem.m))
	for k := range mem.m {
		keys = append(keys, k)
	}
	mem.mu.RUnlock()

	if len(keys) == 0 {
		return
	}

	N := min(mem.cfg.SweepSampleSize, len(keys))
	now := time.Now()

	// random unique sample of up to N keys
	rand.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })
	sampled := keys[:N]

	mem.mu.Lock()
	for _, k := range sampled {
		if e, ok := mem.m[k]; ok && e.expired(now) {
			delete(mem.m, k)
		}
	}
	mem.mu.Unlock()
}

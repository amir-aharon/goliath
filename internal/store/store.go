package store

import (
	"time"
)

type KV interface {
	Get(k string) (string, bool)
	Set(k, v string)
	SetEx(k, v string, ttl time.Duration)
	Del(k string) bool
	TTL(k string) (int64, bool, bool)
	Persist(k string) bool
}

func NewMemory() *memory {
	cfg := LoadMemoryConfig()
	return newMemory(realClock{}, cfg)
}

func NewMemoryWithClock(c Clock) *memory { // <-- for tests
	cfg := LoadMemoryConfig()
	return newMemory(c, cfg)
}

func newMemory(c Clock, cfg MemoryConfig) *memory {
	m := &memory{
		m:     make(map[string]entry),
		cfg:   cfg,
		clock: c,
	}
	go m.startSweeper()
	return m
}


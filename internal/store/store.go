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
	m := &memory{
		m:   make(map[string]entry),
		cfg: cfg,
	}
	go m.startSweeper()
	return m
}

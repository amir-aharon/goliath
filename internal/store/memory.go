package store

import (
	"sync"
	"time"
)

type entry struct {
	val       string
	expiresAt time.Time
}

type memory struct {
	mu               sync.RWMutex
	m                map[string]entry
	cfg MemoryConfig
}

func (mem *memory) getEntry(k string) (entry, bool) {
	now := time.Now()

	mem.mu.RLock()
	e, ok := mem.m[k]
	mem.mu.RUnlock()

	if !ok {
		return entry{}, false
	}

	if e.expired(now) {
		mem.mu.Lock()
		if e2, ok2 := mem.m[k]; ok2 && e2.expired(time.Now()) {
			delete(mem.m, k)
			mem.mu.Unlock()
			return entry{}, false
		}
		e = mem.m[k]
		mem.mu.Unlock()
	}

	return e, true
}

func (mem *memory) Get(k string) (string, bool) {
	e, ok := mem.getEntry(k)
	if !ok {
		return "", false
	}
	return e.val, true
}

func (mem *memory) Set(k, v string) {
	mem.mu.Lock()
	mem.m[k] = entry{val: v}
	mem.mu.Unlock()
}

func (mem *memory) SetEx(k, v string, ttl time.Duration) {
	mem.mu.Lock()
	mem.m[k] = entry{val: v, expiresAt: time.Now().Add(ttl)}
	mem.mu.Unlock()
}

func (mem *memory) Del(k string) bool {
	mem.mu.Lock()
	_, existed := mem.m[k]
	delete(mem.m, k)
	mem.mu.Unlock()
	return existed
}

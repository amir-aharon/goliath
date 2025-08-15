package store

import (
	"sync"
	"time"
)

type KV interface {
	Get(k string) (string, bool)
	Set(k, v string)
	SetEx(k, v string, ttl time.Duration)
	Del(k string) bool
	TTL(k string) (sec int64, exists bool, hasExpiry bool)
}

type entry struct {
	val       string
	expiresAt time.Time
}

func (e entry) expired(now time.Time) bool {
	return !e.expiresAt.IsZero() && now.After(e.expiresAt)
}

type Memory struct {
	mu sync.RWMutex
	m  map[string]entry
}

func NewMemory() *Memory {
	return &Memory{m: make(map[string]entry)}
}

func (s *Memory) Get(k string) (string, bool) {
	now := time.Now()
	s.mu.RLock()
	e, ok := s.m[k]
	s.mu.RUnlock()
	if !ok {
		return "", false
	}
	if e.expired(now) {
		s.mu.Lock()
		if e2, ok2 := s.m[k]; ok2 && e2.expired(now) {
			delete(s.m, k)
		}
		s.mu.Unlock()
		return "", false
	}
	return e.val, true
}

func (s *Memory) Set(k, v string) {
	s.mu.Lock()
	s.m[k] = entry{val: v}
	s.mu.Unlock()
}

func (s *Memory) SetEx(k, v string, ttl time.Duration) {
	s.mu.Lock()
	s.m[k] = entry{val: v, expiresAt: time.Now().Add(ttl)}
	s.mu.Unlock()
}

func (s *Memory) Del(k string) bool {
	s.mu.Lock()
	_, existed := s.m[k]
	delete(s.m, k)
	s.mu.Unlock()
	return existed
}

// TTL(k string) (sec int64, exists bool, hasExpiry bool)
func (s *Memory) TTL(k string) (int64, bool, bool) {
	now := time.Now()

	s.mu.RLock()
	e, ok := s.m[k]
	s.mu.RUnlock()

	if !ok {
		return 0, false, false
	}

	if e.expired(now) {
		s.mu.Lock()
		if e2, ok2 := s.m[k]; ok2 && e2.expired(now) {
			delete(s.m, k)
			s.mu.Unlock()
			return 0, false, false
		}
		e = s.m[k]
		s.mu.Unlock()
	}

	if e.expiresAt.IsZero() {
		return 0, true, false
	}

	secs := int64(time.Until(e.expiresAt).Seconds())
	secs = max(secs, 0)
	return secs, true, true
}

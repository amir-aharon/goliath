package store

import "time"

func (e entry) expired(now time.Time) bool {
	return !e.expiresAt.IsZero() && now.After(e.expiresAt)
}

func (mem *memory) TTL(k string) (int64, bool, bool) {
	e, ok := mem.getEntry(k)
	if !ok {
		return 0, false, false
	}

	if e.expiresAt.IsZero() {
		return 0, true, false
	}

	secs := max(int64(time.Until(e.expiresAt).Seconds()), 0)
	return secs, true, true
}

func (mem *memory) Persist(k string) bool {
	e, ok := mem.getEntry(k)
	if !ok {
		return false
	}

	if e.expiresAt.IsZero() {
		return false
	}

	mem.mu.Lock()
	e.expiresAt = time.Time{}
	mem.m[k] = e
	mem.mu.Unlock()
	return true
}

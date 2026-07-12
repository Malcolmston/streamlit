package st

import (
	"sync"
	"time"
)

// cacheEntry is a memoised value with an optional expiry.
type cacheEntry struct {
	val     any
	expires time.Time // zero means it never expires
}

// cacheStore is the process-wide memoisation table shared by all sessions. It
// is the analogue of Streamlit's @st.cache_data, which caches across reruns and
// across users for the lifetime of the process.
var (
	cacheMu    sync.Mutex
	cacheStore = map[string]cacheEntry{}
)

// Cache returns the value memoised under key, computing it with compute on the
// first call (or after the entry has expired) and returning the stored value on
// subsequent calls. The cache is process-wide and shared across sessions, so an
// expensive computation runs once and is reused by every rerun and every
// browser, mirroring Streamlit's st.cache_data.
//
// An optional time-to-live may be supplied as the final argument; a positive
// TTL causes the entry to be recomputed once it has aged past the duration. A
// zero or omitted TTL caches for the lifetime of the process.
//
//	rows := s.Cache("report", func() any { return loadReport() }, time.Minute)
//	report := rows.([]Row)
func (s *Session) Cache(key string, compute func() any, ttl ...time.Duration) any {
	var d time.Duration
	if len(ttl) > 0 {
		d = ttl[0]
	}

	cacheMu.Lock()
	if e, ok := cacheStore[key]; ok {
		if e.expires.IsZero() || time.Now().Before(e.expires) {
			cacheMu.Unlock()
			return e.val
		}
	}
	cacheMu.Unlock()

	// Compute outside the lock so a slow computation does not block other keys.
	val := compute()

	entry := cacheEntry{val: val}
	if d > 0 {
		entry.expires = time.Now().Add(d)
	}
	cacheMu.Lock()
	cacheStore[key] = entry
	cacheMu.Unlock()
	return val
}

// CacheClear removes all entries from the process-wide cache used by
// [Session.Cache]. It is primarily useful in tests and for manual
// invalidation.
func CacheClear() {
	cacheMu.Lock()
	cacheStore = map[string]cacheEntry{}
	cacheMu.Unlock()
}

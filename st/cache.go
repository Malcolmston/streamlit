package st

import (
	"sync"
	"time"
)

// DefaultMaxCacheEntries bounds the process-wide table used by
// [Session.Cache]. Streamlit's @st.cache_data defaults to max_entries=None
// (unbounded); this port bounds it because a cache whose keys are derived from
// user input otherwise grows until the process dies. Change it with
// [CacheSetMaxEntries].
const DefaultMaxCacheEntries = 1024

// cacheEntry is a memoised value with an optional expiry.
type cacheEntry struct {
	val     any
	expires time.Time // zero means it never expires
	// seq orders insertions so the oldest entry can be evicted when the table
	// is full. A counter is used rather than a timestamp because two entries
	// inserted within the same clock tick must still have a defined order.
	seq uint64
}

// flight tracks an in-progress computation for a key so that concurrent callers
// wait for it rather than each running compute themselves. Streamlit documents
// cache_data as running the function once; without this, N simultaneous
// sessions hitting a cold cache would run it N times.
type flight struct {
	done chan struct{}
}

// cacheStore is the process-wide memoisation table shared by all sessions. It
// is the analogue of Streamlit's @st.cache_data, which caches across reruns and
// across users for the lifetime of the process. resourceStore is the analogue
// of @st.cache_resource and is never expired or evicted.
var (
	cacheMu       sync.Mutex
	cacheStore    = map[string]cacheEntry{}
	cacheFlights  = map[string]*flight{}
	cacheSeq      uint64
	cacheMax      = DefaultMaxCacheEntries
	resourceStore = map[string]any{}
)

// resourceFlightKey namespaces resource in-flight markers so that a data key
// and a resource key of the same name cannot collide.
const resourceFlightKey = "\x00resource\x00"

// CacheSetMaxEntries sets how many entries [Session.Cache] retains. When the
// table is full the entry inserted longest ago is evicted. A value of zero or
// less restores [DefaultMaxCacheEntries]. It affects [Session.Cache] only;
// [Session.CacheResource] entries are never evicted.
func CacheSetMaxEntries(n int) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if n <= 0 {
		n = DefaultMaxCacheEntries
	}
	cacheMax = n
	evictCacheLocked()
}

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
//
// Concurrent callers asking for the same missing key do not each run compute:
// the first caller computes while the others wait for its result. The table
// holds at most [DefaultMaxCacheEntries] entries (see [CacheSetMaxEntries]);
// when it is full the entry inserted longest ago is evicted. Derive keys that
// include everything the computation depends on — this port takes an explicit
// key rather than hashing the arguments the way Python's decorator does.
func (s *Session) Cache(key string, compute func() any, ttl ...time.Duration) any {
	var d time.Duration
	if len(ttl) > 0 {
		d = ttl[0]
	}

	cacheMu.Lock()
	for {
		if e, ok := cacheStore[key]; ok && (e.expires.IsZero() || time.Now().Before(e.expires)) {
			cacheMu.Unlock()
			return e.val
		}
		f, inFlight := cacheFlights[key]
		if !inFlight {
			break
		}
		// Someone else is already computing this key: wait for them, then
		// re-check in case their result has already expired or been evicted.
		cacheMu.Unlock()
		<-f.done
		cacheMu.Lock()
	}
	f := &flight{done: make(chan struct{})}
	cacheFlights[key] = f
	cacheMu.Unlock()

	// Compute outside the lock so a slow computation does not block other keys.
	// The deferred cleanup runs even if compute panics, so a panicking
	// computation cannot wedge every other caller waiting on this key.
	defer func() {
		cacheMu.Lock()
		delete(cacheFlights, key)
		cacheMu.Unlock()
		close(f.done)
	}()
	val := compute()

	entry := cacheEntry{val: val}
	if d > 0 {
		entry.expires = time.Now().Add(d)
	}
	cacheMu.Lock()
	cacheSeq++
	entry.seq = cacheSeq
	cacheStore[key] = entry
	evictCacheLocked()
	cacheMu.Unlock()
	return val
}

// CacheResource returns the singleton registered under key, creating it with
// create on the first call, mirroring Streamlit's @st.cache_resource. Use it
// for objects that are expensive to build and safe to share — database
// handles, model weights, clients — rather than for data.
//
// Unlike [Session.Cache] a resource has no TTL and is never evicted, so create
// runs exactly once per key for the lifetime of the process. create must return
// a value that is safe for concurrent use, because every session shares it.
//
//	db := s.CacheResource("db", func() any { return mustOpenDB() }).(*sql.DB)
func (s *Session) CacheResource(key string, create func() any) any {
	fk := resourceFlightKey + key
	cacheMu.Lock()
	for {
		if v, ok := resourceStore[key]; ok {
			cacheMu.Unlock()
			return v
		}
		f, inFlight := cacheFlights[fk]
		if !inFlight {
			break
		}
		cacheMu.Unlock()
		<-f.done
		cacheMu.Lock()
	}
	f := &flight{done: make(chan struct{})}
	cacheFlights[fk] = f
	cacheMu.Unlock()

	defer func() {
		cacheMu.Lock()
		delete(cacheFlights, fk)
		cacheMu.Unlock()
		close(f.done)
	}()
	val := create()

	cacheMu.Lock()
	resourceStore[key] = val
	cacheMu.Unlock()
	return val
}

// evictCacheLocked trims the data cache down to cacheMax entries, dropping the
// entries inserted longest ago first. The caller must hold cacheMu.
func evictCacheLocked() {
	for len(cacheStore) > cacheMax {
		var oldestKey string
		var oldestSeq uint64
		first := true
		for k, e := range cacheStore {
			if first || e.seq < oldestSeq {
				oldestKey, oldestSeq, first = k, e.seq, false
			}
		}
		delete(cacheStore, oldestKey)
	}
}

// CacheDelete removes a single entry from the data cache used by
// [Session.Cache], forcing the next call for that key to recompute. It reports
// whether an entry was present. It is the targeted counterpart to [CacheClear],
// matching the per-function .clear() Streamlit exposes on a cached function.
func CacheDelete(key string) bool {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	_, ok := cacheStore[key]
	delete(cacheStore, key)
	return ok
}

// CacheLen reports how many entries the data cache currently holds. Resources
// registered with [Session.CacheResource] are not counted.
func CacheLen() int {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	return len(cacheStore)
}

// CacheClear removes all entries from the process-wide cache used by
// [Session.Cache]. It is primarily useful in tests and for manual
// invalidation. Resources registered with [Session.CacheResource] survive;
// clear those with [CacheResourceClear].
func CacheClear() {
	cacheMu.Lock()
	cacheStore = map[string]cacheEntry{}
	cacheMu.Unlock()
}

// CacheResourceClear discards every singleton registered with
// [Session.CacheResource], so the next call for each key builds a fresh one.
// It mirrors st.cache_resource.clear() and is chiefly useful in tests.
func CacheResourceClear() {
	cacheMu.Lock()
	resourceStore = map[string]any{}
	cacheMu.Unlock()
}

package st

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"
)

// State is a per-session key/value store that persists across reruns of the
// app function. It is the Go analogue of Streamlit's st.session_state and is a
// convenient place to keep data (counters, accumulated history, cached
// computations) that must survive the top-to-bottom re-execution of the app.
//
// State is safe to use only from within a single run of the app function; the
// server serialises runs of a given session, so no additional locking is
// required by app code.
type State struct {
	m map[string]any
}

// newState returns an empty State.
func newState() *State { return &State{m: map[string]any{}} }

// Get returns the value stored under key and whether it was present.
func (s *State) Get(key string) (any, bool) {
	v, ok := s.m[key]
	return v, ok
}

// GetString returns the string stored under key, or def if absent or not a
// string.
func (s *State) GetString(key, def string) string {
	if v, ok := s.m[key]; ok {
		if sv, ok := v.(string); ok {
			return sv
		}
	}
	return def
}

// GetInt returns the int stored under key, or def if absent or not numeric.
func (s *State) GetInt(key string, def int) int {
	if v, ok := s.m[key]; ok {
		return asInt(v, def)
	}
	return def
}

// GetFloat returns the float64 stored under key, or def if absent or not
// numeric.
func (s *State) GetFloat(key string, def float64) float64 {
	if v, ok := s.m[key]; ok {
		return asFloat(v, def)
	}
	return def
}

// GetBool returns the bool stored under key, or def if absent or not a bool.
func (s *State) GetBool(key string, def bool) bool {
	if v, ok := s.m[key]; ok {
		return asBool(v, def)
	}
	return def
}

// Set stores value under key.
func (s *State) Set(key string, value any) { s.m[key] = value }

// SetDefault stores value under key only if the key is absent and reports
// whether it was stored. It is the analogue of Streamlit's common
// `if "k" not in st.session_state: st.session_state.k = v` idiom, which is how
// per-session state is seeded on the first of many reruns.
func (s *State) SetDefault(key string, value any) bool {
	if _, ok := s.m[key]; ok {
		return false
	}
	s.m[key] = value
	return true
}

// Has reports whether key is present, mirroring `key in st.session_state`.
func (s *State) Has(key string) bool {
	_, ok := s.m[key]
	return ok
}

// Len returns the number of keys held in the store.
func (s *State) Len() int { return len(s.m) }

// Keys returns the store's keys in ascending lexical order. The order is
// sorted rather than map order so that iterating state is deterministic across
// reruns.
func (s *State) Keys() []string {
	out := make([]string, 0, len(s.m))
	for k := range s.m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Delete removes key from the store.
func (s *State) Delete(key string) { delete(s.m, key) }

// Clear removes every key from the store.
func (s *State) Clear() { s.m = map[string]any{} }

// Session represents a single browser connection to the app. It embeds the
// root [Container], so every display and widget method may be called directly
// on the session (for example s.Title or s.Slider). A fresh Session is created
// for each new browser and reused for every rerun triggered by that browser.
//
// The app function receives a *Session and builds the page by calling methods
// on it. On each run the element tree is rebuilt from scratch while widget
// values and [State] persist, faithfully reproducing Streamlit's
// rerun-on-interaction model.
type Session struct {
	// The root container is embedded through an unexported alias so that all
	// display and widget methods are promoted onto Session while leaving the
	// exported Container method (from layout.go) reachable and unshadowed.
	*ctr

	// State is the persistent per-session key/value store.
	State *State

	id      string
	mu      sync.Mutex
	widgets map[string]any            // persisted widget values keyed by widget key
	clicked map[string]bool           // buttons/transient triggers fired for this run
	uploads map[string][]UploadedFile // files received per file-uploader key
	seen    map[string]struct{}       // widget keys registered during the current run
	counter int                       // auto-key ordinal counter, reset each run

	// lastSeen is the time of the most recent request for this session. It is
	// read and written by the server under the server's own lock (never by
	// app code) and drives idle eviction.
	lastSeen time.Time

	rootEl  *Element
	sidebar *Container
}

// ctr is an unexported alias of Container used solely for embedding in Session
// (see the Session definition for why).
type ctr = Container

// newSession creates a Session with a random identifier and empty state.
func newSession() *Session {
	s := &Session{
		State:    newState(),
		id:       randomID(),
		widgets:  map[string]any{},
		clicked:  map[string]bool{},
		uploads:  map[string][]UploadedFile{},
		seen:     map[string]struct{}{},
		lastSeen: time.Now(),
	}
	s.reset()
	return s
}

// ID returns the session's opaque identifier.
func (s *Session) ID() string { return s.id }

// Sidebar returns the container for the app's sidebar region. Elements added
// to it render in a fixed panel beside the main content.
func (s *Session) Sidebar() *Container { return s.sidebar }

// reset rebuilds the element tree for a new run and resets per-run counters.
// Widget values and [State] are intentionally preserved.
func (s *Session) reset() {
	sidebarEl := &Element{Type: "sidebar"}
	mainEl := &Element{Type: "main"}
	s.rootEl = &Element{Type: "root", Children: []*Element{sidebarEl, mainEl}}
	s.ctr = &Container{s: s, node: mainEl}
	s.sidebar = &Container{s: s, node: sidebarEl}
	s.counter = 0
	s.seen = make(map[string]struct{}, len(s.seen))
}

// maxRerunChain bounds how many times a single request may be restarted by
// [Session.Rerun]. Streamlit has no such bound because each rerun is a fresh
// script execution driven by the browser; here the loop is synchronous, so an
// app that unconditionally calls Rerun would otherwise spin forever inside one
// HTTP request.
const maxRerunChain = 32

// run executes the app function to produce a fresh element tree. Transient
// button clicks are cleared afterwards so a button reads true for exactly one
// run following its click, and widget state belonging to widgets that were not
// rendered by this run is discarded (see pruneStaleWidgets).
//
// If the app calls [Session.Rerun] the tree built so far is thrown away and the
// app function is executed again from the top, up to maxRerunChain times.
func (s *Session) run(app func(*Session)) *Element {
	for i := 0; ; i++ {
		s.reset()
		rerun := s.runApp(app)
		// Transient triggers fire for exactly one execution. Clearing them
		// inside the loop is what stops `if s.Button(x) { s.Rerun() }` from
		// restarting forever: the run started by Rerun sees the button as
		// unclicked, just as a fresh script run does in Streamlit.
		s.clicked = map[string]bool{}
		if !rerun || i >= maxRerunChain-1 {
			break
		}
	}
	s.pruneStaleWidgets()
	return s.rootEl
}

// runApp invokes the app function, recovering the sentinels panicked by
// [Session.Stop] and [Session.Rerun] so that halting or restarting a run
// mid-way is clean rather than fatal. Any other panic is re-raised unchanged.
// It reports whether the app asked for an immediate rerun.
func (s *Session) runApp(app func(*Session)) (rerun bool) {
	defer func() {
		if r := recover(); r != nil {
			switch r.(type) {
			case stopSignal:
				return
			case rerunSignal:
				rerun = true
				return
			}
			panic(r)
		}
	}()
	app(s)
	return false
}

// pruneStaleWidgets drops persisted state for widget keys that the run just
// finished did not render, mirroring Streamlit's behaviour: a widget hidden by
// a branch loses its value, so revealing it again shows the default rather than
// a stale value from an earlier, differently-shaped run.
//
// It also bounds memory. Widget keys arrive from the browser, so without this
// the per-session maps would grow without limit as a client posted events for
// keys the app never renders.
func (s *Session) pruneStaleWidgets() {
	for k := range s.widgets {
		if _, ok := s.seen[k]; !ok {
			delete(s.widgets, k)
		}
	}
	for k := range s.uploads {
		if _, ok := s.seen[k]; !ok {
			delete(s.uploads, k)
		}
	}
}

// widgetStateBytes returns the approximate size of the values held in the
// session's widget state. It is what the per-session byte budget
// (Options.MaxWidgetStateBytes) is measured against; the state is capped at a
// small number of entries, so recomputing it per event is cheap.
func (s *Session) widgetStateBytes() int64 {
	var total int64
	for k, v := range s.widgets {
		total += int64(len(k)) + valueSize(v)
	}
	return total
}

// valueSize approximates the memory held by a decoded JSON value. It is an
// estimate, not an exact accounting: what matters is that it grows with the
// input, so a client cannot make a large value look small. Nesting deeper than
// maxValueDepth is not descended into, which keeps the walk linear on
// adversarial input.
func valueSize(v any) int64 { return valueSizeAt(v, 0) }

// maxValueDepth bounds how deep valueSize descends into nested containers.
const maxValueDepth = 32

func valueSizeAt(v any, depth int) int64 {
	if depth > maxValueDepth {
		return 0
	}
	switch t := v.(type) {
	case nil:
		return 1
	case string:
		return int64(len(t))
	case bool:
		return 1
	case float64, int, int64:
		return 8
	case []any:
		total := int64(16)
		for _, e := range t {
			total += 16 + valueSizeAt(e, depth+1)
		}
		return total
	case map[string]any:
		total := int64(16)
		for k, e := range t {
			total += int64(len(k)) + 16 + valueSizeAt(e, depth+1)
		}
		return total
	default:
		return 16
	}
}

// key resolves the stable identity for a widget. A caller-supplied key is used
// verbatim; otherwise a deterministic ordinal key is generated. Provided the
// app's structure is stable across reruns, generated keys are stable too,
// which is what allows widget values to be restored on each run.
func (s *Session) key(userKey, typ string) string {
	if userKey != "" {
		return userKey
	}
	k := fmt.Sprintf("auto-%s-%d", typ, s.counter)
	s.counter++
	return k
}

// optKey extracts an optional trailing key argument from a variadic slice.
func optKey(key []string) string {
	if len(key) > 0 {
		return key[0]
	}
	return ""
}

// randomID returns a 16-byte hex identifier.
func randomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand should not fail; fall back to a fixed prefix.
		return "session-fallback"
	}
	return hex.EncodeToString(b)
}

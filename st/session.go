package st

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
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

// Set stores value under key.
func (s *State) Set(key string, value any) { s.m[key] = value }

// Delete removes key from the store.
func (s *State) Delete(key string) { delete(s.m, key) }

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
	counter int                       // auto-key ordinal counter, reset each run

	rootEl  *Element
	sidebar *Container
}

// ctr is an unexported alias of Container used solely for embedding in Session
// (see the Session definition for why).
type ctr = Container

// newSession creates a Session with a random identifier and empty state.
func newSession() *Session {
	s := &Session{
		State:   newState(),
		id:      randomID(),
		widgets: map[string]any{},
		clicked: map[string]bool{},
		uploads: map[string][]UploadedFile{},
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
}

// run executes the app function to produce a fresh element tree. Transient
// button clicks are cleared afterwards so a button reads true for exactly one
// run following its click.
func (s *Session) run(app func(*Session)) *Element {
	s.reset()
	s.runApp(app)
	s.clicked = map[string]bool{}
	return s.rootEl
}

// runApp invokes the app function, recovering the sentinel panicked by
// [Session.Stop] so that halting a run mid-way is clean rather than fatal. Any
// other panic is re-raised unchanged.
func (s *Session) runApp(app func(*Session)) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(stopSignal); ok {
				return
			}
			panic(r)
		}
	}()
	app(s)
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

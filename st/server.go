package st

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

//go:embed frontend/index.html frontend/app.js frontend/style.css
var frontendFS embed.FS

// Defaults applied by [Handler] and [Run]. Every one of them exists because the
// value it bounds is ultimately chosen by a remote client: without a ceiling a
// single caller could pin unbounded memory (sessions, uploaded bytes) on the
// server. Use [Options] with [HandlerWithOptions] to tune them.
const (
	// DefaultMaxSessions is the largest number of concurrent sessions kept in
	// memory. When the limit is reached the least recently used session is
	// evicted.
	DefaultMaxSessions = 1000
	// DefaultSessionIdleTimeout is how long a session survives without a
	// request before it is dropped.
	DefaultSessionIdleTimeout = 30 * time.Minute
	// DefaultMaxUploadBytes is the largest accepted body for a single
	// multipart upload request, across all of its file parts.
	DefaultMaxUploadBytes int64 = 32 << 20 // 32 MiB
	// DefaultMaxRequestBytes is the largest accepted body for POST /api/run.
	DefaultMaxRequestBytes int64 = 1 << 20 // 1 MiB
	// DefaultMaxWidgetEntries is the largest number of widget-state entries a
	// single session may retain.
	DefaultMaxWidgetEntries = 1024
	// DefaultMaxWidgetStateBytes is the largest total size of the values held
	// in one session's widget state.
	DefaultMaxWidgetStateBytes int64 = 1 << 20 // 1 MiB
)

// maxWidgetKeys bounds how many widget keys a single event may commit, and
// maxWidgetKeyLen bounds the length of each. Both come straight off the wire,
// so they are validated before anything is stored.
const (
	maxWidgetKeys   = 512
	maxWidgetKeyLen = 256
)

// defaultMultipartMemory is how much of a multipart body is buffered in memory
// before parts spill to temporary files. It mirrors net/http's own default.
const defaultMultipartMemory = 32 << 20

// Options configures the HTTP surface returned by [HandlerWithOptions]. The
// zero Options is valid and selects the documented default for every field.
type Options struct {
	// MaxSessions caps concurrently retained sessions; <= 0 selects
	// [DefaultMaxSessions].
	MaxSessions int
	// SessionIdleTimeout is how long an idle session is retained; <= 0
	// selects [DefaultSessionIdleTimeout].
	SessionIdleTimeout time.Duration
	// MaxUploadBytes caps a single upload request body; <= 0 selects
	// [DefaultMaxUploadBytes].
	MaxUploadBytes int64
	// MaxRequestBytes caps a single /api/run request body; <= 0 selects
	// [DefaultMaxRequestBytes].
	MaxRequestBytes int64
	// MaxWidgetEntries caps how many widget-state entries one session
	// retains; <= 0 selects [DefaultMaxWidgetEntries].
	MaxWidgetEntries int
	// MaxWidgetStateBytes caps the total size of one session's widget-state
	// values; <= 0 selects [DefaultMaxWidgetStateBytes].
	MaxWidgetStateBytes int64
	// AllowedOrigins lists extra browser origins ("https://app.example.com")
	// permitted to drive the app. Same-origin requests are always allowed;
	// requests carrying any other Origin are rejected with 403. Leave it nil
	// unless the app is deliberately embedded in another site.
	AllowedOrigins []string
	// AllowAllOrigins disables origin checking entirely. Only set it when the
	// app is deliberately public and stateless — with it on, any web page a
	// user visits can drive their session (cross-site request forgery).
	AllowAllOrigins bool
}

// withDefaults returns a copy of o with every unset field filled in.
func (o Options) withDefaults() Options {
	if o.MaxSessions <= 0 {
		o.MaxSessions = DefaultMaxSessions
	}
	if o.SessionIdleTimeout <= 0 {
		o.SessionIdleTimeout = DefaultSessionIdleTimeout
	}
	if o.MaxUploadBytes <= 0 {
		o.MaxUploadBytes = DefaultMaxUploadBytes
	}
	if o.MaxRequestBytes <= 0 {
		o.MaxRequestBytes = DefaultMaxRequestBytes
	}
	if o.MaxWidgetEntries <= 0 {
		o.MaxWidgetEntries = DefaultMaxWidgetEntries
	}
	if o.MaxWidgetStateBytes <= 0 {
		o.MaxWidgetStateBytes = DefaultMaxWidgetStateBytes
	}
	return o
}

// server holds the running app function and the live sessions.
type server struct {
	app      func(*Session)
	opts     Options
	mu       sync.Mutex
	sessions map[string]*Session
}

// event is a single widget interaction posted by the frontend. Form
// submissions additionally carry Form (the form's key) and Values (every staged
// widget value in that form), so the whole form commits atomically.
type event struct {
	Key    string         `json:"key"`
	Value  any            `json:"value"`
	Button bool           `json:"button"`
	Form   string         `json:"form,omitempty"`
	Values map[string]any `json:"values,omitempty"`
}

// runRequest is the body of POST /api/run.
type runRequest struct {
	SessionID string `json:"sessionId"`
	Event     *event `json:"event"`
}

// runResponse is the reply to POST /api/run.
type runResponse struct {
	SessionID string   `json:"sessionId"`
	Tree      *Element `json:"tree"`
}

// Handler returns the http.Handler that serves the app with the default
// [Options]. It is exposed so the app can be mounted inside a larger server or
// driven in tests; most callers should use [Run] instead.
func Handler(app func(*Session)) http.Handler {
	return HandlerWithOptions(app, Options{})
}

// HandlerWithOptions is [Handler] with explicit resource limits and origin
// policy. See [Options].
func HandlerWithOptions(app func(*Session), opts Options) http.Handler {
	s := &server{app: app, opts: opts.withDefaults(), sessions: map[string]*Session{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/run", s.handleRun)
	mux.HandleFunc("/api/upload", s.handleUpload)
	mux.HandleFunc("/app.js", serveAsset("frontend/app.js", "text/javascript"))
	mux.HandleFunc("/style.css", serveAsset("frontend/style.css", "text/css"))
	mux.HandleFunc("/", s.handleIndex)
	return &appHandler{srv: s, mux: mux}
}

// appHandler is the concrete http.Handler returned by [HandlerWithOptions]. It
// keeps the routing mux and the server side by side so the session table stays
// reachable from the handler value (which the package's own tests rely on)
// rather than being captured only inside closures.
type appHandler struct {
	srv *server
	mux *http.ServeMux
}

// ServeHTTP routes a request to the app's endpoints and embedded assets.
func (h *appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

// Run starts an HTTP server that serves the app on addr (for example
// ":8501"). Each browser connection is assigned its own [Session]; widget
// interactions rerun the app function and push the updated element tree back.
// Run blocks until the server exits.
func Run(app func(*Session), addr string) error {
	return RunWithOptions(app, addr, Options{})
}

// RunWithOptions is [Run] with explicit resource limits and origin policy.
func RunWithOptions(app func(*Session), addr string, opts Options) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           HandlerWithOptions(app, opts),
		ReadHeaderTimeout: 10 * time.Second,
	}
	return srv.ListenAndServe()
}

// handleIndex serves the single-page frontend for any non-API path.
func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	serveAsset("frontend/index.html", "text/html; charset=utf-8")(w, r)
}

// serveAsset returns a handler that serves an embedded asset with the given
// content type.
func serveAsset(name, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := frontendFS.ReadFile(name)
		if err != nil {
			http.Error(w, "asset not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write(b)
	}
}

// originAllowed reports whether a state-changing request may proceed.
//
// The session identifier is the only credential needed to drive a session, so
// an unchecked POST endpoint lets any page a user visits create sessions and
// replay widget events — the HTTP analogue of cross-site WebSocket hijacking.
// A request with no Origin header (a non-browser client such as curl, or a Go
// test) is allowed; a request that declares an Origin must match the host it
// was sent to, or appear in Options.AllowedOrigins.
func (s *server) originAllowed(r *http.Request) bool {
	if s.opts.AllowAllOrigins {
		return true
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	if origin == "null" {
		// What sandboxed iframes and file:// pages send. Treat it as unknown,
		// never as same-origin.
		return false
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	if strings.EqualFold(u.Host, r.Host) {
		return true
	}
	for _, allowed := range s.opts.AllowedOrigins {
		if strings.EqualFold(strings.TrimSuffix(allowed, "/"), strings.TrimSuffix(origin, "/")) {
			return true
		}
	}
	return false
}

// handleRun creates or resumes a session, applies an optional widget event,
// reruns the app function, and returns the fresh element tree as JSON.
func (s *server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.originAllowed(r) {
		http.Error(w, "forbidden origin", http.StatusForbidden)
		return
	}
	var req runRequest
	body := http.MaxBytesReader(w, r.Body, s.opts.MaxRequestBytes)
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		// An over-long body is reported as 413 rather than 400 so the caller
		// can tell "too big" from "malformed".
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	sess := s.session(req.SessionID)

	// Serialise runs of a given session.
	sess.mu.Lock()
	if req.Event != nil {
		if err := applyEvent(sess, req.Event, s.opts); err != nil {
			sess.mu.Unlock()
			http.Error(w, "widget state too large", http.StatusRequestEntityTooLarge)
			return
		}
	}
	tree := sess.run(s.app)
	sess.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(runResponse{SessionID: sess.id, Tree: tree})
}

// handleUpload receives a multipart file upload for a file-uploader widget,
// stores the files in the session keyed by the widget key, reruns the app, and
// returns the fresh element tree. The request must be a multipart POST with a
// "session" and "key" form field and one or more "files" parts.
//
// The request body is capped at Options.MaxUploadBytes. Without the cap a
// caller could stream an arbitrarily large body: ParseMultipartForm's argument
// bounds only what is buffered in memory, spilling the remainder to temporary
// files on disk.
func (s *server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.originAllowed(r) {
		http.Error(w, "forbidden origin", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, s.opts.MaxUploadBytes)
	// Spilled parts are backed by temporary files; remove them before
	// returning either way.
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()
	if err := r.ParseMultipartForm(defaultMultipartMemory); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			http.Error(w, "upload too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "malformed upload", http.StatusBadRequest)
		return
	}

	key := r.FormValue("key")
	if !validWidgetKey(key) {
		http.Error(w, "missing or oversized key", http.StatusBadRequest)
		return
	}
	sess := s.session(r.FormValue("session"))

	var files []UploadedFile
	if r.MultipartForm != nil {
		for _, fh := range r.MultipartForm.File["files"] {
			f, err := fh.Open()
			if err != nil {
				continue
			}
			data, err := io.ReadAll(io.LimitReader(f, s.opts.MaxUploadBytes))
			_ = f.Close()
			if err != nil {
				continue
			}
			files = append(files, UploadedFile{Name: fh.Filename, Size: len(data), Data: data})
		}
	}

	sess.mu.Lock()
	sess.uploads[key] = files
	tree := sess.run(s.app)
	sess.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(runResponse{SessionID: sess.id, Tree: tree})
}

// session returns the session for id, creating a new one when id is empty or
// unknown. Every call refreshes the session's last-seen time and reaps sessions
// that have gone idle or pushed the table past its limit.
func (s *server) session(id string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.evictLocked(now)

	if id != "" {
		if sess, ok := s.sessions[id]; ok {
			sess.lastSeen = now
			return sess
		}
	}
	sess := newSession()
	sess.lastSeen = now
	s.sessions[sess.id] = sess
	return sess
}

// evictLocked drops idle sessions and, if the table is still at capacity, the
// least recently used ones, leaving room for one new session. The caller must
// hold s.mu.
//
// A session is created for every request presenting an unknown identifier, so
// without eviction the table is an unbounded allocation driven entirely by
// remote callers.
func (s *server) evictLocked(now time.Time) {
	for id, sess := range s.sessions {
		if now.Sub(sess.lastSeen) > s.opts.SessionIdleTimeout {
			delete(s.sessions, id)
		}
	}
	if len(s.sessions) < s.opts.MaxSessions {
		return
	}
	ids := make([]string, 0, len(s.sessions))
	for id := range s.sessions {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return s.sessions[ids[i]].lastSeen.Before(s.sessions[ids[j]].lastSeen)
	})
	for _, id := range ids[:len(s.sessions)-s.opts.MaxSessions+1] {
		delete(s.sessions, id)
	}
}

// sessionCount reports how many sessions the handler currently retains.
func (s *server) sessionCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sessions)
}

// errWidgetStateFull reports that an event would push a session's widget state
// past Options.MaxWidgetEntries or Options.MaxWidgetStateBytes. The handler
// turns it into 413 rather than allocating.
var errWidgetStateFull = errors.New("widget state limit exceeded")

// applyEvent records a widget interaction on the session. A form submission
// carries a Values map that commits every staged widget in the form at once;
// button clicks are transient (true for a single run); all other widgets
// persist their value.
//
// Everything here arrives from the browser, so every dimension it can grow is
// bounded: empty and over-long keys and oversized form batches are ignored,
// and a value that would push the session past its entry or byte budget is
// refused with errWidgetStateFull instead of being stored. Whatever the app
// does not re-render is dropped at the end of the run (see pruneStaleWidgets),
// so this path cannot be used to accumulate state either.
func applyEvent(sess *Session, ev *event, opts Options) error {
	// Commit any batched form values first so they are visible on the rerun
	// triggered by the submit button.
	if len(ev.Values) <= maxWidgetKeys {
		for _, k := range sortedKeys(ev.Values) {
			if !validWidgetKey(k) {
				continue
			}
			if err := setWidget(sess, k, ev.Values[k], opts); err != nil {
				return err
			}
		}
	}
	if !validWidgetKey(ev.Key) {
		return nil
	}
	if ev.Button {
		sess.clicked[ev.Key] = true
		return nil
	}
	return setWidget(sess, ev.Key, ev.Value, opts)
}

// sortedKeys returns m's keys in lexical order, so a batch that trips a limit
// does so deterministically rather than depending on map iteration order.
func sortedKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// setWidget stores one client-supplied widget value, refusing it when the
// session's widget state is already at its entry or byte budget. Replacing an
// existing key is always allowed to proceed as far as the entry count goes;
// only the byte budget applies, computed with the old value removed.
func setWidget(sess *Session, key string, value any, opts Options) error {
	_, replacing := sess.widgets[key]
	if !replacing && len(sess.widgets) >= opts.MaxWidgetEntries {
		return fmt.Errorf("%w: %d entries", errWidgetStateFull, len(sess.widgets))
	}
	size := valueSize(value)
	if size > opts.MaxWidgetStateBytes {
		return fmt.Errorf("%w: value of %d bytes", errWidgetStateFull, size)
	}
	total := sess.widgetStateBytes()
	if replacing {
		total -= valueSize(sess.widgets[key])
	}
	if total+size > opts.MaxWidgetStateBytes {
		return fmt.Errorf("%w: %d bytes held, %d more requested",
			errWidgetStateFull, total, size)
	}
	sess.widgets[key] = value
	return nil
}

// validWidgetKey reports whether a client-supplied widget key is usable.
func validWidgetKey(k string) bool { return k != "" && len(k) <= maxWidgetKeyLen }

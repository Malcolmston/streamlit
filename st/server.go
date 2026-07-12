package st

import (
	"embed"
	"encoding/json"
	"io"
	"net/http"
	"sync"
)

//go:embed frontend/index.html frontend/app.js frontend/style.css
var frontendFS embed.FS

// server holds the running app function and the live sessions.
type server struct {
	app      func(*Session)
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

// Handler returns the http.Handler that serves the app. It is exposed so the
// app can be mounted inside a larger server or driven in tests; most callers
// should use [Run] instead.
func Handler(app func(*Session)) http.Handler {
	s := &server{app: app, sessions: map[string]*Session{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/run", s.handleRun)
	mux.HandleFunc("/api/upload", s.handleUpload)
	mux.HandleFunc("/app.js", serveAsset("frontend/app.js", "text/javascript"))
	mux.HandleFunc("/style.css", serveAsset("frontend/style.css", "text/css"))
	mux.HandleFunc("/", s.handleIndex)
	return mux
}

// Run starts an HTTP server that serves the app on addr (for example
// ":8501"). Each browser connection is assigned its own [Session]; widget
// interactions rerun the app function and push the updated element tree back.
// Run blocks until the server exits.
func Run(app func(*Session), addr string) error {
	return http.ListenAndServe(addr, Handler(app))
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

// handleRun creates or resumes a session, applies an optional widget event,
// reruns the app function, and returns the fresh element tree as JSON.
func (s *server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req runRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	sess := s.session(req.SessionID)

	// Serialise runs of a given session.
	sess.mu.Lock()
	if req.Event != nil {
		applyEvent(sess, req.Event)
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
func (s *server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	sess := s.session(r.FormValue("session"))
	key := r.FormValue("key")
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	var files []UploadedFile
	if r.MultipartForm != nil {
		for _, fh := range r.MultipartForm.File["files"] {
			f, err := fh.Open()
			if err != nil {
				continue
			}
			data, err := io.ReadAll(f)
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
// unknown.
func (s *server) session(id string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id != "" {
		if sess, ok := s.sessions[id]; ok {
			return sess
		}
	}
	sess := newSession()
	s.sessions[sess.id] = sess
	return sess
}

// applyEvent records a widget interaction on the session. A form submission
// carries a Values map that commits every staged widget in the form at once;
// button clicks are transient (true for a single run); all other widgets
// persist their value.
func applyEvent(sess *Session, ev *event) {
	// Commit any batched form values first so they are visible on the rerun
	// triggered by the submit button.
	for k, v := range ev.Values {
		sess.widgets[k] = v
	}
	if ev.Key == "" {
		return
	}
	if ev.Button {
		sess.clicked[ev.Key] = true
		return
	}
	sess.widgets[ev.Key] = ev.Value
}

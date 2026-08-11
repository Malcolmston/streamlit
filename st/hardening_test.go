package st

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Rerun semantics: state and widget identity across two (and more) executions.
// ---------------------------------------------------------------------------

// TestRerunRestartsAppFromTop checks the defining behaviour of st.rerun: the
// elements added before the call are discarded and the app runs again with the
// state change visible.
func TestRerunRestartsAppFromTop(t *testing.T) {
	runs := 0
	app := func(s *Session) {
		runs++
		s.Text("pass-" + s.State.GetString("phase", "one"))
		if s.State.GetString("phase", "one") == "one" {
			s.State.Set("phase", "two")
			s.Rerun()
		}
		s.Text("tail")
	}
	s := newSession()
	tree := s.run(app)

	if runs != 2 {
		t.Fatalf("app executed %d times, want 2", runs)
	}
	texts := findAll(tree, "text")
	if len(texts) != 2 {
		t.Fatalf("tree kept %d text elements, want 2 (first pass discarded)", len(texts))
	}
	if texts[0].Props["text"] != "pass-two" || texts[1].Props["text"] != "tail" {
		t.Errorf("texts = %v / %v", texts[0].Props["text"], texts[1].Props["text"])
	}
}

// TestRerunIsBoundedAndSessionStaysUsable checks that an app that always calls
// Rerun terminates instead of spinning forever, and leaves the session usable.
func TestRerunIsBoundedAndSessionStaysUsable(t *testing.T) {
	runs := 0
	s := newSession()
	done := make(chan *Element, 1)
	go func() {
		done <- s.run(func(s *Session) {
			runs++
			s.Text("again")
			s.Rerun()
		})
	}()
	select {
	case tree := <-done:
		if runs != maxRerunChain {
			t.Errorf("app executed %d times, want the %d-run cap", runs, maxRerunChain)
		}
		if find(tree, "text") == nil {
			t.Error("final tree should hold the last run's elements")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("unbounded Rerun did not terminate")
	}

	tree := s.run(func(s *Session) { s.Text("ok") })
	if find(tree, "text").Props["text"] != "ok" {
		t.Error("session not reusable after a bounded Rerun chain")
	}
}

// TestRerunAfterButtonClickDoesNotLoop pins the interaction between transient
// triggers and Rerun: the run started by Rerun must see the button as
// unclicked, exactly as a fresh script run does in Streamlit. Otherwise the
// most common Rerun idiom (`if Button { mutate; Rerun }`) would spin.
func TestRerunAfterButtonClickDoesNotLoop(t *testing.T) {
	runs := 0
	app := func(s *Session) {
		runs++
		if s.Button("save", "save") {
			s.State.Set("saved", true)
			s.Rerun()
		}
		s.Text(fmt.Sprint(s.State.GetBool("saved", false)))
	}
	s := newSession()
	s.clicked["save"] = true
	tree := s.run(app)

	if runs != 2 {
		t.Fatalf("app executed %d times, want exactly 2 (click run + rerun)", runs)
	}
	if find(tree, "text").Props["text"] != "true" {
		t.Error("rerun did not observe the state written before it")
	}
}

// TestStaleWidgetStateIsDropped is the classic Streamlit correctness case: a
// widget hidden by a branch loses its value, so revealing it later shows the
// default rather than a value left over from an earlier, differently shaped
// run. It exercises three successive runs.
func TestStaleWidgetStateIsDropped(t *testing.T) {
	app := func(s *Session) {
		if s.State.GetBool("show", true) {
			s.TextInput("name", "anon", "name")
		}
	}
	s := newSession()

	// Run 1: widget is rendered, browser sets a value.
	s.run(app)
	s.widgets["name"] = "alice"
	if got := find(s.run(app), "text_input").Props["value"]; got != "alice" {
		t.Fatalf("run 2 value = %v, want alice (value must survive a plain rerun)", got)
	}

	// Run 3: the widget is no longer rendered, so its state is discarded.
	s.State.Set("show", false)
	s.run(app)
	if _, ok := s.widgets["name"]; ok {
		t.Error("state for an unrendered widget survived the run")
	}

	// Run 4: revealing it again shows the default, not the stale value.
	s.State.Set("show", true)
	if got := find(s.run(app), "text_input").Props["value"]; got != "anon" {
		t.Errorf("revealed widget = %v, want the default anon", got)
	}
}

// TestStaleWidgetPruningBoundsClientDrivenState checks the resource-consumption
// side of the same mechanism: widget keys arrive from the browser, and keys the
// app never renders must not accumulate.
func TestStaleWidgetPruningBoundsClientDrivenState(t *testing.T) {
	h := Handler(func(s *Session) { s.TextInput("real", "", "real") })
	first := post(t, h, runRequest{})
	post(t, h, runRequest{SessionID: first.SessionID, Event: &event{Key: "real", Value: "kept"}})

	for i := 0; i < 200; i++ {
		post(t, h, runRequest{
			SessionID: first.SessionID,
			Event:     &event{Key: fmt.Sprintf("junk-%d", i), Value: strings.Repeat("x", 100)},
		})
	}

	sess := sessionOf(t, h, first.SessionID)
	sess.mu.Lock()
	defer sess.mu.Unlock()
	if got := len(sess.widgets); got != 1 {
		t.Errorf("session retained %d widget keys, want 1 (only the rendered one)", got)
	}
	if got := sess.widgets["real"]; got != "kept" {
		t.Errorf("the rendered widget lost its value: %v", got)
	}
}

// sessionOf reaches into a handler's session table. It exists because the
// resource-bounding tests need to assert on server-internal state.
func sessionOf(t *testing.T, h http.Handler, id string) *Session {
	t.Helper()
	srv := serverOf(t, h)
	srv.mu.Lock()
	defer srv.mu.Unlock()
	sess, ok := srv.sessions[id]
	if !ok {
		t.Fatalf("session %q not found", id)
	}
	return sess
}

// ---------------------------------------------------------------------------
// Session isolation and concurrency.
// ---------------------------------------------------------------------------

// postAsync is post for use inside goroutines, where t.Fatalf is illegal: it
// returns an error instead of aborting the calling goroutine.
func postAsync(h http.Handler, req runRequest) (runResponse, error) {
	body, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPost, "/api/run", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var resp runResponse
	if w.Code != http.StatusOK {
		return resp, fmt.Errorf("status = %d", w.Code)
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// TestConcurrentSessionsStayIsolated drives many sessions at once through the
// real HTTP handler. Each writes a distinct value into its own widget and
// session state, and must read back exactly what it wrote — one user's state
// leaking into another's is the most severe failure mode for this package.
// Run with -race to also cover the shared maps.
func TestConcurrentSessionsStayIsolated(t *testing.T) {
	app := func(s *Session) {
		v := s.TextInput("who", "anon", "who")
		s.State.Set("seen", v)
		s.Text(v)
	}
	h := Handler(app)

	const n = 32
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			want := fmt.Sprintf("user-%d", i)
			first, err := postAsync(h, runRequest{})
			if err != nil {
				errs <- err
				return
			}
			if _, err := postAsync(h, runRequest{SessionID: first.SessionID,
				Event: &event{Key: "who", Value: want}}); err != nil {
				errs <- err
				return
			}
			// Two further reruns: the value must be stable, and must never be
			// another goroutine's.
			for r := 0; r < 2; r++ {
				got, err := postAsync(h, runRequest{SessionID: first.SessionID})
				if err != nil {
					errs <- err
					return
				}
				if v := find(got.Tree, "text").Props["text"]; v != want {
					errs <- fmt.Errorf("rerun %d: got %v, want %v", r, v, want)
					return
				}
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

// TestConcurrentRunsOnOneSessionAreSerialised hammers a single session from
// several goroutines. The counter increment inside the app is unsynchronised,
// so it is only safe if the server serialises runs of a session.
func TestConcurrentRunsOnOneSessionAreSerialised(t *testing.T) {
	app := func(s *Session) {
		s.State.Set("n", s.State.GetInt("n", 0)+1)
		s.Text(fmt.Sprint(s.State.GetInt("n", 0)))
	}
	h := Handler(app)
	first := post(t, h, runRequest{})

	const n = 24
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = postAsync(h, runRequest{SessionID: first.SessionID})
		}()
	}
	wg.Wait()

	sess := sessionOf(t, h, first.SessionID)
	sess.mu.Lock()
	got := sess.State.GetInt("n", 0)
	sess.mu.Unlock()
	if got != n+1 { // +1 for the initial connect
		t.Errorf("counter = %d, want %d (a lost update means runs were not serialised)", got, n+1)
	}
}

// ---------------------------------------------------------------------------
// Origin policy.
// ---------------------------------------------------------------------------

// serverOf recovers the concrete *server behind a Handler; used by tests that
// assert on server-internal state such as the session table.
func serverOf(t *testing.T, h http.Handler) *server {
	t.Helper()
	ah, ok := h.(*appHandler)
	if !ok {
		t.Fatalf("handler is %T, not one created by Handler/HandlerWithOptions", h)
	}
	return ah.srv
}

// postOrigin issues POST /api/run with an explicit Origin header and returns
// the status code.
func postOrigin(h http.Handler, host, origin string) int {
	body, _ := json.Marshal(runRequest{})
	r := httptest.NewRequest(http.MethodPost, "/api/run", bytes.NewReader(body))
	r.Host = host
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w.Code
}

func TestOriginPolicy(t *testing.T) {
	app := func(s *Session) { s.Text("hi") }
	plain := Handler(app)
	withAllow := HandlerWithOptions(app, Options{AllowedOrigins: []string{"https://embed.example.com"}})
	open := HandlerWithOptions(app, Options{AllowAllOrigins: true})

	cases := []struct {
		name   string
		h      http.Handler
		host   string
		origin string
		want   int
	}{
		{"no origin (curl, tests)", plain, "app.example.com", "", http.StatusOK},
		{"same origin", plain, "app.example.com", "https://app.example.com", http.StatusOK},
		{"same origin, other scheme", plain, "app.example.com", "http://app.example.com", http.StatusOK},
		{"same host, different port", plain, "app.example.com:8501", "http://app.example.com:8501", http.StatusOK},
		{"cross site", plain, "app.example.com", "https://evil.example.net", http.StatusForbidden},
		{"port mismatch", plain, "app.example.com:8501", "http://app.example.com:9999", http.StatusForbidden},
		{"suffix confusion", plain, "app.example.com", "https://notapp.example.com", http.StatusForbidden},
		{"null origin (sandboxed frame)", plain, "app.example.com", "null", http.StatusForbidden},
		{"garbage origin", plain, "app.example.com", "::not a url::", http.StatusForbidden},
		{"listed origin", withAllow, "app.example.com", "https://embed.example.com", http.StatusOK},
		{"unlisted origin", withAllow, "app.example.com", "https://other.example.com", http.StatusForbidden},
		{"checks disabled", open, "app.example.com", "https://evil.example.net", http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := postOrigin(tc.h, tc.host, tc.origin); got != tc.want {
				t.Errorf("status = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestUploadRejectsForbiddenOrigin(t *testing.T) {
	h := Handler(func(s *Session) { s.FileUploader("f", "f") })
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("key", "f")
	_ = mw.Close()

	r := httptest.NewRequest(http.MethodPost, "/api/upload", &buf)
	r.Host = "app.example.com"
	r.Header.Set("Origin", "https://evil.example.net")
	r.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("cross-site upload = %d, want 403", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Resource bounds.
// ---------------------------------------------------------------------------

func TestSessionTableIsBounded(t *testing.T) {
	h := HandlerWithOptions(func(s *Session) {}, Options{MaxSessions: 5})
	srv := serverOf(t, h)
	for i := 0; i < 50; i++ {
		post(t, h, runRequest{}) // every request presents no id: a new session
	}
	if got := srv.sessionCount(); got > 5 {
		t.Errorf("retained %d sessions, want at most 5", got)
	}
}

func TestIdleSessionsAreEvicted(t *testing.T) {
	h := HandlerWithOptions(func(s *Session) {}, Options{SessionIdleTimeout: time.Nanosecond})
	srv := serverOf(t, h)
	first := post(t, h, runRequest{})
	time.Sleep(time.Millisecond)
	second := post(t, h, runRequest{SessionID: first.SessionID})
	if second.SessionID == first.SessionID {
		t.Error("an idle session should have been reaped, yielding a fresh id")
	}
	if got := srv.sessionCount(); got != 1 {
		t.Errorf("retained %d sessions, want 1", got)
	}
}

func TestActiveSessionSurvivesEviction(t *testing.T) {
	h := HandlerWithOptions(func(s *Session) { s.TextInput("v", "def", "v") },
		Options{MaxSessions: 4, SessionIdleTimeout: time.Hour})
	mine := post(t, h, runRequest{})
	post(t, h, runRequest{SessionID: mine.SessionID, Event: &event{Key: "v", Value: "kept"}})

	// Two other sessions connect, staying under the cap.
	post(t, h, runRequest{})
	post(t, h, runRequest{})

	again := post(t, h, runRequest{SessionID: mine.SessionID})
	if again.SessionID != mine.SessionID {
		t.Fatal("active session was evicted")
	}
	if got := find(again.Tree, "text_input").Props["value"]; got != "kept" {
		t.Errorf("value = %v, want kept", got)
	}
}

func TestOversizedRunBodyRejected(t *testing.T) {
	h := HandlerWithOptions(func(s *Session) {}, Options{MaxRequestBytes: 64})
	r := httptest.NewRequest(http.MethodPost, "/api/run", strings.NewReader(
		`{"sessionId":"`+strings.Repeat("a", 4096)+`"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413", w.Code)
	}
}

func TestOversizedUploadRejected(t *testing.T) {
	h := HandlerWithOptions(func(s *Session) { s.FileUploader("f", "f") },
		Options{MaxUploadBytes: 1024})

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("key", "f")
	fw, _ := mw.CreateFormFile("files", "big.bin")
	_, _ = fw.Write(bytes.Repeat([]byte("x"), 8192))
	_ = mw.Close()

	r := httptest.NewRequest(http.MethodPost, "/api/upload", &buf)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413 for an oversized upload", w.Code)
	}
}

// TestWidgetStateEntryCapRejected checks the per-session ceiling on the number
// of widget-state entries: the entry that would exceed it is refused with 413
// instead of being allocated.
func TestWidgetStateEntryCapRejected(t *testing.T) {
	// The app renders every key it is asked about, so pruning cannot be what
	// bounds the state here — only the cap can.
	h := HandlerWithOptions(func(s *Session) {
		for _, k := range s.State.Keys() {
			s.TextInput(k, "", k)
		}
	}, Options{MaxWidgetEntries: 4})

	first, err := postAsync(h, runRequest{})
	if err != nil {
		t.Fatal(err)
	}
	sid := first.SessionID
	for i := 0; i < 4; i++ {
		key := fmt.Sprintf("k%d", i)
		// Declare the key before the event so the app renders it and pruning
		// keeps it: the cap, not pruning, is what this test exercises.
		sessionOf(t, h, sid).State.Set(key, true)
		if _, err := postAsync(h, runRequest{SessionID: sid, Event: &event{Key: key, Value: "v"}}); err != nil {
			t.Fatalf("event %d: %v", i, err)
		}
	}
	if got := len(sessionOf(t, h, sid).widgets); got != 4 {
		t.Fatalf("session holds %d widget entries before the cap is hit, want 4", got)
	}

	code := postStatus(h, runRequest{SessionID: sid, Event: &event{Key: "one-too-many", Value: "v"}})
	if code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413 for the entry past the cap", code)
	}
	if got := len(sessionOf(t, h, sid).widgets); got > 4 {
		t.Errorf("session holds %d widget entries, want at most 4", got)
	}
}

// TestWidgetStateByteCapRejected checks the per-session ceiling on the total
// size of widget values: a value small enough for the request-body limit but
// large enough to blow the state budget is refused.
func TestWidgetStateByteCapRejected(t *testing.T) {
	h := HandlerWithOptions(func(s *Session) { s.TextInput("v", "", "v") },
		Options{MaxWidgetStateBytes: 256})

	resp, err := postAsync(h, runRequest{})
	if err != nil {
		t.Fatal(err)
	}
	sid := resp.SessionID

	if code := postStatus(h, runRequest{SessionID: sid,
		Event: &event{Key: "v", Value: strings.Repeat("x", 128)}}); code != http.StatusOK {
		t.Fatalf("status = %d for a value inside the budget, want 200", code)
	}
	if code := postStatus(h, runRequest{SessionID: sid,
		Event: &event{Key: "v", Value: strings.Repeat("x", 4096)}}); code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d for a value past the budget, want 413", code)
	}
	if got := sessionOf(t, h, sid).widgetStateBytes(); got > 256 {
		t.Errorf("widget state holds %d bytes, want at most 256", got)
	}
	// The value stored before the refusal is untouched.
	if got := sessionOf(t, h, sid).widgets["v"]; got != strings.Repeat("x", 128) {
		t.Errorf("the accepted value was disturbed by the refusal: %v", got)
	}
}

// TestNormalSessionIsUnaffectedByLimits is the positive case: an everyday
// session, request and widget interaction stay well inside every default bound.
func TestNormalSessionIsUnaffectedByLimits(t *testing.T) {
	h := Handler(func(s *Session) {
		s.Title("hello")
		name := s.TextInput("Name", "world", "name")
		s.Text("hi " + name)
	})
	first := post(t, h, runRequest{})
	second := post(t, h, runRequest{SessionID: first.SessionID,
		Event: &event{Key: "name", Value: "alice"}})
	if second.SessionID != first.SessionID {
		t.Error("a normal session was not resumed")
	}
	if got := find(second.Tree, "text").Props["text"]; got != "hi alice" {
		t.Errorf("text = %v, want \"hi alice\"", got)
	}
	if got := sessionOf(t, h, second.SessionID).widgetStateBytes(); got == 0 {
		t.Error("the widget value was not stored at all")
	}
}

// postStatus issues POST /api/run and returns only the status code.
func postStatus(h http.Handler, req runRequest) int {
	body, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPost, "/api/run", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w.Code
}

func TestApplyEventRejectsAbusiveKeys(t *testing.T) {
	s := newSession()
	long := strings.Repeat("k", maxWidgetKeyLen+1)

	_ = applyEvent(s, &event{Key: long, Value: "x"}, Options{}.withDefaults())
	if _, ok := s.widgets[long]; ok {
		t.Error("an over-long widget key was stored")
	}

	_ = applyEvent(s, &event{Key: "", Value: "x"}, Options{}.withDefaults())
	if len(s.widgets) != 0 {
		t.Errorf("empty key stored something: %v", s.widgets)
	}

	batch := map[string]any{}
	for i := 0; i <= maxWidgetKeys; i++ {
		batch[fmt.Sprintf("k%d", i)] = i
	}
	_ = applyEvent(s, &event{Key: "submit", Button: true, Values: batch}, Options{}.withDefaults())
	if len(s.widgets) != 0 {
		t.Errorf("an oversized form batch stored %d keys, want 0", len(s.widgets))
	}
	if !s.clicked["submit"] {
		t.Error("the submit click itself should still register")
	}
}

// ---------------------------------------------------------------------------
// Cache.
// ---------------------------------------------------------------------------

func TestCacheRunsComputeOnceUnderConcurrency(t *testing.T) {
	CacheClear()
	var mu sync.Mutex
	calls := 0
	start := make(chan struct{})

	const n = 16
	var wg sync.WaitGroup
	results := make([]int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s := newSession()
			<-start
			results[i] = s.Cache("shared", func() any {
				mu.Lock()
				calls++
				mu.Unlock()
				time.Sleep(5 * time.Millisecond) // widen the race window
				return 7
			}).(int)
		}(i)
	}
	close(start)
	wg.Wait()

	if calls != 1 {
		t.Errorf("compute ran %d times, want 1 (concurrent callers must share one flight)", calls)
	}
	for i, got := range results {
		if got != 7 {
			t.Errorf("caller %d got %v, want 7", i, got)
		}
	}
}

func TestCacheEvictsOldestWhenFull(t *testing.T) {
	CacheClear()
	CacheSetMaxEntries(3)
	defer CacheSetMaxEntries(0)

	s := newSession()
	for i := 0; i < 10; i++ {
		i := i
		s.Cache(fmt.Sprintf("k%d", i), func() any { return i })
	}
	if got := CacheLen(); got != 3 {
		t.Fatalf("cache holds %d entries, want 3", got)
	}
	// The three most recent keys survive; the oldest are gone.
	for _, k := range []string{"k7", "k8", "k9"} {
		if !CacheDelete(k) {
			t.Errorf("expected %s to still be cached", k)
		}
	}
}

func TestCacheDeleteAndLen(t *testing.T) {
	CacheClear()
	s := newSession()
	calls := 0
	compute := func() any { calls++; return calls }

	s.Cache("k", compute)
	if CacheLen() != 1 {
		t.Fatalf("len = %d, want 1", CacheLen())
	}
	if !CacheDelete("k") {
		t.Error("CacheDelete should report the entry was present")
	}
	if CacheDelete("k") {
		t.Error("CacheDelete on a missing key should report false")
	}
	if s.Cache("k", compute).(int) != 2 {
		t.Error("delete did not force a recompute")
	}
	CacheClear()
	if CacheLen() != 0 {
		t.Errorf("len after clear = %d", CacheLen())
	}
}

func TestCacheResourceIsASingleton(t *testing.T) {
	CacheResourceClear()
	defer CacheResourceClear()

	type conn struct{ id int }
	var mu sync.Mutex
	made := 0

	const n = 12
	var wg sync.WaitGroup
	got := make([]*conn, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s := newSession()
			got[i] = s.CacheResource("db", func() any {
				mu.Lock()
				made++
				c := &conn{id: made}
				mu.Unlock()
				return c
			}).(*conn)
		}(i)
	}
	wg.Wait()

	if made != 1 {
		t.Errorf("resource built %d times, want 1", made)
	}
	for i := 1; i < n; i++ {
		if got[i] != got[0] {
			t.Fatalf("caller %d received a different instance", i)
		}
	}

	// Resources are unaffected by data-cache clearing, and are never evicted.
	CacheClear()
	CacheSetMaxEntries(1)
	defer CacheSetMaxEntries(0)
	s := newSession()
	if s.CacheResource("db", func() any { return &conn{id: -1} }).(*conn) != got[0] {
		t.Error("CacheClear or eviction discarded a resource")
	}
}

func TestCachePanicDoesNotWedgeOtherCallers(t *testing.T) {
	CacheClear()
	s := newSession()

	func() {
		defer func() { _ = recover() }()
		s.Cache("boom", func() any { panic("compute failed") })
	}()

	done := make(chan any, 1)
	go func() { done <- s.Cache("boom", func() any { return "recovered" }) }()
	select {
	case got := <-done:
		if got != "recovered" {
			t.Errorf("got %v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("a panicking computation left the key permanently in flight")
	}
}

// ---------------------------------------------------------------------------
// Chart robustness against non-finite input.
// ---------------------------------------------------------------------------

func TestChartsSkipNonFiniteValues(t *testing.T) {
	nan, pinf, ninf := math.NaN(), math.Inf(1), math.Inf(-1)
	svgs := map[string]string{
		"line":      renderChart("line", [][]float64{{1, nan, 3}}, 640, 320),
		"area":      renderChart("area", [][]float64{{1, pinf, 3}}, 640, 320),
		"bar":       renderChart("bar", [][]float64{{1, ninf, 3}}, 640, 320),
		"all-nan":   renderChart("line", [][]float64{{nan, nan}}, 640, 320),
		"scatter":   renderScatter([]float64{1, nan, 3}, []float64{1, 2, nan}, 640, 320),
		"pie":       renderPie([]float64{1, nan, pinf, 2}, []string{"a", "b", "c", "d"}, 640, 320),
		"histogram": renderHistogram([]float64{1, nan, 2, pinf, 3}, 4, 640, 320),
		"map":       renderMap([]MapPoint{{Lat: 1, Lng: 2}, {Lat: nan, Lng: 0}}, 640, 320),
	}
	for name, svg := range svgs {
		if strings.Contains(svg, "NaN") || strings.Contains(svg, "Inf") {
			t.Errorf("%s SVG contains a non-finite coordinate: %s", name, svg)
		}
		if !strings.HasPrefix(svg, "<svg") || !strings.HasSuffix(svg, "</svg>") {
			t.Errorf("%s SVG is malformed", name)
		}
	}
}

func TestChartWithFiniteAndNonFiniteStillPlotsGoodPoints(t *testing.T) {
	clean := renderChart("line", [][]float64{{1, 2, 3}}, 640, 320)
	dirty := renderChart("line", [][]float64{{1, math.NaN(), 3}}, 640, 320)
	if !strings.Contains(dirty, "<polyline") {
		t.Error("a series with one bad sample should still draw its good points")
	}
	if clean == dirty {
		t.Error("the NaN sample should have been dropped, changing the path")
	}
}

func TestHistogramCountsIgnoresNonFinite(t *testing.T) {
	// Five samples, two of them unplottable; the three real ones are 1, 2, 3.
	// Two bins over [1,3] have width 1, so int((v-1)/1) gives 0, 1 and 2, and
	// the last is clamped into the final bin: counts are [1, 2].
	got := histogramCounts([]float64{1, math.NaN(), 2, math.Inf(1), 3}, 2)
	want := []float64{1, 2}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("counts = %v, want %v", got, want)
		}
	}
	// An all-non-finite input must not report phantom counts.
	if all := histogramCounts([]float64{math.NaN(), math.Inf(-1)}, 3); all[0] != 0 {
		t.Errorf("all-non-finite counts = %v, want all zero", all)
	}
}

// ---------------------------------------------------------------------------
// Frontend safety invariants.
//
// The Markdown renderer runs in the browser, so these tests pin the shipped
// asset rather than executing it. They exist because the previous renderer
// interpolated a link target straight into href, which made
// `[x](javascript:alert(1))` — and `[x](" onmouseover="…)` — execute in every
// viewer's page.
// ---------------------------------------------------------------------------

func TestFrontendLinkRenderingIsGuarded(t *testing.T) {
	js, err := frontendFS.ReadFile("frontend/app.js")
	if err != nil {
		t.Fatal(err)
	}
	src := string(js)

	if strings.Contains(src, `'<a href="$2"`) || strings.Contains(src, `"<a href=\"$2\"`) {
		t.Error("markdown link targets are interpolated into href without validation")
	}
	for _, needle := range []string{
		`function safeURL(`,           // the allowlist helper exists
		`attrEscape(safeURL(url))`,    // markdown links go through both
		`a.href = safeURL(p.url)`,     // link_button does too
		`.replace(/"/g, "&quot;")`,    // attribute breakout is closed
		`img.src = mediaURL(p.src)`,   // media sources are validated
		`const LINK_SCHEMES = ["http`, // it is an allowlist, not a denylist
		`[\u0000-\u0020\u007f]/g, ""`, // control chars stripped before the scheme test
		`(?:[^()]|\([^()]*\))*`,       // link targets may contain parentheses
	} {
		if !strings.Contains(src, needle) {
			t.Errorf("frontend is missing the guard %q", needle)
		}
	}
	// The decision must be "is this scheme on the list", never "is this scheme
	// one of the bad ones" — a denylist misses whatever it has not heard of.
	if !strings.Contains(src, `allowed.indexOf(m[1].toLowerCase()) === -1`) {
		t.Error("scheme filtering is not expressed as an allowlist membership test")
	}
}

func TestFrontendRendersEveryElementTypeThePackageEmits(t *testing.T) {
	js, err := frontendFS.ReadFile("frontend/app.js")
	if err != nil {
		t.Fatal(err)
	}
	src := string(js)

	// Build one of everything, then check the renderer has a case for each
	// type. An unhandled type falls through to a "[type]" placeholder, which
	// is how the extended widget and effect surface silently failed to render.
	_, tree := render(func(s *Session) {
		s.Title("t")
		s.Header("h")
		s.Subheader("sh")
		s.Text("x")
		s.Caption("c")
		s.Markdown("m")
		s.Divider()
		s.Code("x", "go")
		s.JSON(map[string]int{"a": 1})
		s.Metric("l", "1", "+1")
		s.Success("ok")
		s.Spinner("w")
		s.Progress(0.5)
		s.Table([][]string{{"a"}, {"b"}})
		s.LineChart([]float64{1, 2})
		s.Map([]MapPoint{{Lat: 1, Lng: 2}})
		s.Image([]byte("x"))
		s.Logo([]byte("x"))
		s.Audio([]byte("x"))
		s.Video([]byte("x"))
		s.Latex("x")
		s.Html("<b>x</b>")
		s.Badge("b")
		s.Echo("code")
		s.Exception(fmt.Errorf("e"))
		s.Help(1)
		s.Toast("t")
		s.Balloons()
		s.LinkButton("l", "https://example.com")
		s.PageLink("https://example.com", "l")
		s.Columns(2)[0].Text("in")
		s.Container().Text("in")
		s.BorderedContainer().Text("in")
		s.Empty()
		s.Expander("e", false).Text("in")
		s.Tabs([]string{"a"})[0].Text("in")
		s.Popover("p").Text("in")
		s.Status("s").Text("in")
		s.ChatMessage("user").Text("in")
		s.ChatInput("p")
		s.Button("b")
		s.Checkbox("c", false)
		s.Toggle("t", false)
		s.Slider("s", 0, 1, 0, 0.1)
		s.SelectSlider("ss", []string{"a", "b"})
		s.NumberInput("n", 1)
		s.TextInput("ti", "")
		s.TextArea("ta", "")
		s.SelectBox("sb", []string{"a"})
		s.Radio("r", []string{"a"})
		s.MultiSelect("ms", []string{"a"})
		s.DateInput("d", "")
		s.TimeInput("tm", "")
		s.ColorPicker("cp", "")
		s.Feedback("stars")
		s.DownloadButton("d", "f.txt", []byte("x"))
		s.FileUploader("fu")
		s.CameraInput("ci")
		s.AudioInput("ai")
		s.Pills("p", []string{"a"})
		s.SegmentedControl("sc", []string{"a"})
		s.SliderRange("sr", 0, 1, 0, 1, 0.1)
		s.SelectSliderRange("ssr", []string{"a", "b"})
		s.DateRangeInput("dr", "", "")
		f := s.Form("f")
		f.TextInput("in", "")
		f.FormSubmitButton("go")
	})

	types := map[string]bool{}
	var walk func(*Element)
	walk = func(e *Element) {
		types[e.Type] = true
		for _, c := range e.Children {
			walk(c)
		}
	}
	walk(tree)
	// Structural types the renderer consumes rather than dispatching on.
	for _, structural := range []string{"root", "main", "sidebar", "tab"} {
		delete(types, structural)
	}

	for typ := range types {
		if !strings.Contains(src, `case "`+typ+`"`) {
			t.Errorf("frontend has no renderer for element type %q", typ)
		}
	}
}

func TestMarkdownUnsafeIsOptIn(t *testing.T) {
	_, tree := render(func(s *Session) {
		s.Markdown("<b>hi</b>")
		s.MarkdownUnsafe("<b>hi</b>")
	})
	mds := findAll(tree, "markdown")
	if len(mds) != 2 {
		t.Fatalf("got %d markdown elements", len(mds))
	}
	if _, ok := mds[0].Props["unsafeAllowHTML"]; ok {
		t.Error("plain Markdown must not set unsafeAllowHTML at all")
	}
	if mds[1].Props["unsafeAllowHTML"] != true {
		t.Error("MarkdownUnsafe must set unsafeAllowHTML")
	}
}

// ---------------------------------------------------------------------------
// Newly added API surface.
// ---------------------------------------------------------------------------

func TestColumnsWeighted(t *testing.T) {
	_, tree := render(func(s *Session) {
		cols := s.ColumnsWeighted([]float64{3, 1, 0, math.NaN()})
		for i, c := range cols {
			c.Text(fmt.Sprint(i))
		}
	})
	cols := findAll(tree, "column")
	if len(cols) != 4 {
		t.Fatalf("got %d columns", len(cols))
	}
	want := []float64{3, 1, 1, 1} // degenerate weights fall back to 1
	for i, w := range want {
		if got := cols[i].Props["weight"]; got != w {
			t.Errorf("column %d weight = %v, want %v", i, got, w)
		}
	}
	// The count-based form still produces equal weights.
	_, tree2 := render(func(s *Session) { s.Columns(2) })
	for i, c := range findAll(tree2, "column") {
		if c.Props["weight"] != 1.0 {
			t.Errorf("Columns(2) column %d weight = %v, want 1", i, c.Props["weight"])
		}
	}
	// An empty weight list is one full-width column.
	_, tree3 := render(func(s *Session) { s.ColumnsWeighted(nil) })
	if got := len(findAll(tree3, "column")); got != 1 {
		t.Errorf("ColumnsWeighted(nil) made %d columns, want 1", got)
	}
}

func TestStateHelpers(t *testing.T) {
	st := newState()
	if st.Has("a") || st.Len() != 0 {
		t.Fatal("fresh state should be empty")
	}
	if !st.SetDefault("a", 1) {
		t.Error("SetDefault should report it stored a missing key")
	}
	if st.SetDefault("a", 2) {
		t.Error("SetDefault must not overwrite")
	}
	if st.GetInt("a", 0) != 1 {
		t.Errorf("a = %v", st.GetInt("a", 0))
	}
	st.Set("c", true)
	st.Set("b", "x")
	if got := st.Keys(); len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("Keys() = %v, want sorted [a b c]", got)
	}
	if !st.GetBool("c", false) || st.GetBool("missing", true) != true {
		t.Error("GetBool")
	}
	st.Clear()
	if st.Len() != 0 || st.Has("a") {
		t.Error("Clear did not empty the store")
	}
}

// TestStatePersistsAcrossRerunsButWidgetsResetToDefault contrasts the two
// lifetimes an app author has to reason about, over three runs.
func TestStatePersistsAcrossRerunsButWidgetsResetToDefault(t *testing.T) {
	app := func(s *Session) {
		s.State.SetDefault("runs", 0)
		s.State.Set("runs", s.State.GetInt("runs", 0)+1)
		s.Checkbox("opt", false, "opt")
	}
	s := newSession()
	s.run(app)
	s.widgets["opt"] = true
	tree := s.run(app)

	if got := s.State.GetInt("runs", 0); got != 2 {
		t.Errorf("state counter = %d after two runs, want 2", got)
	}
	if find(tree, "checkbox").Props["value"] != true {
		t.Error("widget value should survive a rerun that still renders it")
	}
}

package st

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// post issues a POST /api/run against the handler and decodes the response.
func post(t *testing.T, h http.Handler, req runRequest) runResponse {
	t.Helper()
	body, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPost, "/api/run", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var resp runResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return resp
}

func TestServerRunAndInteract(t *testing.T) {
	app := func(s *Session) {
		s.State.Set("count", s.State.GetInt("count", 0))
		if s.Button("inc") {
			s.State.Set("count", s.State.GetInt("count", 0)+1)
		}
		s.Metric("count", intToStr(s.State.GetInt("count", 0)), "")
	}
	h := Handler(app)

	// Initial connect.
	first := post(t, h, runRequest{})
	if first.SessionID == "" {
		t.Fatal("no session id assigned")
	}
	if find(first.Tree, "metric").Props["value"] != "0" {
		t.Fatal("initial count should be 0")
	}

	// Click the button (auto key of the only button).
	btn := find(first.Tree, "button")
	resp := post(t, h, runRequest{
		SessionID: first.SessionID,
		Event:     &event{Key: btn.Key, Button: true},
	})
	if resp.SessionID != first.SessionID {
		t.Fatal("session id changed")
	}
	if find(resp.Tree, "metric").Props["value"] != "1" {
		t.Fatal("count should be 1 after click")
	}

	// A subsequent run without the click keeps state but button is transient.
	resp2 := post(t, h, runRequest{SessionID: first.SessionID})
	if find(resp2.Tree, "metric").Props["value"] != "1" {
		t.Fatal("count should remain 1 (button was transient)")
	}
}

func TestServerSessionsAreIsolated(t *testing.T) {
	app := func(s *Session) {
		v := s.TextInput("name", "anon")
		s.Text(v)
	}
	h := Handler(app)

	a := post(t, h, runRequest{})
	b := post(t, h, runRequest{})
	if a.SessionID == b.SessionID {
		t.Fatal("distinct connects should get distinct sessions")
	}

	key := find(a.Tree, "text_input").Key
	post(t, h, runRequest{SessionID: a.SessionID, Event: &event{Key: key, Value: "alice"}})

	// Session b must be unaffected.
	bAgain := post(t, h, runRequest{SessionID: b.SessionID})
	if find(bAgain.Tree, "text_input").Props["value"] != "anon" {
		t.Fatal("session b leaked session a's value")
	}
}

func TestServerServesFrontend(t *testing.T) {
	h := Handler(func(s *Session) {})
	for _, tc := range []struct{ path, needle string }{
		{"/", "<div id=\"app\">"},
		{"/app.js", "renderTree"},
		{"/style.css", ".st-metric"},
	} {
		r := httptest.NewRequest(http.MethodGet, tc.path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Errorf("%s status = %d", tc.path, w.Code)
			continue
		}
		if !strings.Contains(w.Body.String(), tc.needle) {
			t.Errorf("%s missing %q", tc.path, tc.needle)
		}
	}
}

func TestServerRejectsBadMethodAndBody(t *testing.T) {
	h := Handler(func(s *Session) {})
	// GET on /api/run is not allowed.
	r := httptest.NewRequest(http.MethodGet, "/api/run", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /api/run = %d", w.Code)
	}
	// Malformed JSON.
	r = httptest.NewRequest(http.MethodPost, "/api/run", strings.NewReader("{bad"))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("bad body = %d", w.Code)
	}
	// Unknown path -> 404.
	r = httptest.NewRequest(http.MethodGet, "/nope", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("unknown path = %d", w.Code)
	}
}

// intToStr avoids pulling strconv into the test's hot path expectations.
func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

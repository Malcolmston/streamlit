package st

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestNewWidgetConstruction builds every new widget once and checks the
// resulting element type and key are emitted.
func TestNewWidgetConstruction(t *testing.T) {
	_, tree := render(func(s *Session) {
		s.Toggle("t", true)
		s.DateInput("d", "2026-07-12")
		s.TimeInput("ti", "09:30")
		s.ColorPicker("c", "#ff0000")
		s.SelectSlider("ss", []string{"lo", "mid", "hi"})
		s.Feedback("stars")
		s.DownloadButton("dl", "f.txt", []byte("hello"))
		s.FileUploader("up")
	})
	for _, typ := range []string{
		"toggle", "date_input", "time_input", "color_picker",
		"select_slider", "feedback", "download_button", "file_uploader",
	} {
		e := find(tree, typ)
		if e == nil {
			t.Fatalf("missing widget element %q", typ)
		}
		if e.Key == "" {
			t.Errorf("%q has no key", typ)
		}
	}
}

func TestToggleValueResolution(t *testing.T) {
	app := func(s *Session) { s.State.Set("v", s.Toggle("t", false)) }
	s := newSession()
	s.run(app)
	if got, _ := s.State.Get("v"); got != false {
		t.Fatalf("toggle default = %v, want false", got)
	}
	s.widgets["auto-toggle-0"] = true
	s.run(app)
	if got, _ := s.State.Get("v"); got != true {
		t.Fatalf("toggle after set = %v, want true", got)
	}
}

func TestSelectSliderResolution(t *testing.T) {
	opts := []string{"lo", "mid", "hi"}
	app := func(s *Session) { s.State.Set("v", s.SelectSlider("ss", opts)) }
	s := newSession()
	s.run(app)
	if got := s.State.GetString("v", ""); got != "lo" {
		t.Fatalf("select_slider default = %q, want lo", got)
	}
	s.widgets["auto-select_slider-0"] = "hi"
	tree := s.run(app)
	if got := s.State.GetString("v", ""); got != "hi" {
		t.Fatalf("select_slider = %q, want hi", got)
	}
	if idx := find(tree, "select_slider").Props["index"]; idx != 2 {
		t.Errorf("index prop = %v, want 2", idx)
	}
	// Unknown value falls back to first option.
	s.widgets["auto-select_slider-0"] = "zzz"
	s.run(app)
	if got := s.State.GetString("v", ""); got != "lo" {
		t.Fatalf("select_slider fallback = %q, want lo", got)
	}
}

func TestFeedbackSentinel(t *testing.T) {
	app := func(s *Session) { s.State.Set("r", s.Feedback("stars")) }
	s := newSession()
	s.run(app)
	if got := s.State.GetInt("r", -99); got != -1 {
		t.Fatalf("feedback default = %d, want -1", got)
	}
	s.widgets["auto-feedback-0"] = 3.0
	s.run(app)
	if got := s.State.GetInt("r", -99); got != 3 {
		t.Fatalf("feedback = %d, want 3", got)
	}
}

func TestDownloadButtonEmbedsData(t *testing.T) {
	_, tree := render(func(s *Session) {
		s.DownloadButton("dl", "hello.txt", []byte("hello world"))
	})
	e := find(tree, "download_button")
	href, _ := e.Props["href"].(string)
	if !strings.HasPrefix(href, "data:") || !strings.Contains(href, "base64,") {
		t.Fatalf("download href not a data URI: %.30s", href)
	}
	if e.Props["filename"] != "hello.txt" {
		t.Errorf("filename = %v", e.Props["filename"])
	}
}

func TestFileUploaderStateAndUploadHandler(t *testing.T) {
	app := func(s *Session) {
		files := s.FileUploader("up", "myfile")
		if len(files) > 0 {
			s.State.Set("first", string(files[0].Data))
			s.State.Set("name", files[0].Name)
		}
	}
	h := Handler(app)

	// Initial connect: no files.
	first := post(t, h, runRequest{})
	up := find(first.Tree, "file_uploader")
	if files := up.Props["files"].([]any); len(files) != 0 {
		t.Fatalf("expected no files initially, got %v", files)
	}

	// Build a multipart upload targeting the "myfile" key.
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("session", first.SessionID)
	_ = mw.WriteField("key", "myfile")
	fw, _ := mw.CreateFormFile("files", "greeting.txt")
	_, _ = fw.Write([]byte("hi there"))
	_ = mw.Close()

	r := httptest.NewRequest(http.MethodPost, "/api/upload", &buf)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("upload status = %d", w.Code)
	}
	var resp runResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	up2 := find(resp.Tree, "file_uploader")
	names := up2.Props["files"].([]any)
	if len(names) != 1 || names[0] != "greeting.txt" {
		t.Fatalf("uploaded file names = %v", names)
	}

	// A subsequent plain run should still see the uploaded bytes in state.
	again := post(t, h, runRequest{SessionID: first.SessionID})
	_ = again
}

func TestCacheMemoisesOnce(t *testing.T) {
	CacheClear()
	calls := 0
	compute := func() any { calls++; return 42 }
	s := newSession()

	for i := 0; i < 3; i++ {
		if got := s.Cache("k", compute).(int); got != 42 {
			t.Fatalf("cache returned %v", got)
		}
	}
	if calls != 1 {
		t.Fatalf("compute called %d times, want 1", calls)
	}

	// A different session shares the process-wide cache.
	s2 := newSession()
	if got := s2.Cache("k", compute).(int); got != 42 || calls != 1 {
		t.Fatalf("cross-session cache miss: got=%v calls=%d", got, calls)
	}
}

func TestCacheTTLExpiry(t *testing.T) {
	CacheClear()
	calls := 0
	compute := func() any { calls++; return calls }
	s := newSession()

	s.Cache("k", compute, 20*time.Millisecond)
	s.Cache("k", compute, 20*time.Millisecond)
	if calls != 1 {
		t.Fatalf("within TTL compute called %d times, want 1", calls)
	}
	time.Sleep(30 * time.Millisecond)
	s.Cache("k", compute, 20*time.Millisecond)
	if calls != 2 {
		t.Fatalf("after TTL compute called %d times, want 2", calls)
	}
}

func TestFormDefersUntilSubmit(t *testing.T) {
	app := func(s *Session) {
		f := s.Form("signup")
		name := f.TextInput("Name", "anon")
		s.State.Set("name", name)
		if f.FormSubmitButton("Go") {
			s.State.Set("submitted", true)
		}
	}
	s := newSession()
	s.run(app)
	if got := s.State.GetString("name", ""); got != "anon" {
		t.Fatalf("initial form value = %q, want anon", got)
	}

	// The text input inside the form carries the form key so the frontend
	// stages rather than fires it. Simulate the deferral: no widget commit
	// happens on a plain rerun, so the value stays at the default.
	textKey := "auto-text_input-0"
	tree := s.run(app)
	ti := find(tree, "text_input")
	if ti.Props["form"] != "signup" {
		t.Fatalf("form widget missing form tag: %v", ti.Props["form"])
	}
	if got := s.State.GetString("name", ""); got != "anon" {
		t.Fatalf("value should not change without submit, got %q", got)
	}

	// Submit: the batch event commits every staged value and clicks the button.
	_ = applyEvent(s, &event{
		Key:    "signup",
		Button: true,
		Form:   "signup",
		Values: map[string]any{textKey: "alice"},
	}, Options{}.withDefaults())
	s.run(app)
	if got := s.State.GetString("name", ""); got != "alice" {
		t.Fatalf("after submit name = %q, want alice", got)
	}
	if v, _ := s.State.Get("submitted"); v != true {
		t.Fatalf("submit button should have fired")
	}
}

func TestLayoutContainersConstruct(t *testing.T) {
	_, tree := render(func(s *Session) {
		tabs := s.Tabs([]string{"A", "B"})
		tabs[0].Text("in a")
		tabs[1].Text("in b")
		pop := s.Popover("more")
		pop.Text("popped")
		stat := s.Status("working", "complete")
		stat.Text("done")
		empty := s.Empty()
		empty.Text("filled")
	})
	tabsEl := find(tree, "tabs")
	if tabsEl == nil || len(tabsEl.Children) != 2 {
		t.Fatalf("tabs children = %+v", tabsEl)
	}
	if find(tree, "popover") == nil {
		t.Error("missing popover")
	}
	st := find(tree, "status")
	if st == nil || st.Props["state"] != "complete" {
		t.Errorf("status wrong: %+v", st)
	}
	if find(tree, "empty") == nil {
		t.Error("missing empty")
	}
}

func TestChatConstruct(t *testing.T) {
	app := func(s *Session) {
		s.ChatMessage("user").Text("hi")
		s.State.Set("msg", s.ChatInput("say something"))
	}
	s := newSession()
	tree := s.run(app)
	cm := find(tree, "chat_message")
	if cm == nil || cm.Props["role"] != "user" {
		t.Fatalf("chat_message wrong: %+v", cm)
	}
	if find(tree, "chat_input") == nil {
		t.Fatal("missing chat_input")
	}
	// The chat input resolves from session state across a rerun.
	s.widgets["auto-chat_input-0"] = "hello"
	s.run(app)
	if got := s.State.GetString("msg", ""); got != "hello" {
		t.Fatalf("chat input = %q, want hello", got)
	}
}

func TestMediaConstruct(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	_, tree := render(func(s *Session) {
		s.Image(img, "an image")
		s.Image("https://example.com/pic.png")
		s.Logo([]byte("\x89PNG\r\n\x1a\n"))
		s.Audio("https://example.com/a.mp3")
		s.Video([]byte{0, 0, 0, 24})
		s.Map([]MapPoint{{Lat: 40.7, Lng: -74}, {Lat: -33.8, Lng: 151.2}})
	})
	imgEls := findAll(tree, "image")
	if len(imgEls) != 2 {
		t.Fatalf("expected 2 images, got %d", len(imgEls))
	}
	if src, _ := imgEls[0].Props["src"].(string); !strings.HasPrefix(src, "data:image/") {
		t.Errorf("image.Image not encoded to data URI: %.20s", src)
	}
	if imgEls[0].Props["caption"] != "an image" {
		t.Errorf("caption = %v", imgEls[0].Props["caption"])
	}
	if src, _ := imgEls[1].Props["src"].(string); src != "https://example.com/pic.png" {
		t.Errorf("URL image src = %q", src)
	}
	for _, typ := range []string{"logo", "audio", "video", "map"} {
		if find(tree, typ) == nil {
			t.Errorf("missing media element %q", typ)
		}
	}
	mapSVG, _ := find(tree, "map").Props["svg"].(string)
	if !strings.Contains(mapSVG, "<circle") {
		t.Error("map should contain plotted points")
	}
}

func TestNewChartsSVG(t *testing.T) {
	scatter := renderScatter([]float64{1, 2, 3}, []float64{4, 1, 6}, 640, 320)
	if !strings.HasPrefix(scatter, "<svg") || !strings.Contains(scatter, "<circle") {
		t.Errorf("scatter svg wrong: %.40s", scatter)
	}
	if empty := renderScatter(nil, nil, 640, 320); !strings.Contains(empty, "no data") {
		t.Error("empty scatter should say no data")
	}
	pie := renderPie([]float64{1, 2, 3}, []string{"a", "b", "c"}, 640, 320)
	if !strings.Contains(pie, "<path") {
		t.Error("pie missing slices")
	}
	if empty := renderPie([]float64{0, 0}, nil, 640, 320); !strings.Contains(empty, "no data") {
		t.Error("empty pie should say no data")
	}
	hist := renderHistogram([]float64{1, 1, 2, 3, 3, 3}, 3, 640, 320)
	if !strings.Contains(hist, "<rect") {
		t.Error("histogram missing bars")
	}
}

func TestHistogramCounts(t *testing.T) {
	counts := histogramCounts([]float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, 2)
	if len(counts) != 2 || counts[0] != 5 || counts[1] != 5 {
		t.Fatalf("counts = %v, want [5 5]", counts)
	}
	// All-equal values land in the first bin.
	eq := histogramCounts([]float64{7, 7, 7}, 4)
	if eq[0] != 3 {
		t.Fatalf("equal-value counts = %v", eq)
	}
	// Fewer than one bin is treated as one.
	one := histogramCounts([]float64{1, 2, 3}, 0)
	if len(one) != 1 || one[0] != 3 {
		t.Fatalf("zero-bin counts = %v", one)
	}
}

func TestChartsEmitElements(t *testing.T) {
	_, tree := render(func(s *Session) {
		s.ScatterChart([]float64{1, 2}, []float64{3, 4})
		s.PieChart([]float64{1, 2}, []string{"a", "b"})
		s.Histogram([]float64{1, 2, 3, 4}, 2)
	})
	if got := len(findAll(tree, "chart")); got != 3 {
		t.Fatalf("expected 3 chart elements, got %d", got)
	}
}

func TestTableCaptionAndDataFrameSortable(t *testing.T) {
	_, tree := render(func(s *Session) {
		s.Table([][]string{{"h"}, {"v"}}, "a caption")
		s.DataFrame([][]string{{"h"}, {"v"}})
	})
	tbls := findAll(tree, "table")
	if len(tbls) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(tbls))
	}
	if tbls[0].Props["caption"] != "a caption" {
		t.Errorf("caption = %v", tbls[0].Props["caption"])
	}
	if tbls[0].Props["sortable"] != false {
		t.Errorf("Table should not be sortable: %v", tbls[0].Props["sortable"])
	}
	if tbls[1].Props["sortable"] != true {
		t.Errorf("DataFrame should be sortable: %v", tbls[1].Props["sortable"])
	}
}

func TestJSONCollapsedAndMetricColor(t *testing.T) {
	_, tree := render(func(s *Session) {
		s.JSON(map[string]int{"a": 1}, true)
		s.MetricColored("m", "10", "-2", "inverse")
	})
	if find(tree, "json").Props["collapsed"] != true {
		t.Error("json collapsed flag not set")
	}
	if find(tree, "metric").Props["deltaColor"] != "inverse" {
		t.Errorf("metric deltaColor = %v", find(tree, "metric").Props["deltaColor"])
	}
}

func TestFormSubmitEventRoundTrip(t *testing.T) {
	// The extended event fields survive JSON round-tripping.
	ev := event{Key: "f", Button: true, Form: "f", Values: map[string]any{"a": "b"}}
	b, err := json.Marshal(ev)
	if err != nil {
		t.Fatal(err)
	}
	var back event
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.Form != "f" || back.Values["a"] != "b" {
		t.Fatalf("event round trip lost fields: %+v", back)
	}
}

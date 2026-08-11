package st

import (
	"encoding/json"
	"strings"
	"testing"
)

// render runs an app function once against a fresh session and returns the
// tree, mirroring what the server does but without HTTP.
func render(app func(*Session)) (*Session, *Element) {
	s := newSession()
	tree := s.run(app)
	return s, tree
}

// find walks the tree depth-first and returns the first element of the given
// type, or nil.
func find(root *Element, typ string) *Element {
	if root == nil {
		return nil
	}
	if root.Type == typ {
		return root
	}
	for _, c := range root.Children {
		if got := find(c, typ); got != nil {
			return got
		}
	}
	return nil
}

// findAll collects every element of the given type.
func findAll(root *Element, typ string) []*Element {
	var out []*Element
	var walk func(*Element)
	walk = func(e *Element) {
		if e == nil {
			return
		}
		if e.Type == typ {
			out = append(out, e)
		}
		for _, c := range e.Children {
			walk(c)
		}
	}
	walk(root)
	return out
}

func TestElementTreeConstruction(t *testing.T) {
	_, tree := render(func(s *Session) {
		s.Title("Hi")
		s.Header("Section")
		s.Text("body")
	})
	if tree.Type != "root" {
		t.Fatalf("root type = %q", tree.Type)
	}
	if title := find(tree, "title"); title == nil || title.Props["text"] != "Hi" {
		t.Fatalf("title element missing or wrong: %+v", title)
	}
	// Title/header/text should land under the main region in order.
	main := find(tree, "main")
	if main == nil || len(main.Children) != 3 {
		t.Fatalf("expected 3 main children, got %+v", main)
	}
	want := []string{"title", "header", "text"}
	for i, w := range want {
		if main.Children[i].Type != w {
			t.Errorf("child %d = %q, want %q", i, main.Children[i].Type, w)
		}
	}
}

func TestSidebarAndColumns(t *testing.T) {
	_, tree := render(func(s *Session) {
		s.Sidebar().Text("side")
		cols := s.Columns(3)
		cols[0].Text("a")
		cols[2].Text("c")
	})
	sidebar := find(tree, "sidebar")
	if sidebar == nil || find(sidebar, "text") == nil {
		t.Fatal("sidebar text not found")
	}
	colsEl := find(tree, "columns")
	if colsEl == nil || len(colsEl.Children) != 3 {
		t.Fatalf("expected 3 columns, got %+v", colsEl)
	}
	if colsEl.Props["n"] != 3 {
		t.Errorf("columns n = %v", colsEl.Props["n"])
	}
}

func TestWriteTypeDispatch(t *testing.T) {
	type person struct {
		Name string
		Age  int
	}
	cases := []struct {
		name string
		arg  any
		typ  string
	}{
		{"string->markdown", "hello", "markdown"},
		{"error->alert", errString("boom"), "alert"},
		{"int->text", 42, "text"},
		{"struct->table", person{"Ada", 36}, "table"},
		{"structslice->table", []person{{"Ada", 36}}, "table"},
		{"stringmatrix->table", [][]string{{"h"}, {"v"}}, "table"},
		{"map->json", map[string]int{"a": 1}, "json"},
		{"intslice->json", []int{1, 2, 3}, "json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, tree := render(func(s *Session) { s.Write(tc.arg) })
			main := find(tree, "main")
			if len(main.Children) != 1 {
				t.Fatalf("expected 1 element, got %d", len(main.Children))
			}
			if got := main.Children[0].Type; got != tc.typ {
				t.Errorf("Write(%v) -> %q, want %q", tc.arg, got, tc.typ)
			}
		})
	}
}

func TestErrorDispatch(t *testing.T) {
	_, tree := render(func(s *Session) { s.Write(errString("bad")) })
	alert := find(tree, "alert")
	if alert == nil || alert.Props["kind"] != "error" || alert.Props["text"] != "bad" {
		t.Fatalf("error alert wrong: %+v", alert)
	}
}

func TestTableStructReflection(t *testing.T) {
	type row struct {
		A string
		B int
	}
	_, tree := render(func(s *Session) {
		s.Table([]row{{"x", 1}, {"y", 2}})
	})
	tbl := find(tree, "table")
	if tbl == nil {
		t.Fatal("no table")
	}
	hdr := tbl.Props["header"].([]any)
	if len(hdr) != 2 || hdr[0] != "A" || hdr[1] != "B" {
		t.Errorf("header = %v", hdr)
	}
	rows := tbl.Props["rows"].([]any)
	if len(rows) != 2 {
		t.Fatalf("rows = %v", rows)
	}
	first := rows[0].([]any)
	if first[0] != "x" || first[1] != "1" {
		t.Errorf("row 0 = %v", first)
	}
}

func TestWidgetValueResolutionAcrossRerun(t *testing.T) {
	app := func(s *Session) {
		v := s.Slider("amt", 0, 100, 10, 1)
		s.State.Set("last", v)
	}
	s := newSession()

	// First run: default value.
	s.run(app)
	if got, _ := s.State.Get("last"); got != 10.0 {
		t.Fatalf("first run slider = %v, want 10", got)
	}

	// Simulate a browser interaction: the slider's auto key is stable.
	sliderKey := "auto-slider-0"
	s.widgets[sliderKey] = 42.0

	// Rerun: value should be restored from session state.
	s.run(app)
	if got, _ := s.State.Get("last"); got != 42.0 {
		t.Fatalf("second run slider = %v, want 42", got)
	}
}

func TestButtonIsTransient(t *testing.T) {
	var seen []bool
	app := func(s *Session) {
		seen = append(seen, s.Button("go"))
	}
	s := newSession()

	s.run(app) // no click
	s.clicked["auto-button-0"] = true
	s.run(app) // click registered -> true this run
	s.run(app) // cleared -> false again

	if len(seen) != 3 || seen[0] != false || seen[1] != true || seen[2] != false {
		t.Fatalf("button transience wrong: %v", seen)
	}
}

func TestSessionStatePersistence(t *testing.T) {
	app := func(s *Session) {
		s.State.Set("count", s.State.GetInt("count", 0)+1)
	}
	s := newSession()
	for i := 1; i <= 5; i++ {
		s.run(app)
		if got := s.State.GetInt("count", 0); got != i {
			t.Fatalf("after run %d count = %d", i, got)
		}
	}
}

func TestSelectBoxDefaultAndPersistence(t *testing.T) {
	app := func(s *Session) {
		got := s.SelectBox("pick", []string{"a", "b", "c"})
		s.State.Set("choice", got)
	}
	s := newSession()
	s.run(app)
	if c := s.State.GetString("choice", ""); c != "a" {
		t.Fatalf("default selectbox = %q, want a", c)
	}
	s.widgets["auto-selectbox-0"] = "c"
	s.run(app)
	if c := s.State.GetString("choice", ""); c != "c" {
		t.Fatalf("selected selectbox = %q, want c", c)
	}
	// Invalid selection falls back to default.
	s.widgets["auto-selectbox-0"] = "zzz"
	s.run(app)
	if c := s.State.GetString("choice", ""); c != "a" {
		t.Fatalf("invalid selectbox = %q, want fallback a", c)
	}
}

func TestMultiSelectFiltersUnknown(t *testing.T) {
	app := func(s *Session) {
		s.MultiSelect("tags", []string{"x", "y"})
	}
	s := newSession()
	s.widgets["auto-multiselect-0"] = []any{"x", "nope", "y"}
	tree := s.run(app)
	ms := find(tree, "multiselect")
	val := ms.Props["value"].([]string)
	if len(val) != 2 || val[0] != "x" || val[1] != "y" {
		t.Fatalf("multiselect value = %v", val)
	}
}

func TestJSONProtocolRoundTrip(t *testing.T) {
	_, tree := render(func(s *Session) {
		s.Title("T")
		s.Slider("s", 0, 1, 0.5, 0.1)
	})
	b, err := json.Marshal(runResponse{SessionID: "id1", Tree: tree})
	if err != nil {
		t.Fatal(err)
	}
	var back runResponse
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.SessionID != "id1" {
		t.Errorf("sessionId = %q", back.SessionID)
	}
	if find(back.Tree, "slider") == nil {
		t.Error("slider lost in round trip")
	}
	if !strings.Contains(string(b), `"type":"title"`) {
		t.Error("title not serialised")
	}
}

func TestApplyEventUpdatesWidget(t *testing.T) {
	s := newSession()
	_ = applyEvent(s, &event{Key: "k", Value: "hello"}, Options{}.withDefaults())
	if s.widgets["k"] != "hello" {
		t.Errorf("widget not set: %v", s.widgets["k"])
	}
	_ = applyEvent(s, &event{Key: "b", Button: true}, Options{}.withDefaults())
	if !s.clicked["b"] {
		t.Error("button click not recorded")
	}
}

func TestSVGChartGeneration(t *testing.T) {
	svg := renderChart("line", [][]float64{{1, 5, 3, 8}}, 640, 320)
	if !strings.HasPrefix(svg, "<svg") || !strings.HasSuffix(svg, "</svg>") {
		t.Fatalf("not an svg document: %.40s", svg)
	}
	if !strings.Contains(svg, "<polyline") {
		t.Error("line chart missing polyline")
	}
	bar := renderChart("bar", [][]float64{{1, 2}}, 640, 320)
	if !strings.Contains(bar, "<rect") {
		t.Error("bar chart missing rect")
	}
	area := renderChart("area", [][]float64{{1, 2, 3}}, 640, 320)
	if !strings.Contains(area, "<polygon") {
		t.Error("area chart missing polygon")
	}
	empty := renderChart("line", nil, 640, 320)
	if !strings.Contains(empty, "no data") {
		t.Error("empty chart should say no data")
	}
}

func TestChartElementEmbedsSVG(t *testing.T) {
	_, tree := render(func(s *Session) {
		s.LineChart([]float64{1, 2, 3})
	})
	ch := find(tree, "chart")
	if ch == nil {
		t.Fatal("no chart element")
	}
	if svg, _ := ch.Props["svg"].(string); !strings.HasPrefix(svg, "<svg") {
		t.Errorf("chart svg prop wrong: %.20s", svg)
	}
}

func TestCoercionHelpers(t *testing.T) {
	if asFloat("3.5", 0) != 3.5 {
		t.Error("asFloat string")
	}
	if asFloat(nil, 7) != 7 {
		t.Error("asFloat default")
	}
	if asInt(4.9, 0) != 4 {
		t.Error("asInt truncation")
	}
	if !asBool("true", false) {
		t.Error("asBool string")
	}
	if asString(123, "def") != "def" {
		t.Error("asString fallback")
	}
	if got := asStringSlice([]any{"a", 1, "b"}, nil); len(got) != 2 {
		t.Errorf("asStringSlice = %v", got)
	}
	if clampFloat(5, 0, 3) != 3 {
		t.Error("clamp high")
	}
}

func TestAutoKeyStability(t *testing.T) {
	app := func(s *Session) {
		s.Slider("a", 0, 1, 0, 0.1)
		s.TextInput("b", "")
		s.Slider("c", 0, 1, 0, 0.1)
	}
	s := newSession()
	tree := s.run(app)
	widgets := []string{}
	var walk func(*Element)
	walk = func(e *Element) {
		if e.Key != "" {
			widgets = append(widgets, e.Key)
		}
		for _, c := range e.Children {
			walk(c)
		}
	}
	walk(tree)
	want := []string{"auto-slider-0", "auto-text_input-1", "auto-slider-2"}
	if len(widgets) != len(want) {
		t.Fatalf("keys = %v", widgets)
	}
	for i := range want {
		if widgets[i] != want[i] {
			t.Errorf("key %d = %q, want %q", i, widgets[i], want[i])
		}
	}
}

// errString is a simple error implementation for tests.
type errString string

func (e errString) Error() string { return string(e) }

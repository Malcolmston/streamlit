package st

import (
	"reflect"
	"testing"
)

func TestExFloatPairOrdering(t *testing.T) {
	cases := []struct {
		name           string
		in             any
		defLo, defHi   float64
		wantLo, wantHi float64
	}{
		{"default", nil, 2, 8, 2, 8},
		{"pair", []any{3.0, 5.0}, 0, 10, 3, 5},
		{"unordered", []any{9.0, 1.0}, 0, 10, 1, 9},
		{"float64slice", []float64{4, 7}, 0, 10, 4, 7},
		{"short", []any{5.0}, 1, 6, 1, 6},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lo, hi := exFloatPair(tc.in, tc.defLo, tc.defHi)
			if lo != tc.wantLo || hi != tc.wantHi {
				t.Errorf("exFloatPair(%v) = (%v,%v), want (%v,%v)", tc.in, lo, hi, tc.wantLo, tc.wantHi)
			}
		})
	}
}

func TestNewWidgetsConstruction(t *testing.T) {
	_, tree := render(func(s *Session) {
		s.Pills("p", []string{"a", "b"})
		s.SegmentedControl("sc", []string{"x", "y"})
		s.PrimaryButton("go")
		s.PasswordInput("pw", "")
		s.TextInputMax("tm", "", 5)
		s.NumberInputRange("nr", 0, 10, 5)
		s.SliderRange("sr", 0, 100, 20, 80, 1)
		s.SelectSliderRange("ssr", []string{"lo", "mid", "hi"})
		s.DateRangeInput("dr", "2026-01-01", "2026-12-31")
		s.CameraInput("cam")
		s.AudioInput("aud")
	})
	for _, typ := range []string{
		"pills", "segmented_control", "text_input", "number",
		"slider_range", "select_slider_range", "date_range_input",
		"camera_input", "audio_input",
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

func TestSegmentedControlResolution(t *testing.T) {
	opts := []string{"x", "y", "z"}
	app := func(s *Session) { s.State.Set("v", s.SegmentedControl("sc", opts)) }
	s := newSession()
	s.run(app)
	if got := s.State.GetString("v", ""); got != "x" {
		t.Fatalf("default = %q, want x", got)
	}
	s.widgets["auto-segmented_control-0"] = "z"
	s.run(app)
	if got := s.State.GetString("v", ""); got != "z" {
		t.Fatalf("selected = %q, want z", got)
	}
	s.widgets["auto-segmented_control-0"] = "unknown"
	s.run(app)
	if got := s.State.GetString("v", ""); got != "x" {
		t.Fatalf("fallback = %q, want x", got)
	}
}

func TestPillsFiltersUnknown(t *testing.T) {
	opts := []string{"a", "b", "c"}
	s := newSession()
	s.widgets["mine"] = []any{"a", "zzz", "c"}
	var got []string
	s.run(func(s *Session) { got = s.Pills("p", opts, "mine") })
	want := []string{"a", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Pills = %v, want %v", got, want)
	}
}

func TestNumberInputRangeClamp(t *testing.T) {
	cases := []struct {
		def  float64
		want float64
	}{
		{5, 5}, {-3, 0}, {99, 10},
	}
	for _, tc := range cases {
		var got float64
		render(func(s *Session) { got = s.NumberInputRange("nr", 0, 10, tc.def) })
		if got != tc.want {
			t.Errorf("NumberInputRange def=%v = %v, want %v", tc.def, got, tc.want)
		}
	}
	// Swapped bounds are normalised.
	var got float64
	render(func(s *Session) { got = s.NumberInputRange("nr", 10, 0, 5) })
	if got != 5 {
		t.Errorf("swapped bounds = %v, want 5", got)
	}
}

func TestSliderRangeClampAndOrder(t *testing.T) {
	// Defaults within range.
	var lo, hi float64
	render(func(s *Session) { lo, hi = s.SliderRange("sr", 0, 100, 20, 80, 0) })
	if lo != 20 || hi != 80 {
		t.Fatalf("default = (%v,%v), want (20,80)", lo, hi)
	}
	// Restored state out of order and out of bounds.
	s := newSession()
	s.widgets["mine"] = []any{150.0, -10.0}
	render2(s, func(s *Session) { lo, hi = s.SliderRange("sr", 0, 100, 20, 80, 0, "mine") })
	if lo != 0 || hi != 100 {
		t.Fatalf("clamped = (%v,%v), want (0,100)", lo, hi)
	}
	// Default step derives from range.
	var stepTree *Element
	_, stepTree = render(func(s *Session) { s.SliderRange("sr", 0, 100, 20, 80, 0) })
	if got := find(stepTree, "slider_range").Props["step"]; got != 1.0 {
		t.Errorf("default step = %v, want 1", got)
	}
}

func TestSelectSliderRangeOrder(t *testing.T) {
	opts := []string{"a", "b", "c", "d"}
	// Default spans full range.
	var lo, hi string
	render(func(s *Session) { lo, hi = s.SelectSliderRange("ssr", opts) })
	if lo != "a" || hi != "d" {
		t.Fatalf("default = (%q,%q), want (a,d)", lo, hi)
	}
	// Restored reversed selection is reordered by position.
	s := newSession()
	s.widgets["mine"] = []any{"d", "b"}
	render2(s, func(s *Session) { lo, hi = s.SelectSliderRange("ssr", opts, "mine") })
	if lo != "b" || hi != "d" {
		t.Fatalf("reordered = (%q,%q), want (b,d)", lo, hi)
	}
}

func TestDateRangeInputOrder(t *testing.T) {
	var start, end string
	render(func(s *Session) { start, end = s.DateRangeInput("dr", "2026-05-01", "2026-01-01") })
	if start != "2026-01-01" || end != "2026-05-01" {
		t.Fatalf("DateRangeInput = (%q,%q), want chronological", start, end)
	}
}

func TestTextInputMaxTruncates(t *testing.T) {
	s := newSession()
	s.widgets["mine"] = "abcdefgh"
	var got string
	s.run(func(s *Session) { got = s.TextInputMax("tm", "", 3, "mine") })
	if got != "abc" {
		t.Fatalf("TextInputMax = %q, want abc", got)
	}
	// Zero limit means no truncation.
	s.widgets["mine2"] = "abcdefgh"
	s.run(func(s *Session) { got = s.TextInputMax("tm", "", 0, "mine2") })
	if got != "abcdefgh" {
		t.Fatalf("no-limit TextInputMax = %q, want full", got)
	}
}

func TestPrimaryButtonClick(t *testing.T) {
	app := func(s *Session) { s.State.Set("c", s.PrimaryButton("go", "btn")) }
	s := newSession()
	s.run(app)
	if got, _ := s.State.Get("c"); got != false {
		t.Fatalf("unclicked = %v, want false", got)
	}
	s.clicked["btn"] = true
	s.run(app)
	if got, _ := s.State.Get("c"); got != true {
		t.Fatalf("clicked = %v, want true", got)
	}
}

// render2 runs an app against an existing session (to exercise state
// restoration) and returns the tree.
func render2(s *Session, app func(*Session)) *Element { return s.run(app) }

func BenchmarkSliderRange(b *testing.B) {
	s := newSession()
	s.widgets["k"] = []any{33.0, 77.0}
	for i := 0; i < b.N; i++ {
		s.run(func(s *Session) { s.SliderRange("sr", 0, 100, 20, 80, 1, "k") })
	}
}

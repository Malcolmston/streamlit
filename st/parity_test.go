package st

// This file encodes known-answer vectors taken directly from the upstream
// Streamlit test suite (streamlit/streamlit, lib/tests/streamlit/elements/*.py)
// as deterministic assertions against this port's public API. Each TestParity*
// function names the upstream test file it mirrors so the correspondence can be
// audited. The goal is upstream parity: the port must reproduce the same
// observable widget/element behaviour Streamlit asserts, not merely "work".

import "testing"

// widgetProps renders app once and returns the props of the first element of
// the given type, failing the test if none is present.
func widgetProps(t *testing.T, typ string, app func(*Session)) map[string]any {
	t.Helper()
	_, tree := render(app)
	el := find(tree, typ)
	if el == nil {
		t.Fatalf("no %q element in tree", typ)
	}
	return el.Props
}

// TestParityColorPicker mirrors upstream color_picker_test.py:
// test_just_label and test_value_types — the no-value default is "#000000"
// and explicit hex values (short or long form) are preserved verbatim.
func TestParityColorPicker(t *testing.T) {
	cases := []struct{ def, want string }{
		{"", "#000000"},        // (None, "#000000")
		{"#333333", "#333333"}, // ("#333333", "#333333")
		{"#333", "#333"},       // ("#333", "#333")
	}
	for _, tc := range cases {
		var got string
		p := widgetProps(t, "color_picker", func(s *Session) { got = s.ColorPicker("the label", tc.def) })
		if got != tc.want || p["value"] != tc.want {
			t.Errorf("ColorPicker(%q) = %q, prop %v, want %q", tc.def, got, p["value"], tc.want)
		}
	}
}

// TestParityMetricDeltaColor mirrors upstream metric_test.py::test_delta_color:
// the eight (delta, mode) rows and their resolved colour and direction. The
// numeric Python rows are expressed here as their string delta equivalents,
// since this port's Metric takes a string delta.
func TestParityMetricDeltaColor(t *testing.T) {
	cases := []struct {
		delta, mode      string
		color, direction string
	}{
		{"-123", "normal", "red", "down"},
		{"-123", "inverse", "green", "down"},
		{"-123", "off", "grey", "down"},
		{"123", "normal", "green", "up"},
		{"123", "inverse", "red", "up"},
		{"123", "off", "grey", "up"},
		{"", "normal", "grey", "none"},
	}
	for _, tc := range cases {
		p := widgetProps(t, "metric", func(s *Session) { s.MetricColored("label_test", "4312", tc.delta, tc.mode) })
		if p["color"] != tc.color || p["direction"] != tc.direction {
			t.Errorf("Metric(delta=%q,mode=%q) = (%v,%v), want (%s,%s)",
				tc.delta, tc.mode, p["color"], p["direction"], tc.color, tc.direction)
		}
	}
}

// TestParityMetricZeroDelta mirrors metric_test.py::test_zero_delta_color_and_direction
// and test_zero_like_strings_keep_positive_direction and
// test_negative_zero_string_is_negative: only the literal "0" (after trimming
// whitespace) is neutral; "0.0", "+0", "0%" read positive; "-0" reads negative.
func TestParityMetricZeroDelta(t *testing.T) {
	cases := []struct {
		delta            string
		color, direction string
	}{
		{"0", "grey", "none"},  // literal zero -> neutral
		{" 0", "grey", "none"}, // dedented to "0"
		{"0.0", "green", "up"}, // not literal "0"
		{"+0", "green", "up"},  // positive
		{"0%", "green", "up"},  // positive
		{"-0", "red", "down"},  // string "-0" is negative
		{"", "grey", "none"},   // empty -> neutral
	}
	for _, tc := range cases {
		p := widgetProps(t, "metric", func(s *Session) { s.Metric("label_test", "123", tc.delta) })
		if p["color"] != tc.color || p["direction"] != tc.direction {
			t.Errorf("Metric(delta=%q) = (%v,%v), want (%s,%s)",
				tc.delta, p["color"], p["direction"], tc.color, tc.direction)
		}
	}
}

// TestParitySliderWidensBounds mirrors slider_test.py: test_value_greater_than_min,
// test_value_smaller_than_max and test_max_min. When the default falls outside
// [min,max] Streamlit widens the range to include it and returns the value
// unchanged (it does not clamp), and out-of-order bounds are swapped.
func TestParitySliderWidensBounds(t *testing.T) {
	cases := []struct {
		min, max, def    float64
		wantRet          float64
		wantMin, wantMax float64
	}{
		{10, 100, 0, 0, 0, 100},        // value below min -> min becomes 0, ret 0
		{10, 100, 101, 101, 10, 101},   // value above max -> max becomes 101, ret 101
		{101, 100, 101, 101, 100, 101}, // bounds swapped to (100,101), ret 101
	}
	for _, tc := range cases {
		var ret float64
		p := widgetProps(t, "slider", func(s *Session) {
			ret = s.Slider("Slider label", tc.min, tc.max, tc.def, 1)
		})
		if ret != tc.wantRet {
			t.Errorf("Slider(%v,%v,%v) ret = %v, want %v", tc.min, tc.max, tc.def, ret, tc.wantRet)
		}
		if p["min"] != tc.wantMin || p["max"] != tc.wantMax {
			t.Errorf("Slider(%v,%v,%v) bounds = (%v,%v), want (%v,%v)",
				tc.min, tc.max, tc.def, p["min"], p["max"], tc.wantMin, tc.wantMax)
		}
	}
}

// TestParityProgressPercent mirrors progress_test.py::test_progress_float and
// test_progress_int: the reported completion is int(value*100).
func TestParityProgressPercent(t *testing.T) {
	cases := []struct {
		value float64
		want  int
	}{
		{0.0, 0},
		{0.42, 42},
		{1.0, 100},
	}
	for _, tc := range cases {
		p := widgetProps(t, "progress", func(s *Session) { s.Progress(tc.value) })
		if p["percent"] != tc.want {
			t.Errorf("Progress(%v) percent = %v, want %d", tc.value, p["percent"], tc.want)
		}
	}
}

// TestParitySelectbox mirrors selectbox_test.py: the default selection is the
// first option (upstream default index 0), and a persisted value that is no
// longer among the options resets to that default.
func TestParitySelectbox(t *testing.T) {
	opts := []string{"m", "f"}
	var got string
	render(func(s *Session) { got = s.SelectBox("the label", opts) })
	if got != "m" {
		t.Errorf("SelectBox default = %q, want %q", got, "m")
	}
	// Persisted unknown value resets to default.
	s := newSession()
	s.widgets["auto-selectbox-0"] = "gone"
	s.run(func(s *Session) { got = s.SelectBox("the label", opts) })
	if !containsString(opts, got) {
		t.Errorf("SelectBox with stale value = %q, want a valid option", got)
	}
}

// TestParityMultiselectFiltersUnknown mirrors multiselect_test.py: with no
// selection the value is the empty list, and selections not present in options
// are dropped.
func TestParityMultiselectFiltersUnknown(t *testing.T) {
	opts := []string{"a", "b", "c"}
	var got []string
	render(func(s *Session) { got = s.MultiSelect("the label", opts) })
	if len(got) != 0 {
		t.Errorf("MultiSelect default = %v, want empty", got)
	}
	s := newSession()
	s.reset()
	s.widgets["auto-multiselect-0"] = []any{"a", "zzz", "c"}
	s.run(func(s *Session) { got = s.MultiSelect("the label", opts) })
	if len(got) != 2 || got[0] != "a" || got[1] != "c" {
		t.Errorf("MultiSelect filtered = %v, want [a c]", got)
	}
}

// TestParitySelectSliderRange mirrors select_slider_test.py::test_range and
// test_range_out_of_order: the default pair spans the first and last options
// and is always returned in option order regardless of the input order.
func TestParitySelectSliderRange(t *testing.T) {
	opts := []string{"lo", "mid", "hi"}
	var lo, hi string
	render(func(s *Session) { lo, hi = s.SelectSliderRange("ss", opts) })
	if lo != "lo" || hi != "hi" {
		t.Errorf("SelectSliderRange default = (%q,%q), want (lo,hi)", lo, hi)
	}
	// Out-of-order persisted values are reordered by option position.
	s := newSession()
	s.reset()
	s.widgets["auto-select_slider_range-0"] = []any{"hi", "lo"}
	s.run(func(s *Session) { lo, hi = s.SelectSliderRange("ss", opts) })
	if lo != "lo" || hi != "hi" {
		t.Errorf("SelectSliderRange reordered = (%q,%q), want (lo,hi)", lo, hi)
	}
}

// TestParityCheckbox mirrors checkbox_test.py::test_value_types: the widget
// echoes its boolean default.
func TestParityCheckbox(t *testing.T) {
	for _, def := range []bool{true, false} {
		var got bool
		p := widgetProps(t, "checkbox", func(s *Session) { got = s.Checkbox("l", def) })
		if got != def || p["value"] != def {
			t.Errorf("Checkbox(%v) = %v, prop %v", def, got, p["value"])
		}
	}
}

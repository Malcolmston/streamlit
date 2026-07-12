package st

import "testing"

// stringerT exercises the fmt.Stringer branch of Write.
type stringerT struct{}

func (stringerT) String() string { return "I am a stringer" }

// TestAllDisplayAndWidgets calls every public display, layout, status, chart
// and widget method once to ensure they build without panicking and emit the
// expected element types.
func TestAllDisplayAndWidgets(t *testing.T) {
	_, tree := render(func(s *Session) {
		s.Header("h")
		s.Subheader("sh")
		s.Caption("cap")
		s.Markdown("**md**")
		s.Code("x := 1", "go")
		s.JSON(map[string]int{"a": 1})
		s.Metric("m", "1", "-2")
		s.Divider()
		s.Success("ok")
		s.Info("info")
		s.Warning("warn")
		s.Error("err")
		s.Spinner("working")
		s.Progress(0.5)
		s.DataFrame([][]string{{"h"}, {"v"}})

		s.BarChart([]float64{1, 2, 3})
		s.AreaChart([]float64{3, 2, 1})

		// Layout.
		cont := s.Container()
		cont.Text("in container")
		exp := s.Expander("more", true)
		exp.Text("hidden")

		// Widgets.
		s.Checkbox("cb", true)
		s.NumberInput("num", 5)
		s.TextArea("ta", "text")
		s.Radio("r", []string{"a", "b"})

		// Write branches.
		s.Write(stringerT{})
		s.Write(nil)
		x := 7
		s.Write(&x)
	})

	for _, typ := range []string{
		"header", "subheader", "caption", "markdown", "code", "json",
		"metric", "divider", "alert", "spinner", "progress", "table",
		"chart", "container", "expander", "checkbox", "number",
		"text_area", "radio",
	} {
		if find(tree, typ) == nil {
			t.Errorf("missing element type %q", typ)
		}
	}
	if len(findAll(tree, "chart")) != 2 {
		t.Errorf("expected 2 charts, got %d", len(findAll(tree, "chart")))
	}
	if len(findAll(tree, "alert")) != 4 {
		t.Errorf("expected 4 alerts, got %d", len(findAll(tree, "alert")))
	}
}

// TestProgressClamp verifies progress values are clamped to [0,1].
func TestProgressClamp(t *testing.T) {
	_, tree := render(func(s *Session) { s.Progress(5) })
	if v := find(tree, "progress").Props["value"]; v != 1.0 {
		t.Errorf("progress clamp = %v, want 1", v)
	}
}

// TestColumnsMinimum ensures Columns(0) yields one column.
func TestColumnsMinimum(t *testing.T) {
	_, tree := render(func(s *Session) { s.Columns(0) })
	if n := len(find(tree, "columns").Children); n != 1 {
		t.Errorf("Columns(0) children = %d, want 1", n)
	}
}

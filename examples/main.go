// Command examples serves a demo Streamlit-Go application on localhost.
//
// Run it with:
//
//	go run ./examples
//
// then open http://localhost:8501 in a browser. The demo shows a title,
// st.Write, a slider and button that update a metric and a line chart, a
// selectbox, and a session-state counter that persists across reruns.
package main

import (
	"errors"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/malcolmston/streamlit/st"
)

func main() {
	log.Println("serving demo on http://localhost:8501")
	if err := st.Run(app, ":8501"); err != nil {
		log.Fatal(err)
	}
}

// app builds the demo page. It is re-executed top-to-bottom on every widget
// interaction.
func app(s *st.Session) {
	// Sidebar controls.
	sb := s.Sidebar()
	sb.Header("Controls")
	wave := sb.SelectBox("Waveform", []string{"sine", "cosine", "sawtooth"})
	amp := sb.Slider("Amplitude", 1, 10, 5, 0.5)

	// Main content.
	s.Title("Streamlit-Go Demo")
	s.Write("A **from-scratch**, standard-library-only port of Streamlit. " +
		"Interacting with any widget reruns this script top-to-bottom.")
	s.Divider()

	points := int(s.Slider("Number of points", 10, 200, 80, 1))
	series := generate(wave, amp, points)

	cols := s.Columns(3)
	cols[0].Metric("Points", itoa(points), "")
	cols[1].Metric("Amplitude", ftoa(amp), "")
	cols[2].Metric("Peak", ftoa(peak(series)), "")

	s.Subheader("Line chart")
	s.LineChart(series)

	// A button plus a session-state counter that survives reruns.
	s.Divider()
	c1, c2 := twoCols(s)
	if c1.Button("Increment counter") {
		s.State.Set("count", s.State.GetInt("count", 0)+1)
	}
	c2.Metric("Counter", itoa(s.State.GetInt("count", 0)), "")

	// Status + data views inside an expander.
	if peak(series) > 8 {
		s.Warning("Amplitude is high!")
	} else {
		s.Info("Adjust the amplitude in the sidebar.")
	}

	exp := s.Expander("Show raw data", false)
	exp.Write([][]string{
		{"i", "value"},
		{"0", ftoa(series[0])},
		{"1", ftoa(series[1])},
		{"2", ftoa(series[2])},
	})
	exp.Code("s.LineChart(series)", "go")

	// New in 0.2.0: tabs grouping more charts and widgets.
	s.Divider()
	s.Header("More components")
	tabs := s.Tabs([]string{"Charts", "Widgets", "Form", "Chat", "Extras"})

	// Charts tab: scatter, pie, histogram, all server-side SVG.
	xs, ys := scatterData(series)
	tabs[0].Subheader("Scatter")
	tabs[0].ScatterChart(xs, ys)
	tabs[0].Subheader("Distribution")
	tabs[0].Histogram(series, 12)
	tabs[0].Subheader("Waveform mix")
	tabs[0].PieChart([]float64{3, 5, 2}, []string{"sine", "cosine", "sawtooth"})

	// Widgets tab: toggle, color picker, select-slider, feedback, download.
	w := tabs[1]
	if w.Toggle("Show a cached fact", false) {
		fact := s.Cache("fact", func() any { return expensiveFact() })
		w.Info(fact.(string))
	}
	col := w.ColorPicker("Pick a colour", "#4c78a8")
	w.Metric("Selected colour", col, "")
	size := w.SelectSlider("T-shirt size", []string{"XS", "S", "M", "L", "XL"})
	w.Write("You chose size **" + size + "**.")
	if r := w.Feedback("stars"); r >= 0 {
		w.Caption("Thanks for rating: " + itoa(r+1) + "/5")
	}
	w.DownloadButton("Download series as CSV", "series.csv", []byte(seriesCSV(series)))

	// Form tab: batched submission.
	f := tabs[2].Form("profile")
	name := f.TextInput("Name", "")
	age := int(f.NumberInput("Age", 30))
	if f.FormSubmitButton("Save") {
		s.State.Set("profile", name)
	}
	if saved := s.State.GetString("profile", ""); saved != "" {
		tabs[2].Success("Saved profile for " + saved + " (age " + itoa(age) + ")")
	}

	// Chat tab: a tiny echo assistant backed by session state.
	chat := tabs[3]
	history := getHistory(s)
	for _, msg := range history {
		chat.ChatMessage(msg.role).Markdown(msg.text)
	}
	if msg := chat.ChatInput("Say hi to the echo bot"); msg != "" {
		history = append(history,
			chatMsg{"user", msg},
			chatMsg{"assistant", "You said: _" + msg + "_"})
		s.State.Set("history", history)
	}

	extras(s, tabs[4], series)
}

// extras exercises the widgets, effects and layout helpers that round out the
// port's parity with Streamlit: chip selectors, range sliders, weighted
// columns, badges, link buttons, LaTeX, toasts and the one-shot animations.
func extras(s *st.Session, t *st.Container, series []float64) {
	t.Subheader("Chips and ranges")

	// st.pills / st.segmented_control.
	langs := t.Pills("Languages you use", []string{"Go", "Python", "Rust", "TypeScript"})
	mode := t.SegmentedControl("View", []string{"Compact", "Comfortable", "Spacious"})
	t.Write("Selected " + itoa(len(langs)) + " language(s); view is **" + mode + "**.")

	// st.slider with a tuple default, and st.select_slider with a tuple.
	lo, hi := t.SliderRange("Value window", -10, 10, -3, 6, 0.5)
	from, to := t.SelectSliderRange("Size range", []string{"XS", "S", "M", "L", "XL"})
	t.Caption("Window " + ftoa(lo) + " to " + ftoa(hi) + "; sizes " + from + "–" + to)

	// Values in the window, plotted in a 3:1 split (st.columns([3, 1])).
	inWindow := filterRange(series, lo, hi)
	cols := t.ColumnsWeighted([]float64{3, 1})
	cols[0].AreaChart(inWindow)
	cols[1].Metric("In window", itoa(len(inWindow)), "")

	// st.date_input with a tuple default.
	start, end := t.DateRangeInput("Reporting period", "2026-01-01", "2026-03-31")
	t.Caption("Period: " + start + " → " + end)

	t.Divider()
	t.Subheader("Labels, links and maths")
	t.Badge("stable", "green")
	t.Badge("std-lib only", "violet")
	t.LinkButton("Streamlit docs", "https://docs.streamlit.io")
	t.PageLink("https://pkg.go.dev/github.com/malcolmston/streamlit/st", "API reference")
	t.Latex(`f(x) = A\sin(2\pi x)`)

	// st.markdown is safe by default: this renders as literal text, not a tag.
	t.Markdown("User-supplied text is escaped: `<b>not bold</b>`")

	t.Divider()
	t.Subheader("Status and effects")
	if t.Button("Celebrate") {
		t.Toast("Nice one!", "🎉")
		t.Balloons()
	}
	if t.Button("Let it snow") {
		t.Snow()
	}
	t.Exception(errDemo)
	t.Help(struct {
		Points int
		Peak   float64
	}{len(series), peak(series)})

	// st.rerun: flip a mode and repaint the whole page immediately.
	if t.Button("Reset the counter and rerun") {
		s.State.Set("count", 0)
		s.Rerun()
	}
}

// errDemo is a stand-in error for the st.exception demonstration.
var errDemo = errors.New("something went wrong (this is a demo)")

// filterRange returns the values of xs that fall inside [lo, hi].
func filterRange(xs []float64, lo, hi float64) []float64 {
	out := make([]float64, 0, len(xs))
	for _, x := range xs {
		if x >= lo && x <= hi {
			out = append(out, x)
		}
	}
	return out
}

// chatMsg is a single stored chat turn.
type chatMsg struct {
	role string
	text string
}

// getHistory returns the chat history stored in session state.
func getHistory(s *st.Session) []chatMsg {
	if v, ok := s.State.Get("history"); ok {
		if h, ok := v.([]chatMsg); ok {
			return h
		}
	}
	return nil
}

// scatterData pairs a series against its own shifted copy for a demo scatter.
func scatterData(series []float64) (xs, ys []float64) {
	for i := range series {
		xs = append(xs, series[i])
		ys = append(ys, series[(i+3)%len(series)])
	}
	return xs, ys
}

// seriesCSV formats a series as CSV text for the download button.
func seriesCSV(series []float64) string {
	var b strings.Builder
	b.WriteString("i,value\n")
	for i, v := range series {
		b.WriteString(itoa(i))
		b.WriteByte(',')
		b.WriteString(ftoa(v))
		b.WriteByte('\n')
	}
	return b.String()
}

// expensiveFact simulates a costly computation memoised by st.Cache.
func expensiveFact() string {
	time.Sleep(150 * time.Millisecond)
	return "This sentence was computed once and cached across every rerun."
}

// twoCols returns two equal columns.
func twoCols(s *st.Session) (*st.Container, *st.Container) {
	cols := s.Columns(2)
	return cols[0], cols[1]
}

// generate produces a numeric series for the chosen waveform.
func generate(wave string, amp float64, n int) []float64 {
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		x := float64(i) / float64(n) * 4 * math.Pi
		switch wave {
		case "cosine":
			out[i] = amp * math.Cos(x)
		case "sawtooth":
			out[i] = amp * (2 * (x/(2*math.Pi) - math.Floor(0.5+x/(2*math.Pi))))
		default:
			out[i] = amp * math.Sin(x)
		}
	}
	return out
}

// peak returns the maximum absolute value of a series.
func peak(xs []float64) float64 {
	m := 0.0
	for _, x := range xs {
		if math.Abs(x) > m {
			m = math.Abs(x)
		}
	}
	return m
}

// itoa formats an int.
func itoa(n int) string { return strconv.Itoa(n) }

// ftoa formats a float with one decimal place.
func ftoa(f float64) string { return strconv.FormatFloat(f, 'f', 1, 64) }

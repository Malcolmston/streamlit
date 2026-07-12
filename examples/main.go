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
	"log"
	"math"
	"strconv"

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

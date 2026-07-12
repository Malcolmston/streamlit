package st

import (
	"fmt"
	"math"
	"strings"
)

// chartPalette provides distinguishable series colours.
var chartPalette = []string{
	"#4c78a8", "#f58518", "#54a24b", "#e45756",
	"#72b7b2", "#b279a2", "#ff9da6", "#9d755d",
}

// LineChart adds a line chart. Each argument is one numeric series; series are
// plotted against a shared 0..N-1 x-axis and rendered to inline SVG on the
// server using only the standard library.
func (c *Container) LineChart(series ...[]float64) {
	c.add("chart", props{"svg": renderChart("line", series, 640, 320)})
}

// AreaChart adds a filled area chart. See [Container.LineChart] for the series
// convention.
func (c *Container) AreaChart(series ...[]float64) {
	c.add("chart", props{"svg": renderChart("area", series, 640, 320)})
}

// BarChart adds a bar chart. See [Container.LineChart] for the series
// convention. Multiple series are drawn as grouped bars.
func (c *Container) BarChart(series ...[]float64) {
	c.add("chart", props{"svg": renderChart("bar", series, 640, 320)})
}

// renderChart produces a self-contained SVG document string for the given
// chart kind and series. It is deliberately free of external dependencies so
// it can be unit-tested and embedded directly into the element tree.
//
// The rendering is intentionally simple but complete: it computes shared value
// bounds across all series, draws a framed plot area with a baseline, and
// emits polylines, filled areas, or grouped bars accordingly.
func renderChart(kind string, series [][]float64, width, height int) string {
	const padL, padR, padT, padB = 40, 20, 20, 30
	plotW := float64(width - padL - padR)
	plotH := float64(height - padT - padB)

	maxLen := 0
	minV, maxV := math.Inf(1), math.Inf(-1)
	for _, s := range series {
		if len(s) > maxLen {
			maxLen = len(s)
		}
		for _, v := range s {
			minV = math.Min(minV, v)
			maxV = math.Max(maxV, v)
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img">`,
		width, height, width, height)

	if maxLen == 0 {
		// Nothing to plot; return an empty framed chart.
		fmt.Fprintf(&b, `<rect x="%d" y="%d" width="%.0f" height="%.0f" fill="none" stroke="#ddd"/>`,
			padL, padT, plotW, plotH)
		b.WriteString(`<text x="50%" y="50%" text-anchor="middle" fill="#999" font-family="sans-serif" font-size="12">no data</text></svg>`)
		return b.String()
	}

	if math.IsInf(minV, 1) {
		minV, maxV = 0, 1
	}
	// Include the zero baseline so bars/areas are anchored sensibly.
	if minV > 0 {
		minV = 0
	}
	if maxV < 0 {
		maxV = 0
	}
	if maxV == minV {
		maxV = minV + 1
	}

	// Coordinate mappers.
	xAt := func(i int) float64 {
		if maxLen == 1 {
			return float64(padL) + plotW/2
		}
		return float64(padL) + plotW*float64(i)/float64(maxLen-1)
	}
	yAt := func(v float64) float64 {
		return float64(padT) + plotH*(1-(v-minV)/(maxV-minV))
	}

	// Plot frame and zero baseline.
	fmt.Fprintf(&b, `<rect x="%d" y="%d" width="%.0f" height="%.0f" fill="none" stroke="#e0e0e0"/>`,
		padL, padT, plotW, plotH)
	zeroY := yAt(0)
	fmt.Fprintf(&b, `<line x1="%d" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#ccc" stroke-dasharray="2,2"/>`,
		padL, zeroY, float64(padL)+plotW, zeroY)

	switch kind {
	case "bar":
		renderBars(&b, series, maxLen, xAt, yAt, zeroY, plotW)
	case "area":
		renderAreas(&b, series, xAt, yAt, zeroY)
		renderLines(&b, series, xAt, yAt)
	default: // line
		renderLines(&b, series, xAt, yAt)
	}

	// y-axis min/max labels.
	fmt.Fprintf(&b, `<text x="%d" y="%.1f" text-anchor="end" fill="#666" font-family="sans-serif" font-size="10">%s</text>`,
		padL-4, float64(padT)+8, trimNum(maxV))
	fmt.Fprintf(&b, `<text x="%d" y="%.1f" text-anchor="end" fill="#666" font-family="sans-serif" font-size="10">%s</text>`,
		padL-4, float64(padT)+plotH, trimNum(minV))

	b.WriteString(`</svg>`)
	return b.String()
}

// renderLines draws a polyline per series.
func renderLines(b *strings.Builder, series [][]float64, xAt func(int) float64, yAt func(float64) float64) {
	for si, s := range series {
		if len(s) == 0 {
			continue
		}
		col := chartPalette[si%len(chartPalette)]
		var pts strings.Builder
		for i, v := range s {
			if i > 0 {
				pts.WriteByte(' ')
			}
			fmt.Fprintf(&pts, "%.1f,%.1f", xAt(i), yAt(v))
		}
		fmt.Fprintf(b, `<polyline fill="none" stroke="%s" stroke-width="2" points="%s"/>`, col, pts.String())
	}
}

// renderAreas draws a filled polygon down to the zero baseline per series.
func renderAreas(b *strings.Builder, series [][]float64, xAt func(int) float64, yAt func(float64) float64, zeroY float64) {
	for si, s := range series {
		if len(s) == 0 {
			continue
		}
		col := chartPalette[si%len(chartPalette)]
		var pts strings.Builder
		fmt.Fprintf(&pts, "%.1f,%.1f", xAt(0), zeroY)
		for i, v := range s {
			fmt.Fprintf(&pts, " %.1f,%.1f", xAt(i), yAt(v))
		}
		fmt.Fprintf(&pts, " %.1f,%.1f", xAt(len(s)-1), zeroY)
		fmt.Fprintf(b, `<polygon fill="%s" fill-opacity="0.2" stroke="none" points="%s"/>`, col, pts.String())
	}
}

// renderBars draws grouped bars for all series.
func renderBars(b *strings.Builder, series [][]float64, maxLen int, xAt func(int) float64, yAt func(float64) float64, zeroY, plotW float64) {
	n := len(series)
	if n == 0 {
		return
	}
	slot := plotW / float64(maxLen)
	groupW := slot * 0.7
	barW := groupW / float64(n)
	for si, s := range series {
		col := chartPalette[si%len(chartPalette)]
		for i, v := range s {
			cx := xAt(i) - groupW/2 + barW*float64(si)
			y := yAt(v)
			top := math.Min(y, zeroY)
			h := math.Abs(zeroY - y)
			fmt.Fprintf(b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s"/>`,
				cx, top, barW, h, col)
		}
	}
}

// trimNum formats a float without trailing zeros for compact axis labels.
func trimNum(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-" {
		return "0"
	}
	return s
}

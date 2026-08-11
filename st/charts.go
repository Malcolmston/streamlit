package st

import (
	"fmt"
	"math"
	"strings"
)

// finite reports whether v is a real, plottable number. NaN and ±Inf are
// filtered out of every chart path: without the guard a single NaN in a series
// poisons the shared min/max scan (math.Min(NaN, x) is NaN), and every
// coordinate in the emitted SVG becomes the literal text "NaN", which browsers
// discard — one bad sample would blank the whole chart rather than skip a point.
func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

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

// ScatterChart adds a scatter plot of the paired xs and ys values, rendered to
// inline SVG on the server. Points beyond the shorter of the two slices are
// ignored. Unlike the line/bar/area charts, the axes are scaled to the data
// rather than anchored at zero.
func (c *Container) ScatterChart(xs, ys []float64) {
	c.add("chart", props{"svg": renderScatter(xs, ys, 640, 320)})
}

// PieChart adds a pie chart. values gives each slice's magnitude and the
// optional labels name the slices in order. Non-positive totals render an empty
// chart.
func (c *Container) PieChart(values []float64, labels []string) {
	c.add("chart", props{"svg": renderPie(values, labels, 640, 320)})
}

// Histogram adds a histogram of data grouped into the given number of bins
// (clamped to at least one), rendered as a bar chart. It is a convenience over
// bucketing the data yourself and calling [Container.BarChart].
func (c *Container) Histogram(data []float64, bins int) {
	c.add("chart", props{"svg": renderHistogram(data, bins, 640, 320)})
}

// histogramCounts buckets data into bins equal-width bins spanning its range
// and returns the per-bin counts. With fewer than one bin it uses one; when all
// values are equal every value falls in the first bin.
func histogramCounts(data []float64, bins int) []float64 {
	if bins < 1 {
		bins = 1
	}
	counts := make([]float64, bins)
	if len(data) == 0 {
		return counts
	}
	minV, maxV := math.Inf(1), math.Inf(-1)
	n := 0
	for _, v := range data {
		if !finite(v) {
			continue
		}
		n++
		minV = math.Min(minV, v)
		maxV = math.Max(maxV, v)
	}
	if n == 0 {
		return counts
	}
	if maxV == minV {
		counts[0] = float64(n)
		return counts
	}
	width := (maxV - minV) / float64(bins)
	for _, v := range data {
		if !finite(v) {
			continue
		}
		idx := int((v - minV) / width)
		if idx >= bins {
			idx = bins - 1
		}
		if idx < 0 {
			idx = 0
		}
		counts[idx]++
	}
	return counts
}

// renderHistogram buckets data and draws the counts as a bar chart.
func renderHistogram(data []float64, bins, width, height int) string {
	return renderChart("bar", [][]float64{histogramCounts(data, bins)}, width, height)
}

// renderScatter draws paired points as circles with data-scaled axes.
func renderScatter(xs, ys []float64, width, height int) string {
	const padL, padR, padT, padB = 40, 20, 20, 30
	plotW := float64(width - padL - padR)
	plotH := float64(height - padT - padB)

	n := len(xs)
	if len(ys) < n {
		n = len(ys)
	}
	// Only pairs with two finite coordinates are plottable.
	pairs := make([][2]float64, 0, n)
	for i := 0; i < n; i++ {
		if finite(xs[i]) && finite(ys[i]) {
			pairs = append(pairs, [2]float64{xs[i], ys[i]})
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img">`,
		width, height, width, height)
	fmt.Fprintf(&b, `<rect x="%d" y="%d" width="%.0f" height="%.0f" fill="none" stroke="#e0e0e0"/>`,
		padL, padT, plotW, plotH)

	if len(pairs) == 0 {
		b.WriteString(`<text x="50%" y="50%" text-anchor="middle" fill="#999" font-family="sans-serif" font-size="12">no data</text></svg>`)
		return b.String()
	}

	minX, maxX := math.Inf(1), math.Inf(-1)
	minY, maxY := math.Inf(1), math.Inf(-1)
	for _, pt := range pairs {
		minX, maxX = math.Min(minX, pt[0]), math.Max(maxX, pt[0])
		minY, maxY = math.Min(minY, pt[1]), math.Max(maxY, pt[1])
	}
	if maxX == minX {
		maxX = minX + 1
	}
	if maxY == minY {
		maxY = minY + 1
	}
	xAt := func(v float64) float64 { return float64(padL) + plotW*(v-minX)/(maxX-minX) }
	yAt := func(v float64) float64 { return float64(padT) + plotH*(1-(v-minY)/(maxY-minY)) }

	for _, pt := range pairs {
		fmt.Fprintf(&b, `<circle cx="%.1f" cy="%.1f" r="3.5" fill="%s" fill-opacity="0.7"/>`,
			xAt(pt[0]), yAt(pt[1]), chartPalette[0])
	}
	fmt.Fprintf(&b, `<text x="%d" y="%.1f" text-anchor="end" fill="#666" font-family="sans-serif" font-size="10">%s</text>`,
		padL-4, float64(padT)+8, trimNum(maxY))
	fmt.Fprintf(&b, `<text x="%d" y="%.1f" text-anchor="end" fill="#666" font-family="sans-serif" font-size="10">%s</text>`,
		padL-4, float64(padT)+plotH, trimNum(minY))
	b.WriteString(`</svg>`)
	return b.String()
}

// renderPie draws a pie chart from values and optional labels.
func renderPie(values []float64, labels []string, width, height int) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img">`,
		width, height, width, height)

	total := 0.0
	for _, v := range values {
		if finite(v) && v > 0 {
			total += v
		}
	}
	cx, cy := float64(height)/2, float64(height)/2
	r := float64(height)/2 - 20
	if total <= 0 {
		fmt.Fprintf(&b, `<circle cx="%.1f" cy="%.1f" r="%.1f" fill="none" stroke="#ddd"/>`, cx, cy, r)
		b.WriteString(`<text x="50%" y="50%" text-anchor="middle" fill="#999" font-family="sans-serif" font-size="12">no data</text></svg>`)
		return b.String()
	}

	angle := -math.Pi / 2 // start at 12 o'clock
	legendX := float64(height) + 10
	legendY := 24.0
	for i, v := range values {
		if !finite(v) || v <= 0 {
			continue
		}
		frac := v / total
		next := angle + frac*2*math.Pi
		col := chartPalette[i%len(chartPalette)]
		x1, y1 := cx+r*math.Cos(angle), cy+r*math.Sin(angle)
		x2, y2 := cx+r*math.Cos(next), cy+r*math.Sin(next)
		large := 0
		if frac > 0.5 {
			large = 1
		}
		fmt.Fprintf(&b, `<path d="M%.1f,%.1f L%.1f,%.1f A%.1f,%.1f 0 %d 1 %.1f,%.1f Z" fill="%s"/>`,
			cx, cy, x1, y1, r, r, large, x2, y2, col)
		label := fmt.Sprintf("%d", i)
		if i < len(labels) {
			label = labels[i]
		}
		fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="12" height="12" fill="%s"/>`, legendX, legendY-10, col)
		fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" fill="#333" font-family="sans-serif" font-size="12">%s (%s%%)</text>`,
			legendX+18, legendY, escapeText(label), trimNum(frac*100))
		legendY += 20
		angle = next
	}
	b.WriteString(`</svg>`)
	return b.String()
}

// escapeText escapes the handful of characters that are unsafe inside SVG text.
func escapeText(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
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
			if !finite(v) {
				continue
			}
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
			if !finite(v) {
				continue
			}
			if pts.Len() > 0 {
				pts.WriteByte(' ')
			}
			fmt.Fprintf(&pts, "%.1f,%.1f", xAt(i), yAt(v))
		}
		if pts.Len() == 0 {
			continue
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
		drawn := 0
		for i, v := range s {
			if !finite(v) {
				continue
			}
			drawn++
			fmt.Fprintf(&pts, " %.1f,%.1f", xAt(i), yAt(v))
		}
		if drawn == 0 {
			continue
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
			if !finite(v) {
				continue
			}
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

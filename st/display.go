package st

import (
	"encoding/json"
	"fmt"
)

// Container is a region of the page into which elements are appended. The root
// container is embedded in [Session] so its methods are reachable directly on
// the session; layout helpers such as [Container.Columns] and
// [Container.Expander] return additional containers that render nested regions.
//
// Every display and widget method lives on *Container, which is what lets the
// same API be used for the main body, the sidebar, and individual columns.
type Container struct {
	s    *Session
	node *Element
	// form is non-empty when this container (or an ancestor it was derived
	// from) is a form created by [Container.Form]. Widgets appended to a form
	// container defer their state updates until the form is submitted; see
	// forms.go.
	form string
}

// child creates a sub-container rooted at node, inheriting this container's
// form association so that layout regions nested inside a form still defer
// their widgets' updates.
func (c *Container) child(node *Element) *Container {
	return &Container{s: c.s, node: node, form: c.form}
}

// Title adds a top-level page title.
func (c *Container) Title(text string) { c.add("title", props{"text": text}) }

// Header adds a section header.
func (c *Container) Header(text string) { c.add("header", props{"text": text}) }

// Subheader adds a smaller section header.
func (c *Container) Subheader(text string) { c.add("subheader", props{"text": text}) }

// Text adds fixed-width, unformatted text.
func (c *Container) Text(text string) { c.add("text", props{"text": text}) }

// Caption adds small, muted caption text (rendered as markdown).
func (c *Container) Caption(text string) { c.add("caption", props{"text": text}) }

// Markdown adds text rendered with a small, safe subset of Markdown
// (headings, bold, italics, inline code, links and lists) by the frontend.
func (c *Container) Markdown(text string) { c.add("markdown", props{"text": text}) }

// Divider adds a horizontal rule.
func (c *Container) Divider() { c.add("divider", nil) }

// Code adds a syntax-highlight-styled code block. lang is an advisory
// language label (for example "go" or "python") and may be empty.
func (c *Container) Code(code, lang string) {
	c.add("code", props{"code": code, "lang": lang})
}

// JSON adds a pretty-printed JSON view of value. Values that cannot be
// marshalled are rendered using their Go default formatting instead. When the
// optional collapsed argument is true the view starts collapsed behind a
// disclosure toggle in the browser.
func (c *Container) JSON(value any, collapsed ...bool) {
	col := len(collapsed) > 0 && collapsed[0]
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		c.add("json", props{"json": fmt.Sprintf("%v", value), "collapsed": col})
		return
	}
	c.add("json", props{"json": string(b), "collapsed": col})
}

// Metric adds a big-number metric with an optional delta indicator. A delta
// beginning with '-' renders as a negative (downward) change; the coloring can
// be flipped or disabled with [Container.MetricColored].
func (c *Container) Metric(label, value, delta string) {
	c.add("metric", props{"label": label, "value": value, "delta": delta, "deltaColor": "normal"})
}

// MetricColored is like [Container.Metric] but controls how the delta is
// coloured. deltaColor is one of "normal" (positive green, negative red),
// "inverse" (the reverse, useful when lower is better), or "off" (grey).
func (c *Container) MetricColored(label, value, delta, deltaColor string) {
	switch deltaColor {
	case "inverse", "off":
	default:
		deltaColor = "normal"
	}
	c.add("metric", props{"label": label, "value": value, "delta": delta, "deltaColor": deltaColor})
}

// Success adds a green success message box.
func (c *Container) Success(text string) { c.add("alert", props{"kind": "success", "text": text}) }

// Info adds a blue informational message box.
func (c *Container) Info(text string) { c.add("alert", props{"kind": "info", "text": text}) }

// Warning adds a yellow warning message box.
func (c *Container) Warning(text string) { c.add("alert", props{"kind": "warning", "text": text}) }

// Error adds a red error message box.
func (c *Container) Error(text string) { c.add("alert", props{"kind": "error", "text": text}) }

// Spinner adds a spinner with a label. In this synchronous MVP the spinner is
// primarily decorative: the element tree is delivered only after the run
// completes, so a spinner marks a region that performed work rather than
// animating during it.
func (c *Container) Spinner(label string) { c.add("spinner", props{"label": label}) }

// Progress adds a progress bar. value is clamped to the range [0, 1].
func (c *Container) Progress(value float64) {
	c.add("progress", props{"value": clampFloat(value, 0, 1)})
}

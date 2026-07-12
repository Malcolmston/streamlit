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
// marshalled are rendered using their Go default formatting instead.
func (c *Container) JSON(value any) {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		c.add("json", props{"json": fmt.Sprintf("%v", value)})
		return
	}
	c.add("json", props{"json": string(b)})
}

// Metric adds a big-number metric with an optional delta indicator. A delta
// beginning with '-' renders as a negative (downward) change.
func (c *Container) Metric(label, value, delta string) {
	c.add("metric", props{"label": label, "value": value, "delta": delta})
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

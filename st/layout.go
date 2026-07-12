package st

// Columns splits the current region into n side-by-side columns and returns a
// container for each. Elements added to a returned container render within that
// column. If n is less than one it is treated as one.
//
//	c1, c2 := s.Columns(2)[0], s.Columns(2)[1] // (illustrative)
//	cols := s.Columns(2)
//	cols[0].Metric("Left", "1", "")
//	cols[1].Metric("Right", "2", "")
func (c *Container) Columns(n int) []*Container {
	if n < 1 {
		n = 1
	}
	row := c.add("columns", props{"n": n})
	cols := make([]*Container, n)
	for i := range cols {
		col := &Element{Type: "column"}
		row.Children = append(row.Children, col)
		cols[i] = c.child(col)
	}
	return cols
}

// Container adds a nested, undecorated grouping region and returns a container
// for it. It is useful for grouping elements or for building output out of
// order.
func (c *Container) Container() *Container {
	node := c.add("container", nil)
	return c.child(node)
}

// Expander adds a collapsible section with the given label and returns a
// container for its body. When expanded is true the section starts open.
func (c *Container) Expander(label string, expanded bool) *Container {
	node := c.add("expander", props{"label": label, "expanded": expanded})
	return c.child(node)
}

// Tabs adds a tabbed region with one tab per label and returns a container for
// each tab's body, in the same order as labels. Switching between tabs happens
// entirely in the browser and does not rerun the app. If labels is empty a
// single unnamed tab is created.
//
//	tabs := s.Tabs([]string{"Chart", "Data"})
//	tabs[0].LineChart(series)
//	tabs[1].DataFrame(rows)
func (c *Container) Tabs(labels []string) []*Container {
	if len(labels) == 0 {
		labels = []string{""}
	}
	row := c.add("tabs", props{"labels": toAnySlice(labels)})
	out := make([]*Container, len(labels))
	for i := range labels {
		tab := &Element{Type: "tab", Props: map[string]any{"label": labels[i]}}
		row.Children = append(row.Children, tab)
		out[i] = c.child(tab)
	}
	return out
}

// Popover adds a button that reveals a small floating panel when clicked and
// returns a container for the panel's body. The panel is toggled in the browser
// without rerunning the app, making it a convenient home for extra controls or
// help text.
func (c *Container) Popover(label string) *Container {
	node := c.add("popover", props{"label": label})
	return c.child(node)
}

// Status adds a collapsible status box with a label and a visual state and
// returns a container for its body. The optional state is one of "running"
// (default), "complete", or "error"; it selects the icon shown beside the
// label. Because the app reruns top to bottom, updating a status is simply a
// matter of calling Status again with a new state on the next run.
//
//	st := s.Status("Crunching numbers…", "running")
//	st.Write("step 1 done")
func (c *Container) Status(label string, state ...string) *Container {
	st := "running"
	if len(state) > 0 && state[0] != "" {
		st = state[0]
	}
	node := c.add("status", props{"label": label, "state": st})
	return c.child(node)
}

// Empty adds a single placeholder region and returns a container for it.
// Because the element tree is rebuilt on every run, an Empty is most useful as
// a stable slot whose contents are decided later in the same run.
func (c *Container) Empty() *Container {
	node := c.add("empty", nil)
	return c.child(node)
}

// toAnySlice converts a string slice to []any for JSON-friendly props.
func toAnySlice(xs []string) []any {
	out := make([]any, len(xs))
	for i, x := range xs {
		out[i] = x
	}
	return out
}

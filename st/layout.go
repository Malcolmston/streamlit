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
		cols[i] = &Container{s: c.s, node: col}
	}
	return cols
}

// Container adds a nested, undecorated grouping region and returns a container
// for it. It is useful for grouping elements or for building output out of
// order.
func (c *Container) Container() *Container {
	node := c.add("container", nil)
	return &Container{s: c.s, node: node}
}

// Expander adds a collapsible section with the given label and returns a
// container for its body. When expanded is true the section starts open.
func (c *Container) Expander(label string, expanded bool) *Container {
	node := c.add("expander", props{"label": label, "expanded": expanded})
	return &Container{s: c.s, node: node}
}

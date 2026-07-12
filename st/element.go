package st

// Element is a single node in the page's element tree.
//
// The element tree is the serializable description of everything the app
// wants the browser to display for the current run. Each call such as
// [Container.Title] or [Container.Slider] appends one Element to the tree.
// Elements are marshalled to JSON and shipped to the embedded frontend, which
// renders them and, for widgets, sends interaction events back to the server.
//
// An Element carries a Type (interpreted by the frontend renderer), an
// arbitrary Props bag, an optional Key (stable identity used for widgets), and
// nested Children (used by layout containers such as columns and expanders).
type Element struct {
	// Type identifies how the frontend should render the node, for example
	// "title", "slider" or "columns".
	Type string `json:"type"`
	// Key is the stable widget identity. It is empty for non-widget
	// elements. Widget interaction events reference this value.
	Key string `json:"key,omitempty"`
	// Props holds the element's rendered attributes (labels, values, SVG
	// markup, table rows, and so on).
	Props map[string]any `json:"props,omitempty"`
	// Children holds nested elements for layout containers.
	Children []*Element `json:"children,omitempty"`
}

// props is a small constructor helper for building an Element's property bag.
type props map[string]any

// add appends a new child Element to the container's current node and returns
// it so callers may attach children of their own.
func (c *Container) add(typ string, p props) *Element {
	el := &Element{Type: typ, Props: map[string]any(p)}
	c.node.Children = append(c.node.Children, el)
	return el
}

// addWidget appends a keyed widget Element, recording its key in Props so the
// frontend can echo it back on interaction.
func (c *Container) addWidget(key, typ string, p props) *Element {
	if p == nil {
		p = props{}
	}
	// Widgets appended inside a form carry the form's key so the frontend can
	// stage their changes and only commit them on submission.
	if c.form != "" {
		p["form"] = c.form
	}
	el := &Element{Type: typ, Key: key, Props: map[string]any(p)}
	c.node.Children = append(c.node.Children, el)
	return el
}

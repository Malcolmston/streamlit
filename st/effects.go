package st

import "fmt"

// Latex adds a block of mathematics written in LaTeX. expr is the raw LaTeX
// source without surrounding delimiters, for example `\int_a^b f(x)\,dx`. It
// mirrors Streamlit's st.latex; the embedded frontend renders the expression in
// a dedicated math block.
func (c *Container) Latex(expr string) { c.add("latex", props{"expr": expr}) }

// Html adds a block of raw, unescaped HTML, mirroring Streamlit's st.html. The
// markup is inserted verbatim, so only pass HTML you trust — untrusted input can
// inject script into the page.
func (c *Container) Html(html string) { c.add("html", props{"html": html}) }

// Badge adds a small coloured pill label, mirroring Streamlit's st.badge. color
// is an optional colour name (for example "blue", "green", "red", "orange",
// "violet" or "grey"); an empty or unrecognised colour defaults to "blue".
func (c *Container) Badge(label string, color ...string) {
	col := "blue"
	if len(color) > 0 {
		switch color[0] {
		case "blue", "green", "red", "orange", "violet", "grey", "gray":
			col = color[0]
			if col == "gray" {
				col = "grey"
			}
		}
	}
	c.add("badge", props{"label": label, "color": col})
}

// Exception adds a formatted error box for err, mirroring Streamlit's
// st.exception. The error's message and its Go type name are shown. A nil error
// renders an empty exception box.
func (c *Container) Exception(err error) {
	msg, typ := "", ""
	if err != nil {
		msg = err.Error()
		typ = fmt.Sprintf("%T", err)
	}
	c.add("exception", props{"message": msg, "type": typ})
}

// Echo adds a read-only display of Go source code, mirroring Streamlit's
// st.echo (which shows the code inside its block). The code is rendered as a
// syntax-styled block tagged as Go.
func (c *Container) Echo(code string) {
	c.add("echo", props{"code": code, "lang": "go"})
}

// Toast adds a transient notification message, mirroring Streamlit's st.toast.
// An optional leading icon (typically an emoji) may be supplied as the second
// argument.
func (c *Container) Toast(message string, icon ...string) {
	c.add("toast", props{"message": message, "icon": optKey(icon)})
}

// Balloons triggers the celebratory balloons animation, mirroring Streamlit's
// st.balloons. It adds a one-shot effect element to the page.
func (c *Container) Balloons() { c.add("effect", props{"kind": "balloons"}) }

// Snow triggers the falling-snow animation, mirroring Streamlit's st.snow. It
// adds a one-shot effect element to the page.
func (c *Container) Snow() { c.add("effect", props{"kind": "snow"}) }

// LinkButton adds a button that navigates to url when clicked, mirroring
// Streamlit's st.link_button. Unlike [Container.Button] it does not rerun the
// app; it is a styled hyperlink.
func (c *Container) LinkButton(label, url string) {
	c.add("link_button", props{"label": label, "url": url})
}

// PageLink adds a navigational link to url shown with the given label,
// mirroring Streamlit's st.page_link. It is rendered as a prominent link rather
// than a button.
func (c *Container) PageLink(url, label string) {
	c.add("page_link", props{"url": url, "label": label})
}

// Help adds an introspective description of value, mirroring Streamlit's
// st.help. The value's Go type and its formatted representation are shown, which
// is handy for quickly inspecting a variable while building an app.
func (c *Container) Help(value any) {
	c.add("help", props{"type": fmt.Sprintf("%T", value), "value": fmt.Sprintf("%+v", value)})
}

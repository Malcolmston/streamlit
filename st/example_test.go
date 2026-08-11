package st

import "fmt"

// ExampleSession demonstrates building a page and inspecting the resulting
// element tree without an HTTP server. Rendering an app function against a
// session produces a deterministic tree that can be walked and asserted on.
func ExampleSession() {
	app := func(s *Session) {
		s.Title("Report")
		n := s.Slider("Count", 0, 10, 3, 1)
		s.Metric("Count", fmt.Sprintf("%.0f", n), "+1")
	}

	s := newSession()
	tree := s.run(app)

	// The main region holds the three elements in call order.
	main := tree.Children[1]
	for _, el := range main.Children {
		fmt.Println(el.Type)
	}

	// Output:
	// title
	// slider
	// metric
}

// ExampleSession_rerun shows st.rerun's control flow: the run is abandoned at
// the call and the app executes again from the top, so the second pass sees the
// state written just before it and the discarded elements never reach the page.
func ExampleSession_rerun() {
	app := func(s *Session) {
		if !s.State.GetBool("ready", false) {
			s.Text("loading…") // discarded: this pass is thrown away
			s.State.Set("ready", true)
			s.Rerun()
		}
		s.Title("Dashboard")
	}

	s := newSession()
	tree := s.run(app)

	main := tree.Children[1]
	fmt.Println("elements:", len(main.Children))
	fmt.Println("first:", main.Children[0].Type)

	// Output:
	// elements: 1
	// first: title
}

// ExampleSession_stop shows st.stop: elements added before the call stay on the
// page and nothing after it runs.
func ExampleSession_stop() {
	app := func(s *Session) {
		s.Title("Private area")
		if !s.State.GetBool("authenticated", false) {
			s.Error("Please log in")
			s.Stop()
		}
		s.Text("secrets")
	}

	s := newSession()
	main := s.run(app).Children[1]
	for _, el := range main.Children {
		fmt.Println(el.Type)
	}

	// Output:
	// title
	// alert
}

// ExampleSession_state shows the two lifetimes an app has to reason about.
// State survives every rerun; a widget's value survives only as long as the app
// keeps rendering that widget.
func ExampleSession_state() {
	app := func(s *Session) {
		s.State.SetDefault("runs", 0)
		s.State.Set("runs", s.State.GetInt("runs", 0)+1)
		s.TextInput("Name", "anon", "name")
	}

	s := newSession()
	s.run(app)
	s.widgets["name"] = "ada" // stand-in for a browser interaction
	tree := s.run(app)

	fmt.Println("runs:", s.State.GetInt("runs", 0))
	fmt.Println("name:", tree.Children[1].Children[0].Props["value"])
	fmt.Println("keys:", s.State.Keys())

	// Output:
	// runs: 2
	// name: ada
	// keys: [runs]
}

// ExampleContainer_columnsWeighted lays out a wide chart beside a narrow
// summary, mirroring st.columns([3, 1]).
func ExampleContainer_columnsWeighted() {
	app := func(s *Session) {
		cols := s.ColumnsWeighted([]float64{3, 1})
		cols[0].LineChart([]float64{1, 4, 9})
		cols[1].Metric("Peak", "9", "+5")
	}

	s := newSession()
	row := s.run(app).Children[1].Children[0]
	for _, col := range row.Children {
		fmt.Printf("%v -> %s\n", col.Props["weight"], col.Children[0].Type)
	}

	// Output:
	// 3 -> chart
	// 1 -> metric
}

// ExampleSession_cacheResource shows the @st.cache_resource analogue: a
// singleton built once and shared by every session, unlike Cache which holds
// data and can expire.
func ExampleSession_cacheResource() {
	CacheResourceClear()

	builds := 0
	open := func() any {
		builds++
		return "connection"
	}

	a, b := newSession(), newSession()
	fmt.Println(a.CacheResource("db", open))
	fmt.Println(b.CacheResource("db", open))
	fmt.Println("builds:", builds)

	// Output:
	// connection
	// connection
	// builds: 1
}

// ExampleContainer_markdownUnsafe contrasts the safe default with the opt-in.
// Markdown ships the text for escaping in the browser; MarkdownUnsafe flags it
// to be passed through as HTML, which is why it must never carry untrusted
// input.
func ExampleContainer_markdownUnsafe() {
	app := func(s *Session) {
		s.Markdown(`<b>from a user</b>`)
		s.MarkdownUnsafe(`<b>from me</b>`)
	}

	s := newSession()
	for _, el := range s.run(app).Children[1].Children {
		fmt.Println(el.Props["text"], "unsafe:", el.Props["unsafeAllowHTML"])
	}

	// Output:
	// <b>from a user</b> unsafe: <nil>
	// <b>from me</b> unsafe: true
}

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

package st

// Form adds a form region identified by key and returns a container for its
// body. Widgets appended to the returned container defer their state updates:
// interacting with them does not rerun the app. Instead the browser stages
// their values locally and commits them all at once when a
// [Container.FormSubmitButton] inside the form is clicked, at which point the
// app reruns with every staged value applied together.
//
// This batching is the point of a form: several inputs can be changed and then
// submitted as a unit, avoiding a rerun per keystroke.
//
//	f := s.Form("signup")
//	name := f.TextInput("Name", "")
//	age := f.NumberInput("Age", 18)
//	if f.FormSubmitButton("Register") {
//		s.Success("Welcome, " + name)
//		_ = age
//	}
func (c *Container) Form(key string) *Container {
	node := c.add("form", props{"key": key})
	child := c.child(node)
	child.form = key
	return child
}

// FormSubmitButton adds the submit button for the enclosing form and returns
// true on the single run triggered by its click. It must be called on a
// container returned by [Container.Form]; used elsewhere it behaves like a
// disconnected button that never batches values.
func (c *Container) FormSubmitButton(label string) bool {
	// A form submit button uses the form's key as its own widget key so the
	// frontend can associate the staged values with the click.
	k := c.form
	if k == "" {
		k = c.s.key("", "form_submit")
	}
	clicked := c.s.clicked[k]
	c.addWidget(k, "form_submit", props{"label": label})
	return clicked
}

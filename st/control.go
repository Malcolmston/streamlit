package st

// stopSignal is the sentinel value panicked by [Session.Stop] to unwind the
// app function. It is recovered by the session's run loop (see runApp in
// session.go) so that halting a run is a clean, non-fatal operation.
type stopSignal struct{}

// Stop immediately halts execution of the current run of the app function,
// mirroring Streamlit's st.stop. Elements added before the call remain on the
// page; nothing after it runs. It is implemented by panicking with an internal
// sentinel that the run loop recovers, so it must be called from within the app
// function (not from a separate goroutine).
//
//	if !authenticated {
//		s.Error("Please log in")
//		s.Stop() // the rest of the app does not run
//	}
func (s *Session) Stop() { panic(stopSignal{}) }

// rerunSignal is the sentinel value panicked by [Session.Rerun] to unwind and
// restart the app function. Like stopSignal it is recovered by the run loop.
type rerunSignal struct{}

// Rerun abandons the current run and immediately re-executes the app function
// from the top, mirroring Streamlit's st.rerun. Every element added so far is
// discarded; [State] and widget values persist, so the fresh run observes any
// change made before the call.
//
// It is the escape hatch for "I have just mutated state that earlier code
// already rendered" — the canonical example being a login form that must
// repaint the whole page once authentication succeeds:
//
//	if s.Button("Log in") {
//		s.State.Set("user", name)
//		s.Rerun() // repaint the page as the logged-in user
//	}
//
// Like [Session.Stop] it is implemented by panicking with an internal sentinel
// and so must be called from within the app function, not a separate goroutine.
// A chain of reruns within a single request is capped (see maxRerunChain); an
// app that calls Rerun unconditionally therefore terminates with the tree from
// the final permitted run instead of hanging.
func (s *Session) Rerun() { panic(rerunSignal{}) }

// SetPageConfig sets page-level metadata for the app, mirroring Streamlit's
// st.set_page_config. title is the browser tab title and icon is an optional
// favicon (typically an emoji). The values are attached to the root of the
// element tree. Call it once, at the top of the app function.
func (s *Session) SetPageConfig(title, icon string) {
	if s.rootEl == nil {
		return
	}
	if s.rootEl.Props == nil {
		s.rootEl.Props = map[string]any{}
	}
	s.rootEl.Props["pageTitle"] = title
	s.rootEl.Props["pageIcon"] = icon
}

// BorderedContainer adds a nested grouping region drawn with a visible border
// and returns a container for it, mirroring Streamlit's st.container(border=
// True). It behaves like [Container.Container] but is visually delimited, which
// is useful for grouping related output into a card.
func (c *Container) BorderedContainer() *Container {
	node := c.add("container", props{"border": true})
	return c.child(node)
}

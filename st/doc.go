// Package st is a from-scratch, standard-library-only Go port of Python's
// Streamlit: a framework for building interactive web and data apps by writing
// a plain script.
//
// # The model
//
// You write an app function with the signature func(s *st.Session). Inside it
// you call methods on the session to build the page: s.Title, s.Write,
// s.Slider, s.Button and so on. Each call appends one element to the page's
// element tree, and each widget call returns the widget's current value. The
// framework runs your app function top to bottom to produce the page.
//
// The defining behaviour of Streamlit, reproduced faithfully here, is
// rerun-on-interaction. When the user interacts with any widget, the entire
// app function runs again from the top. Widget values are restored from
// per-session state, so a slider you dragged reads its new position on the
// next run, and any [State] you stored persists too. The freshly built element
// tree is diffed against the page and the browser updates. This means app code
// can be written as a simple straight-line script: there are no callbacks,
// event handlers, or manual DOM updates.
//
// # Sessions and containers
//
// A [Session] represents one browser connection. It embeds the root
// [Container], so all display and widget methods are available directly on the
// session. Layout helpers such as [Container.Columns] and [Container.Expander]
// return further containers that render nested regions; [Session.Sidebar]
// returns the sidebar container. Because every method lives on *Container, the
// same API works uniformly for the main body, the sidebar and each column.
//
// # Transport and frontend
//
// [Run] starts an HTTP server. It serves a small, dependency-free single-page
// frontend (embedded via go:embed) that renders the JSON element tree and
// posts widget-change events back to POST /api/run. The server applies the
// event, reruns the app function for that session, and returns the new tree.
// The whole protocol is plain JSON request/response — no third-party modules,
// no JavaScript frameworks, no external charting library (charts are rendered
// to inline SVG on the server).
//
// # Example
//
//	package main
//
//	import "github.com/malcolmston/streamlit/st"
//
//	func main() {
//		st.Run(app, ":8501")
//	}
//
//	func app(s *st.Session) {
//		s.Title("Hello, Streamlit-Go")
//		n := s.Slider("Points", 0, 100, 50, 1)
//		s.LineChart(series(int(n)))
//		if s.Button("Celebrate") {
//			s.Success("🎉")
//		}
//	}
//
// # Deferred features
//
// This is an MVP of a large framework. Deliberately deferred: custom
// components, caching decorators (st.cache_data/st.cache_resource), file
// uploads, multipage apps, real-time streaming/async widgets, dataframe
// editing, theming APIs and forms. The synchronous long-poll-free transport
// also means [Container.Spinner] and [Container.Progress] are snapshots of the
// completed run rather than live-updating during work.
package st

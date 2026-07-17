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
// # Widgets, layout, media and charts
//
// Beyond the basics the package offers a broad surface, each piece backed by
// per-session state and rendered by the embedded frontend:
//
//   - Display and data: [Container.Title], [Container.Header],
//     [Container.Subheader], [Container.Markdown], [Container.Write],
//     [Container.Metric], [Container.Code], [Container.JSON],
//     [Container.Table] and [Container.DataFrame]. Tabular structs and slices
//     render automatically, and alerts ([Container.Success], [Container.Info],
//     [Container.Warning], [Container.Error]) draw the callout boxes.
//   - Widgets: [Container.Button], [Container.Checkbox], [Container.Toggle],
//     [Container.Slider], [Container.SelectSlider], [Container.NumberInput],
//     [Container.TextInput], [Container.TextArea], [Container.SelectBox],
//     [Container.Radio], [Container.MultiSelect], [Container.DateInput],
//     [Container.TimeInput], [Container.ColorPicker], [Container.Feedback],
//     [Container.DownloadButton] and [Container.FileUploader].
//   - Layout: [Container.Columns], [Container.Container], [Container.Expander],
//     [Container.Tabs], [Container.Popover], [Container.Status],
//     [Container.Empty] and [Session.Sidebar].
//   - Forms: [Container.Form] with [Container.FormSubmitButton] batch widget
//     updates until the form is submitted.
//   - Chat: [Container.ChatMessage] and [Container.ChatInput].
//   - Media: [Container.Image], [Container.Logo], [Container.Audio],
//     [Container.Video] and [Container.Map]. Bytes and [image.Image] values are
//     embedded as base64 data URIs.
//   - Charts (server-side SVG): [Container.LineChart], [Container.AreaChart],
//     [Container.BarChart], [Container.ScatterChart], [Container.PieChart] and
//     [Container.Histogram].
//   - Caching: [Session.Cache] memoises an expensive computation process-wide,
//     across reruns and sessions, with an optional TTL — the analogue of
//     st.cache_data. [CacheClear] evicts every cached entry.
//
// # Deferred features
//
// This is a compact reimplementation of a large framework. Deliberately
// deferred: custom components, resource caching (st.cache_resource),
// multipage apps, real-time streaming/async widgets, in-place dataframe
// editing and theming APIs. The synchronous long-poll-free transport also
// means [Container.Spinner] and [Container.Progress] are snapshots of the
// completed run rather than live-updating during work.
package st

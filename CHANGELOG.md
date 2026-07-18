# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] - 2026-07-18
### Added
- Display & effects (closer to Streamlit's `st` surface): `Latex`, `Html`,
  `Badge`, `Exception`, `Echo`, `Toast`, `Balloons`, `Snow`, `LinkButton`,
  `PageLink`, and `Help`.
- Widgets: `Pills` and `SegmentedControl` (chip selectors), `PrimaryButton`,
  `PasswordInput`, `TextInputMax` (server-side max-length), `NumberInputRange`
  (bounded/clamped numeric entry), `SliderRange` and `SelectSliderRange`
  (two-handle range selectors), `DateRangeInput`, and the media-capture inputs
  `CameraInput` and `AudioInput` (reuse the existing `/api/upload` endpoint).
- Control & layout: `Session.Stop` (halts the current run like `st.stop`, via a
  recovered sentinel), `Session.SetPageConfig` (page title/icon, like
  `st.set_page_config`), and `Container.BorderedContainer`
  (`st.container(border=True)`).
- Tests: deterministic known-answer table tests for every new symbol, plus a
  `SliderRange` benchmark.

## [0.2.0] - 2026-07-12
### Added
- Widgets: `Toggle`, `SelectSlider`, `DateInput`, `TimeInput`, `ColorPicker`,
  `Feedback` (stars/thumbs), `DownloadButton` (serves bytes as a data URI), and
  `FileUploader` (multipart upload into per-session state).
- Forms: `Form` and `FormSubmitButton` — widgets inside a form stage their
  values in the browser and commit atomically on submit (new `/api/upload`
  endpoint and batched form-submit events extend the JSON protocol additively).
- Layout: `Tabs`, `Popover`, `Status` (with state), and `Empty`.
- Chat: `ChatMessage` and `ChatInput`.
- Media: `Image` (accepts `image.Image` or PNG/JPEG bytes), `Logo`, `Audio`,
  `Video` (bytes or URL), and `Map` (lat/lng points to inline SVG).
- Charts: `ScatterChart`, `PieChart`, and `Histogram`, rendered server-side to
  inline SVG like the existing charts.
- Caching: `Session.Cache` memoises a computation process-wide across reruns and
  sessions, with an optional TTL — the analogue of `@st.cache_data`.
- Data touches: `Table`/`DataFrame` captions and a client-side sort hint,
  collapsible `JSON`, and `MetricColored` for normal/inverse/off delta coloring.
- Frontend renderers and styling for every new element type, plus an expanded
  `examples/` demo showcasing tabs, the new charts, widgets, a form, and chat.

## [0.1.0] - 2026-07-12
### Added
- Initial release — a standard-library-only Go port of Streamlit for building
  interactive web and data apps by writing a plain Go script.
- Rerun-on-interaction model: an app function `func(s *st.Session)` is re-executed
  top to bottom on every widget interaction, with widget values and per-session
  [State] restored across runs.
- Display API: `Title`, `Header`, `Subheader`, `Text`, `Markdown`, `Caption`,
  `Code`, `JSON`, `Metric`, `Divider`, `Success`/`Info`/`Warning`/`Error`
  alerts, and a type-dispatching `Write`.
- Widget API: `Button`, `Checkbox`, `Slider`, `NumberInput`, `TextInput`,
  `TextArea`, `SelectBox`, `Radio`, and `MultiSelect`.
- Chart API: `LineChart`, `AreaChart`, and `BarChart`, rendered to inline SVG
  on the server with the standard library only.
- Layout: `Columns`, `Container`, `Expander`, and `Sidebar`.
- HTTP transport (`Run` / `Handler`) serving a dependency-free single-page
  frontend embedded via `go:embed`.
- CI: gofmt · vet · build gate a `-race` + coverage run, plus golangci-lint v2,
  govulncheck, cross-compile, dependency review, and VERSION-driven releases.

[Unreleased]: https://github.com/malcolmston/streamlit/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/malcolmston/streamlit/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/malcolmston/streamlit/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/malcolmston/streamlit/releases/tag/v0.1.0

# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/malcolmston/streamlit/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/malcolmston/streamlit/releases/tag/v0.1.0

# streamlit

[![Go Test](https://github.com/Malcolmston/streamlit/actions/workflows/go-test.yml/badge.svg)](https://github.com/Malcolmston/streamlit/actions/workflows/go-test.yml)
[![Go Lint](https://github.com/Malcolmston/streamlit/actions/workflows/go-lint.yml/badge.svg)](https://github.com/Malcolmston/streamlit/actions/workflows/go-lint.yml)
[![Go Vuln](https://github.com/Malcolmston/streamlit/actions/workflows/go-vuln.yml/badge.svg)](https://github.com/Malcolmston/streamlit/actions/workflows/go-vuln.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/malcolmston/streamlit.svg)](https://pkg.go.dev/github.com/malcolmston/streamlit)
[![Go Report Card](https://goreportcard.com/badge/github.com/malcolmston/streamlit)](https://goreportcard.com/report/github.com/malcolmston/streamlit)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Malcolmston/streamlit)](go.mod)
[![Release](https://img.shields.io/github/v/release/Malcolmston/streamlit?sort=semver)](https://github.com/Malcolmston/streamlit/releases)
[![Last Commit](https://img.shields.io/github/last-commit/Malcolmston/streamlit)](https://github.com/Malcolmston/streamlit/commits)
[![Code Size](https://img.shields.io/github/languages/code-size/Malcolmston/streamlit)](https://github.com/Malcolmston/streamlit)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)
[![Docs](https://img.shields.io/badge/docs-pages-2f9bff)](https://malcolmston.github.io/streamlit/)

A standard-library-only Go port of Streamlit — build interactive web & data apps by writing plain Go.

## What it is

[Streamlit](https://streamlit.io) lets you turn a plain script into an interactive
web app: you call display and widget functions top to bottom, and the framework
handles rendering, state, and re-execution. This is a from-scratch,
dependency-free Go port of that idea. You write an app function with the
signature `func(s *st.Session)`, call methods on the session to build the page,
and `st.Run` serves it over HTTP with a small embedded web UI. Nothing but the
Go standard library is used — no web framework, no JavaScript build step, no
external charting library.

## Installation

```sh
go get github.com/malcolmston/streamlit
```

Requires Go 1.24+.

## Quick start

Build a page, wire up a button and a slider, and serve it on `:8501`:

```go
package main

import (
	"fmt"
	"log"

	"github.com/malcolmston/streamlit/st"
)

func main() {
	if err := st.Run(app, ":8501"); err != nil {
		log.Fatal(err)
	}
}

func app(s *st.Session) {
	s.Title("Hello, Streamlit-Go")

	if s.Button("go") {
		s.Success("Button clicked!")
	}

	// Slider(label, min, max, default, step): returns the current value.
	n := s.Slider("n", 0, 100, 50, 1)
	s.Metric("value", fmt.Sprint(n), "")
}
```

Run it and open <http://localhost:8501>. Drag the slider or click the button and
the whole app function reruns — the metric updates to the slider's new value,
with no callbacks or manual DOM code on your part.

## Features

- **Rerun-on-interaction model** — the defining Streamlit behaviour, reproduced
  faithfully: any widget interaction reruns your entire app function from the
  top. App code stays a simple straight-line script — no callbacks, event
  handlers, or manual DOM updates.
- **Session state** — a per-session key/value store (`s.State`, the analogue of
  `st.session_state`) that survives reruns, so counters, history, and cached
  computations persist. Widget values are restored automatically on each run.
- **Display API** — `Title`, `Header`, `Subheader`, `Text`, `Markdown`,
  `Caption`, `Code`, `JSON`, `Metric`, `Divider`, the `Success`/`Info`/
  `Warning`/`Error` alerts, and a Swiss-army `Write` that dispatches on an
  argument's dynamic type.
- **Widget API** — `Button`, `Checkbox`, `Slider`, `NumberInput`, `TextInput`,
  `TextArea`, `SelectBox`, `Radio`, and `MultiSelect`. Every widget returns its
  current value.
- **Chart API** — `LineChart`, `AreaChart`, and `BarChart`, each taking one or
  more numeric series.
- **Layout** — `Columns`, `Container`, `Expander`, and a `Sidebar`. Because
  every method lives on `*Container`, the same API works for the main body, the
  sidebar, and each column.
- **Embedded, dependency-free web UI** — a small single-page frontend is
  embedded with `go:embed` and served by the Go binary. There is no npm build
  and no JavaScript framework.
- **SVG charts** — charts are rendered to inline SVG on the server using only
  the standard library; there is no client-side charting dependency.
- **Zero dependencies** — pure Go standard library; nothing to audit but the
  toolchain.

## How it works

`st.Run` starts an HTTP server that serves the embedded single-page frontend.
The frontend renders the JSON element tree the server produces and posts
widget-change events back to `POST /api/run`. On each event the server restores
that browser's [Session], applies the event to session state, and reruns your
app function from the top. Running the app rebuilds the page's element tree from
scratch while widget values and `s.State` persist — a slider you dragged reads
its new position on the next run, and anything you stored in state carries over.
The fresh tree is returned as JSON and the browser updates. This is what lets
app code be written as a plain top-to-bottom script instead of a graph of
callbacks.

## Documentation

- Full API reference on [pkg.go.dev](https://pkg.go.dev/github.com/malcolmston/streamlit/st).
- See [`examples/main.go`](examples/main.go) for a complete app exercising the
  display, widget, chart, and layout APIs.
- Docs site: <https://malcolmston.github.io/streamlit/>.

## License

MIT

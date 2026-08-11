# API deviations from upstream Streamlit

This port mirrors [streamlit/streamlit](https://github.com/streamlit/streamlit)'s
API on purpose: names and semantics follow the Python original rather than being
Go-ified. Where the two differ, it is either because Go has no equivalent of a
Python feature, or because a Python default is unsafe in a compiled server that
several people share. Every such difference is listed here.

## Naming

Python's `snake_case` module functions become exported Go methods on
`*st.Container` (which `*st.Session` embeds), so `st.line_chart(...)` is
`s.LineChart(...)`. Keyword arguments have no Go equivalent; they appear as
either positional parameters, a trailing variadic, or a differently named
method:

| Streamlit | This port |
| --- | --- |
| `st.markdown(body, unsafe_allow_html=True)` | `MarkdownUnsafe(body)` |
| `st.button(label, type="primary")` | `PrimaryButton(label)` |
| `st.text_input(label, type="password")` | `PasswordInput(label, def)` |
| `st.text_input(label, max_chars=n)` | `TextInputMax(label, def, n)` |
| `st.number_input(label, min_value, max_value)` | `NumberInputRange(label, min, max, def)` |
| `st.slider(label, value=(lo, hi))` | `SliderRange(label, min, max, lo, hi, step)` |
| `st.select_slider(label, value=(lo, hi))` | `SelectSliderRange(label, options)` |
| `st.date_input(label, value=(a, b))` | `DateRangeInput(label, a, b)` |
| `st.columns([3, 1])` | `ColumnsWeighted([]float64{3, 1})` |
| `st.container(border=True)` | `BorderedContainer()` |
| `st.metric(..., delta_color=...)` | `MetricColored(label, value, delta, mode)` |
| `st.cache_data` | `Session.Cache(key, compute, ttl…)` |
| `st.cache_resource` | `Session.CacheResource(key, create)` |

The optional `key=` argument that Streamlit accepts on every widget is the
trailing variadic `key ...string` here.

## Behavioural differences

### Cache keys are explicit

Python derives a cache key by hashing the decorated function's arguments. Go has
no equivalent of that (no decorators, no universal hashing of arbitrary values),
so `Cache` and `CacheResource` take the key directly. Build it out of everything
the computation depends on.

### The data cache is bounded

`@st.cache_data` defaults to `max_entries=None`. Here the table holds at most
`DefaultMaxCacheEntries` (1024) entries and evicts oldest-first; change it with
`CacheSetMaxEntries`. An unbounded cache whose keys derive from user input grows
until the process dies.

`CacheResource` values are never evicted or expired, matching upstream.

### Concurrent cache callers share one computation

Concurrent callers for the same missing key do not each run the compute
function: the first computes and the rest wait for its result. Streamlit's
behaviour here depends on its own locking; this port makes the documented
"runs once" guarantee true under concurrency.

### `Rerun` chains are capped

`st.rerun` restarts the script, and in Streamlit each restart is a fresh
execution driven by the browser. Here the loop is synchronous inside one HTTP
request, so a chain of reruns is capped at 32. An app that calls `Rerun`
unconditionally returns the last permitted run's tree instead of hanging the
request. Transient triggers are cleared between iterations, so the common
`if s.Button(x) { …; s.Rerun() }` idiom performs exactly two executions.

### `Spinner` and `Progress` are snapshots

The transport is a plain request/response, so the element tree is delivered only
once the run has finished. `Spinner` marks a region that performed work and
`Progress` records the value it was last called with; neither animates during
the run.

### Origin checking is on by default

Streamlit's server accepts requests from any origin by default and offers
`server.enableCORS` / `server.enableXsrfProtection` to change that. Here both
POST endpoints reject a request whose `Origin` header names another site, unless
you list it in `Options.AllowedOrigins` or set `Options.AllowAllOrigins`.
Requests with no `Origin` header at all (curl, tests, server-to-server) are
allowed.

### Charts skip non-finite samples

`NaN` and `±Inf` in a series are dropped rather than plotted. Python's plotting
stack treats them as gaps; here they would otherwise poison the shared min/max
scan and emit literal `NaN` coordinates, blanking the whole chart.

### `Write` dispatch

`st.write` inspects Python types (DataFrames, dicts, Matplotlib figures, …).
`Write` dispatches on Go dynamic types instead: `string` → Markdown, `error` →
`Error`, `fmt.Stringer` → `Text`, structs and slices of structs and `[][]string`
→ `Table`, maps and other slices → `JSON`, anything else → `Text`.

## Not implemented

Deliberately deferred, in rough order of how often they are missed:

* **Custom components** (`st.components.v1`) — there is no JavaScript build step
  to hang them off.
* **Multipage apps** (`st.navigation`, `st.Page`, `pages/`) — a single app
  function is the whole surface.
* **`st.data_editor`** — tables are read-only; `DataFrame` adds a client-side
  sort only.
* **Fragments** (`@st.fragment`) and async/streaming widgets
  (`st.write_stream`) — every run is synchronous and whole-page.
* **`st.query_params`, `st.context`, `st.user`** — the app function receives no
  request handle.
* **Theming APIs and `.streamlit/config.toml`** — styling is the embedded
  stylesheet.
* **Connections** (`st.connection`) — use `CacheResource` with your own client.
* **Plotting-library passthroughs** (`st.pyplot`, `st.altair_chart`,
  `st.plotly_chart`) — charts are rendered to SVG on the server by this package.

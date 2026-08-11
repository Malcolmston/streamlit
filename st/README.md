# st — from-scratch, standard-library-only Go port of Python's Streamlit

[![Go Reference](https://pkg.go.dev/badge/github.com/malcolmston/streamlit/st.svg)](https://pkg.go.dev/github.com/malcolmston/streamlit/st)

Package st is a from-scratch, standard-library-only Go port of Python's
Streamlit: a framework for building interactive web and data apps by writing a
plain script.

You write an app function with the signature func(s *st.Session). Inside it
you call methods on the session to build the page: s.Title, s.Write, s.Slider,
s.Button and so on. Each call appends one element to the page's element tree,
and each widget call returns the widget's current value. The framework runs
your app function top to bottom to produce the page.

## Install

```sh
go get github.com/malcolmston/streamlit@v0.4.0
```

```go
import "github.com/malcolmston/streamlit/st"
```

## Exported surface

### Functions

| Function | What it does |
| --- | --- |
| `func CacheClear()` | CacheClear removes all entries from the process-wide cache used by `Session.Cache`. |
| `func CacheDelete(key string) bool` | CacheDelete removes a single entry from the data cache used by `Session.Cache`, forcing the next call for that key to recompute. |
| `func CacheLen() int` | CacheLen reports how many entries the data cache currently holds. |
| `func CacheResourceClear()` | CacheResourceClear discards every singleton registered with `Session.CacheResource`, so the next call for each key builds a fresh one. |
| `func CacheSetMaxEntries(n int)` | CacheSetMaxEntries sets how many entries `Session.Cache` retains. |
| `func Handler(app func(*Session)) http.Handler` | Handler returns the http.Handler that serves the app with the default `Options`. |
| `func HandlerWithOptions(app func(*Session), opts Options) http.Handler` | HandlerWithOptions is `Handler` with explicit resource limits and origin policy. |
| `func Run(app func(*Session), addr string) error` | Run starts an HTTP server that serves the app on addr (for example ":8501"). |
| `func RunWithOptions(app func(*Session), addr string, opts Options) error` | RunWithOptions is `Run` with explicit resource limits and origin policy. |

### Types

| Type | What it is |
| --- | --- |
| `Container` | Container is a region of the page into which elements are appended. |
| `Element` | Element is a single node in the page's element tree. |
| `MapPoint` | MapPoint is a single latitude/longitude coordinate plotted by `Container.Map`. |
| `Options` | Options configures the HTTP surface returned by `HandlerWithOptions`. |
| `Session` | Session represents a single browser connection to the app. |
| `State` | State is a per-session key/value store that persists across reruns of the app function. |
| `UploadedFile` | UploadedFile is a single file received from a `Container.FileUploader`. |

<details>
<summary><code>Container</code> — constructors and methods</summary>

| Signature | What it does |
| --- | --- |
| `func (c *Container) AreaChart(series ...[]float64)` | AreaChart adds a filled area chart. |
| `func (c *Container) Audio(src any)` | Audio adds an audio player. |
| `func (c *Container) AudioInput(label string, key ...string) []UploadedFile` | AudioInput adds a control that records audio from the user's microphone and returns the recorded files for the current session, mirroring Streamlit's… |
| `func (c *Container) Badge(label string, color ...string)` | Badge adds a small coloured pill label, mirroring Streamlit's st.badge. |
| `func (c *Container) Balloons()` | Balloons triggers the celebratory balloons animation, mirroring Streamlit's st.balloons. |
| `func (c *Container) BarChart(series ...[]float64)` | BarChart adds a bar chart. |
| `func (c *Container) BorderedContainer() *Container` | BorderedContainer adds a nested grouping region drawn with a visible border and returns a container for it, mirroring Streamlit's… |
| `func (c *Container) Button(label string, key ...string) bool` | Button adds a clickable button and returns true on the single run triggered by a click. |
| `func (c *Container) CameraInput(label string, key ...string) []UploadedFile` | CameraInput adds a control that captures a photo from the user's webcam and returns the captured image files for the current session, mirroring… |
| `func (c *Container) Caption(text string)` | Caption adds small, muted caption text (rendered as markdown). |
| `func (c *Container) ChatInput(placeholder string, key ...string) string` | ChatInput adds a chat entry box pinned to the bottom of its region and returns the message the user last submitted, or the empty string before any… |
| `func (c *Container) ChatMessage(role string) *Container` | ChatMessage adds a chat message bubble attributed to role (commonly "user" or "assistant") and returns a container for the message body. |
| `func (c *Container) Checkbox(label string, def bool, key ...string) bool` | Checkbox adds a checkbox initialised to def and returns its current state. |
| `func (c *Container) Code(code, lang string)` | Code adds a syntax-highlight-styled code block. |
| `func (c *Container) ColorPicker(label, def string, key ...string) string` | ColorPicker adds a colour selector and returns the chosen colour as a "#rrggbb" hex string. |
| `func (c *Container) Columns(n int) []*Container` | Columns splits the current region into n side-by-side columns and returns a container for each. |
| `func (c *Container) ColumnsWeighted(weights []float64) []*Container` | ColumnsWeighted splits the current region into columns whose widths are proportional to weights, mirroring Streamlit's st.columns([2, 1]) — which… |
| `func (c *Container) Container() *Container` | Container adds a nested, undecorated grouping region and returns a container for it. |
| `func (c *Container) DataFrame(data any, caption ...string)` | DataFrame renders tabular data like `Container.Table` but adds a client-side sorting hint, so column headers can be clicked to sort the rows in the… |
| `func (c *Container) DateInput(label, def string, key ...string) string` | DateInput adds a calendar date field and returns the selected date as an ISO-8601 string ("2006-01-02"). |
| `func (c *Container) DateRangeInput(label, defStart, defEnd string, key ...string) (string, string)` | DateRangeInput adds a calendar control that selects a start and end date and returns them as ISO-8601 strings ("2006-01-02"), mirroring Streamlit's… |
| `func (c *Container) Divider()` | Divider adds a horizontal rule. |
| `func (c *Container) DownloadButton(label, filename string, data []byte, key ...string) bool` | DownloadButton adds a button that downloads data as a file named filename when clicked, and returns true on the single run triggered by the click. |
| `func (c *Container) Echo(code string)` | Echo adds a read-only display of Go source code, mirroring Streamlit's st.echo (which shows the code inside its block). |
| `func (c *Container) Empty() *Container` | Empty adds a single placeholder region and returns a container for it. |
| `func (c *Container) Error(text string)` | Error adds a red error message box. |
| `func (c *Container) Exception(err error)` | Exception adds a formatted error box for err, mirroring Streamlit's st.exception. |
| `func (c *Container) Expander(label string, expanded bool) *Container` | Expander adds a collapsible section with the given label and returns a container for its body. |
| `func (c *Container) Feedback(kind string, key ...string) int` | Feedback adds a rating control and returns the selected score, or -1 when nothing has been chosen yet. |
| `func (c *Container) FileUploader(label string, key ...string) []UploadedFile` | FileUploader adds a file selection control and returns the files uploaded for it in the current session. |
| `func (c *Container) Form(key string) *Container` | Form adds a form region identified by key and returns a container for its body. |
| `func (c *Container) FormSubmitButton(label string) bool` | FormSubmitButton adds the submit button for the enclosing form and returns true on the single run triggered by its click. |
| `func (c *Container) Header(text string)` | Header adds a section header. |
| `func (c *Container) Help(value any)` | Help adds an introspective description of value, mirroring Streamlit's st.help. |
| `func (c *Container) Histogram(data []float64, bins int)` | Histogram adds a histogram of data grouped into the given number of bins (clamped to at least one), rendered as a bar chart. |
| `func (c *Container) Html(html string)` | Html adds a block of raw, unescaped HTML, mirroring Streamlit's st.html. |
| `func (c *Container) Image(src any, caption ...string)` | Image adds an image. |
| `func (c *Container) Info(text string)` | Info adds a blue informational message box. |
| `func (c *Container) JSON(value any, collapsed ...bool)` | JSON adds a pretty-printed JSON view of value. |
| `func (c *Container) Latex(expr string)` | Latex adds a block of mathematics written in LaTeX. expr is the raw LaTeX source without surrounding delimiters, for example `\int_a^b f(x)\,dx`. |
| `func (c *Container) LineChart(series ...[]float64)` | LineChart adds a line chart. |
| `func (c *Container) LinkButton(label, url string)` | LinkButton adds a button that navigates to url when clicked, mirroring Streamlit's st.link_button. |
| `func (c *Container) Logo(src any)` | Logo adds a small brand image, typically shown at the top of the app or sidebar. |
| `func (c *Container) Map(points []MapPoint)` | Map adds a simple point map. |
| `func (c *Container) Markdown(text string)` | Markdown adds text rendered with a small, safe subset of Markdown (ATX headings, bold, italics, inline code, links and unordered lists) by the… |
| `func (c *Container) MarkdownUnsafe(text string)` | MarkdownUnsafe is `Container.Markdown` with HTML passed through instead of escaped, mirroring st.markdown(body, unsafe_allow_html=True). |
| `func (c *Container) Metric(label, value, delta string)` | Metric adds a big-number metric with an optional delta indicator. |
| `func (c *Container) MetricColored(label, value, delta, deltaColor string)` | MetricColored is like `Container.Metric` but controls how the delta is coloured. |
| `func (c *Container) MultiSelect(label string, options []string, key ...string) []string` | MultiSelect adds a multiple-selection control and returns the currently selected options. |
| `func (c *Container) NumberInput(label string, def float64, key ...string) float64` | NumberInput adds a numeric entry field initialised to def and returns its current value. |
| `func (c *Container) NumberInputRange(label string, min, max, def float64, key ...string) float64` | NumberInputRange adds a numeric entry field bounded to the inclusive range [min, max] and returns its current value clamped to that range, mirroring… |
| `func (c *Container) PageLink(url, label string)` | PageLink adds a navigational link to url shown with the given label, mirroring Streamlit's st.page_link. |
| `func (c *Container) PasswordInput(label, def string, key ...string) string` | PasswordInput adds a single-line text field whose contents are masked in the browser and returns its current value, mirroring Streamlit's… |
| `func (c *Container) PieChart(values []float64, labels []string)` | PieChart adds a pie chart. |
| `func (c *Container) Pills(label string, options []string, key ...string) []string` | Pills adds a row of selectable "pill" chips allowing multiple selections and returns the currently selected options, mirroring Streamlit's st.pills. |
| `func (c *Container) Popover(label string) *Container` | Popover adds a button that reveals a small floating panel when clicked and returns a container for the panel's body. |
| `func (c *Container) PrimaryButton(label string, key ...string) bool` | PrimaryButton adds an emphasised call-to-action button and returns true on the single run triggered by a click, mirroring Streamlit's st.button(type=… |
| `func (c *Container) Progress(value float64)` | Progress adds a progress bar. |
| `func (c *Container) Radio(label string, options []string, key ...string) string` | Radio adds a group of mutually exclusive radio options and returns the selected option. |
| `func (c *Container) ScatterChart(xs, ys []float64)` | ScatterChart adds a scatter plot of the paired xs and ys values, rendered to inline SVG on the server. |
| `func (c *Container) SegmentedControl(label string, options []string, key ...string) string` | SegmentedControl adds a single-choice segmented button group and returns the selected option, mirroring Streamlit's st.segmented_control. |
| `func (c *Container) SelectBox(label string, options []string, key ...string) string` | SelectBox adds a drop-down of options and returns the selected option. |
| `func (c *Container) SelectSlider(label string, options []string, key ...string) string` | SelectSlider adds a slider that moves across a set of discrete options and returns the currently selected option. |
| `func (c *Container) SelectSliderRange(label string, options []string, key ...string) (string, string)` | SelectSliderRange adds a slider that selects a contiguous range across a set of discrete options and returns the currently selected low and high… |
| `func (c *Container) Slider(label string, min, max, def, step float64, key ...string) float64` | Slider adds a numeric slider bounded by min and max, initialised to def, and returns its current value. |
| `func (c *Container) SliderRange(label string, min, max, low, high, step float64, key ...string) (float64, float64)` | SliderRange adds a two-handle range slider bounded by min and max and returns the currently selected low and high values, mirroring Streamlit's… |
| `func (c *Container) Snow()` | Snow triggers the falling-snow animation, mirroring Streamlit's st.snow. |
| `func (c *Container) Spinner(label string)` | Spinner adds a spinner with a label. |
| `func (c *Container) Status(label string, state ...string) *Container` | Status adds a collapsible status box with a label and a visual state and returns a container for its body. |
| `func (c *Container) Subheader(text string)` | Subheader adds a smaller section header. |
| `func (c *Container) Success(text string)` | Success adds a green success message box. |
| `func (c *Container) Table(data any, caption ...string)` | Table renders tabular data. |
| `func (c *Container) Tabs(labels []string) []*Container` | Tabs adds a tabbed region with one tab per label and returns a container for each tab's body, in the same order as labels. |
| `func (c *Container) Text(text string)` | Text adds fixed-width, unformatted text. |
| `func (c *Container) TextArea(label, def string, key ...string) string` | TextArea adds a multi-line text field initialised to def and returns its current value. |
| `func (c *Container) TextInput(label, def string, key ...string) string` | TextInput adds a single-line text field initialised to def and returns its current value. |
| `func (c *Container) TextInputMax(label, def string, maxChars int, key ...string) string` | TextInputMax adds a single-line text field that accepts at most maxChars characters and returns its current value, truncated to maxChars runes,… |
| `func (c *Container) TimeInput(label, def string, key ...string) string` | TimeInput adds a time-of-day field and returns the selected time as a 24-hour "15:04" string. |
| `func (c *Container) Title(text string)` | Title adds a top-level page title. |
| `func (c *Container) Toast(message string, icon ...string)` | Toast adds a transient notification message, mirroring Streamlit's st.toast. |
| `func (c *Container) Toggle(label string, def bool, key ...string) bool` | Toggle adds an on/off switch initialised to def and returns its current state. |
| `func (c *Container) Video(src any)` | Video adds a video player. |
| `func (c *Container) Warning(text string)` | Warning adds a yellow warning message box. |
| `func (c *Container) Write(args ...any)` | Write is the Swiss-army display method, mirroring Streamlit's st.write. |

</details>

<details>
<summary><code>Session</code> — constructors and methods</summary>

| Signature | What it does |
| --- | --- |
| `func (s *Session) Cache(key string, compute func() any, ttl ...time.Duration) any` | Cache returns the value memoised under key, computing it with compute on the first call (or after the entry has expired) and returning the stored… |
| `func (s *Session) CacheResource(key string, create func() any) any` | CacheResource returns the singleton registered under key, creating it with create on the first call, mirroring Streamlit's @st.cache_resource. |
| `func (s *Session) ID() string` | ID returns the session's opaque identifier. |
| `func (s *Session) Rerun()` | Rerun abandons the current run and immediately re-executes the app function from the top, mirroring Streamlit's st.rerun. |
| `func (s *Session) SetPageConfig(title, icon string)` | SetPageConfig sets page-level metadata for the app, mirroring Streamlit's st.set_page_config. |
| `func (s *Session) Sidebar() *Container` | Sidebar returns the container for the app's sidebar region. |
| `func (s *Session) Stop()` | Stop immediately halts execution of the current run of the app function, mirroring Streamlit's st.stop. |

</details>

<details>
<summary><code>State</code> — constructors and methods</summary>

| Signature | What it does |
| --- | --- |
| `func (s *State) Clear()` | Clear removes every key from the store. |
| `func (s *State) Delete(key string)` | Delete removes key from the store. |
| `func (s *State) Get(key string) (any, bool)` | Get returns the value stored under key and whether it was present. |
| `func (s *State) GetBool(key string, def bool) bool` | GetBool returns the bool stored under key, or def if absent or not a bool. |
| `func (s *State) GetFloat(key string, def float64) float64` | GetFloat returns the float64 stored under key, or def if absent or not numeric. |
| `func (s *State) GetInt(key string, def int) int` | GetInt returns the int stored under key, or def if absent or not numeric. |
| `func (s *State) GetString(key, def string) string` | GetString returns the string stored under key, or def if absent or not a string. |
| `func (s *State) Has(key string) bool` | Has reports whether key is present, mirroring `key in st.session_state`. |
| `func (s *State) Keys() []string` | Keys returns the store's keys in ascending lexical order. |
| `func (s *State) Len() int` | Len returns the number of keys held in the store. |
| `func (s *State) Set(key string, value any)` | Set stores value under key. |
| `func (s *State) SetDefault(key string, value any) bool` | SetDefault stores value under key only if the key is absent and reports whether it was stored. |

</details>

### Constants

`DefaultMaxSessions`, `DefaultSessionIdleTimeout`, `DefaultMaxUploadBytes`, `DefaultMaxRequestBytes`, `DefaultMaxWidgetEntries`, `DefaultMaxWidgetStateBytes`, `DefaultMaxCacheEntries`

Full signatures, doc comments and every runnable example are on
[pkg.go.dev](https://pkg.go.dev/github.com/malcolmston/streamlit/st).

## Deviations from upstream

Deliberate differences for this package, where any exist, are recorded in the
module-wide [`API-DEVIATIONS.md`](../API-DEVIATIONS.md).

## License

MIT, as part of [`github.com/malcolmston/streamlit`](..).
An independent re-implementation, not affiliated with or endorsed by the
original project.

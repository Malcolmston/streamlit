# The execution model

Streamlit's defining idea is that your app is a *script*, not a callback graph.
This port reproduces that faithfully, and almost every surprise people hit with
Streamlit comes from the same place: understanding exactly what is re-executed,
what persists, and for how long. This page is that model, written down.

## One request, one run

```
browser event ──▶ POST /api/run ──▶ apply event to session ──▶ run app(s) ──▶ element tree ──▶ JSON
```

`st.Handler` keeps a `*Session` per browser. On every request it:

1. looks the session up by the opaque id the browser echoes back (creating one
   if the id is unknown or absent);
2. applies the incoming widget event to that session's widget map;
3. runs your app function from the top, building a brand-new element tree;
4. serialises the tree and returns it.

Runs of a *given* session are serialised by a mutex, so your app function never
executes concurrently with itself for one user. Different sessions do run
concurrently — anything you share between them (a `CacheResource` singleton, a
package-level variable) must be safe for concurrent use.

## Three lifetimes

| Thing | Lives for | Cleared when |
| --- | --- | --- |
| Element tree | one run | every run — it is rebuilt from scratch |
| Widget value | as long as the app keeps rendering that widget | the widget stops being rendered, or the session ends |
| `s.State` | the session | you delete the key, or the session is evicted |
| `Cache` entry | the process | TTL expiry, eviction, `CacheDelete`, `CacheClear` |
| `CacheResource` value | the process | `CacheResourceClear` only |

The second row is the one that bites. Consider:

```go
if s.Checkbox("Advanced", false, "adv") {
    note := s.TextInput("Note", "", "note")   // only rendered sometimes
    _ = note
}
```

Type something into the note, untick the checkbox, tick it again: the note is
empty. That is deliberate and matches Streamlit — state for a widget that a run
did not draw is discarded at the end of that run. It also means a browser cannot
make the server accumulate widget state for keys the app never renders.

If a value must outlive its widget, copy it into `s.State`:

```go
if s.Checkbox("Advanced", false, "adv") {
    s.State.Set("note", s.TextInput("Note", s.State.GetString("note", ""), "note"))
}
```

## Widget identity

Every widget needs a stable key across runs, because the key is how its value is
found again. There are two ways to get one:

* **Explicit** — pass a key as the trailing argument: `s.Slider("n", 0, 10, 5, 1, "n")`.
  Always do this for a widget inside a conditional, a loop, or a function that
  may be called a varying number of times.
* **Automatic** — omit it and the key becomes `auto-<type>-<ordinal>`, where the
  ordinal counts *all* widgets rendered so far in the run.

Automatic keys are stable only while the app's shape is stable. This shifts every
subsequent widget's key, resetting their values:

```go
if s.Checkbox("extra", false) {   // auto-checkbox-0
    s.TextInput("sometimes", "")  // auto-text_input-1 … only sometimes
}
s.Slider("always", 0, 1, 0, 0.1)  // auto-slider-1 or auto-slider-2!
```

Rule of thumb: any widget that is not unconditionally rendered, exactly once,
in call order, should carry an explicit key.

## Control flow: `Stop` and `Rerun`

`s.Stop()` halts the current run. Everything already added stays on the page;
nothing after the call executes. Use it as a guard.

`s.Rerun()` throws away the current run *including the elements it already
added* and executes the app again from the top. Use it when you have just
changed state that earlier code already rendered:

```go
if f.FormSubmitButton("Log in") && name != "" {
    s.State.Set("user", name)
    s.Rerun()          // repaint the page as the logged-in user
}
```

Both are implemented by panicking with an internal sentinel that the run loop
recovers, so both must be called from the app function itself, never from a
goroutine you started.

Transient triggers (`Button`, `PrimaryButton`, `DownloadButton`,
`FormSubmitButton`) read `true` for exactly one execution. The run that `Rerun`
starts sees them as unclicked, which is what makes `if s.Button(x) { …; s.Rerun() }`
terminate instead of looping. A chain of reruns inside one request is capped at
32; an app that calls `Rerun` unconditionally therefore returns the last
permitted run's tree rather than hanging the request.

## Caching

`s.Cache(key, compute, ttl…)` is `@st.cache_data`: process-wide, shared by every
session, optionally expiring. Concurrent callers asking for the same missing key
do not each run `compute` — the first computes and the rest wait. The table is
bounded (see `CacheSetMaxEntries`); when full, the entry inserted longest ago is
evicted.

`s.CacheResource(key, create)` is `@st.cache_resource`: a singleton that is never
expired or evicted, for connections, clients and models. Because it is shared by
every session, whatever `create` returns must be safe for concurrent use.

Python derives the cache key by hashing the decorated function's arguments; this
port takes an explicit key instead, so build the key out of everything the
computation depends on:

```go
rows := s.Cache("report:"+region+":"+day, func() any { return load(region, day) }, time.Hour)
```

## Sessions

A session is created on first contact and identified by a 128-bit random id the
browser stores in memory. Sessions are evicted when they go idle
(`Options.SessionIdleTimeout`, 30 minutes by default) and, if the table is full,
least-recently-used first (`Options.MaxSessions`, 1000 by default). Nothing in a
session is persisted, so treat eviction as a logout.

See [security.md](security.md) for the request-level policy that protects a
session from other origins.

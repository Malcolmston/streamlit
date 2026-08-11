# Security notes

This page describes what the package defends against, what it does not, and the
knobs you have. It is short on purpose — the surface is small.

## Threat model

An app built with this package serves HTML and JavaScript to a browser and
accepts widget events back. The two parties worth worrying about are:

* **the page's viewer**, who controls every byte of every event posted to
  `/api/run` and `/api/upload`; and
* **any other site the viewer happens to have open**, which can make their
  browser issue requests to your app.

The *app author* is trusted: strings you pass to `Html`, `MarkdownUnsafe`,
`LinkButton` and friends are your own.

## Rendering: escaped by default

`Markdown`, `Caption`, `Write(string)` and the alert helpers all render through
the frontend's Markdown subset, which **escapes the entire source string before
applying any formatting**. Text arriving from a user cannot introduce a tag.

Link targets get a second check. `[label](url)` emits an `href` only when the
URL's scheme is on an allowlist — `http`, `https`, `mailto`, `tel`, `ftp`,
`ftps`, `sms` — and otherwise emits `#`. Media sources (`Image`, `Logo`,
`Audio`, `Video`, `DownloadButton`) use the same check with `data:` and `blob:`
added, which is how server-encoded payloads are delivered.

Two details matter and are covered by tests:

* The check is an **allowlist**, not a list of dangerous schemes. A denylist
  fails for whatever it has not heard of.
* URLs are stripped of spaces and control characters *before* the scheme is
  read, because browsers ignore tabs and newlines while parsing a URL. Without
  that step `java&#9;script:alert(1)` would look scheme-less, be passed through
  as a relative URL, and then execute.

Quotes are escaped along with `&`, `<` and `>`. Because the renderer builds an
`href` attribute by string concatenation, leaving `"` unescaped would let a link
target close the attribute and add an event handler.

### The opt-ins

`Container.Html` and `Container.MarkdownUnsafe` deliberately bypass all of that,
mirroring `st.html` and `st.markdown(..., unsafe_allow_html=True)`. They are off
unless you call them, and anything you pass becomes live markup in every
viewer's page. Never hand them a string that contains user input.

## Requests: origin policy

The session id is the only thing needed to drive a session, so both POST
endpoints check the `Origin` header:

| Origin header | Result |
| --- | --- |
| absent (curl, tests, server-to-server) | allowed |
| matches the request's `Host` | allowed |
| listed in `Options.AllowedOrigins` | allowed |
| anything else, including `null` | `403 Forbidden` |

This is what stops another site from creating sessions or replaying widget
events in a viewer's browser — the HTTP analogue of cross-site WebSocket
hijacking. `Options.AllowAllOrigins` turns the check off; only set it for an app
that is deliberately public and stateless.

Host matching is exact, so `https://notapp.example.com` does not satisfy a
request to `app.example.com`, and a port mismatch is a mismatch.

## Resource limits

Everything a remote caller can grow is bounded. All of these are
`Options` fields with the documented defaults:

| Bound | Default | Why |
| --- | --- | --- |
| `MaxSessions` | 1000 | every request with an unknown id creates a session |
| `SessionIdleTimeout` | 30m | otherwise sessions are only ever added |
| `MaxUploadBytes` | 32 MiB | `ParseMultipartForm`'s argument bounds memory only; the rest spills to disk |
| `MaxRequestBytes` | 1 MiB | `/api/run` bodies |
| `MaxWidgetEntries` | 1024 | widget keys are chosen by the client |
| `MaxWidgetStateBytes` | 1 MiB | so are widget values, and they persist across requests |
| `CacheSetMaxEntries` | 1024 | a cache keyed on user input grows until the process dies |

A body over `MaxRequestBytes` or `MaxUploadBytes`, and an event that would push a
session past `MaxWidgetEntries` or `MaxWidgetStateBytes`, are answered with **413
Request Entity Too Large** — the limit is checked before the value is stored, so
the refusal costs nothing.

Two more bounds are not configurable because there is no reason to raise them:
widget keys are limited to 256 bytes, and a single form submission may commit at
most 512 of them. Beyond those limits the values are ignored rather than stored.

Finally, widget state for anything the app did not render is discarded at the end
of every run (see [execution-model.md](execution-model.md)). That is primarily a
correctness rule, but it also means a client cannot use `/api/run` to accumulate
state for keys your app never draws.

## What is not covered

* **No authentication or authorisation.** The package has no notion of a user.
  Put it behind whatever your deployment already uses, and treat session
  eviction as a logout.
* **No transport security.** `Run` serves plain HTTP; terminate TLS in front of
  it, or build your own `http.Server` around `Handler`.
* **No CSP.** The embedded page ships its own inline-free script and stylesheet,
  but the package does not set a `Content-Security-Policy` header. If you mount
  `Handler` inside a larger server, adding one is cheap and worthwhile.
* **Uploaded bytes are held in memory** for the life of the session (until the
  uploader widget stops being rendered). Size `MaxUploadBytes` accordingly.

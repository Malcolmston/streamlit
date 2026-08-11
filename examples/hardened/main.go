// Command hardened serves a small Streamlit-Go application configured the way
// a multi-user deployment should be, and demonstrates the control-flow and
// caching primitives that only make sense once more than one browser is
// involved.
//
// Run it with:
//
//	go run ./examples/hardened
//
// then open http://localhost:8502. Open a second browser (or a private window)
// to see that the two sessions are completely independent while the shared
// resource behind them is built exactly once.
//
// What it demonstrates:
//
//   - st.RunWithOptions with explicit resource limits and an origin policy, so
//     the app cannot be driven from another site and cannot be made to retain
//     unbounded state.
//   - st.Rerun as the repaint-the-whole-page primitive after a login.
//   - st.CacheResource (@st.cache_resource) for a process-wide singleton, next
//     to st.Cache (@st.cache_data) for per-key data with a TTL.
//   - st.Stop as the guard that keeps the rest of the script from running.
package main

import (
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/malcolmston/streamlit/st"
)

func main() {
	opts := st.Options{
		// Only pages served from this app may drive it. Requests carrying any
		// other Origin are rejected, which is what stops a page the user
		// happens to be visiting from replaying widget events into their
		// session.
		AllowedOrigins: nil, // same-origin only

		// Bound everything a remote caller controls.
		MaxSessions:        200,
		SessionIdleTimeout: 15 * time.Minute,
		MaxUploadBytes:     4 << 20, // 4 MiB
		MaxRequestBytes:    256 << 10,
	}
	log.Println("serving hardened demo on http://localhost:8502")
	if err := st.RunWithOptions(app, ":8502", opts); err != nil {
		log.Fatal(err)
	}
}

func app(s *st.Session) {
	s.SetPageConfig("Hardened demo", "🔒")
	s.Title("Per-user state, shared resources")

	// --- Login gate ------------------------------------------------------
	//
	// The classic st.rerun case: once the user logs in, everything already
	// drawn is stale, so abandon this pass and repaint from the top.
	if !s.State.GetBool("authed", false) {
		s.Info("Any non-empty name works; nothing is sent anywhere.")
		f := s.Form("login")
		name := strings.TrimSpace(f.TextInput("Your name", ""))
		if f.FormSubmitButton("Log in") && name != "" {
			s.State.Set("user", name)
			s.State.Set("authed", true)
			s.Rerun()
		}
		// Nothing below this point should run for a logged-out visitor.
		s.Stop()
	}

	user := s.State.GetString("user", "")
	s.Success("Signed in as " + user)

	// --- Shared resource vs per-session state ----------------------------
	//
	// CacheResource builds the value once for the whole process; every session
	// shares this exact instance, so it must be safe for concurrent use. That
	// is why the counter below is an atomic.
	hits := s.CacheResource("hit-counter", func() any {
		log.Println("building the shared hit counter (this prints once)")
		return new(atomic.Int64)
	}).(*atomic.Int64)
	hits.Add(1)

	// Cache holds data rather than a resource, and may expire.
	quote := s.Cache("quote-of-the-minute", func() any {
		return "Computed at " + time.Now().Format(time.TimeOnly)
	}, time.Minute).(string)

	cols := s.ColumnsWeighted([]float64{2, 1, 1})
	cols[0].Metric("Cached data (1m TTL)", quote, "")
	cols[1].Metric("Runs, all sessions", fmt.Sprint(hits.Load()), "")
	cols[2].Metric("Runs, this session", fmt.Sprint(bump(s)), "")

	s.Caption("Open a second browser: the left two tiles are shared, the right one is not.")

	// --- Per-session widgets ---------------------------------------------
	s.Divider()
	s.Subheader("Your settings")

	theme := s.SegmentedControl("Theme", []string{"Light", "Dark", "System"}, "theme")
	tags := s.Pills("Interests", []string{"charts", "tables", "maps", "chat"}, "tags")
	lo, hi := s.SliderRange("Range of interest", 0, 100, 20, 80, 1, "range")

	s.Write(fmt.Sprintf("Theme **%s**, %d interest(s), range %.0f–%.0f.",
		theme, len(tags), lo, hi))

	// A widget that only exists in one branch: hiding it discards its value,
	// so revealing it again starts from the default. That is Streamlit's
	// behaviour, and it is why keys should be stable and explicit.
	if s.Checkbox("Show the advanced field", false, "advanced") {
		note := s.TextInput("Note (cleared whenever it is hidden)", "", "note")
		s.Caption("Note: " + note)
	}

	s.Divider()
	if s.Button("Log out") {
		s.State.Clear()
		s.Rerun()
	}
}

// bump increments and returns this session's own run counter.
func bump(s *st.Session) int {
	n := s.State.GetInt("runs", 0) + 1
	s.State.Set("runs", n)
	return n
}

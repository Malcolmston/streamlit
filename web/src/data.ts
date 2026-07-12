// Library content for the streamlit documentation site. Mirrors the shape used
// by the malcolmston/go landing site's data.ts so the sibling sites stay in
// sync.
export interface Lib {
  id: string; name: string; icon: string; accent: string; pkg: string; node: string;
  repo: string; docs: string; tagline: string; blurb: string; tags: string[];
  features: string[]; node_code: string; go_code: string; integrate: string;
}

export const NODE_ACCENT = '#8cc84b';

export const STREAMLIT: Lib = {
  id:"streamlit", name:"streamlit", icon:'<i class="fa-solid fa-chart-line"></i>', accent:"#ff4b4b",
  pkg:"github.com/malcolmston/streamlit", node:"streamlit/streamlit",
  repo:"https://github.com/malcolmston/streamlit", docs:"https://malcolmston.github.io/streamlit/",
  tagline:"Interactive data apps in Go.",
  blurb:"A from-scratch, standard-library-only Go port of Python's Streamlit: build interactive data and web "+
    "apps by writing a plain Go function. Call methods on a session — Title, Slider, Metric, LineChart — to build "+
    "the page; each widget call returns its current value, and every interaction reruns your function from the top. "+
    "Streamlit's signature rerun-on-interaction model, persistent session state, layout containers and "+
    "server-rendered SVG charts, reproduced faithfully with no third-party modules and no JavaScript framework.",
  tags:["rerun-on-interaction","session state","widgets","layout","SVG charts","go:embed","zero-deps"],
  features:[
    "<code>Rerun-on-interaction</code> — the whole app function re-executes on every widget change, so app code stays a straight-line script",
    "Persistent <code>session state</code> and restored widget values survive each rerun, exactly like Python's <code>st.session_state</code>",
    "A broad API: display (<code>Title</code>, <code>Write</code>, <code>Markdown</code>, <code>Metric</code>), widgets (<code>Slider</code>, <code>Button</code>, <code>SelectBox</code>) and layout (<code>Columns</code>, <code>Expander</code>, <code>Sidebar</code>)",
    "Charts (<code>LineChart</code>, <code>AreaChart</code>, <code>BarChart</code>) rendered to inline <code>SVG</code> on the server — no JavaScript charting library",
    "A small, dependency-free single-page frontend embedded via <code>go:embed</code>; the wire protocol is plain JSON over <code>POST /api/run</code>",
    "Zero dependencies — pure Go standard library, nothing to audit but the toolchain"
  ],
  node_code:
`import streamlit as st

st.title("Demo")
n = st.slider("n", 0, 100, 50)
st.metric("value", n)

if st.button("Celebrate"):
    st.success("🎉")`,
  go_code:
`import (
    "fmt"

    "github.com/malcolmston/streamlit/st"
)

func main() {
    st.Run(app, ":8501")
}

func app(s *st.Session) {
    s.Title("Demo")
    n := s.Slider("n", 0, 100, 50, 1)
    s.Metric("value", fmt.Sprint(int(n)), "")

    if s.Button("Celebrate") {
        s.Success("🎉")
    }
}`,
  integrate:
`<span class="tok-c">// Layout columns, session state, an SVG chart, then a collapsible section</span>
cols := s.Columns(2)
cols[0].Metric("Users", "1,204", "+8%")
cols[1].Metric("Latency", "38ms", "-12%")

<span class="tok-c">// session state persists across the rerun-on-interaction loop</span>
count := s.State.GetInt("count", 0)
if s.Button("Increment") {
    count++
    s.State.Set("count", count)
}
s.Write("clicked", count, "times")

<span class="tok-c">// charts render to inline SVG on the server — no JS charting library</span>
s.LineChart([]float64{3, 1, 4, 1, 5, 9, 2, 6})

<span class="tok-c">// group elements into an expandable section</span>
exp := s.Expander("Details", false)
exp.Markdown("**All standard library.** Nothing to audit but the toolchain.")`
};

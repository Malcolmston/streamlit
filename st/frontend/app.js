"use strict";

// Minimal, dependency-free Streamlit-Go client.
//
// Protocol: POST /api/run with {sessionId, event?}. The server returns
// {sessionId, tree}. An event is {key, value, button}. On any widget change we
// post the event and re-render the returned tree, reproducing Streamlit's
// rerun-on-interaction model.

let sessionId = null;
const status = document.getElementById("status");

async function run(event) {
  status.textContent = "running…";
  try {
    const res = await fetch("/api/run", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ sessionId, event }),
    });
    const data = await res.json();
    sessionId = data.sessionId;
    renderTree(data.tree);
    status.textContent = "ready";
  } catch (err) {
    status.textContent = "error: " + err;
  }
}

function renderTree(root) {
  const sidebar = document.getElementById("sidebar");
  const main = document.getElementById("main");
  sidebar.innerHTML = "";
  main.innerHTML = "";
  if (!root || !root.children) return;
  for (const child of root.children) {
    if (child.type === "sidebar") renderInto(sidebar, child.children || []);
    else if (child.type === "main") renderInto(main, child.children || []);
  }
}

function renderInto(parent, children) {
  for (const el of children) parent.appendChild(renderElement(el));
}

function el(tag, cls, text) {
  const n = document.createElement(tag);
  if (cls) n.className = cls;
  if (text != null) n.textContent = text;
  return n;
}

// A tiny, safe Markdown subset: escaping first, then inline formatting.
function escapeHTML(s) {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}
function markdown(s) {
  let h = escapeHTML(s);
  h = h.replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>");
  h = h.replace(/\*([^*]+)\*/g, "<em>$1</em>");
  h = h.replace(/`([^`]+)`/g, '<code class="inline">$1</code>');
  h = h.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" rel="noopener">$1</a>');
  h = h.replace(/\n/g, "<br>");
  return h;
}

function fire(key, value, button) {
  run({ key, value, button: !!button });
}

function renderElement(e) {
  const p = e.props || {};
  switch (e.type) {
    case "title": return el("h1", "st", p.text);
    case "header": return el("h2", "st", p.text);
    case "subheader": return el("h3", "st", p.text);
    case "text": return el("div", "st-text", p.text);
    case "caption": { const n = el("div", "st-caption"); n.innerHTML = markdown(p.text || ""); return n; }
    case "markdown": { const n = el("div", "st-markdown"); n.innerHTML = "<p>" + markdown(p.text || "") + "</p>"; return n; }
    case "divider": return el("hr", "st");
    case "code": { const pre = el("pre", "st-code"); pre.appendChild(el("code", null, p.code)); return pre; }
    case "json": return el("pre", "st-json", p.json);
    case "alert": { const n = el("div", "st-alert " + p.kind); n.innerHTML = markdown(p.text || ""); return n; }
    case "metric": return renderMetric(p);
    case "spinner": return el("div", "st-spinner", "⏳ " + (p.label || ""));
    case "progress": return renderProgress(p);
    case "table": return renderTable(p);
    case "chart": { const n = el("div", "st-chart"); n.innerHTML = p.svg || ""; return n; }
    case "columns": return renderColumns(e);
    case "column": { const n = el("div", "st-column"); renderInto(n, e.children || []); return n; }
    case "container": { const n = el("div", "st-container"); renderInto(n, e.children || []); return n; }
    case "expander": return renderExpander(e);
    // widgets
    case "button": return renderButton(e);
    case "checkbox": return renderCheckbox(e);
    case "slider": return renderSlider(e);
    case "number": return renderNumber(e);
    case "text_input": return renderTextInput(e);
    case "text_area": return renderTextArea(e);
    case "selectbox": return renderSelect(e, false);
    case "radio": return renderRadio(e);
    case "multiselect": return renderSelect(e, true);
    default: return el("div", null, "[" + e.type + "]");
  }
}

function renderMetric(p) {
  const n = el("div", "st-metric");
  n.appendChild(el("div", "label", p.label));
  n.appendChild(el("div", "value", p.value));
  if (p.delta) {
    const down = String(p.delta).trim().startsWith("-");
    n.appendChild(el("div", "delta " + (down ? "down" : "up"), (down ? "▼ " : "▲ ") + p.delta));
  }
  return n;
}

function renderProgress(p) {
  const n = el("div", "st-progress");
  const bar = el("div");
  bar.style.width = Math.round((p.value || 0) * 100) + "%";
  n.appendChild(bar);
  return n;
}

function renderTable(p) {
  const t = el("table", "st");
  if (p.header && p.header.length) {
    const tr = el("tr");
    for (const h of p.header) tr.appendChild(el("th", null, h));
    t.appendChild(tr);
  }
  for (const row of p.rows || []) {
    const tr = el("tr");
    for (const cell of row) tr.appendChild(el("td", null, cell));
    t.appendChild(tr);
  }
  return t;
}

function renderColumns(e) {
  const n = el("div", "st-columns");
  renderInto(n, e.children || []);
  return n;
}

function renderExpander(e) {
  const p = e.props || {};
  const d = el("details", "st-expander");
  if (p.expanded) d.open = true;
  d.appendChild(el("summary", null, p.label));
  const body = el("div");
  renderInto(body, e.children || []);
  d.appendChild(body);
  return d;
}

function widgetWrap(cls, labelText, control) {
  const n = el("div", "st-widget " + cls);
  if (labelText) n.appendChild(el("label", null, labelText));
  n.appendChild(control);
  return n;
}

function renderButton(e) {
  const b = el("button", "st", e.props.label);
  b.onclick = () => fire(e.key, true, true);
  return widgetWrap("st-button", null, b);
}

function renderCheckbox(e) {
  const label = el("label");
  const cb = document.createElement("input");
  cb.type = "checkbox";
  cb.checked = !!e.props.value;
  cb.onchange = () => fire(e.key, cb.checked, false);
  label.appendChild(cb);
  label.appendChild(document.createTextNode(" " + e.props.label));
  const n = el("div", "st-widget st-checkbox");
  n.appendChild(label);
  return n;
}

function renderSlider(e) {
  const p = e.props;
  const r = document.createElement("input");
  r.type = "range";
  r.min = p.min; r.max = p.max; r.step = p.step; r.value = p.value;
  const out = el("span", "st-range-value", String(p.value));
  r.oninput = () => (out.textContent = r.value);
  r.onchange = () => fire(e.key, parseFloat(r.value), false);
  const wrap = widgetWrap("st-slider", p.label, r);
  wrap.appendChild(out);
  return wrap;
}

function renderNumber(e) {
  const p = e.props;
  const inp = document.createElement("input");
  inp.type = "number";
  inp.value = p.value;
  inp.onchange = () => fire(e.key, parseFloat(inp.value), false);
  return widgetWrap("st-number", p.label, inp);
}

function renderTextInput(e) {
  const p = e.props;
  const inp = document.createElement("input");
  inp.type = "text";
  inp.value = p.value || "";
  inp.onchange = () => fire(e.key, inp.value, false);
  return widgetWrap("st-text-input", p.label, inp);
}

function renderTextArea(e) {
  const p = e.props;
  const ta = document.createElement("textarea");
  ta.rows = 4;
  ta.value = p.value || "";
  ta.onchange = () => fire(e.key, ta.value, false);
  return widgetWrap("st-text-area", p.label, ta);
}

function renderSelect(e, multi) {
  const p = e.props;
  const sel = document.createElement("select");
  if (multi) sel.multiple = true;
  const chosen = multi ? new Set(p.value || []) : null;
  for (const opt of p.options || []) {
    const o = document.createElement("option");
    o.value = opt; o.textContent = opt;
    if (multi ? chosen.has(opt) : opt === p.value) o.selected = true;
    sel.appendChild(o);
  }
  sel.onchange = () => {
    if (multi) {
      const vals = Array.from(sel.selectedOptions).map((o) => o.value);
      fire(e.key, vals, false);
    } else {
      fire(e.key, sel.value, false);
    }
  };
  return widgetWrap(multi ? "st-multiselect" : "st-selectbox", p.label, sel);
}

function renderRadio(e) {
  const p = e.props;
  const n = el("div", "st-widget st-radio");
  if (p.label) n.appendChild(el("label", null, p.label));
  for (const opt of p.options || []) {
    const lab = el("label");
    lab.style.display = "block";
    const rb = document.createElement("input");
    rb.type = "radio";
    rb.name = e.key;
    rb.value = opt;
    rb.checked = opt === p.value;
    rb.onchange = () => fire(e.key, opt, false);
    lab.appendChild(rb);
    lab.appendChild(document.createTextNode(" " + opt));
    n.appendChild(lab);
  }
  return n;
}

// Kick off the first run.
run(null);

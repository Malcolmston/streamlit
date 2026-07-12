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

// Form staging: widgets that belong to a form (props.form set) do not fire on
// change. Their values are collected here and submitted together when the
// form's submit button is clicked.
const formStage = {};

function commit(e, value) {
  const f = e.props && e.props.form;
  if (f) {
    (formStage[f] = formStage[f] || {})[e.key] = value;
  } else {
    fire(e.key, value, false);
  }
}

function submitForm(e) {
  const f = e.props.form || e.key;
  const values = formStage[f] || {};
  run({ key: e.key, button: true, form: f, values });
  formStage[f] = {};
}

// uploadFiles posts a multipart form to /api/upload and re-renders the tree.
async function uploadFiles(key, fileList) {
  if (!fileList || !fileList.length) return;
  status.textContent = "uploading…";
  const fd = new FormData();
  fd.append("session", sessionId || "");
  fd.append("key", key);
  for (const file of fileList) fd.append("files", file);
  try {
    const res = await fetch("/api/upload", { method: "POST", body: fd });
    const data = await res.json();
    sessionId = data.sessionId;
    renderTree(data.tree);
    status.textContent = "ready";
  } catch (err) {
    status.textContent = "error: " + err;
  }
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
    case "json": return renderJSON(p);
    case "alert": { const n = el("div", "st-alert " + p.kind); n.innerHTML = markdown(p.text || ""); return n; }
    case "metric": return renderMetric(p);
    case "spinner": return el("div", "st-spinner", "⏳ " + (p.label || ""));
    case "progress": return renderProgress(p);
    case "table": return renderTable(p);
    case "chart": { const n = el("div", "st-chart"); n.innerHTML = p.svg || ""; return n; }
    case "map": { const n = el("div", "st-map st-chart"); n.innerHTML = p.svg || ""; return n; }
    case "columns": return renderColumns(e);
    case "column": { const n = el("div", "st-column"); renderInto(n, e.children || []); return n; }
    case "container": { const n = el("div", "st-container"); renderInto(n, e.children || []); return n; }
    case "empty": { const n = el("div", "st-empty"); renderInto(n, e.children || []); return n; }
    case "expander": return renderExpander(e);
    case "tabs": return renderTabs(e);
    case "popover": return renderPopover(e);
    case "status": return renderStatus(e);
    case "form": { const n = el("div", "st-form"); renderInto(n, e.children || []); return n; }
    // media
    case "image": return renderImage(p);
    case "logo": { const img = el("img", "st-logo"); img.src = p.src || ""; img.alt = "logo"; return img; }
    case "audio": { const a = document.createElement("audio"); a.controls = true; a.src = p.src || ""; a.className = "st-audio"; return a; }
    case "video": { const v = document.createElement("video"); v.controls = true; v.src = p.src || ""; v.className = "st-video"; return v; }
    // chat
    case "chat_message": return renderChatMessage(e);
    case "chat_input": return renderChatInput(e);
    // widgets
    case "button": return renderButton(e);
    case "checkbox": return renderCheckbox(e);
    case "toggle": return renderToggle(e);
    case "slider": return renderSlider(e);
    case "select_slider": return renderSelectSlider(e);
    case "number": return renderNumber(e);
    case "text_input": return renderTextInput(e);
    case "text_area": return renderTextArea(e);
    case "selectbox": return renderSelect(e, false);
    case "radio": return renderRadio(e);
    case "multiselect": return renderSelect(e, true);
    case "date_input": return renderDateInput(e);
    case "time_input": return renderTimeInput(e);
    case "color_picker": return renderColorPicker(e);
    case "feedback": return renderFeedback(e);
    case "download_button": return renderDownloadButton(e);
    case "file_uploader": return renderFileUploader(e);
    case "form_submit": return renderFormSubmit(e);
    default: return el("div", null, "[" + e.type + "]");
  }
}

function renderMetric(p) {
  const n = el("div", "st-metric");
  n.appendChild(el("div", "label", p.label));
  n.appendChild(el("div", "value", p.value));
  if (p.delta) {
    const down = String(p.delta).trim().startsWith("-");
    // deltaColor: "normal" (up=green), "inverse" (up=red), "off" (grey).
    let cls = down ? "down" : "up";
    if (p.deltaColor === "inverse") cls = down ? "up" : "down";
    else if (p.deltaColor === "off") cls = "off";
    n.appendChild(el("div", "delta " + cls, (down ? "▼ " : "▲ ") + p.delta));
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
  const wrap = el("div", "st-table-wrap");
  const t = el("table", "st");
  let rows = (p.rows || []).map((r) => r.slice());
  let sortCol = -1, sortAsc = true;

  function body() {
    // Remove existing body rows, keep header.
    while (t.rows.length > 1) t.deleteRow(1);
    for (const row of rows) {
      const tr = el("tr");
      for (const cell of row) tr.appendChild(el("td", null, cell));
      t.appendChild(tr);
    }
  }

  if (p.header && p.header.length) {
    const tr = el("tr");
    p.header.forEach((h, i) => {
      const th = el("th", null, h);
      if (p.sortable) {
        th.classList.add("sortable");
        th.onclick = () => {
          sortAsc = sortCol === i ? !sortAsc : true;
          sortCol = i;
          rows.sort((a, b) => {
            const av = a[i], bv = b[i];
            const an = parseFloat(av), bn = parseFloat(bv);
            let c;
            if (!isNaN(an) && !isNaN(bn)) c = an - bn;
            else c = String(av).localeCompare(String(bv));
            return sortAsc ? c : -c;
          });
          body();
        };
      }
      tr.appendChild(th);
    });
    t.appendChild(tr);
  }
  body();
  wrap.appendChild(t);
  if (p.caption) wrap.appendChild(el("div", "st-caption", p.caption));
  return wrap;
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
  cb.onchange = () => commit(e, cb.checked);
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
  r.onchange = () => commit(e, parseFloat(r.value));
  const wrap = widgetWrap("st-slider", p.label, r);
  wrap.appendChild(out);
  return wrap;
}

function renderNumber(e) {
  const p = e.props;
  const inp = document.createElement("input");
  inp.type = "number";
  inp.value = p.value;
  inp.onchange = () => commit(e, parseFloat(inp.value));
  return widgetWrap("st-number", p.label, inp);
}

function renderTextInput(e) {
  const p = e.props;
  const inp = document.createElement("input");
  inp.type = "text";
  inp.value = p.value || "";
  inp.onchange = () => commit(e, inp.value);
  return widgetWrap("st-text-input", p.label, inp);
}

function renderTextArea(e) {
  const p = e.props;
  const ta = document.createElement("textarea");
  ta.rows = 4;
  ta.value = p.value || "";
  ta.onchange = () => commit(e, ta.value);
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
      commit(e, vals);
    } else {
      commit(e, sel.value);
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
    rb.onchange = () => commit(e, opt);
    lab.appendChild(rb);
    lab.appendChild(document.createTextNode(" " + opt));
    n.appendChild(lab);
  }
  return n;
}

function renderJSON(p) {
  if (p.collapsed) {
    const d = el("details", "st-json-details");
    d.appendChild(el("summary", null, "JSON"));
    d.appendChild(el("pre", "st-json", p.json));
    return d;
  }
  return el("pre", "st-json", p.json);
}

function renderImage(p) {
  const fig = el("figure", "st-image");
  const img = el("img");
  img.src = p.src || "";
  img.alt = p.caption || "image";
  fig.appendChild(img);
  if (p.caption) fig.appendChild(el("figcaption", "st-caption", p.caption));
  return fig;
}

function renderTabs(e) {
  const p = e.props || {};
  const n = el("div", "st-tabs");
  const bar = el("div", "st-tab-bar");
  const panels = el("div", "st-tab-panels");
  const tabs = e.children || [];
  const buttons = [];
  tabs.forEach((tab, i) => {
    const label = (tab.props && tab.props.label) || "";
    const b = el("button", "st-tab", label);
    const panel = el("div", "st-tab-panel");
    renderInto(panel, tab.children || []);
    if (i !== 0) panel.style.display = "none";
    else b.classList.add("active");
    b.onclick = () => {
      buttons.forEach((btn) => btn.classList.remove("active"));
      b.classList.add("active");
      Array.from(panels.children).forEach((pn, j) => {
        pn.style.display = j === i ? "" : "none";
      });
    };
    buttons.push(b);
    bar.appendChild(b);
    panels.appendChild(panel);
  });
  n.appendChild(bar);
  n.appendChild(panels);
  return n;
}

function renderPopover(e) {
  const p = e.props || {};
  const n = el("div", "st-popover");
  const btn = el("button", "st", (p.label || "") + " ▾");
  const panel = el("div", "st-popover-panel");
  panel.style.display = "none";
  renderInto(panel, e.children || []);
  btn.onclick = () => {
    panel.style.display = panel.style.display === "none" ? "block" : "none";
  };
  n.appendChild(btn);
  n.appendChild(panel);
  return n;
}

function renderStatus(e) {
  const p = e.props || {};
  const icons = { running: "⏳", complete: "✅", error: "❌" };
  const d = el("details", "st-status " + (p.state || "running"));
  d.open = p.state === "running";
  d.appendChild(el("summary", null, (icons[p.state] || "⏳") + " " + (p.label || "")));
  const body = el("div");
  renderInto(body, e.children || []);
  d.appendChild(body);
  return d;
}

function renderChatMessage(e) {
  const p = e.props || {};
  const role = p.role || "assistant";
  const n = el("div", "st-chat-message " + role);
  const avatar = el("div", "st-chat-avatar", role === "user" ? "🧑" : "🤖");
  const body = el("div", "st-chat-body");
  renderInto(body, e.children || []);
  n.appendChild(avatar);
  n.appendChild(body);
  return n;
}

function renderChatInput(e) {
  const p = e.props || {};
  const n = el("div", "st-widget st-chat-input");
  const inp = document.createElement("input");
  inp.type = "text";
  inp.placeholder = p.placeholder || "";
  inp.onkeydown = (ev) => {
    if (ev.key === "Enter" && inp.value.trim() !== "") {
      fire(e.key, inp.value, false);
      inp.value = "";
    }
  };
  n.appendChild(inp);
  return n;
}

function renderToggle(e) {
  const p = e.props || {};
  const label = el("label", "st-toggle-label");
  const cb = document.createElement("input");
  cb.type = "checkbox";
  cb.className = "st-toggle-input";
  cb.checked = !!p.value;
  cb.onchange = () => commit(e, cb.checked);
  const track = el("span", "st-toggle-track");
  label.appendChild(cb);
  label.appendChild(track);
  label.appendChild(document.createTextNode(" " + (p.label || "")));
  const n = el("div", "st-widget st-toggle");
  n.appendChild(label);
  return n;
}

function renderSelectSlider(e) {
  const p = e.props || {};
  const opts = p.options || [];
  const r = document.createElement("input");
  r.type = "range";
  r.min = 0;
  r.max = Math.max(0, opts.length - 1);
  r.step = 1;
  r.value = p.index || 0;
  const out = el("span", "st-range-value", String(p.value));
  r.oninput = () => (out.textContent = String(opts[parseInt(r.value, 10)]));
  r.onchange = () => commit(e, opts[parseInt(r.value, 10)]);
  const wrap = widgetWrap("st-select-slider", p.label, r);
  wrap.appendChild(out);
  return wrap;
}

function renderDateInput(e) {
  const p = e.props || {};
  const inp = document.createElement("input");
  inp.type = "date";
  inp.value = p.value || "";
  inp.onchange = () => commit(e, inp.value);
  return widgetWrap("st-date-input", p.label, inp);
}

function renderTimeInput(e) {
  const p = e.props || {};
  const inp = document.createElement("input");
  inp.type = "time";
  inp.value = p.value || "";
  inp.onchange = () => commit(e, inp.value);
  return widgetWrap("st-time-input", p.label, inp);
}

function renderColorPicker(e) {
  const p = e.props || {};
  const inp = document.createElement("input");
  inp.type = "color";
  inp.value = p.value || "#000000";
  inp.onchange = () => commit(e, inp.value);
  return widgetWrap("st-color-picker", p.label, inp);
}

function renderFeedback(e) {
  const p = e.props || {};
  const n = el("div", "st-widget st-feedback");
  const cur = typeof p.value === "number" ? p.value : -1;
  if (p.kind === "thumbs") {
    [["👎", 0], ["👍", 1]].forEach(([icon, val]) => {
      const b = el("button", "st-feedback-btn" + (cur === val ? " active" : ""), icon);
      b.onclick = () => commit(e, val);
      n.appendChild(b);
    });
  } else {
    for (let i = 0; i < 5; i++) {
      const b = el("button", "st-feedback-btn" + (i <= cur ? " active" : ""), i <= cur ? "★" : "☆");
      b.onclick = () => commit(e, i);
      n.appendChild(b);
    }
  }
  return n;
}

function renderDownloadButton(e) {
  const p = e.props || {};
  const a = document.createElement("a");
  a.className = "st st-download";
  a.textContent = p.label || "Download";
  a.href = p.href || "#";
  a.download = p.filename || "download";
  a.onclick = () => fire(e.key, true, true);
  return widgetWrap("st-download-button", null, a);
}

function renderFileUploader(e) {
  const p = e.props || {};
  const inp = document.createElement("input");
  inp.type = "file";
  inp.multiple = true;
  inp.onchange = () => uploadFiles(e.key, inp.files);
  const wrap = widgetWrap("st-file-uploader", p.label, inp);
  if (p.files && p.files.length) {
    const list = el("ul", "st-file-list");
    for (const name of p.files) list.appendChild(el("li", null, name));
    wrap.appendChild(list);
  }
  return wrap;
}

function renderFormSubmit(e) {
  const b = el("button", "st st-form-submit", (e.props && e.props.label) || "Submit");
  b.onclick = () => submitForm(e);
  return widgetWrap("st-form-submit-wrap", null, b);
}

// Kick off the first run.
run(null);

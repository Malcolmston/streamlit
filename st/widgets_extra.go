package st

// This file adds interactive widgets that bring the port closer to Streamlit's
// input surface: single/multi chip selectors, range sliders, bounded numeric
// and text inputs, a password field, a date-range picker, and media capture
// controls. Each follows the same resolve-key / read-state / append-element /
// return-value pattern as the widgets in widgets.go.

// exFloatPair coerces a value (typically a two-element []any decoded from JSON)
// into an ordered pair of floats, falling back to (defLo, defHi). The lower of
// the two resolved values is always returned first.
func exFloatPair(v any, defLo, defHi float64) (float64, float64) {
	lo, hi := defLo, defHi
	switch xs := v.(type) {
	case []any:
		if len(xs) >= 2 {
			lo = asFloat(xs[0], defLo)
			hi = asFloat(xs[1], defHi)
		}
	case []float64:
		if len(xs) >= 2 {
			lo, hi = xs[0], xs[1]
		}
	}
	if lo > hi {
		lo, hi = hi, lo
	}
	return lo, hi
}

// exStringPair coerces a value into an ordered pair of strings, falling back to
// (defLo, defHi). Ordering is by the supplied options slice when both values are
// members; otherwise the values are returned as decoded.
func exStringPair(v any, defLo, defHi string) (string, string) {
	lo, hi := defLo, defHi
	switch xs := v.(type) {
	case []any:
		if len(xs) >= 2 {
			lo = asString(xs[0], defLo)
			hi = asString(xs[1], defHi)
		}
	case []string:
		if len(xs) >= 2 {
			lo, hi = xs[0], xs[1]
		}
	}
	return lo, hi
}

// indexOfString returns the position of s in xs, or -1 if absent.
func indexOfString(xs []string, s string) int {
	for i, x := range xs {
		if x == s {
			return i
		}
	}
	return -1
}

// Pills adds a row of selectable "pill" chips allowing multiple selections and
// returns the currently selected options, mirroring Streamlit's st.pills.
// Selections not present in options are dropped. An optional stable key may be
// supplied as the final argument.
func (c *Container) Pills(label string, options []string, key ...string) []string {
	k := c.s.key(optKey(key), "pills")
	sel := asStringSlice(c.s.widgets[k], nil)
	filtered := make([]string, 0, len(sel))
	for _, s := range sel {
		if containsString(options, s) {
			filtered = append(filtered, s)
		}
	}
	c.addWidget(k, "pills", props{"label": label, "options": toAnySlice(options), "value": filtered})
	return filtered
}

// SegmentedControl adds a single-choice segmented button group and returns the
// selected option, mirroring Streamlit's st.segmented_control. The first option
// is selected by default; if options is empty the empty string is returned.
func (c *Container) SegmentedControl(label string, options []string, key ...string) string {
	k := c.s.key(optKey(key), "segmented_control")
	def := ""
	if len(options) > 0 {
		def = options[0]
	}
	val := asString(c.s.widgets[k], def)
	if !containsString(options, val) {
		val = def
	}
	c.addWidget(k, "segmented_control", props{"label": label, "options": toAnySlice(options), "value": val})
	return val
}

// PrimaryButton adds an emphasised call-to-action button and returns true on the
// single run triggered by a click, mirroring Streamlit's st.button(type=
// "primary"). It behaves exactly like [Container.Button] but is styled as the
// page's primary action.
func (c *Container) PrimaryButton(label string, key ...string) bool {
	k := c.s.key(optKey(key), "button")
	clicked := c.s.clicked[k]
	c.addWidget(k, "button", props{"label": label, "kind": "primary"})
	return clicked
}

// PasswordInput adds a single-line text field whose contents are masked in the
// browser and returns its current value, mirroring Streamlit's
// st.text_input(type="password"). def is the initial value.
func (c *Container) PasswordInput(label, def string, key ...string) string {
	k := c.s.key(optKey(key), "password_input")
	val := asString(c.s.widgets[k], def)
	c.addWidget(k, "text_input", props{"label": label, "value": val, "type": "password"})
	return val
}

// TextInputMax adds a single-line text field that accepts at most maxChars
// characters and returns its current value, truncated to maxChars runes,
// mirroring Streamlit's st.text_input(max_chars=…). A maxChars of zero or less
// imposes no limit.
func (c *Container) TextInputMax(label, def string, maxChars int, key ...string) string {
	k := c.s.key(optKey(key), "text_input")
	val := asString(c.s.widgets[k], def)
	if maxChars > 0 {
		if r := []rune(val); len(r) > maxChars {
			val = string(r[:maxChars])
		}
	}
	c.addWidget(k, "text_input", props{"label": label, "value": val, "maxChars": maxChars})
	return val
}

// NumberInputRange adds a numeric entry field bounded to the inclusive range
// [min, max] and returns its current value clamped to that range, mirroring
// Streamlit's st.number_input(min_value=…, max_value=…). def is the initial
// value and is itself clamped.
func (c *Container) NumberInputRange(label string, min, max, def float64, key ...string) float64 {
	k := c.s.key(optKey(key), "number")
	if min > max {
		min, max = max, min
	}
	val := clampFloat(asFloat(c.s.widgets[k], def), min, max)
	c.addWidget(k, "number", props{"label": label, "value": val, "min": min, "max": max})
	return val
}

// SliderRange adds a two-handle range slider bounded by min and max and returns
// the currently selected low and high values, mirroring Streamlit's st.slider
// used with a tuple default. low and high are the initial handle positions; both
// returned values are clamped to [min, max] and the lower is always returned
// first. step controls the increment; a non-positive step defaults to
// (max-min)/100.
func (c *Container) SliderRange(label string, min, max, low, high, step float64, key ...string) (float64, float64) {
	k := c.s.key(optKey(key), "slider_range")
	if min > max {
		min, max = max, min
	}
	if step <= 0 {
		if max > min {
			step = (max - min) / 100
		} else {
			step = 1
		}
	}
	if low > high {
		low, high = high, low
	}
	lo, hi := exFloatPair(c.s.widgets[k], low, high)
	lo = clampFloat(lo, min, max)
	hi = clampFloat(hi, min, max)
	if lo > hi {
		lo, hi = hi, lo
	}
	c.addWidget(k, "slider_range", props{
		"label": label, "min": min, "max": max, "step": step,
		"value": []any{lo, hi},
	})
	return lo, hi
}

// SelectSliderRange adds a slider that selects a contiguous range across a set
// of discrete options and returns the currently selected low and high options,
// mirroring Streamlit's st.select_slider used with a tuple default. By default
// the full range (first to last option) is selected. The returned pair is
// ordered by position in options. If options is empty two empty strings are
// returned.
func (c *Container) SelectSliderRange(label string, options []string, key ...string) (string, string) {
	k := c.s.key(optKey(key), "select_slider_range")
	defLo, defHi := "", ""
	if len(options) > 0 {
		defLo = options[0]
		defHi = options[len(options)-1]
	}
	lo, hi := exStringPair(c.s.widgets[k], defLo, defHi)
	if !containsString(options, lo) {
		lo = defLo
	}
	if !containsString(options, hi) {
		hi = defHi
	}
	iLo, iHi := indexOfString(options, lo), indexOfString(options, hi)
	if iLo > iHi {
		lo, hi = hi, lo
		iLo, iHi = iHi, iLo
	}
	c.addWidget(k, "select_slider_range", props{
		"label": label, "options": toAnySlice(options),
		"value": []any{lo, hi}, "indexLow": iLo, "indexHigh": iHi,
	})
	return lo, hi
}

// DateRangeInput adds a calendar control that selects a start and end date and
// returns them as ISO-8601 strings ("2006-01-02"), mirroring Streamlit's
// st.date_input used with a tuple default. defStart and defEnd are the initial
// values and may be empty. The earlier date is always returned first.
func (c *Container) DateRangeInput(label, defStart, defEnd string, key ...string) (string, string) {
	k := c.s.key(optKey(key), "date_range_input")
	start, end := exStringPair(c.s.widgets[k], defStart, defEnd)
	// ISO-8601 dates sort chronologically as plain strings.
	if start != "" && end != "" && start > end {
		start, end = end, start
	}
	c.addWidget(k, "date_range_input", props{"label": label, "value": []any{start, end}})
	return start, end
}

// CameraInput adds a control that captures a photo from the user's webcam and
// returns the captured image files for the current session, mirroring
// Streamlit's st.camera_input. Like [Container.FileUploader] the bytes arrive
// over the multipart /api/upload endpoint keyed by the widget key and are held
// in session state; the returned slice is empty until a photo is taken.
func (c *Container) CameraInput(label string, key ...string) []UploadedFile {
	k := c.s.key(optKey(key), "camera_input")
	files := c.s.uploads[k]
	names := make([]any, len(files))
	for i, f := range files {
		names[i] = f.Name
	}
	c.addWidget(k, "camera_input", props{"label": label, "files": names})
	return files
}

// AudioInput adds a control that records audio from the user's microphone and
// returns the recorded files for the current session, mirroring Streamlit's
// st.audio_input. It works like [Container.CameraInput], receiving bytes over
// the multipart /api/upload endpoint keyed by the widget key.
func (c *Container) AudioInput(label string, key ...string) []UploadedFile {
	k := c.s.key(optKey(key), "audio_input")
	files := c.s.uploads[k]
	names := make([]any, len(files))
	for i, f := range files {
		names[i] = f.Name
	}
	c.addWidget(k, "audio_input", props{"label": label, "files": names})
	return files
}

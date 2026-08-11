package st

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"math"
	"net/http"
	"strings"
)

// dataURIFromBytes encodes b as a base64 "data:" URI, detecting the content
// type from the leading bytes. Empty input yields an empty string.
func dataURIFromBytes(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	mime := http.DetectContentType(b)
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b)
}

// mediaSrc resolves a media source into a string usable as an HTML src
// attribute. It accepts:
//
//   - string: returned unchanged (a URL or an existing data URI);
//   - []byte: encoded as a base64 data URI with a detected content type;
//   - image.Image: PNG-encoded and returned as a data URI.
//
// Unsupported types yield the empty string.
func mediaSrc(src any) string {
	switch v := src.(type) {
	case string:
		return v
	case []byte:
		return dataURIFromBytes(v)
	case image.Image:
		var buf bytes.Buffer
		if err := png.Encode(&buf, v); err != nil {
			return ""
		}
		return dataURIFromBytes(buf.Bytes())
	}
	return ""
}

// Image adds an image. src may be a URL string, PNG/JPEG (or other) bytes, or
// an [image.Image] value, which is encoded to PNG. An optional caption is shown
// beneath the image.
func (c *Container) Image(src any, caption ...string) {
	c.add("image", props{"src": mediaSrc(src), "caption": optKey(caption)})
}

// Logo adds a small brand image, typically shown at the top of the app or
// sidebar. src accepts the same values as [Container.Image].
func (c *Container) Logo(src any) {
	c.add("logo", props{"src": mediaSrc(src)})
}

// Audio adds an audio player. src may be a URL string or raw audio bytes (for
// example WAV or MP3), which are embedded as a data URI.
func (c *Container) Audio(src any) {
	c.add("audio", props{"src": mediaSrc(src)})
}

// Video adds a video player. src may be a URL string or raw video bytes (for
// example MP4), which are embedded as a data URI.
func (c *Container) Video(src any) {
	c.add("video", props{"src": mediaSrc(src)})
}

// MapPoint is a single latitude/longitude coordinate plotted by
// [Container.Map]. Latitude is in the range [-90, 90] and longitude in
// [-180, 180].
type MapPoint struct {
	Lat float64
	Lng float64
}

// Map adds a simple point map. Each point is projected with an equirectangular
// projection and drawn as a dot over a framed world extent, rendered to inline
// SVG on the server using only the standard library.
func (c *Container) Map(points []MapPoint) {
	c.add("map", props{"svg": renderMap(points, 640, 320)})
}

// renderMap projects lat/lng points onto a width×height SVG canvas.
func renderMap(points []MapPoint, width, height int) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img">`,
		width, height, width, height)
	fmt.Fprintf(&b, `<rect x="0" y="0" width="%d" height="%d" fill="#eef3f8" stroke="#cdd7e2"/>`, width, height)

	// Reference graticule at the equator and prime meridian.
	midX := width / 2
	midY := height / 2
	fmt.Fprintf(&b, `<line x1="0" y1="%d" x2="%d" y2="%d" stroke="#dbe4ee"/>`, midY, width, midY)
	fmt.Fprintf(&b, `<line x1="%d" y1="0" x2="%d" y2="%d" stroke="#dbe4ee"/>`, midX, midX, height)

	for _, pt := range points {
		// A NaN coordinate would survive the clamp (math.Min(90, NaN) is NaN)
		// and emit cx="NaN", which browsers drop; skip it instead.
		if !finite(pt.Lat) || !finite(pt.Lng) {
			continue
		}
		lat := math.Max(-90, math.Min(90, pt.Lat))
		lng := math.Max(-180, math.Min(180, pt.Lng))
		x := (lng + 180) / 360 * float64(width)
		y := (90 - lat) / 180 * float64(height)
		fmt.Fprintf(&b, `<circle cx="%.1f" cy="%.1f" r="4" fill="#e45756" fill-opacity="0.75" stroke="#a32f2f"/>`, x, y)
	}
	b.WriteString(`</svg>`)
	return b.String()
}

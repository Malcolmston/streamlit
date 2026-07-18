package st

import (
	"errors"
	"testing"
)

// TestEffectsElementConstruction checks each new display/effect method emits
// the expected element type with its key properties.
func TestEffectsElementConstruction(t *testing.T) {
	_, tree := render(func(s *Session) {
		s.Latex(`\int_a^b f(x)\,dx`)
		s.Html("<b>hi</b>")
		s.Echo("x := 1")
		s.Toast("saved", "✅")
		s.Balloons()
		s.Snow()
		s.LinkButton("Docs", "https://example.com")
		s.PageLink("/next", "Next")
		s.Help(42)
	})
	for _, typ := range []string{
		"latex", "html", "echo", "toast", "link_button", "page_link", "help",
	} {
		if find(tree, typ) == nil {
			t.Errorf("missing element type %q", typ)
		}
	}
	if got := len(findAll(tree, "effect")); got != 2 {
		t.Errorf("effect count = %d, want 2 (balloons+snow)", got)
	}
}

func TestLatexAndHtmlProps(t *testing.T) {
	_, tree := render(func(s *Session) {
		s.Latex("E=mc^2")
		s.Html("<i>raw</i>")
	})
	if got := find(tree, "latex").Props["expr"]; got != "E=mc^2" {
		t.Errorf("latex expr = %v", got)
	}
	if got := find(tree, "html").Props["html"]; got != "<i>raw</i>" {
		t.Errorf("html = %v", got)
	}
}

func TestBadgeColorNormalisation(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", "blue"},
		{"green", "green"},
		{"gray", "grey"},
		{"grey", "grey"},
		{"nonsense", "blue"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			_, tree := render(func(s *Session) {
				if tc.in == "" {
					s.Badge("b")
				} else {
					s.Badge("b", tc.in)
				}
			})
			if got := find(tree, "badge").Props["color"]; got != tc.want {
				t.Errorf("Badge(%q) color = %v, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestExceptionCapturesTypeAndMessage(t *testing.T) {
	_, tree := render(func(s *Session) { s.Exception(errors.New("boom")) })
	e := find(tree, "exception")
	if e.Props["message"] != "boom" {
		t.Errorf("message = %v, want boom", e.Props["message"])
	}
	if e.Props["type"] == "" {
		t.Errorf("type not captured")
	}

	_, tree = render(func(s *Session) { s.Exception(nil) })
	e = find(tree, "exception")
	if e.Props["message"] != "" || e.Props["type"] != "" {
		t.Errorf("nil error should be empty, got %+v", e.Props)
	}
}

func TestHelpDescribesValue(t *testing.T) {
	_, tree := render(func(s *Session) { s.Help("hello") })
	e := find(tree, "help")
	if e.Props["type"] != "string" {
		t.Errorf("help type = %v, want string", e.Props["type"])
	}
	if e.Props["value"] != "hello" {
		t.Errorf("help value = %v, want hello", e.Props["value"])
	}
}

func TestToastIconOptional(t *testing.T) {
	_, tree := render(func(s *Session) { s.Toast("hi") })
	if got := find(tree, "toast").Props["icon"]; got != "" {
		t.Errorf("toast icon = %v, want empty", got)
	}
	_, tree = render(func(s *Session) { s.Toast("hi", "🔥") })
	if got := find(tree, "toast").Props["icon"]; got != "🔥" {
		t.Errorf("toast icon = %v, want 🔥", got)
	}
}

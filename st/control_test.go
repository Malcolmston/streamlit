package st

import "testing"

func TestStopHaltsRun(t *testing.T) {
	_, tree := render(func(s *Session) {
		s.Text("before")
		s.Stop()
		s.Text("after")
	})
	texts := findAll(tree, "text")
	if len(texts) != 1 {
		t.Fatalf("expected 1 text element (Stop halts), got %d", len(texts))
	}
	if texts[0].Props["text"] != "before" {
		t.Errorf("kept element = %v, want 'before'", texts[0].Props["text"])
	}
}

func TestStopDoesNotLeakPanic(t *testing.T) {
	// A run using Stop must return normally and remain reusable.
	s := newSession()
	s.run(func(s *Session) { s.Stop() })
	tree := s.run(func(s *Session) { s.Text("ok") })
	if find(tree, "text") == nil {
		t.Fatal("session not reusable after Stop")
	}
}

func TestNonStopPanicPropagates(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected a real panic to propagate")
		}
	}()
	s := newSession()
	s.run(func(s *Session) { panic("real failure") })
}

func TestSetPageConfig(t *testing.T) {
	_, tree := render(func(s *Session) { s.SetPageConfig("My App", "🚀") })
	if tree.Props["pageTitle"] != "My App" {
		t.Errorf("pageTitle = %v, want 'My App'", tree.Props["pageTitle"])
	}
	if tree.Props["pageIcon"] != "🚀" {
		t.Errorf("pageIcon = %v, want 🚀", tree.Props["pageIcon"])
	}
}

func TestBorderedContainer(t *testing.T) {
	_, tree := render(func(s *Session) {
		bc := s.BorderedContainer()
		bc.Text("inside")
	})
	cont := find(tree, "container")
	if cont == nil {
		t.Fatal("no container element")
	}
	if cont.Props["border"] != true {
		t.Errorf("border = %v, want true", cont.Props["border"])
	}
	if find(cont, "text") == nil {
		t.Error("nested text not found in bordered container")
	}
}

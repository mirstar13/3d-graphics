package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mirstar13/go-3d-graphics/internal/project"
)

func TestBodyAuthoringReopenAndPlay(t *testing.T) {
	m := inputEditor(t)
	ed := m.document
	if err := ed.Set("box", "velocity.x", "NaN"); err == nil || ed.Dirty() {
		t.Fatal("invalid velocity changed document")
	}
	m.selectID("box")
	values := map[string]string{"velocity.x": "4", "angular_velocity.z": "-2", "sensor": "true"}
	for i, prop := range project.Properties {
		value, ok := values[prop.Key]
		if !ok {
			continue
		}
		m.property = i
		m, _ = editorRune(m, "e")
		for len(m.draft) > 0 {
			m = resourceKey(m, tea.KeyBackspace)
		}
		m, _ = editorRune(m, value)
		m = resourceKey(m, tea.KeyEnter)
		if m.modal != "" {
			t.Fatalf("edit %s: %s", prop.Key, m.message)
		}
	}
	if !ed.Undo() || !ed.Redo() {
		t.Fatal("missing edit history")
	}
	m.save()
	p, err := project.Open(m.project.Root)
	if err != nil {
		t.Fatal(err)
	}
	if err = m.install(p); err != nil {
		t.Fatal(err)
	}
	m.startPlay()
	id, ok := m.play.Find("box")
	if !ok {
		t.Fatal("missing box")
	}
	b, err := m.play.Inspect(id)
	if err != nil {
		t.Fatal(err)
	}
	if b.Body.Velocity.X != 4 || b.Body.AngularVelocity.Z != -2 || !b.Body.Collider.Sensor {
		t.Fatalf("unexpected initial body %+v", b.Body)
	}
}

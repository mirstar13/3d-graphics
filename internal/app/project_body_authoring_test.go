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
	for key, value := range map[string]string{"friction": "-1", "restitution": "1.01", "collision_layer": "-1", "collision_mask": "4294967296"} {
		if ed.Set("box", key, value) == nil || ed.Dirty() {
			t.Fatal("invalid material changed document", key)
		}
	}
	m.selectID("box")
	values := map[string]string{"velocity.x": "4", "angular_velocity.z": "-2", "sensor": "true", "friction": "0", "restitution": "0.85", "collision_layer": "0x80000000", "collision_mask": "0"}
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
	if b.Body.Velocity.X != 4 || b.Body.AngularVelocity.Z != -2 || !b.Body.Collider.Sensor || b.Body.Collider.Friction != 0 || b.Body.Collider.Restitution != .85 || b.Body.Collider.Layer != 1<<31 || b.Body.Collider.Mask != 0 {
		t.Fatalf("unexpected initial body %+v", b.Body)
	}
}

func TestColliderMaterialDefaults(t *testing.T) {
	m := inputEditor(t)
	for _, key := range []string{"friction", "restitution", "collision_layer", "collision_mask"} {
		if err := m.document.Set("box", key, "0"); err != nil {
			t.Fatal(err)
		}
		if err := m.document.Set("box", key, "default"); err != nil {
			t.Fatal(err)
		}
	}
	if m.document.Dirty() {
		t.Fatal("restoring both defaults should restore the authored scene")
	}
	m.startPlay()
	id, _ := m.play.Find("box")
	info, err := m.play.Inspect(id)
	if err != nil {
		t.Fatal(err)
	}
	if info.Body.Collider.Friction != .6 || info.Body.Collider.Restitution != .1 || info.Body.Collider.Layer != 1 || info.Body.Collider.Mask != ^uint32(0) {
		t.Fatalf("legacy material changed: %+v", info.Body.Collider)
	}
}

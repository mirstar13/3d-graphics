package game

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"github.com/mirstar13/go-3d-graphics/internal/project"
)

func TestLinkedInstancesOwnRuntimeState(t *testing.T) {
	p, err := project.Create(filepath.Join(t.TempDir(), "Game"))
	if err != nil {
		t.Fatal(err)
	}
	path, err := p.Extract(p.Scene, "box", "Box")
	if err != nil {
		t.Fatal(err)
	}
	ed, _ := project.NewEditor(p.Scene)
	first, err := ed.Instantiate(p, path)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ed.Instantiate(p, path)
	if err != nil {
		t.Fatal(err)
	}
	if err = ed.Set(first, "position.x", "-3"); err != nil {
		t.Fatal(err)
	}
	if err = ed.Set(second, "position.x", "3"); err != nil {
		t.Fatal(err)
	}
	authored := ed.Document()
	s, err := NewSession(authored, physics.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	a, _ := s.Find(first)
	b, _ := s.Find(second)
	if a == b {
		t.Fatal("shared runtime handle")
	}
	s.World.SetPaused(true)
	if err = s.QueueDestroy(a); err != nil {
		t.Fatal(err)
	}
	if err = s.Step(); err != nil {
		t.Fatal(err)
	}
	if _, err = s.State(a); err == nil {
		t.Fatal("destroyed handle still live")
	}
	if state, err := s.State(b); err != nil || state.Position.X != 3 {
		t.Fatal("other instance affected", state, err)
	}
	if !reflect.DeepEqual(authored, ed.Document()) {
		t.Fatal("runtime changed authored instance")
	}
	restarted, err := NewSession(authored, physics.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := restarted.Find(first); !ok {
		t.Fatal("restart lost destroyed instance")
	}
	if err = p.Save(authored); err != nil {
		t.Fatal(err)
	}
	reopened, err := project.Open(p.Root)
	if err != nil || !reflect.DeepEqual(authored, reopened.Scene) {
		t.Fatal("play changed round trip", err)
	}
}

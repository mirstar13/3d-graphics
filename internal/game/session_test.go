package game

import (
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"github.com/mirstar13/go-3d-graphics/internal/project"
	"reflect"
	"testing"
)

func TestPlayIsolatedFromAuthoredDocument(t *testing.T) {
	doc := project.NewDocument()
	before := doc.Clone()
	s, err := NewSession(doc, physics.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	id, _ := s.Find("box")
	for i := 0; i < 60; i++ {
		if err = s.Advance(physics.DefaultOptions().FixedStep); err != nil {
			t.Fatal(err)
		}
	}
	state, err := s.State(id)
	if err != nil || state.Position.Y >= 3 {
		t.Fatal("body did not fall", state, err)
	}
	if !reflect.DeepEqual(doc, before) {
		t.Fatal("play modified authored document")
	}
	reopened, _ := NewSession(doc, physics.DefaultOptions())
	if _, err = reopened.State(id); err == nil {
		t.Fatal("handle survived new session")
	}
	newID, _ := reopened.Find("box")
	state, _ = reopened.State(newID)
	if state.Position.Y != 3 {
		t.Fatal("restart did not restore authored state")
	}
}
func TestRuntimeCommandsPauseAndGeneration(t *testing.T) {
	s, err := NewSession(project.NewDocument(), physics.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	id, _ := s.Find("box")
	s.World.SetPaused(true)
	s.QueueImpulse(id, engine.Point{Y: 5})
	before, _ := s.State(id)
	s.Advance(physics.DefaultOptions().FixedStep)
	after, _ := s.State(id)
	if after != before || s.PendingCommands() != 1 {
		t.Fatal("paused command applied")
	}
	if err = s.Step(); err != nil {
		t.Fatal(err)
	}
	after, _ = s.State(id)
	if after.Position.Y <= before.Position.Y || s.PendingCommands() != 0 || s.World.Stats().Tick != 1 {
		t.Fatal("single step command semantics")
	}
	s.QueueDestroy(id)
	if err = s.Step(); err != nil {
		t.Fatal(err)
	}
	if _, err = s.State(id); err == nil {
		t.Fatal("destroyed handle still valid")
	}
	s.QueueSpawn(project.NewEntity("new", "New"))
	if err = s.Step(); err != nil {
		t.Fatal(err)
	}
	replacement, ok := s.Find("new")
	if !ok || replacement == id {
		t.Fatal("stale handle resurrected")
	}
	state, _ := s.State(replacement)
	if state.Position.Y >= 3 {
		t.Fatal("spawned body did not integrate on commanded tick")
	}
	if err = s.QueueImpulse(id, engine.Point{Y: 1}); err == nil {
		t.Fatal("stale command accepted")
	}
}

package game

import (
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"github.com/mirstar13/go-3d-graphics/internal/project"
	"math"
	"testing"
)

func TestNestedRuntimeTransformsAndDestruction(t *testing.T) {
	root := project.NewEntity("root", "Same name")
	root.Position = engine.Point{X: 4, Y: 8}
	root.Rotation.Z = 90
	child := project.NewEntity("child", "Same name")
	child.Parent = "root"
	child.Body = "none"
	child.Position = engine.Point{X: 2}
	leaf := project.NewEntity("leaf", "Same name")
	leaf.Parent = "child"
	leaf.Body = "none"
	leaf.Position = engine.Point{X: 1}
	d := project.Document{Version: project.Version, Name: "Nested", Entities: []project.Entity{leaf, child, root}}
	s, err := NewSession(d, physics.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	rootID, _ := s.Find("root")
	childID, _ := s.Find("child")
	leafID, _ := s.Find("leaf")
	got, err := s.State(leafID)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got.Position.X-4) > 1e-8 || math.Abs(got.Position.Y-11) > 1e-8 || math.Abs(got.Rotation.Z-90) > 1e-8 {
		t.Fatal("nested world pose", got)
	}
	if len(s.Scene.AllNodes) != 4 {
		t.Fatal("duplicate names lost registry entries", len(s.Scene.AllNodes))
	}
	s.World.SetPaused(true)
	if err = s.Step(); err != nil {
		t.Fatal(err)
	}
	moved, _ := s.State(leafID)
	parent, _ := s.State(rootID)
	if moved.Position.Y >= got.Position.Y || math.Abs(moved.Position.Y-parent.Position.Y-3) > 1e-8 {
		t.Fatal("child did not follow", moved, parent)
	}
	lines := s.SelectionLines(leafID)
	var center engine.Point
	for _, line := range lines {
		center.X += line.Start.X + line.End.X
		center.Y += line.Start.Y + line.End.Y
	}
	center.X /= 24
	center.Y /= 24
	if math.Abs(center.X-moved.Position.X) > 1e-8 || math.Abs(center.Y-moved.Position.Y) > 1e-8 {
		t.Fatal("highlight not world-space", center, moved.Position)
	}
	if err = s.QueueDestroy(rootID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.State(leafID); err != nil {
		t.Fatal("destroy applied before tick")
	}
	if err = s.Step(); err != nil {
		t.Fatal(err)
	}
	for _, id := range []EntityID{rootID, childID, leafID} {
		if _, err = s.State(id); err == nil {
			t.Fatal("stale descendant survived")
		}
	}
	if len(s.Scene.AllNodes) != 1 || len(s.Scene.Root.Children) != 0 {
		t.Fatal("visual registry leaked")
	}
	if err = s.QueueSpawn(root); err != nil {
		t.Fatal(err)
	}
	if err = s.Step(); err != nil {
		t.Fatal(err)
	}
	if _, err = s.State(rootID); err == nil {
		t.Fatal("reused slot resurrected handle")
	}
	if d.Entities[0].Position != leaf.Position {
		t.Fatal("runtime mutated document")
	}
}
func TestSpawnChildAndQueuedParentRemoval(t *testing.T) {
	root := project.NewEntity("root", "Root")
	s, err := NewSession(project.Document{Version: project.Version, Name: "Spawn", Entities: []project.Entity{root}}, physics.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	s.World.SetPaused(true)
	child := project.NewEntity("child", "Child")
	child.Parent = "root"
	child.Body = "none"
	if err = s.QueueSpawn(child); err != nil {
		t.Fatal(err)
	}
	if err = s.Step(); err != nil {
		t.Fatal(err)
	}
	id, _ := s.Find("root")
	if err = s.QueueDestroy(id); err != nil {
		t.Fatal(err)
	}
	child.ID = "later"
	if err = s.QueueSpawn(child); err != nil {
		t.Fatal(err)
	}
	if err = s.Step(); err == nil {
		t.Fatal("spawn accepted removed parent")
	}
	if _, ok := s.Find("later"); ok {
		t.Fatal("orphan leaked")
	}
}

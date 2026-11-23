package game

import (
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"github.com/mirstar13/go-3d-graphics/internal/project"
	"testing"
)

func TestPickVisualChildAndAuthoritativeBody(t *testing.T) {
	parent := project.NewEntity("parent", "Parent")
	parent.Position = engine.Point{X: 4}
	parent.Rotation.Z = 90
	child := project.NewEntity("child", "Child")
	child.Parent = "parent"
	child.Body = "none"
	child.Position = engine.Point{X: 2}
	d := project.Document{Version: project.Version, Name: "Pick", Entities: []project.Entity{parent, child}}
	s, err := NewSession(d, physics.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	ray := physics.Ray{Origin: engine.Point{X: 4, Y: 2, Z: -5}, Direction: engine.Point{Z: 1}, MaxDistance: 10}
	hit, ok, err := s.Pick(ray)
	if err != nil || !ok || hit.FileID != "child" || !hit.VisualProxy {
		t.Fatal(hit, ok, err)
	}
	parentID, _ := s.Find("parent")
	entry, _ := s.lookup(parentID)
	// Physics is authoritative even before the node's next visual synchronization.
	if err = s.World.Teleport(entry.body, engine.Point{X: 4, Y: 2, Z: -2}, engine.IdentityQuaternion()); err != nil {
		t.Fatal(err)
	}
	hit, ok, err = s.Pick(ray)
	if err != nil || !ok || hit.FileID != "parent" || hit.VisualProxy {
		t.Fatal("physics picking used stale node", hit, err)
	}
	before := s.World.Stats()
	s.World.SetPaused(true)
	s.QueueDestroy(parentID)
	s.Step()
	if _, err = s.State(hit.Entity); err == nil {
		t.Fatal("stale picked identity survived")
	}
	if _, ok, err = s.Pick(ray); err != nil || ok {
		t.Fatal("removed entity pick", ok, err)
	}
	if before.Tick != 0 {
		t.Fatal("pick advanced simulation")
	}
}

func TestPickDoesNotReturnAnUnownedHandle(t *testing.T) {
	s, err := NewSession(project.Document{Version: project.Version, Name: "Empty"}, physics.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.World.AddBody(physics.NewBodyDesc(physics.Static, physics.Sphere(1))); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := s.Pick(physics.Ray{Origin: engine.Point{Z: -3}, Direction: engine.Point{Z: 1}, MaxDistance: 10}); err != nil || ok {
		t.Fatal("unowned body mapped to an entity", ok, err)
	}
}

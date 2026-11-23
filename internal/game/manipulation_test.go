package game

import (
	"math"
	"reflect"
	"testing"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"github.com/mirstar13/go-3d-graphics/internal/project"
)

func TestPickedAnchorFollowsBodyAtExecution(t *testing.T) {
	d := project.Document{Version: project.Version, Name: "Anchor", Entities: []project.Entity{project.NewEntity("box", "Box")}}
	d.Entities[0].Position = engine.Point{}
	opts := physics.DefaultOptions()
	opts.Gravity = engine.Point{}
	s, err := NewSession(d, opts)
	if err != nil {
		t.Fatal(err)
	}
	s.World.SetPaused(true)
	id, _ := s.Find("box")
	anchor, err := s.AnchorAt(id, engine.Point{X: .5})
	if err != nil {
		t.Fatal(err)
	}
	entry, _ := s.lookup(id)
	if err = s.World.Teleport(entry.body, engine.Point{X: 10}, engine.QuaternionFromEuler(0, 0, math.Pi/2)); err != nil {
		t.Fatal(err)
	}
	before, _ := s.World.Body(entry.body)
	if err = s.QueueImpulseAtAnchor(anchor, engine.Point{X: 1}); err != nil {
		t.Fatal(err)
	}
	after, _ := s.World.Body(entry.body)
	if !reflect.DeepEqual(before, after) || s.PendingCommands() != 1 {
		t.Fatal("queue changed physics")
	}
	if err = s.Step(); err != nil {
		t.Fatal(err)
	}
	inspected, err := s.Inspect(id)
	if err != nil {
		t.Fatal(err)
	}
	if inspected.Body.AngularVelocity.Z >= -.1 || inspected.Body.Velocity.X <= 0 || inspected.Spawned {
		t.Fatal("local anchor angular response", inspected)
	}
	if err = s.QueueDestroy(id); err != nil {
		t.Fatal(err)
	}
	if err = s.Step(); err != nil {
		t.Fatal(err)
	}
	if err = s.QueueImpulseAtAnchor(anchor, engine.Point{Y: 1}); err == nil {
		t.Fatal("stale anchor accepted")
	}
}
func TestCheckedSpawnsRevalidateAtTickAndProtectAuthored(t *testing.T) {
	doc := project.NewDocument()
	s, err := NewSession(doc, physics.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	s.World.SetPaused(true)
	box, _ := s.Find("box")
	if s.QueueDeleteSpawned(box) == nil {
		t.Fatal("authored deletion accepted")
	}
	d := project.NewEntity("runtime", "Box") // same label as authored, independent ownership
	d.Position = engine.Point{X: 4, Y: 5}
	if err = s.QueueCheckedSpawn(d, 4); err != nil {
		t.Fatal(err)
	}
	second := d
	second.ID = "other"
	if err = s.QueueCheckedSpawn(second, 4); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Find(d.ID); ok {
		t.Fatal("spawn activated while paused")
	}
	if err = s.Step(); err == nil {
		t.Fatal("competing overlap was not rejected")
	}
	id, ok := s.Find(d.ID)
	if !ok {
		t.Fatal("first spawn missing")
	}
	if _, ok = s.Find(second.ID); ok {
		t.Fatal("overlapping queued spawn activated")
	}
	info, err := s.Inspect(id)
	if err != nil || !info.Spawned || info.Body == nil {
		t.Fatal("spawn ownership", info, err)
	}
	if err = s.QueueDeleteSpawned(id); err != nil {
		t.Fatal(err)
	}
	if err = s.Step(); err != nil {
		t.Fatal(err)
	}
	if _, err = s.State(id); err == nil {
		t.Fatal("deleted handle survived")
	}
	// Queue two placements with a budget that permits only the first.
	d.ID = "one"
	second.ID = "two"
	second.Position.X = -4
	if err = s.QueueCheckedSpawn(d, 3); err != nil {
		t.Fatal(err)
	}
	if err = s.QueueCheckedSpawn(second, 3); err != nil {
		t.Fatal(err)
	}
	if err = s.Step(); err == nil {
		t.Fatal("budget race not caught")
	}
	if len(s.Hierarchy()) != 3 {
		t.Fatal("budget exceeded")
	}
	if !reflect.DeepEqual(doc, project.NewDocument()) {
		t.Fatal("authoring mutated")
	}
}
func TestSpawnPreflightAndInspection(t *testing.T) {
	s, err := NewSession(project.NewDocument(), physics.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	d := project.NewEntity("candidate", "Candidate")
	d.Position = engine.Point{} // overlaps floor
	if s.CheckSpawn(d, 128) == nil {
		t.Fatal("overlap allowed")
	}
	d.Position = engine.Point{X: 4, Y: 5}
	if s.CheckSpawn(d, 2) == nil || s.CheckSpawn(d, 129) == nil {
		t.Fatal("invalid budget")
	}
	d.Mass = math.NaN()
	if s.CheckSpawn(d, 128) == nil {
		t.Fatal("invalid spawn accepted")
	}
	floor, _ := s.Find("floor")
	if _, err = s.AnchorAt(floor, engine.Point{}); err == nil {
		t.Fatal("static anchor accepted")
	}
	body, _ := s.Find("box")
	info, err := s.Inspect(body)
	if err != nil {
		t.Fatal(err)
	}
	info.Body.Mass = 999
	again, err := s.Inspect(body)
	if err != nil || again.Body.Mass == 999 {
		t.Fatal("inspection aliases state")
	}
}

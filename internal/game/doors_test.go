package game

import (
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"github.com/mirstar13/go-3d-graphics/internal/project"
	"testing"
)

func TestDoorCombinesPlatesAndReleasesDeletedOccupants(t *testing.T) {
	d := project.NewDocument()
	plate := d.Entities[0]
	plate.Size.X = 2
	plate.Sensor = true
	plate.PlateSignal = "gate"
	d.Entities[0] = plate
	d.Entities[1].Position.Y = .25
	plate.ID = "plate2"
	plate.Position.X = 10
	d.Entities = append(d.Entities, plate)
	door := project.NewEntity("gate", "Gate")
	door.Body = "static"
	door.DoorSignal = "gate"
	door.Position.X = 20
	d.Entities = append(d.Entities, door)
	options := physics.DefaultOptions()
	options.Gravity = engine.Point{}
	s, err := NewSession(d, options)
	if err != nil {
		t.Fatal(err)
	}
	extra := project.NewEntity("visitor", "Visitor")
	extra.Position = engine.Point{X: 10, Y: .25}
	if err = s.QueueSpawn(extra); err != nil {
		t.Fatal(err)
	}
	id, _ := s.Find("gate")
	check := func(y float64) {
		t.Helper()
		if err := s.Advance(2 * options.FixedStep); err != nil {
			t.Fatal(err)
		}
		info, err := s.Inspect(id)
		if err != nil || info.Body.Position.Y != y {
			t.Fatalf("gate at %v, want %v (%v)", info.Body, y, err)
		}
	}
	check(door.Position.Y + 1)
	for _, name := range []string{"box", "visitor"} {
		occupant, ok := s.Find(name)
		if !ok {
			t.Fatal(name)
		}
		if err = s.QueueDestroy(occupant); err != nil {
			t.Fatal(err)
		}
		if name == "box" {
			check(door.Position.Y + 1)
		} else {
			check(door.Position.Y)
		}
	}
}

package game

import (
	"testing"

	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"github.com/mirstar13/go-3d-graphics/internal/project"
)

func TestSceneLightingAndGravityInheritance(t *testing.T) {
	d := project.NewDocument()
	options := physics.DefaultOptions()
	options.Gravity.X = 3
	zero := 0.
	d.Simulation = &project.SimulationSettings{GravityY: &zero}
	d.Lighting = &project.LightingSettings{}
	s, err := NewSession(d, options)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Lighting.Lights) != 0 {
		t.Fatal("empty rig gained a light")
	}
	if err = s.Advance(options.FixedStep); err != nil {
		t.Fatal(err)
	}
	box, _ := s.Find("box")
	state, err := s.State(box)
	if err != nil || state.Position.Y != 3 || state.Position.X <= 0 {
		t.Fatal("gravity override and inheritance", state.Position, err)
	}
	d.Lighting = nil
	s, err = NewSession(d, options)
	if err != nil || len(s.Lighting.Lights) != 1 {
		t.Fatal("legacy light", err)
	}
}

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

func TestAuthoredCollisionMaskFiltersContact(t *testing.T) {
	for _, tc := range []struct {
		mask    string
		contact bool
	}{{"0x80000000", true}, {"1", false}, {"0", false}} {
		t.Run(tc.mask, func(t *testing.T) {
			ed, err := project.NewEditor(project.NewDocument())
			if err != nil {
				t.Fatal(err)
			}
			for _, setting := range [][3]string{{"floor", "collision_layer", "0x80000000"}, {"box", "collision_mask", tc.mask}, {"box", "position.y", "0.25"}} {
				if err := ed.Set(setting[0], setting[1], setting[2]); err != nil {
					t.Fatal(err)
				}
			}
			options := physics.DefaultOptions()
			s, err := NewSession(ed.Document(), options)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Advance(options.FixedStep); err != nil {
				t.Fatal(err)
			}
			if got := len(s.World.Contacts()) > 0; got != tc.contact {
				t.Fatalf("contact = %v, want %v", got, tc.contact)
			}
		})
	}
}

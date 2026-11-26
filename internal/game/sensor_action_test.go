package game

import (
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"github.com/mirstar13/go-3d-graphics/internal/project"
	"testing"
)

func TestSensorEnterAction(t *testing.T) {
	for _, tc := range []struct {
		name            string
		sensor, batched bool
		want            float64
	}{{"single ticks", true, false, 3}, {"catch up", true, true, 3}, {"ordinary collider", false, true, 0}} {
		t.Run(tc.name, func(t *testing.T) {
			d := project.NewDocument()
			pad := &d.Entities[0]
			pad.Sensor = tc.sensor
			pad.EnterImpulse = engine.Point{X: 6}
			box := &d.Entities[1]
			box.Position.Y = .49
			box.Mass = 2
			opts := physics.DefaultOptions()
			opts.Gravity = engine.Point{}
			ed, err := project.NewEditor(d)
			if err != nil {
				t.Fatal(err)
			}
			if err := ed.Set("box", "linear_damping", "0"); err != nil {
				t.Fatal(err)
			}
			s, err := NewSession(ed.Document(), opts)
			if err != nil {
				t.Fatal(err)
			}
			id, _ := s.Find("box")
			if tc.batched {
				err = s.Advance(3 * opts.FixedStep)
			} else {
				s.World.SetPaused(true)
				if err = s.Step(); err != nil {
					t.Fatal(err)
				}
				first, err := s.Inspect(id)
				if err != nil || first.Body.Velocity.X != 0 {
					t.Fatal("impulse fired before next tick", err)
				}
				if err = s.Step(); err != nil {
					t.Fatal(err)
				}
				err = s.Step()
				if err != nil {
					t.Fatal(err)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			state, err := s.Inspect(id)
			if err != nil {
				t.Fatal(err)
			}
			if state.Body.Velocity.X != tc.want {
				t.Fatalf("velocity X = %v, want %v", state.Body.Velocity.X, tc.want)
			}
		})
	}
}

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

func TestOneShotSensorRecipientsAndRestart(t *testing.T) {
	for _, once := range []bool{false, true} {
		d := project.NewDocument()
		d.Entities[0].Sensor = true
		d.Entities[0].EnterOnce = once
		d.Entities[0].EnterImpulse = engine.Point{Y: 3}
		d.Entities[1].Position = engine.Point{X: -2, Y: .25}
		second := d.Entities[1]
		second.ID = "second"
		second.Position.X = 2
		d.Entities = append(d.Entities, second)
		opts := physics.DefaultOptions()
		opts.Gravity = engine.Point{}
		for run := 0; run < 2; run++ {
			s, err := NewSession(d, opts)
			if err != nil {
				t.Fatal(err)
			}
			if err = s.Advance(2 * opts.FixedStep); err != nil {
				t.Fatal(err)
			}
			recipients := 0
			for _, name := range []string{"box", "second"} {
				id, _ := s.Find(name)
				state, err := s.Inspect(id)
				if err != nil {
					t.Fatal(err)
				}
				if state.Body.Velocity.Y > 0 {
					recipients++
				}
			}
			want := 2
			if once {
				want = 1
			}
			if recipients != want {
				t.Fatalf("once=%v run=%d got %d recipients, want %d", once, run, recipients, want)
			}
		}
	}
}

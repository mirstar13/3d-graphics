package physics

import (
	"fmt"
	"math"
	"testing"
)

// These are metre-scale regression fixtures, not a guarantee for arbitrary stacks.
func TestStackStabilityEnvelope(t *testing.T) {
	for _, ratio := range []float64{1, 10} {
		t.Run(fmt.Sprint(ratio), func(t *testing.T) {
			w := newTestWorld(t)
			w.options.Sleeping = false
			floor := NewBodyDesc(Static, Plane(vec{Y: 1}, 0))
			floor.Collider.Restitution = 0
			addTestBody(t, w, floor)
			var ids []BodyID
			for i := 0; i < 5; i++ {
				d := NewBodyDesc(Dynamic, Box(vec{X: .5, Y: .5, Z: .5}))
				d.Position.Y = .5 + float64(i)
				d.Collider.Restitution = 0
				if i == 0 {
					d.Mass = ratio
				}
				ids = append(ids, addTestBody(t, w, d))
			}
			var penetration, linear, angular float64
			for tick := 0; tick < 1800; tick++ {
				stepN(t, w, 1)
				if tick < 1200 {
					continue
				}
				for _, c := range w.Contacts() {
					penetration = math.Max(penetration, -c.Separation)
				}
				for i, id := range ids {
					s := stateOf(t, w, id)
					if !finitePoint(s.Position) || math.Abs(s.Position.Y-(.5+float64(i))) > .15 {
						t.Fatalf("stack escaped at tick %d: %+v", tick, s.Position)
					}
					linear = math.Max(linear, length(s.Velocity))
					angular = math.Max(angular, length(s.AngularVelocity))
				}
			}
			t.Logf("max penetration %.6fm linear %.6fm/s angular %.6frad/s", penetration, linear, angular)
			if penetration > .04 || linear > .1 || angular > .1 {
				t.Fatal("resting envelope exceeded")
			}
		})
	}
}

func TestOverlapRecoveryDoesNotInjectVelocity(t *testing.T) {
	w := noGravity(t)
	d := NewBodyDesc(Dynamic, Sphere(1))
	d.Collider.Restitution = 0
	a := addTestBody(t, w, d)
	d.Position.X = .5
	b := addTestBody(t, w, d)
	stepN(t, w, 60)
	sa, sb := stateOf(t, w, a), stateOf(t, w, b)
	if length(sub(sa.Position, sb.Position)) < 1.99 || length(sa.Velocity) > 1e-9 || length(sb.Velocity) > 1e-9 {
		t.Fatal(sa, sb)
	}
}

func TestKinematicCrushingRemainsFinite(t *testing.T) {
	w := newTestWorld(t)
	addTestBody(t, w, NewBodyDesc(Static, Plane(vec{Y: 1}, 0)))
	d := NewBodyDesc(Dynamic, Sphere(.5))
	d.Position.Y = .5
	id := addTestBody(t, w, d)
	p := NewBodyDesc(Kinematic, Box(vec{X: 2, Y: .25, Z: 2}))
	p.Position.Y = 2
	pid := addTestBody(t, w, p)
	for i := 0; i < 180; i++ {
		if err := w.SetKinematicTarget(pid, vec{Y: math.Max(.65, 2-float64(i)*.01)}, p.Rotation); err != nil {
			t.Fatal(err)
		}
		stepN(t, w, 1)
		s := stateOf(t, w, id)
		// An impossible gap cannot satisfy both contacts; require bounded recovery.
		if !finitePoint(s.Position) || !finitePoint(s.Velocity) || length(s.Position) > 5 || length(s.Velocity) > 10 {
			t.Fatal(s)
		}
	}
}

func TestDetectionWorkCountsAndSnapshotOwnership(t *testing.T) {
	w := noGravity(t)
	w.options.BroadPhase = BruteForce
	for i := 0; i < 4; i++ {
		d := NewBodyDesc(Dynamic, Sphere(.5))
		d.Position.X = float64(i) * 3
		addTestBody(t, w, d)
	}
	stepN(t, w, 1)
	s := w.SolverStats()
	if s.PairTests != 6*s.DetectionPasses || s.DetectionPasses != 2 || s.CandidatePairs != 0 {
		t.Fatal(s)
	}
	snapshot := w.Bodies()
	first := snapshot[0]
	w.AddImpulse(first.ID, vec{Y: 1})
	stepN(t, w, 1)
	if snapshot[0] != first {
		t.Fatal("snapshot aliased world state")
	}
}

func BenchmarkWorldLayouts(b *testing.B) {
	for _, layout := range []string{"sparse", "dense", "settled"} {
		for _, count := range []int{16, 64, 128} {
			b.Run(fmt.Sprintf("%s/%d", layout, count), func(b *testing.B) {
				w, _ := NewWorld(DefaultOptions())
				floor := NewBodyDesc(Static, Plane(vec{Y: 1}, 0))
				floor.Collider.Restitution = 0
				w.AddBody(floor)
				for i := 0; i < count; i++ {
					d := NewBodyDesc(Dynamic, Sphere(.5))
					d.Collider.Restitution = 0
					spacing := 3.0
					if layout == "dense" {
						spacing = .9
					}
					d.Position = vec{X: float64(i%16) * spacing, Y: .5, Z: float64(i/16) * spacing}
					if layout == "sparse" {
						d.Position.Y = 1000000
					}
					w.AddBody(d)
				}
				if layout == "settled" {
					for i := 0; i < 180; i++ {
						w.Advance(w.options.FixedStep, nil)
					}
				}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					w.Advance(w.options.FixedStep, nil)
				}
				b.StopTimer()
				s := w.SolverStats()
				b.ReportMetric(float64(s.PairTests), "pair-tests/tick")
				b.ReportMetric(float64(s.Contacts), "contacts/tick")
			})
		}
	}
}

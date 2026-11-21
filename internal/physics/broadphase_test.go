package physics

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

func TestSweepAndPruneMatchesOracle(t *testing.T) {
	rng := rand.New(rand.NewSource(71))
	w := noGravity(t)
	addTestBody(t, w, NewBodyDesc(Static, Plane(vec{Y: 1}, 0)))
	for i := 0; i < 48; i++ {
		shape := Sphere(.5 + rng.Float64())
		if i%2 == 0 {
			shape = Box(vec{X: .1 + rng.Float64(), Y: .1 + rng.Float64(), Z: .1 + rng.Float64()})
		}
		d := NewBodyDesc(Dynamic, shape)
		d.Position = vec{X: rng.Float64() * 10, Y: rng.Float64() * 4, Z: rng.Float64() * 4}
		d.Rotation = engine.QuaternionFromEuler(rng.Float64(), rng.Float64(), rng.Float64())
		d.Disabled = i%11 == 0
		if i%7 == 0 {
			d.Collider.Mask = 0
		}
		d.Collider.Sensor = i%5 == 0
		addTestBody(t, w, d)
	}
	for pass := 0; pass < 50; pass++ {
		w.options.BroadPhase = SweepAndPrune
		a := append([]bodyPair(nil), w.candidates()...)
		w.options.BroadPhase = BruteForce
		b := append([]bodyPair(nil), w.candidates()...)
		if !reflect.DeepEqual(a, b) {
			t.Fatalf("pair mismatch pass %d", pass)
		}
		states := w.Bodies()
		id := states[1+pass%(len(states)-1)].ID
		if pass%3 == 0 {
			w.RemoveBody(id)
			d := NewBodyDesc(Dynamic, Sphere(.7))
			d.Position.Y = 2
			addTestBody(t, w, d)
		} else {
			w.Teleport(id, vec{X: rng.Float64() * 10, Y: rng.Float64() * 4}, engine.QuaternionFromEuler(0, rng.Float64(), 0))
		}
	}
}
func TestBroadPhaseDoesNotChangeSimulation(t *testing.T) {
	a, b := newTestWorld(t), newTestWorld(t)
	b.options.BroadPhase = BruteForce
	for _, w := range []*World{a, b} {
		addTestBody(t, w, NewBodyDesc(Static, Plane(vec{Y: 1}, 0)))
		for i := 0; i < 12; i++ {
			d := NewBodyDesc(Dynamic, Sphere(.4))
			d.Position = vec{X: float64(i%4) * .9, Y: 2 + float64(i/4)}
			d.CCD = true
			addTestBody(t, w, d)
		}
	}
	for tick := 0; tick < 180; tick++ {
		stepN(t, a, 1)
		stepN(t, b, 1)
		if !reflect.DeepEqual(a.Bodies(), b.Bodies()) || !reflect.DeepEqual(a.Contacts(), b.Contacts()) || !reflect.DeepEqual(a.ContactEvents(), b.ContactEvents()) {
			t.Fatalf("different tick %d", tick)
		}
	}
	saved := a.Contacts()
	copySaved := append([]Contact(nil), saved...)
	stepN(t, a, 10)
	if !reflect.DeepEqual(saved, copySaved) {
		t.Fatal("published contacts alias scratch")
	}
}

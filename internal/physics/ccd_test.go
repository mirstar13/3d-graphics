package physics

import (
	"math"
	"testing"
)

func TestCCDThinObstacles(t *testing.T) {
	mesh, err := NewTriangleMesh([]vec{{X: -5, Z: -5}, {X: 5, Z: -5}, {X: 5, Z: 5}, {X: -5, Z: 5}}, []int{0, 2, 1, 0, 3, 2})
	if err != nil {
		t.Fatal(err)
	}
	for _, shape := range []Shape{Plane(vec{Y: 1}, 0), Box(vec{X: 5, Y: .01, Z: 5}), Mesh(mesh)} {
		w := noGravity(t)
		floor := NewBodyDesc(Static, shape)
		floor.Collider.Restitution = 0
		addTestBody(t, w, floor)
		d := NewBodyDesc(Dynamic, Sphere(.2))
		d.CCD = true
		d.Position.Y = 3
		d.Velocity.Y = -600
		d.LinearDamping = 0
		d.Collider.Restitution = 0
		id := addTestBody(t, w, d)
		before, after := 0, 0
		w.AdvanceWithHooks(w.options.FixedStep, StepHooks{BeforeStep: func(Tick) { before++ }, AfterStep: func(Tick) { after++ }})
		s := stateOf(t, w, id)
		if s.Position.Y < .19 || s.Position.Y > .22 || math.Abs(s.Velocity.Y) > 1e-6 || before != 1 || after != 1 || w.Stats().Tick != 1 || w.SolverStats().CCDImpacts == 0 {
			t.Fatal(shape.kind, s, w.SolverStats())
		}
	}
}
func TestCCDBounceAndTransientEvents(t *testing.T) {
	w := noGravity(t)
	f := NewBodyDesc(Static, Plane(vec{Y: 1}, 0))
	f.Collider.Restitution = 1
	addTestBody(t, w, f)
	d := NewBodyDesc(Dynamic, Sphere(.2))
	d.CCD = true
	d.Position.Y = 1
	d.Velocity.Y = -120
	d.LinearDamping = 0
	d.Collider.Restitution = 1
	id := addTestBody(t, w, d)
	stepN(t, w, 1)
	s := stateOf(t, w, id)
	if s.Position.Y < 1.3 || math.Abs(s.Velocity.Y-120) > 1e-6 {
		t.Fatal(s)
	}
	events := w.ContactEvents()
	if len(events) != 2 || events[0].Phase != ContactBegin || events[1].Phase != ContactEnd {
		t.Fatal(events)
	}
}
func TestCCDCornersAndBudget(t *testing.T) {
	a, b := collisionBodies(Sphere(.2), Box(vec{X: 1, Y: 1, Z: 1}), vec{})
	a.state.Position = vec{X: -3, Y: 1.19, Z: 1.19}
	a.state.Velocity = vec{X: 600}
	if _, hit := sphereSweep(a, b, 1.0/60); hit {
		t.Fatal("expanded box corner produced false hit")
	}
	a.state.Position = vec{X: -3, Y: 1.1, Z: 1.1}
	if _, hit := sphereSweep(a, b, 1.0/60); !hit {
		t.Fatal("rounded corner missed")
	}
	w := noGravity(t)
	w.options.CCDMaxImpacts = 1
	for _, y := range []float64{0, 1} {
		normal := vec{Y: 1}
		offset := 0.0
		if y == 1 {
			normal.Y = -1
			offset = -1
		}
		f := NewBodyDesc(Static, Plane(normal, offset))
		f.Collider.Restitution = 1
		addTestBody(t, w, f)
	}
	d := NewBodyDesc(Dynamic, Sphere(.1))
	d.CCD = true
	d.Position.Y = .5
	d.Velocity.Y = -600
	d.Collider.Restitution = 1
	d.LinearDamping = 0
	id := addTestBody(t, w, d)
	stepN(t, w, 1)
	s := stateOf(t, w, id)
	if s.Position.Y < .09 || s.Position.Y > .91 || w.SolverStats().CCDClamps != 1 {
		t.Fatal(s, w.SolverStats())
	}
}
func TestCCDForceLifetimeAndValidation(t *testing.T) {
	w := noGravity(t)
	d := NewBodyDesc(Dynamic, Box(vec{X: 1, Y: 1, Z: 1}))
	d.CCD = true
	if _, err := w.AddBody(d); err == nil {
		t.Fatal("unsupported box CCD accepted")
	}
	d = NewBodyDesc(Dynamic, Sphere(.2))
	d.CCD = true
	d.LinearDamping = 0
	d.Position.Y = 1
	d.Velocity.Y = -120
	id := addTestBody(t, w, d)
	floor := NewBodyDesc(Static, Plane(vec{Y: 1}, 0))
	floor.Collider.Friction = 0
	addTestBody(t, w, floor)
	w.AddForce(id, vec{X: 60})
	stepN(t, w, 1)
	s := stateOf(t, w, id)
	if math.Abs(s.Velocity.X-60*w.options.FixedStep.Seconds()) > 1e-8 {
		t.Fatal("force applied per impact", s.Velocity)
	}
	w.SetPaused(true)
	tick := w.Stats().Tick
	w.StepOnce(nil)
	if w.Stats().Tick != tick+1 {
		t.Fatal("CCD changed single-step count")
	}
}

func TestCCDMeshCanLeaveFloorAndHitAnotherTriangle(t *testing.T) {
	// One mesh contains a floor and a vertical wall. A separating floor contact
	// must not suppress the wall sweep for the entire mesh.
	mesh, err := NewTriangleMesh([]vec{{X: -5, Z: -5}, {X: 5, Z: -5}, {X: 5, Z: 5}, {X: 2, Y: 0, Z: -5}, {X: 2, Y: 5, Z: -5}, {X: 2, Y: 0, Z: 5}, {X: 2, Y: 5, Z: 5}}, []int{0, 2, 1, 3, 4, 5, 4, 6, 5})
	if err != nil {
		t.Fatal(err)
	}
	w := noGravity(t)
	f := NewBodyDesc(Static, Mesh(mesh))
	f.Collider.Restitution = 0
	addTestBody(t, w, f)
	d := NewBodyDesc(Dynamic, Sphere(.2))
	d.CCD = true
	d.Position = vec{Y: .2025}
	d.Velocity = vec{X: 300, Y: 10}
	d.LinearDamping = 0
	d.Collider.Restitution = 0
	id := addTestBody(t, w, d)
	stepN(t, w, 1)
	if s := stateOf(t, w, id); s.Position.X > 1.801 {
		t.Fatal("mesh wall crossed", s)
	}
}

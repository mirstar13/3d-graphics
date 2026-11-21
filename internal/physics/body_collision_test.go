package physics

import (
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"math"
	"testing"
)

func addTestBody(t *testing.T, w *World, desc BodyDesc) BodyID {
	t.Helper()
	id, err := w.AddBody(desc)
	if err != nil {
		t.Fatal(err)
	}
	return id
}
func stateOf(t *testing.T, w *World, id BodyID) BodyState {
	t.Helper()
	s, err := w.Body(id)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func stepN(t *testing.T, w *World, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		if err := w.Advance(w.options.FixedStep, nil); err != nil {
			t.Fatal(err)
		}
	}
}
func noGravity(t *testing.T) *World {
	t.Helper()
	o := DefaultOptions()
	o.Gravity = vec{}
	w, err := NewWorld(o)
	if err != nil {
		t.Fatal(err)
	}
	return w
}
func TestBodyLifecycleAndDeferredChanges(t *testing.T) {
	w := noGravity(t)
	desc := NewBodyDesc(Dynamic, Sphere(1))
	id := addTestBody(t, w, desc)
	w.RemoveBody(id)
	replacement := addTestBody(t, w, desc)
	if _, err := w.Body(id); err == nil || replacement == id {
		t.Fatal("stale handle resurrected")
	}
	w.Reset()
	if _, err := w.Body(replacement); err == nil {
		t.Fatal("reset retained handle")
	}
	id = addTestBody(t, w, desc)
	var created BodyID
	w.AdvanceWithHooks(w.options.FixedStep, StepHooks{AfterStep: func(Tick) { w.RemoveBody(id); created = addTestBody(t, w, desc) }})
	if len(w.Bodies()) != 1 {
		t.Fatal("lifecycle mutation applied during hook")
	}
	stepN(t, w, 1)
	if _, err := w.Body(id); err == nil {
		t.Fatal("queued removal not applied")
	}
	if len(w.Bodies()) != 1 || w.Bodies()[0].ID != created {
		t.Fatal(w.Bodies())
	}
}
func TestForcesInertiaAndKinematicTargets(t *testing.T) {
	w := noGravity(t)
	desc := NewBodyDesc(Dynamic, Box(vec{X: 1, Y: 2, Z: 3}))
	desc.Mass = 6
	desc.LinearDamping = 0
	desc.AngularDamping = 0
	id := addTestBody(t, w, desc)
	s := stateOf(t, w, id)
	if s.Inertia != (vec{X: 26, Y: 20, Z: 10}) {
		t.Fatal(s.Inertia)
	}
	w.AddForce(id, vec{X: 60})
	stepN(t, w, 1)
	velocity := stateOf(t, w, id).Velocity
	stepN(t, w, 1)
	if stateOf(t, w, id).Velocity != velocity {
		t.Fatal("force applied twice")
	}
	w.AddImpulseAtPoint(id, vec{X: 6}, vec{Y: 1})
	stepN(t, w, 1)
	s = stateOf(t, w, id)
	if s.AngularVelocity.Z >= 0 || math.Abs(s.Velocity.X-velocity.X-1) > 1e-8 {
		t.Fatal(s)
	}
	kin := addTestBody(t, w, NewBodyDesc(Kinematic, Sphere(.5)))
	w.SetKinematicTarget(kin, vec{X: 3}, engine.QuaternionFromEuler(0, .2, 0))
	stepN(t, w, 1)
	if stateOf(t, w, kin).Position.X != 3 || stateOf(t, w, kin).Velocity.X <= 0 {
		t.Fatal("kinematic target missed")
	}
	stepN(t, w, 1)
	if stateOf(t, w, kin).Velocity != (vec{}) {
		t.Fatal("missing target retained velocity")
	}
	w.Teleport(kin, vec{X: 8}, engine.IdentityQuaternion())
	if stateOf(t, w, kin).Velocity != (vec{}) {
		t.Fatal("teleport generated velocity")
	}
}
func TestInvalidBodyDescriptors(t *testing.T) {
	w := noGravity(t)
	cases := []BodyDesc{NewBodyDesc(Dynamic, Sphere(-1)), NewBodyDesc(Dynamic, Box(vec{X: 1, Y: 0, Z: 1})), NewBodyDesc(Dynamic, Plane(vec{Y: 1}, 0))}
	d := NewBodyDesc(Dynamic, Sphere(1))
	d.Mass = 0
	cases = append(cases, d)
	d = NewBodyDesc(Dynamic, Sphere(1))
	d.Rotation = quat{}
	cases = append(cases, d)
	for _, desc := range cases {
		if _, err := w.AddBody(desc); err == nil {
			t.Fatal("invalid descriptor accepted")
		}
	}
}
func collisionBodies(a, b Shape, p vec) (*body, *body) {
	da, db := NewBodyDesc(Dynamic, a), NewBodyDesc(Static, b)
	da.Position = p
	return &body{state: BodyState{BodyDesc: da}}, &body{state: BodyState{BodyDesc: db}}
}
func TestPrimitiveContactPairs(t *testing.T) {
	cases := []struct {
		name     string
		a, b     Shape
		p        vec
		rotation quat
	}{
		{"spheres", Sphere(1), Sphere(1), vec{X: 1.5}, engine.IdentityQuaternion()},
		{"coincident", Sphere(1), Sphere(1), vec{}, engine.IdentityQuaternion()},
		{"sphere-plane", Sphere(1), Plane(vec{Y: 1}, 0), vec{Y: .8}, engine.IdentityQuaternion()},
		{"sphere-box", Sphere(1), Box(vec{X: 1, Y: 1, Z: 1}), vec{X: 1.5}, engine.IdentityQuaternion()},
		{"inside-box", Sphere(.2), Box(vec{X: 1, Y: 1, Z: 1}), vec{}, engine.IdentityQuaternion()},
		{"box-plane", Box(vec{X: 1, Y: 1, Z: 1}), Plane(vec{Y: 1}, 0), vec{Y: .8}, engine.IdentityQuaternion()},
		{"boxes", Box(vec{X: 1, Y: 1, Z: 1}), Box(vec{X: 1, Y: 1, Z: 1}), vec{X: 1.5}, engine.IdentityQuaternion()},
		{"rotated-boxes", Box(vec{X: 1, Y: 1, Z: 1}), Box(vec{X: 1, Y: 1, Z: 1}), vec{X: 1.5}, engine.QuaternionFromEuler(.2, .4, .1)},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			a, b := collisionBodies(test.a, test.b, test.p)
			a.state.Rotation = test.rotation
			contacts := collide(a, b)
			if len(contacts) == 0 || len(contacts) > 4 {
				t.Fatal(contacts)
			}
			for _, c := range contacts {
				if !finitePoint(c.Normal) || math.Abs(length(c.Normal)-1) > 1e-8 || c.Separation > contactSlop {
					t.Fatal(c)
				}
			}
			reverse := collide(b, a)
			if len(reverse) == 0 {
				t.Fatal("reverse pair missed")
			}
			// Plane is an infinite half-space, so separate upward rather than sideways.
			a.state.Position = vec{X: 20, Y: 20, Z: 20}
			if len(collide(a, b)) != 0 {
				t.Fatal("separated pair collided")
			}
		})
	}
}
func TestSphereRestitutionAndRest(t *testing.T) {
	w := newTestWorld(t)
	floor := NewBodyDesc(Static, Plane(vec{Y: 1}, 0))
	floor.Collider.Restitution = 0
	addTestBody(t, w, floor)
	d := NewBodyDesc(Dynamic, Sphere(1))
	d.Position.Y = 5
	d.Collider.Restitution = 0
	id := addTestBody(t, w, d)
	stepN(t, w, 600)
	s := stateOf(t, w, id)
	if math.Abs(s.Position.Y-1) > .02 || length(s.Velocity) > .02 {
		t.Fatal("sphere failed to settle", s.Position, s.Velocity)
	}
	w.Teleport(id, vec{Y: 1.1}, engine.IdentityQuaternion())
	d.Collider.Restitution = .8
	w.SetCollider(id, d.Collider)
	w.SetVelocity(id, vec{Y: -5}, vec{})
	bounced := false
	for i := 0; i < 10; i++ {
		stepN(t, w, 1)
		if stateOf(t, w, id).Velocity.Y > 3 {
			bounced = true
		}
	}
	if !bounced {
		t.Fatal("restitution produced no bounce")
	}
}
func TestBoxStackAndFriction(t *testing.T) {
	w := newTestWorld(t)
	floor := NewBodyDesc(Static, Plane(vec{Y: 1}, 0))
	floor.Collider.Restitution = 0
	addTestBody(t, w, floor)
	var ids []BodyID
	for i := 0; i < 3; i++ {
		d := NewBodyDesc(Dynamic, Box(vec{X: 1, Y: 1, Z: 1}))
		d.Position.Y = 1.01 + float64(i)*2.01
		d.Collider.Restitution = 0
		ids = append(ids, addTestBody(t, w, d))
	}
	stepN(t, w, 900)
	for i, id := range ids {
		s := stateOf(t, w, id)
		if math.Abs(s.Position.Y-(1+float64(i)*2)) > .2 || length(s.Velocity) > .1 {
			t.Fatalf("box %d unstable: p=%+v v=%+v", i, s.Position, s.Velocity)
		}
	}
	d := NewBodyDesc(Dynamic, Box(vec{X: .5, Y: .5, Z: .5}))
	d.Position = vec{X: 8, Y: .5}
	d.Velocity = vec{X: 3}
	d.LinearDamping = 0
	d.Collider.Restitution = 0
	id := addTestBody(t, w, d)
	stepN(t, w, 180)
	if math.Abs(stateOf(t, w, id).Velocity.X) > .1 {
		t.Fatal("friction did not stop sliding")
	}
}
func TestSensorFilteringAndRemovalEvent(t *testing.T) {
	w := noGravity(t)
	a := NewBodyDesc(Static, Sphere(2))
	a.Collider.Sensor = true
	id := addTestBody(t, w, a)
	b := NewBodyDesc(Dynamic, Sphere(1))
	b.Collider.Mask = 0
	other := addTestBody(t, w, b)
	stepN(t, w, 1)
	if len(w.Contacts()) != 0 {
		t.Fatal("mask ignored")
	}
	b.Collider.Mask = 1
	w.SetCollider(other, b.Collider)
	stepN(t, w, 1)
	if len(w.ContactEvents()) != 1 || w.ContactEvents()[0].Phase != ContactBegin {
		t.Fatal(w.ContactEvents())
	}
	if stateOf(t, w, other).Position != (vec{}) {
		t.Fatal("sensor moved body")
	}
	w.AdvanceWithHooks(w.options.FixedStep, StepHooks{AfterStep: func(Tick) {
		if err := w.RemoveBody(id); err != nil {
			t.Fatal(err)
		}
	}})
	stepN(t, w, 1)
	if len(w.ContactEvents()) != 1 || w.ContactEvents()[0].Phase != ContactEnd {
		t.Fatal(w.ContactEvents())
	}
}
func TestStaticMeshTerrainAndBVH(t *testing.T) {
	vertices := []vec{{X: -10, Z: -10}, {X: 10, Z: -10}, {X: 10, Z: 10}, {X: -10, Z: 10}}
	mesh, err := NewTriangleMesh(vertices, []int{0, 2, 1, 0, 3, 2})
	if err != nil {
		t.Fatal(err)
	}
	vertices[0].Y = 100
	if mesh.vertices[0].Y != 0 {
		t.Fatal("mesh aliased caller storage")
	}
	for _, shape := range []Shape{Sphere(.5), Box(vec{X: .5, Y: .5, Z: .5})} {
		w := newTestWorld(t)
		terrain := NewBodyDesc(Static, Mesh(mesh))
		terrain.Position.Y = 2
		addTestBody(t, w, terrain)
		d := NewBodyDesc(Dynamic, shape)
		d.Position = vec{X: -2, Y: 5}
		d.Velocity.X = 1
		d.Collider.Friction = 0
		d.Collider.Restitution = 0
		id := addTestBody(t, w, d)
		stepN(t, w, 240)
		s := stateOf(t, w, id)
		if math.Abs(s.Position.Y-2.5) > .05 {
			t.Fatal("fell through terrain", shape.kind, s.Position)
		}
	}
	for _, indices := range [][]int{{0, 1}, {0, 1, 99}, {0, 0, 1}} {
		if _, err := NewTriangleMesh(vertices, indices); err == nil {
			t.Fatal("invalid mesh accepted")
		}
	}
}

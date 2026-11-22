package physics

import (
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"math"
	"testing"
)

func testCompound(t *testing.T) Shape {
	t.Helper()
	s, e := Compound([]CompoundChild{{Shape: Sphere(.5), Position: vec{X: -1}, Rotation: engine.IdentityQuaternion(), Weight: 1}, {Shape: Sphere(.5), Position: vec{X: 1}, Rotation: engine.IdentityQuaternion(), Weight: 1}})
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func TestCompoundInertiaAndOwnership(t *testing.T) {
	s := testCompound(t)
	w := noGravity(t)
	d := NewBodyDesc(Dynamic, s)
	d.Mass = 2
	id := addTestBody(t, w, d)
	state := stateOf(t, w, id)
	if math.Abs(state.Inertia.X-.2) > 1e-10 || math.Abs(state.Inertia.Y-2.2) > 1e-10 || math.Abs(state.Inertia.Z-2.2) > 1e-10 {
		t.Fatal(state.InertiaTensor)
	}
	children := s.Children()
	children[0].Position.X = 100
	if s.Children()[0].Position.X != -1 {
		t.Fatal("mutable child storage exposed")
	}
	w.AddAngularImpulse(id, vec{Y: 2.2})
	stepN(t, w, 1)
	if math.Abs(stateOf(t, w, id).AngularVelocity.Y-math.Exp(-d.AngularDamping*w.options.FixedStep.Seconds())) > 1e-8 {
		t.Fatal("wrong compound angular response")
	}
	rotated, e := Compound([]CompoundChild{{Shape: Box(vec{X: 1, Y: 2, Z: 3}), Rotation: engine.QuaternionFromEuler(0, 0, math.Pi/4), Weight: 1}})
	if e != nil {
		t.Fatal(e)
	}
	tensor := tensorFor(rotated, 1)
	if math.Abs(tensor[0][1]) < .1 {
		t.Fatal("rotated child lost off-diagonal inertia")
	}
	inverse, ok := tensorInverse(tensor)
	v := vec{X: .2, Y: .5, Z: .7}
	if !ok || length(sub(tensorVector(inverse, tensorVector(tensor, v)), v)) > 1e-9 {
		t.Fatal("tensor inverse")
	}
}
func TestCompoundContactsAndMassCentre(t *testing.T) {
	w := newTestWorld(t)
	addTestBody(t, w, NewBodyDesc(Static, Mesh(tiledTerrain(t))))
	d := NewBodyDesc(Dynamic, testCompound(t))
	d.Position.Y = 1
	d.Collider.Restitution = 0
	id := addTestBody(t, w, d)
	stepN(t, w, 180)
	state := stateOf(t, w, id)
	if math.Abs(state.Position.Y-.5) > .03 || len(w.Contacts()) < 2 {
		t.Fatal(state, w.Contacts())
	}
	features := map[uint64]bool{}
	for _, c := range w.Contacts() {
		if features[c.Feature] {
			t.Fatal("child feature collision")
		}
		features[c.Feature] = true
	}
	shape, e := Compound([]CompoundChild{{Shape: Sphere(.5), Position: vec{X: -1}, Rotation: engine.IdentityQuaternion(), Weight: 1}, {Shape: Sphere(.5), Position: vec{X: 1}, Rotation: engine.IdentityQuaternion(), Weight: 3}})
	if e != nil || shape.CenterOfMass().X != .5 {
		t.Fatal(shape.CenterOfMass(), e)
	}
	d.CCD = true
	if _, e := w.AddBody(d); e == nil {
		t.Fatal("unsupported compound CCD accepted")
	}
}
func TestCompoundValidation(t *testing.T) {
	if _, e := Compound(nil); e == nil {
		t.Fatal("empty compound accepted")
	}
	if _, e := Compound([]CompoundChild{{Shape: Plane(vec{Y: 1}, 0), Rotation: engine.IdentityQuaternion(), Weight: 1}}); e == nil {
		t.Fatal("plane compound accepted")
	}
}

func TestCompoundJointAnchorAndInertiaChange(t *testing.T) {
	w := noGravity(t)
	d := NewBodyDesc(Dynamic, testCompound(t))
	d.Position = vec{Y: 2}
	id := addTestBody(t, w, d)
	addTestJoint(t, w, JointDesc{Kind: BallJoint, BodyA: id, AnchorA: vec{X: 1}, AnchorB: vec{X: 1, Y: 2}})
	w.AddImpulseAtPoint(id, vec{Z: .3}, vec{X: -1, Y: 2})
	stepN(t, w, 120)
	for _, row := range w.jointRows(w.options.FixedStep.Seconds(), false) {
		if math.Abs(row.error) > .01 {
			t.Fatal("compound anchor drift", row.error)
		}
	}
	old := stateOf(t, w, id).InertiaTensor
	if err := w.SetMass(id, 2); err != nil {
		t.Fatal(err)
	}
	next := stateOf(t, w, id).InertiaTensor
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if next[i][j] != 2*old[i][j] {
				t.Fatal("mass did not update tensor")
			}
		}
	}
}

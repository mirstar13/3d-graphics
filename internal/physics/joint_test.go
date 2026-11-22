package physics

import (
	"math"
	"testing"
)

func addTestJoint(t *testing.T, w *World, d JointDesc) JointID {
	t.Helper()
	id, e := w.AddJoint(d)
	if e != nil {
		t.Fatal(e)
	}
	return id
}
func TestDistancePendulumAndJointLifecycle(t *testing.T) {
	w := newTestWorld(t)
	w.options.Sleeping = false
	d := NewBodyDesc(Dynamic, Sphere(.2))
	d.Position = vec{X: 1, Y: 1}
	d.LinearDamping = 0
	d.AngularDamping = 0
	id := addTestBody(t, w, d)
	jointID := addTestJoint(t, w, JointDesc{Kind: DistanceJoint, BodyA: id, AnchorB: vec{Y: 2}, Length: math.Sqrt(2)})
	initialEnergy := d.Mass * 9.81 * d.Position.Y
	for i := 0; i < 600; i++ {
		stepN(t, w, 1)
		s := stateOf(t, w, id)
		if math.Abs(length(sub(s.Position, vec{Y: 2}))-math.Sqrt(2)) > .01 {
			t.Fatal("pendulum length drift", s.Position)
		}
		energy := s.Mass*9.81*s.Position.Y + .5*s.Mass*dot(s.Velocity, s.Velocity)
		if energy > initialEnergy+.3 {
			t.Fatal("pendulum gained energy", energy)
		}
	}
	w.RemoveBody(id)
	if _, e := w.Joint(jointID); e == nil {
		t.Fatal("body removal retained joint")
	}
	id = addTestBody(t, w, d)
	next := addTestJoint(t, w, JointDesc{Kind: BallJoint, BodyA: id, AnchorB: d.Position})
	if next == jointID {
		t.Fatal("joint handle resurrected")
	}
	w.Reset()
	if _, e := w.Joint(next); e == nil {
		t.Fatal("reset retained joint")
	}
}
func TestBallJointChainAndWake(t *testing.T) {
	w := newTestWorld(t)
	var ids []BodyID
	for i := 0; i < 4; i++ {
		d := NewBodyDesc(Dynamic, Box(vec{X: .2, Y: .5, Z: .2}))
		d.Position.Y = 3.5 - float64(i)
		d.Collider.Restitution = 0
		id := addTestBody(t, w, d)
		desc := JointDesc{Kind: BallJoint, BodyA: id, AnchorA: vec{Y: .5}, AnchorB: vec{Y: 4}}
		if i > 0 {
			desc.BodyB = ids[i-1]
			desc.AnchorB = vec{Y: -.5}
		}
		addTestJoint(t, w, desc)
		ids = append(ids, id)
	}
	stepN(t, w, 600)
	for i, id := range ids {
		s := stateOf(t, w, id)
		if math.Abs(s.Position.Y-(3.5-float64(i))) > .04 {
			t.Fatal("chain stretched", s)
		}
	}
	for _, id := range ids {
		if !stateOf(t, w, id).Sleeping {
			t.Fatal("resting chain did not sleep")
		}
	}
	w.AddImpulse(ids[3], vec{X: 1})
	for _, id := range ids {
		if stateOf(t, w, id).Sleeping {
			t.Fatal("joint island did not wake")
		}
	}
	stepN(t, w, 120)
	for _, row := range w.jointRows(w.options.FixedStep.Seconds(), false) {
		if math.Abs(row.error) > .02 {
			t.Fatal("anchor error", row.error)
		}
	}
}
func TestHingeMotorTorqueAndLimits(t *testing.T) {
	w := noGravity(t)
	w.options.Sleeping = false
	d := NewBodyDesc(Dynamic, Box(vec{X: 1, Y: .5, Z: .2}))
	d.AngularDamping = 0
	id := addTestBody(t, w, d)
	j := addTestJoint(t, w, JointDesc{Kind: HingeJoint, BodyA: id, AxisA: vec{Y: 1}, AxisB: vec{Y: 1}, Limits: true, LowerAngle: -.4, UpperAngle: .4, Motor: true, MotorSpeed: 2, MaxMotorTorque: 1})
	for i := 0; i < 180; i++ {
		stepN(t, w, 1)
		s, e := w.Joint(j)
		if e != nil {
			t.Fatal(e)
		}
		if math.Abs(s.MotorImpulse) > w.options.FixedStep.Seconds()+1e-9 {
			t.Fatal("motor exceeded torque budget", s.MotorImpulse)
		}
		b, _ := w.lookup(id)
		axisY := b.state.Rotation.RotateVector(vec{Y: 1})
		if axisY.Y < .999 {
			t.Fatal("hinge axis drift")
		}
		angle := 2 * math.Atan2(b.state.Rotation.Y, b.state.Rotation.W)
		if math.Abs(angle) > .42 {
			t.Fatal("hinge limit exceeded", angle)
		}
	}
	if err := w.SetJointMotor(j, false, 0, 0); err != nil {
		t.Fatal(err)
	}
}
func TestJointFilteringAndDeferredRemoval(t *testing.T) {
	w := noGravity(t)
	d := NewBodyDesc(Dynamic, Sphere(1))
	a := addTestBody(t, w, d)
	d.Position.X = 1
	b := addTestBody(t, w, d)
	j := addTestJoint(t, w, JointDesc{Kind: DistanceJoint, BodyA: a, BodyB: b, Length: 1})
	stepN(t, w, 1)
	if len(w.Contacts()) != 0 {
		t.Fatal("connected collision not filtered")
	}
	w.AdvanceWithHooks(w.options.FixedStep, StepHooks{AfterStep: func(Tick) { w.RemoveJoint(j) }})
	if _, e := w.Joint(j); e != nil {
		t.Fatal("removed inside hook")
	}
	stepN(t, w, 1)
	if _, e := w.Joint(j); e == nil {
		t.Fatal("deferred removal not applied")
	}
	if len(w.Contacts()) == 0 {
		t.Fatal("unrelated collision state persisted")
	}
}

func TestJointKinematicAttachment(t *testing.T) {
	w := noGravity(t)
	p := NewBodyDesc(Kinematic, Sphere(.2))
	pid := addTestBody(t, w, p)
	d := NewBodyDesc(Dynamic, Sphere(.2))
	d.Position.X = 1
	id := addTestBody(t, w, d)
	addTestJoint(t, w, JointDesc{Kind: BallJoint, BodyA: id, BodyB: pid, AnchorB: vec{X: 1}})
	stepN(t, w, 60)
	for i := 0; i < 60; i++ {
		w.SetKinematicTarget(pid, vec{Y: float64(i+1) * .01}, p.Rotation)
		stepN(t, w, 1)
	}
	s := stateOf(t, w, id)
	if math.Abs(s.Position.Y-.6) > .02 || math.Abs(s.Position.X-1) > .02 {
		t.Fatal(s)
	}
}

func TestWorldAnchorsDoNotConnectSleepingIslands(t *testing.T) {
	w := noGravity(t)
	d := NewBodyDesc(Dynamic, Sphere(.2))
	a := addTestBody(t, w, d)
	d.Position.X = 10
	b := addTestBody(t, w, d)
	ja := addTestJoint(t, w, JointDesc{Kind: BallJoint, BodyA: a})
	addTestJoint(t, w, JointDesc{Kind: BallJoint, BodyA: b, AnchorB: d.Position})
	stepN(t, w, 120)
	if !stateOf(t, w, a).Sleeping || !stateOf(t, w, b).Sleeping {
		t.Fatal("anchored bodies did not sleep")
	}
	w.RemoveJoint(ja)
	if !stateOf(t, w, b).Sleeping {
		t.Fatal("world anchor bridged independent islands")
	}
}

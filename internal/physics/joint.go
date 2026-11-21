package physics

import (
	"fmt"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"math"
)

// JointKind identifies rigid constraints; springs and breakable joints are separate.
type JointKind uint8

const (
	DistanceJoint JointKind = iota
	BallJoint
	HingeJoint
)

// JointID is a generation-checked world-local handle, independent of BodyID.
type JointID struct {
	slot       uint32
	generation uint64
}

// JointDesc uses body-local anchors and axes. A zero BodyB means a world anchor,
// so AnchorB and AxisB then use world coordinates. Hinge angle zero is its creation pose.
type JointDesc struct {
	Kind                       JointKind
	BodyA, BodyB               BodyID
	AnchorA, AnchorB           vec
	AxisA, AxisB               vec
	Length                     float64
	CollideConnected           bool
	Limits                     bool
	LowerAngle, UpperAngle     float64
	Motor                      bool
	MotorSpeed, MaxMotorTorque float64
}

// JointState is a snapshot. MotorImpulse is the last solve's angular impulse.
type JointState struct {
	ID JointID
	JointDesc
	MotorImpulse float64
}
type joint struct {
	state                  JointState
	active                 bool
	referenceA, referenceB vec
	impulses               [7]float64
}
type jointSlot struct {
	generation uint64
	joint      *joint
}

func (w *World) jointBodies(d JointDesc) (*body, *body, error) {
	a, err := w.lookup(d.BodyA)
	if err != nil {
		return nil, nil, err
	}
	b := &body{state: BodyState{BodyDesc: BodyDesc{Type: Static, Rotation: engine.IdentityQuaternion()}}}
	if d.BodyB != (BodyID{}) {
		b, err = w.lookup(d.BodyB)
		if err != nil {
			return nil, nil, err
		}
	}
	if d.BodyA == d.BodyB || a.state.Type != Dynamic && b.state.Type != Dynamic {
		return nil, nil, fmt.Errorf("joint requires distinct bodies and a dynamic participant")
	}
	return a, b, nil
}
func (w *World) validateJoint(d JointDesc) error {
	if _, _, err := w.jointBodies(d); err != nil {
		return err
	}
	if d.Kind > HingeJoint || !finitePoint(d.AnchorA) || !finitePoint(d.AnchorB) || !finite(d.Length) || d.Length < 0 {
		return fmt.Errorf("invalid joint kind, anchors, or length")
	}
	if d.Kind == HingeJoint {
		if !finitePoint(d.AxisA) || !finitePoint(d.AxisB) || length(d.AxisA) < 1e-10 || length(d.AxisB) < 1e-10 || !finite(length(d.AxisA)) || !finite(length(d.AxisB)) {
			return fmt.Errorf("hinge axes must be finite and nonzero")
		}
		if !finite(d.LowerAngle) || !finite(d.UpperAngle) || d.LowerAngle > d.UpperAngle || d.LowerAngle < -math.Pi || d.UpperAngle > math.Pi {
			return fmt.Errorf("hinge limits must lie within -pi..pi")
		}
		if !finite(d.MotorSpeed) || !finite(d.MaxMotorTorque) || d.MaxMotorTorque < 0 {
			return fmt.Errorf("invalid hinge motor")
		}
	} else if d.Limits || d.Motor {
		return fmt.Errorf("limits and motor require hinge")
	}
	return nil
}

// AddJoint reserves a handle immediately; hooks activate it at the next boundary.
func (w *World) AddJoint(desc JointDesc) (JointID, error) {
	if err := w.validateJoint(desc); err != nil {
		return JointID{}, err
	}
	a, b, _ := w.jointBodies(desc)
	if desc.Kind == HingeJoint && dot(unit(a.state.Rotation.RotateVector(desc.AxisA)), unit(b.state.Rotation.RotateVector(desc.AxisB))) < .99 {
		return JointID{}, fmt.Errorf("hinge axes must initially align in world space")
	}
	if desc.Kind == HingeJoint {
		desc.AxisA = unit(desc.AxisA)
		desc.AxisB = unit(desc.AxisB)
	}
	slot := len(w.joints)
	for i, s := range w.joints {
		if s.joint == nil {
			slot = i
			break
		}
	}
	if slot == len(w.joints) {
		w.joints = append(w.joints, jointSlot{generation: 1})
	}
	id := JointID{uint32(slot), w.joints[slot].generation}
	j := &joint{state: JointState{ID: id, JointDesc: desc}, active: !w.advancing}
	if desc.Kind == HingeJoint {
		axis := a.state.Rotation.RotateVector(desc.AxisA)
		ref := perpendicular(axis)
		j.referenceA = inverseRotate(a.state.Rotation, ref)
		j.referenceB = inverseRotate(b.state.Rotation, ref)
	}
	w.joints[slot].joint = j
	activate := func() {
		if live, err := w.lookupJoint(id); err == nil {
			live.active = true
			w.invalidate(desc.BodyA)
			if desc.BodyB != (BodyID{}) {
				w.invalidate(desc.BodyB)
			}
		}
	}
	if w.advancing {
		w.pending = append(w.pending, activate)
	} else {
		activate()
	}
	return id, nil
}
func perpendicular(n vec) vec {
	a := vec{X: 1}
	if math.Abs(n.X) > .8 {
		a = vec{Y: 1}
	}
	return unit(cross(n, a))
}
func (w *World) lookupJoint(id JointID) (*joint, error) {
	if id.generation == 0 || int(id.slot) >= len(w.joints) || w.joints[id.slot].generation != id.generation || w.joints[id.slot].joint == nil {
		return nil, fmt.Errorf("stale joint handle")
	}
	return w.joints[id.slot].joint, nil
}

// Joint returns a copy of a joint's descriptor and diagnostics.
func (w *World) Joint(id JointID) (JointState, error) {
	j, e := w.lookupJoint(id)
	if e != nil {
		return JointState{}, e
	}
	return j.state, nil
}

// RemoveJoint removes between steps or queues removal from a hook.
func (w *World) RemoveJoint(id JointID) error {
	if _, e := w.lookupJoint(id); e != nil {
		return e
	}
	remove := func() { w.removeJoint(id) }
	if w.advancing {
		w.pending = append(w.pending, remove)
	} else {
		remove()
	}
	return nil
}
func (w *World) removeJoint(id JointID) {
	j, e := w.lookupJoint(id)
	if e != nil {
		return
	}
	w.wakeConnected(j.state.BodyA, w.contacts)
	w.wakeConnected(j.state.BodyB, w.contacts)
	w.joints[id.slot].joint = nil
	w.joints[id.slot].generation++
}
func (w *World) removeAttachedJoints(id BodyID) {
	for _, s := range w.joints {
		if s.joint != nil && (s.joint.state.BodyA == id || s.joint.state.BodyB == id) {
			w.removeJoint(s.joint.state.ID)
		}
	}
}

// SetJointMotor updates hinge drive settings at a safe boundary and wakes its island.
func (w *World) SetJointMotor(id JointID, enabled bool, speed, torque float64) error {
	j, e := w.lookupJoint(id)
	if e != nil {
		return e
	}
	d := j.state.JointDesc
	d.Motor = enabled
	d.MotorSpeed = speed
	d.MaxMotorTorque = torque
	if e = w.validateJoint(d); e != nil {
		return e
	}
	apply := func() {
		if live, e := w.lookupJoint(id); e == nil {
			live.state.Motor = enabled
			live.state.MotorSpeed = speed
			live.state.MaxMotorTorque = torque
			live.impulses = [7]float64{}
			w.wakeConnected(d.BodyA, w.contacts)
			w.wakeConnected(d.BodyB, w.contacts)
		}
	}
	if w.advancing {
		w.pending = append(w.pending, apply)
	} else {
		apply()
	}
	return nil
}
func (w *World) jointFiltered(a, b BodyID) bool {
	for _, slot := range w.joints {
		j := slot.joint
		if j != nil && j.active && !j.state.CollideConnected && ((j.state.BodyA == a && j.state.BodyB == b) || (j.state.BodyA == b && j.state.BodyB == a)) {
			return true
		}
	}
	return false
}

// jointNeighbors supplies island graph edges without joining islands via world anchors.
func (w *World) jointNeighbors(id BodyID) []*body {
	var out []*body
	for _, slot := range w.joints {
		j := slot.joint
		if j == nil || !j.active {
			continue
		}
		var other BodyID
		if j.state.BodyA == id {
			other = j.state.BodyB
		} else if j.state.BodyB == id {
			other = j.state.BodyA
		} else {
			continue
		}
		b, e := w.lookup(other)
		if e == nil && !b.state.Disabled {
			out = append(out, b)
		}
	}
	return out
}

// SetJointLimits changes a hinge's angular limits at a safe boundary.
func (w *World) SetJointLimits(id JointID, enabled bool, lower, upper float64) error {
	j, e := w.lookupJoint(id)
	if e != nil {
		return e
	}
	d := j.state.JointDesc
	d.Limits = enabled
	d.LowerAngle = lower
	d.UpperAngle = upper
	if e = w.validateJoint(d); e != nil {
		return e
	}
	apply := func() {
		if live, e := w.lookupJoint(id); e == nil {
			live.state.Limits = enabled
			live.state.LowerAngle = lower
			live.state.UpperAngle = upper
			live.impulses = [7]float64{}
			w.wakeConnected(d.BodyA, w.contacts)
			w.wakeConnected(d.BodyB, w.contacts)
		}
	}
	if w.advancing {
		w.pending = append(w.pending, apply)
	} else {
		apply()
	}
	return nil
}

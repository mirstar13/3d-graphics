package physics

import (
	"fmt"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"math"
)

// BodyType selects pose authority and force response.
type BodyType uint8

const (
	Static BodyType = iota
	Dynamic
	Kinematic
)

// BodyID is a generation-checked world-local handle. Zero is never valid.
type BodyID struct {
	slot       uint32
	generation uint64
}

// CCDPolicy overrides the world's sphere CCD default for a body.
type CCDPolicy uint8

const (
	InheritCCD CCDPolicy = iota
	EnableCCD
	DisableCCD
)

// BodyDesc supplies initial state. Use NewBodyDesc for identity rotation,
// collision masks, and material defaults. Poses have no scale.
type BodyDesc struct {
	CCDPolicy                           CCDPolicy
	Type                                BodyType
	Position                            vec
	Rotation                            quat
	Velocity, AngularVelocity           vec
	Mass, LinearDamping, AngularDamping float64
	Collider                            Collider
	Disabled                            bool
	// CCD enables sphere sweeps against static planes, boxes, and triangle meshes.
	CCD bool
}

// NewBodyDesc supplies default mass, damping, and collision material.
func NewBodyDesc(kind BodyType, shape Shape) BodyDesc {
	return BodyDesc{Type: kind, Rotation: engine.IdentityQuaternion(), Mass: 1, LinearDamping: .02, AngularDamping: .05, Collider: Collider{Shape: shape, Friction: .6, Restitution: .1, Layer: 1, Mask: math.MaxUint32}}
}

// BodyState is a copy; changing it never changes the world.
type BodyState struct {
	InertiaTensor InertiaTensor
	ID            BodyID
	BodyDesc
	Inertia          vec
	PreviousPosition vec
	PreviousRotation quat
	Sleeping         bool
}
type body struct {
	inverseTensor                          InertiaTensor
	targetActive                           bool
	ccdStopped                             bool
	state                                  BodyState
	inverseMass                            float64
	force, torque, impulse, angularImpulse vec
	targetPosition                         vec
	targetRotation                         quat
	hasTarget                              bool
	active                                 bool
	quietTime                              float64
}
type bodySlot struct {
	generation uint64
	body       *body
}

func validateBody(desc BodyDesc) error {
	if desc.CCDPolicy > DisableCCD {
		return fmt.Errorf("invalid CCD policy")
	}
	if desc.CCD && (desc.Type != Dynamic || desc.Collider.Shape.kind != SphereShape) {
		return fmt.Errorf("CCD currently requires a dynamic sphere")
	}

	if desc.Type > Kinematic || !finitePoint(desc.Position) || !finitePoint(desc.Velocity) || !finitePoint(desc.AngularVelocity) || !validRotation(desc.Rotation) {
		return fmt.Errorf("invalid body type, pose, or velocity")
	}
	if !finite(desc.Mass) || desc.Mass <= 0 || !finite(desc.LinearDamping) || desc.LinearDamping < 0 || !finite(desc.AngularDamping) || desc.AngularDamping < 0 {
		return fmt.Errorf("mass must be positive; damping must be nonnegative and finite")
	}
	if err := desc.Collider.validate(desc.Type); err != nil {
		return err
	}
	if desc.Type == Dynamic {
		if _, ok := tensorInverse(tensorFor(desc.Collider.Shape, desc.Mass)); !ok || !finite(1/desc.Mass) {
			return fmt.Errorf("invalid body inertia")
		}

	}
	return nil
}

// AddBody reserves an ID immediately; creation inside a hook activates next tick.
func (w *World) AddBody(desc BodyDesc) (BodyID, error) {
	switch desc.CCDPolicy {
	case EnableCCD:
		desc.CCD = true
	case DisableCCD:
		desc.CCD = false
	case InheritCCD:
		desc.CCD = desc.CCD || (w.options.DefaultCCD && desc.Type == Dynamic && desc.Collider.Shape.kind == SphereShape)
	}

	if err := validateBody(desc); err != nil {
		return BodyID{}, err
	}
	desc.Rotation = desc.Rotation.Normalize()
	if desc.Collider.Shape.kind == PlaneShape {
		n := length(desc.Collider.Shape.normal)
		desc.Collider.Shape.normal = mul(desc.Collider.Shape.normal, 1/n)
		desc.Collider.Shape.offset /= n
	}
	slot := len(w.bodies)
	for i, s := range w.bodies {
		if s.body == nil {
			slot = i
			break
		}
	}
	if slot == len(w.bodies) {
		w.bodies = append(w.bodies, bodySlot{generation: 1})
	}
	id := BodyID{slot: uint32(slot), generation: w.bodies[slot].generation}
	b := &body{state: BodyState{ID: id, BodyDesc: desc, PreviousPosition: desc.Position, PreviousRotation: desc.Rotation}, active: !w.advancing}
	if desc.Type == Dynamic {
		b.inverseMass = 1 / desc.Mass
		b.updateInertia()
	} else {
		b.state.Velocity = vec{}
		b.state.AngularVelocity = vec{}
	}
	w.bodies[slot].body = b
	if w.advancing {
		w.pending = append(w.pending, func() {
			if live, err := w.lookup(id); err == nil {
				live.active = true
			}
		})
	}
	return id, nil
}
func (w *World) lookup(id BodyID) (*body, error) {
	if int(id.slot) >= len(w.bodies) || id.generation == 0 {
		return nil, fmt.Errorf("invalid body handle")
	}
	slot := w.bodies[id.slot]
	if slot.generation != id.generation || slot.body == nil {
		return nil, fmt.Errorf("stale body handle")
	}
	return slot.body, nil
}

// Body returns a snapshot, including reserved bodies not yet active.
func (w *World) Body(id BodyID) (BodyState, error) {
	b, err := w.lookup(id)
	if err != nil {
		return BodyState{}, err
	}
	return b.state, nil
}

// Bodies returns active states in stable slot order.
func (w *World) Bodies() []BodyState {
	states := make([]BodyState, 0, len(w.bodies))
	for _, s := range w.bodies {
		if s.body != nil && s.body.active {
			states = append(states, s.body.state)
		}
	}
	return states
}

// RemoveBody removes immediately between ticks, or defers safely from a hook.
func (w *World) RemoveBody(id BodyID) error {
	if _, err := w.lookup(id); err != nil {
		return err
	}
	remove := func() {
		if _, err := w.lookup(id); err == nil {
			w.invalidate(id)
			w.removeAttachedJoints(id)
			w.bodies[id.slot].body = nil
			w.bodies[id.slot].generation++
		}
	}
	if w.advancing {
		w.pending = append(w.pending, remove)
	} else {
		remove()
	}
	return nil
}
func (w *World) mutate(id BodyID, change func(*body)) error {
	b, err := w.lookup(id)
	if err != nil {
		return err
	}
	if w.advancing {
		w.pending = append(w.pending, func() {
			if b, err := w.lookup(id); err == nil {
				change(b)
				w.invalidate(id)
			}
		})
	} else {
		change(b)
		w.invalidate(id)
	}
	return nil
}

// Teleport changes pose without deriving velocity. Existing velocity is cleared.
func (w *World) Teleport(id BodyID, position vec, rotation quat) error {
	if !finitePoint(position) || !validRotation(rotation) {
		return fmt.Errorf("invalid teleport pose")
	}
	return w.mutate(id, func(b *body) {
		b.state.Position = position
		b.state.Rotation = rotation.Normalize()
		b.state.PreviousPosition = position
		b.state.PreviousRotation = b.state.Rotation
		b.state.Velocity = vec{}
		b.state.AngularVelocity = vec{}
		b.hasTarget = false
	})
}

// SetVelocity sets dynamic linear and angular velocity in world coordinates.
func (w *World) SetVelocity(id BodyID, linear, angular vec) error {
	if !finitePoint(linear) || !finitePoint(angular) {
		return fmt.Errorf("invalid velocity")
	}
	return w.mutate(id, func(b *body) {
		if b.state.Type == Dynamic {
			b.state.Velocity = linear
			b.state.AngularVelocity = angular
		}
	})
}

// SetEnabled controls participation in integration and collision detection.
func (w *World) SetEnabled(id BodyID, enabled bool) error {
	return w.mutate(id, func(b *body) {
		b.state.Disabled = !enabled
		b.state.PreviousPosition = b.state.Position
		b.state.PreviousRotation = b.state.Rotation
	})
}

// SetCollider replaces collision geometry and material, recalculating inertia.
func (w *World) SetCollider(id BodyID, collider Collider) error {
	b, err := w.lookup(id)
	if err != nil {
		return err
	}
	desc := b.state.BodyDesc
	desc.Collider = collider
	if err := validateBody(desc); err != nil {
		return err
	}
	if collider.Shape.kind == PlaneShape {
		n := length(collider.Shape.normal)
		collider.Shape.normal = mul(collider.Shape.normal, 1/n)
		collider.Shape.offset /= n
	}
	return w.mutate(id, func(b *body) {
		b.state.Collider = collider
		if b.state.Type == Dynamic {
			b.updateInertia()
		}
	})
}

// SetMass updates mass and inertia without changing velocity.
func (w *World) SetMass(id BodyID, mass float64) error {
	b, err := w.lookup(id)
	if err != nil {
		return err
	}
	desc := b.state.BodyDesc
	desc.Mass = mass
	if err := validateBody(desc); err != nil {
		return err
	}
	return w.mutate(id, func(b *body) {
		b.state.Mass = mass
		if b.state.Type == Dynamic {
			b.inverseMass = 1 / mass
			b.updateInertia()
		}
	})
}

// AddForce and AddTorque accumulate for the next integration tick; call each
// BeforeStep for continuous forcing. AfterStep submissions apply on the next tick.
func (w *World) AddForce(id BodyID, force vec) error { return w.addVector(id, force, 0) }

// AddTorque accumulates world-space torque for the next integration tick.
func (w *World) AddTorque(id BodyID, torque vec) error { return w.addVector(id, torque, 1) }

// AddImpulse queues a world-space linear impulse for the next integration tick.
func (w *World) AddImpulse(id BodyID, impulse vec) error { return w.addVector(id, impulse, 2) }

// AddAngularImpulse queues a world-space angular impulse for the next integration tick.
func (w *World) AddAngularImpulse(id BodyID, impulse vec) error { return w.addVector(id, impulse, 3) }
func (w *World) addVector(id BodyID, value vec, kind int) error {
	if !finitePoint(value) {
		return fmt.Errorf("force or impulse must be finite")
	}
	b, err := w.lookup(id)
	if err != nil {
		return err
	}
	if b.state.Type != Dynamic {
		return nil
	}
	target := &b.force
	switch kind {
	case 1:
		target = &b.torque
	case 2:
		target = &b.impulse
	case 3:
		target = &b.angularImpulse
	}
	sum := add(*target, value)
	if !finitePoint(sum) {
		return fmt.Errorf("force accumulator overflow")
	}
	if length(value) > 0 {
		w.wakeConnected(id, w.contacts)
	}
	*target = sum
	return nil
}

// SetKinematicTarget prescribes the next fixed-tick pose, deriving contact velocity.
func (w *World) SetKinematicTarget(id BodyID, position vec, rotation quat) error {
	if !finitePoint(position) || !validRotation(rotation) {
		return fmt.Errorf("invalid kinematic target")
	}
	b, err := w.lookup(id)
	if err != nil {
		return err
	}
	if b.state.Type != Kinematic {
		return fmt.Errorf("target requires kinematic body")
	}
	if position != b.state.Position || rotation.Normalize() != b.state.Rotation {
		w.wakeConnected(id, w.contacts)
	}
	b.targetPosition = position
	b.targetRotation = rotation.Normalize()
	b.hasTarget = true
	return nil
}

// AddForceAtPoint also applies torque about the centre of mass.
func (w *World) AddForceAtPoint(id BodyID, force, point vec) error {
	return w.addAtPoint(id, force, point, false)
}

// AddImpulseAtPoint queues a linear impulse and its angular impulse for next tick.
func (w *World) AddImpulseAtPoint(id BodyID, impulse, point vec) error {
	return w.addAtPoint(id, impulse, point, true)
}
func (w *World) addAtPoint(id BodyID, value, point vec, impulse bool) error {
	b, err := w.lookup(id)
	if err != nil {
		return err
	}
	if !finitePoint(value) || !finitePoint(point) {
		return fmt.Errorf("invalid force point")
	}
	if b.state.Type != Dynamic {
		return nil
	}
	linear, angular := &b.force, &b.torque
	if impulse {
		linear, angular = &b.impulse, &b.angularImpulse
	}
	l, a := add(*linear, value), add(*angular, cross(sub(point, b.state.Position), value))
	if !finitePoint(l) || !finitePoint(a) {
		return fmt.Errorf("force or torque overflow")
	}
	if length(value) > 0 {
		w.wakeConnected(id, w.contacts)
	}
	*linear, *angular = l, a
	return nil
}

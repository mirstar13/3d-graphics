package physics

import "math"

func inverseInertia(b *body, v vec) vec {
	if b.inverseMass == 0 {
		return vec{}
	}
	local := inverseRotate(b.state.Rotation, v)
	i := b.state.Inertia
	return b.state.Rotation.RotateVector(vec{X: local.X / i.X, Y: local.Y / i.Y, Z: local.Z / i.Z})
}
func worldInertia(b *body, v vec) vec {
	local := inverseRotate(b.state.Rotation, v)
	i := b.state.Inertia
	return b.state.Rotation.RotateVector(vec{X: local.X * i.X, Y: local.Y * i.Y, Z: local.Z * i.Z})
}
func (w *World) prepareBodies(dt float64) {
	for _, slot := range w.bodies {
		b := slot.body
		if b == nil || !b.active {
			continue
		}
		b.ccdStopped = false
		b.targetActive = b.hasTarget
		s := &b.state
		s.PreviousPosition = s.Position
		s.PreviousRotation = s.Rotation
		if s.Disabled || s.Sleeping {
			b.force = vec{}
			b.torque = vec{}
			b.impulse = vec{}
			b.angularImpulse = vec{}
			b.hasTarget = false
			continue
		}
		switch s.Type {
		case Dynamic:
			s.Velocity = add(s.Velocity, mul(b.impulse, b.inverseMass))
			s.AngularVelocity = add(s.AngularVelocity, inverseInertia(b, b.angularImpulse))
			s.Velocity = add(s.Velocity, mul(add(w.Gravity(), mul(b.force, b.inverseMass)), dt))
			gyro := cross(s.AngularVelocity, worldInertia(b, s.AngularVelocity))
			s.AngularVelocity = add(s.AngularVelocity, mul(inverseInertia(b, sub(b.torque, gyro)), dt))
			s.Velocity = mul(s.Velocity, math.Exp(-s.LinearDamping*dt))
			s.AngularVelocity = mul(s.AngularVelocity, math.Exp(-s.AngularDamping*dt))
		case Kinematic:
			s.Velocity = vec{}
			s.AngularVelocity = vec{}
			if b.hasTarget {
				s.Velocity = mul(sub(b.targetPosition, s.Position), 1/dt)
				s.AngularVelocity = angularBetween(s.Rotation, b.targetRotation, dt)
				b.hasTarget = false
			}
		}
		b.force = vec{}
		b.torque = vec{}
		b.impulse = vec{}
		b.angularImpulse = vec{}
	}
}

func (w *World) advanceBodies(dt float64) {
	for _, slot := range w.bodies {
		b := slot.body
		if b == nil || !b.active || b.state.Disabled || b.state.Sleeping || b.ccdStopped || b.state.Type == Static {
			continue
		}
		b.state.Position = add(b.state.Position, mul(b.state.Velocity, dt))
		b.state.Rotation = rotateBy(b.state.Rotation, b.state.AngularVelocity, dt)
	}
}

func (w *World) finishTargets() {
	for _, slot := range w.bodies {
		b := slot.body
		if b != nil && b.active && !b.state.Disabled && b.state.Type == Kinematic && b.targetActive {
			b.state.Position = b.targetPosition
			b.state.Rotation = b.targetRotation
			b.targetActive = false
		}
	}
}

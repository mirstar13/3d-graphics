package physics

import "math"

type jointRow struct {
	owner                                *joint
	index                                int
	a, b                                 *body
	ra, rb, axis                         vec
	angular                              bool
	error, target, lower, upper, impulse float64
}

func jointPoint(b *body, local vec) vec {
	return add(b.state.Position, b.state.Rotation.RotateVector(local))
}
func (w *World) jointRows(dt float64, warm bool) []jointRow {
	var rows []jointRow
	for _, slot := range w.joints {
		j := slot.joint
		if j == nil || !j.active {
			continue
		}
		a, b, e := w.jointBodies(j.state.JointDesc)
		if e != nil || a.state.Disabled || b.state.Disabled || sleepingContact(a, b) {
			continue
		}
		pa, pb := jointPoint(a, j.state.AnchorA), jointPoint(b, j.state.AnchorB)
		ra, rb := sub(pa, a.state.Position), sub(pb, b.state.Position)
		appendRow := func(index int, n vec, err float64, angular bool, lo, hi, target float64) {
			r := jointRow{owner: j, index: index, a: a, b: b, ra: ra, rb: rb, axis: n, error: err, angular: angular, lower: lo, upper: hi, target: target}
			if warm {
				r.impulse = math.Max(lo, math.Min(hi, j.impulses[index]))
			}
			rows = append(rows, r)
		}
		inf := math.Inf(1)
		delta := sub(pb, pa)
		if j.state.Kind == DistanceJoint {
			n := unit(delta)
			if length(delta) < 1e-10 {
				n = vec{X: 1}
			}
			appendRow(0, n, length(delta)-j.state.Length, false, -inf, inf, 0)
			continue
		}
		for i := 0; i < 3; i++ {
			n := axis(i)
			appendRow(i, n, dot(delta, n), false, -inf, inf, 0)
		}
		if j.state.Kind != HingeJoint {
			continue
		}
		na, nb := a.state.Rotation.RotateVector(j.state.AxisA), b.state.Rotation.RotateVector(j.state.AxisB)
		t := perpendicular(na)
		u := cross(na, t)
		alignment := cross(na, nb)
		appendRow(3, t, dot(alignment, t), true, -inf, inf, 0)
		appendRow(4, u, dot(alignment, u), true, -inf, inf, 0)
		refA, refB := a.state.Rotation.RotateVector(j.referenceA), b.state.Rotation.RotateVector(j.referenceB)
		angle := math.Atan2(dot(na, cross(refA, refB)), dot(refA, refB))
		limited := false
		if j.state.Limits {
			if angle < j.state.LowerAngle {
				appendRow(5, na, angle-j.state.LowerAngle, true, 0, inf, 0)
				limited = true
			}
			if angle > j.state.UpperAngle {
				appendRow(5, na, angle-j.state.UpperAngle, true, -inf, 0, 0)
				limited = true
			}
		}
		if j.state.Motor && !limited {
			limit := j.state.MaxMotorTorque * dt
			appendRow(6, na, 0, true, -limit, limit, j.state.MotorSpeed)
		}
	}
	return rows
}
func (r *jointRow) mass() float64 {
	if r.angular {
		return dot(r.axis, add(inverseInertia(r.a, r.axis), inverseInertia(r.b, r.axis)))
	}
	return r.a.inverseMass + r.b.inverseMass + dot(r.axis, add(cross(inverseInertia(r.a, cross(r.ra, r.axis)), r.ra), cross(inverseInertia(r.b, cross(r.rb, r.axis)), r.rb)))
}
func (r *jointRow) velocity() float64 {
	if r.angular {
		return dot(sub(r.b.state.AngularVelocity, r.a.state.AngularVelocity), r.axis)
	}
	va := add(r.a.state.Velocity, cross(r.a.state.AngularVelocity, r.ra))
	vb := add(r.b.state.Velocity, cross(r.b.state.AngularVelocity, r.rb))
	return dot(sub(vb, va), r.axis)
}
func (r *jointRow) apply(amount float64, position bool) {
	impulse := mul(r.axis, amount)
	for i, b := range []*body{r.a, r.b} {
		if b.inverseMass == 0 {
			continue
		}
		sign := -1.0
		arm := r.ra
		if i == 1 {
			sign = 1
			arm = r.rb
		}
		torque := impulse
		if !r.angular {
			torque = cross(arm, impulse)
		}
		angular := mul(inverseInertia(b, torque), sign)
		if position {
			if !r.angular {
				b.state.Position = add(b.state.Position, mul(impulse, sign*b.inverseMass))
			}
			b.state.Rotation = rotateBy(b.state.Rotation, angular, 1)
		} else {
			if !r.angular {
				b.state.Velocity = add(b.state.Velocity, mul(impulse, sign*b.inverseMass))
			}
			b.state.AngularVelocity = add(b.state.AngularVelocity, angular)
		}
	}
}
func solveJointVelocities(rows []jointRow) {
	for i := range rows {
		r := &rows[i]
		mass := r.mass()
		if mass < 1e-12 {
			continue
		}
		old := r.impulse
		r.impulse = math.Max(r.lower, math.Min(r.upper, old+(r.target-r.velocity())/mass))
		r.apply(r.impulse-old, false)
	}
}
func (w *World) correctJoints() {
	for _, row := range w.jointRows(0, false) {
		if row.index == 6 || math.Abs(row.error) < 1e-5 {
			continue
		}
		mass := row.mass()
		if mass < 1e-12 {
			continue
		}
		amount := math.Max(-.2, math.Min(.2, -.4*row.error)) / mass
		row.apply(math.Max(row.lower, math.Min(row.upper, amount)), true)
	}
}

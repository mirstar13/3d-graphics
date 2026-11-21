package physics

import "math"

const ccdMargin = contactSlop / 2

func (w *World) hasCCD() bool {
	for _, s := range w.bodies {
		if s.body != nil && s.body.active && s.body.state.CCD && !s.body.state.Disabled && !s.body.state.Sleeping {
			return true
		}
	}
	return false
}

// sphereSweep advances by distance/speed, a conservative bound for translation.
// Closest-point distances include rounded box corners and triangle edges/vertices.
// If convergence is exhausted, stop at the last safe position rather than tunnel.
func sphereSweep(a, b *body, dt float64) (float64, bool) {
	p := inverseRotate(b.state.Rotation, sub(a.state.Position, b.state.Position))
	v := inverseRotate(b.state.Rotation, a.state.Velocity)
	speed := length(v)
	if speed < 1e-12 {
		return dt, false
	}
	shape := b.state.Collider.Shape
	if shape.kind == MeshShape {
		query := emptyBounds()
		query.include(p)
		query.include(add(p, mul(v, dt)))
		earliest := dt
		found := false
		for _, index := range shape.mesh.query(query.expanded(a.state.Collider.Shape.radius + contactSlop)) {
			tri := shape.mesh.triangle(index)
			distance := func(point vec) (float64, vec) {
				delta := sub(point, closestTriangle(point, tri[0], tri[1], tri[2]))
				return length(delta) - a.state.Collider.Shape.radius, unit(delta)
			}
			if t, hit := sweepDistance(p, v, dt, distance); hit && (!found || t < earliest) {
				earliest = t
				found = true
			}
		}
		return earliest, found
	}

	distance := func(p vec) (float64, vec) {
		switch shape.kind {
		case PlaneShape:
			return dot(shape.normal, p) - shape.offset - a.state.Collider.Shape.radius, shape.normal
		case BoxShape:
			h := shape.half
			closest := vec{X: math.Max(-h.X, math.Min(h.X, p.X)), Y: math.Max(-h.Y, math.Min(h.Y, p.Y)), Z: math.Max(-h.Z, math.Min(h.Z, p.Z))}
			delta := sub(p, closest)
			return length(delta) - a.state.Collider.Shape.radius, unit(delta)

		}
		return math.Inf(1), vec{}
	}
	return sweepDistance(p, v, dt, distance)
}

func sweepDistance(p, v vec, dt float64, distance func(vec) (float64, vec)) (float64, bool) {
	speed := length(v)
	gap, normal := distance(p)
	if gap < -penetrationSlop {
		return dt, false
	} // Recover existing penetration discretely.
	if gap <= ccdMargin+1e-7 && dot(v, normal) >= -1e-8 {
		return dt, false
	}
	time := 0.0
	for i := 0; i < 96; i++ {
		gap, _ = distance(add(p, mul(v, time)))
		if gap <= ccdMargin+1e-7 {
			return time, true
		}
		increment := (gap - ccdMargin) / speed
		if time+increment > dt {
			return dt, false
		}
		if increment < 1e-10 {
			return time, true
		}
		time += increment
	}
	return time, true
}

func (w *World) earliestImpact(dt float64) (float64, bool) {
	earliest := dt
	found := false
	for _, slot := range w.bodies {
		a := slot.body
		if a == nil || !a.active || !a.state.CCD || a.state.Disabled || a.state.Sleeping || a.state.Collider.Sensor || a.ccdStopped {
			continue
		}
		swept := bodyBounds(a)
		end := swept
		delta := mul(a.state.Velocity, dt)
		end.min = add(end.min, delta)
		end.max = add(end.max, delta)
		swept.include(end.min)
		swept.include(end.max)
		swept = swept.expanded(contactSlop)
		for _, other := range w.bodies {
			b := other.body
			if b == nil || !b.active || b.state.Disabled || b.state.Type != Static || b.state.Collider.Sensor {
				continue
			}
			kind := b.state.Collider.Shape.kind
			if kind != PlaneShape && kind != BoxShape && kind != MeshShape {
				continue
			}
			if a.state.Collider.Layer&b.state.Collider.Mask == 0 || b.state.Collider.Layer&a.state.Collider.Mask == 0 || !swept.overlaps(bodyBounds(b)) {
				continue
			}
			w.solverStats.CCDQueries++
			if time, hit := sphereSweep(a, b, dt); hit && (!found || time < earliest) {
				earliest = time
				found = true
			}
		}
	}
	return earliest, found
}

// advanceCCD keeps all bodies at the same sub-tick time. Outer hooks, forces,
// pose history, and sleep timers still execute once per fixed clock tick.
func (w *World) advanceCCD(dt float64) {
	remaining := dt
	var events []ContactEvent
	seenStay := make(map[bodyPair]bool)
	publish := func() {
		w.updateEvents()
		for _, e := range w.events {
			key := bodyPair{e.A, e.B}
			if e.Phase == ContactStay && seenStay[key] {
				continue
			}
			events = append(events, e)
			seenStay[key] = true
		}
	}
	for remaining > 1e-12 {
		interval, hit := w.earliestImpact(remaining)
		w.advanceBodies(interval)
		remaining -= interval
		// Warm-start impulses belong to a full fixed interval. Avoid reapplying
		// those impulses to repeated CCD solves with different interval lengths.
		w.solveContacts(false)
		publish()
		if !hit {
			break
		}
		w.solverStats.CCDImpacts++
		if w.solverStats.CCDImpacts >= w.options.CCDMaxImpacts {
			if remaining > 1e-12 {
				for _, s := range w.bodies {
					b := s.body
					if b != nil && b.active && b.state.CCD && !b.state.Disabled && !b.state.Sleeping {
						b.ccdStopped = true
						b.state.Velocity = vec{}
						b.state.AngularVelocity = vec{}
						w.solverStats.CCDClamps++
					}
				}
				w.advanceBodies(remaining)
				w.solveContacts(false)
				publish()
			}
			break
		}
	}
	w.events = events
}

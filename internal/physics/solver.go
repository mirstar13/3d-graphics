package physics

import (
	"math"
	"time"
)

type constraint struct {
	contact        Contact
	a, b           *body
	normalImpulse  float64
	tangentImpulse vec
	bounce         float64
}

// SolverStats reports only physics work, excluding drawing and terminal output.
type SolverStats struct {
	Bodies, CandidatePairs, Contacts               int
	AwakeBodies, SleepingBodies, SolvedConstraints int
	Duration                                       time.Duration
	// PairTests includes all enabled pairs visited across every detection pass.
	PairTests, DetectionPasses                                                  int
	IntegrationDuration, BroadPhaseDuration, NarrowPhaseDuration, SolveDuration time.Duration
}

func contactPoint(c Contact) vec { return mul(add(c.PointA, c.PointB), .5) }
func relativeVelocity(c *constraint) vec {
	p := contactPoint(c.contact)
	ra, rb := sub(p, c.a.state.Position), sub(p, c.b.state.Position)
	return sub(add(c.b.state.Velocity, cross(c.b.state.AngularVelocity, rb)), add(c.a.state.Velocity, cross(c.a.state.AngularVelocity, ra)))
}
func effectiveMass(a, b *body, p, n vec) float64 {
	ra, rb := sub(p, a.state.Position), sub(p, b.state.Position)
	return a.inverseMass + b.inverseMass + dot(n, add(cross(inverseInertia(a, cross(ra, n)), ra), cross(inverseInertia(b, cross(rb, n)), rb)))
}
func applyImpulse(a, b *body, p, impulse vec) {
	if a.inverseMass > 0 {
		a.state.Velocity = sub(a.state.Velocity, mul(impulse, a.inverseMass))
		a.state.AngularVelocity = sub(a.state.AngularVelocity, inverseInertia(a, cross(sub(p, a.state.Position), impulse)))
	}
	if b.inverseMass > 0 {
		b.state.Velocity = add(b.state.Velocity, mul(impulse, b.inverseMass))
		b.state.AngularVelocity = add(b.state.AngularVelocity, inverseInertia(b, cross(sub(p, b.state.Position), impulse)))
	}
}
func (w *World) stepBodies(dt float64) {
	start := time.Now()
	w.solverStats = SolverStats{}
	w.integrateBodies(dt)
	w.solverStats.IntegrationDuration = time.Since(start)
	contacts, candidates := w.detect()
	w.wakeImpacts(contacts)
	constraints := make([]constraint, 0, len(contacts))
	for _, contact := range contacts {
		if contact.Sensor {
			continue
		}
		a, _ := w.lookup(contact.A)
		b, _ := w.lookup(contact.B)
		if a.inverseMass+b.inverseMass == 0 || sleepingContact(a, b) {
			continue
		}
		c := constraint{contact: contact, a: a, b: b}
		speed := dot(relativeVelocity(&c), contact.Normal)
		if speed < -restitutionSpeedThreshold {
			c.bounce = -math.Max(a.state.Collider.Restitution, b.state.Collider.Restitution) * speed
		}
		if previous, ok := w.cache[contactKey{contact.A, contact.B, contact.Feature}]; ok && dot(previous.normal, contact.Normal) > cacheNormalAlignment && length(sub(previous.point, contactPoint(contact))) < cachePointDistance {
			c.normalImpulse = previous.impulse
			c.tangentImpulse = sub(previous.tangent, mul(contact.Normal, dot(previous.tangent, contact.Normal)))
		}
		constraints = append(constraints, c)
	}
	w.solverStats.SolvedConstraints = len(constraints)
	for i := range constraints {
		c := &constraints[i]
		applyImpulse(c.a, c.b, contactPoint(c.contact), add(mul(c.contact.Normal, c.normalImpulse), c.tangentImpulse))
	}
	for iteration := 0; iteration < w.options.VelocityIterations; iteration++ {
		for i := range constraints {
			c := &constraints[i]
			n, p := c.contact.Normal, contactPoint(c.contact)
			mass := effectiveMass(c.a, c.b, p, n)
			if mass < 1e-12 {
				continue
			}
			delta := (c.bounce - dot(relativeVelocity(c), n)) / mass
			previous := c.normalImpulse
			c.normalImpulse = math.Max(0, previous+delta)
			applyImpulse(c.a, c.b, p, mul(n, c.normalImpulse-previous))
			velocity := relativeVelocity(c)
			tangent := sub(velocity, mul(n, dot(velocity, n)))
			if length(tangent) < 1e-10 {
				continue
			}
			t := unit(tangent)
			mass = effectiveMass(c.a, c.b, p, t)
			if mass < 1e-12 {
				continue
			}
			impulse := sub(c.tangentImpulse, mul(t, dot(velocity, t)/mass))
			limit := math.Sqrt(c.a.state.Collider.Friction*c.b.state.Collider.Friction) * c.normalImpulse
			if length(impulse) > limit {
				impulse = mul(unit(impulse), limit)
			}
			applyImpulse(c.a, c.b, p, sub(impulse, c.tangentImpulse))
			c.tangentImpulse = impulse
		}
	}
	nextCache := make(map[contactKey]cachedImpulse, len(constraints))
	for _, c := range constraints {
		nextCache[contactKey{c.contact.A, c.contact.B, c.contact.Feature}] = cachedImpulse{normal: c.contact.Normal, tangent: c.tangentImpulse, point: contactPoint(c.contact), impulse: c.normalImpulse}
	}
	for key, impulse := range w.cache {
		a, ea := w.lookup(key.a)
		b, eb := w.lookup(key.b)
		if ea == nil && eb == nil && sleepingContact(a, b) {
			nextCache[key] = impulse
		}
	}
	w.cache = nextCache
	for iteration := 0; iteration < w.options.PositionIterations; iteration++ {
		positions, _ := w.detect()
		for _, c := range positions {
			if c.Sensor || c.Separation >= -penetrationSlop {
				continue
			}
			a, _ := w.lookup(c.A)
			b, _ := w.lookup(c.B)
			if sleepingContact(a, b) {
				continue
			}
			p := contactPoint(c)
			mass := effectiveMass(a, b, p, c.Normal)
			if mass < 1e-12 {
				continue
			}
			magnitude := math.Min(maxPositionCorrection, positionCorrectionFraction*(-c.Separation-penetrationSlop)) / mass
			impulse := mul(c.Normal, magnitude)
			ra, rb := sub(p, a.state.Position), sub(p, b.state.Position)
			if a.inverseMass > 0 {
				a.state.Position = sub(a.state.Position, mul(impulse, a.inverseMass))
				a.state.Rotation = rotateBy(a.state.Rotation, mul(inverseInertia(a, cross(ra, impulse)), -1), 1)
			}
			if b.inverseMass > 0 {
				b.state.Position = add(b.state.Position, mul(impulse, b.inverseMass))
				b.state.Rotation = rotateBy(b.state.Rotation, inverseInertia(b, cross(rb, impulse)), 1)
			}
		}
	}
	w.contacts, _ = w.detect()
	w.updateEvents()
	w.updateSleep(dt)
	active := 0
	for _, slot := range w.bodies {
		if slot.body != nil && slot.body.active && !slot.body.state.Disabled {
			active++
			if slot.body.state.Type == Dynamic {
				if slot.body.state.Sleeping {
					w.solverStats.SleepingBodies++
				} else {
					w.solverStats.AwakeBodies++
				}
			}
		}
	}
	w.solverStats.Bodies = active
	w.solverStats.CandidatePairs = candidates
	w.solverStats.Contacts = len(w.contacts)
	w.solverStats.Duration = time.Since(start)
	w.solverStats.SolveDuration = w.solverStats.Duration - w.solverStats.IntegrationDuration - w.solverStats.BroadPhaseDuration - w.solverStats.NarrowPhaseDuration
}

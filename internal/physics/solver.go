package physics

import (
	"math"
	"time"
)

type contactKey struct {
	a, b    BodyID
	feature uint64
}
type cachedImpulse struct {
	normal, tangent, point vec
	impulse                float64
}
type constraint struct {
	contact        Contact
	a, b           *body
	normalImpulse  float64
	tangentImpulse vec
	bounce         float64
}

// SolverStats reports only physics work, excluding drawing and terminal output.
type SolverStats struct {
	Bodies, CandidatePairs, Contacts int
	Duration                         time.Duration
}

// ContactPhase reports a pair beginning, maintaining, or ending contact.
type ContactPhase uint8

const (
	ContactBegin ContactPhase = iota
	ContactStay
	ContactEnd
)

// ContactEvent is buffered once per pair per tick, including sensors.
type ContactEvent struct {
	A, B   BodyID
	Phase  ContactPhase
	Sensor bool
}
type bodyPair struct{ a, b BodyID }

// Contacts returns a copy of the most recent fixed-step manifolds.
func (w *World) Contacts() []Contact { return append([]Contact(nil), w.contacts...) }

// ContactEvents returns the latest tick's events. Hooks may schedule removals safely.
func (w *World) ContactEvents() []ContactEvent { return append([]ContactEvent(nil), w.events...) }
func (w *World) SolverStats() SolverStats      { return w.solverStats }
func (w *World) invalidate(id BodyID) {
	for key := range w.cache {
		if key.a == id || key.b == id {
			delete(w.cache, key)
		}
	}
}
func (w *World) detect() ([]Contact, int) {
	var contacts []Contact
	candidates := 0
	bs := make([]bounds, len(w.bodies))
	for i, s := range w.bodies {
		if s.body != nil && s.body.active && !s.body.state.Disabled {
			bs[i] = bodyBounds(s.body).expanded(contactSlop)
		}
	}
	for i, slot := range w.bodies {
		a := slot.body
		if a == nil || !a.active || a.state.Disabled {
			continue
		}
		for j := i + 1; j < len(w.bodies); j++ {
			b := w.bodies[j].body
			if b == nil || !b.active || b.state.Disabled {
				continue
			}
			ac, bc := a.state.Collider, b.state.Collider
			if ac.Layer&bc.Mask == 0 || bc.Layer&ac.Mask == 0 {
				continue
			}
			if a.state.Type != Dynamic && b.state.Type != Dynamic && !ac.Sensor && !bc.Sensor {
				continue
			}
			if !bs[i].overlaps(bs[j]) {
				continue
			}
			candidates++
			contacts = append(contacts, collide(a, b)...)
		}
	}
	return contacts, candidates
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
	w.integrateBodies(dt)
	contacts, candidates := w.detect()
	constraints := make([]constraint, 0, len(contacts))
	for _, contact := range contacts {
		if contact.Sensor {
			continue
		}
		a, _ := w.lookup(contact.A)
		b, _ := w.lookup(contact.B)
		if a.inverseMass+b.inverseMass == 0 {
			continue
		}
		c := constraint{contact: contact, a: a, b: b}
		speed := dot(relativeVelocity(&c), contact.Normal)
		if speed < -1 {
			c.bounce = -math.Max(a.state.Collider.Restitution, b.state.Collider.Restitution) * speed
		}
		if previous, ok := w.cache[contactKey{contact.A, contact.B, contact.Feature}]; ok && dot(previous.normal, contact.Normal) > .95 && length(sub(previous.point, contactPoint(contact))) < .1 {
			c.normalImpulse = previous.impulse
			c.tangentImpulse = sub(previous.tangent, mul(contact.Normal, dot(previous.tangent, contact.Normal)))
		}
		constraints = append(constraints, c)
	}
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
	w.cache = nextCache
	for iteration := 0; iteration < w.options.PositionIterations; iteration++ {
		positions, _ := w.detect()
		for _, c := range positions {
			if c.Sensor || c.Separation >= -.002 {
				continue
			}
			a, _ := w.lookup(c.A)
			b, _ := w.lookup(c.B)
			p := contactPoint(c)
			mass := effectiveMass(a, b, p, c.Normal)
			if mass < 1e-12 {
				continue
			}
			magnitude := math.Min(.2, .4*(-c.Separation-.002)) / mass
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
	active := 0
	for _, slot := range w.bodies {
		if slot.body != nil && slot.body.active && !slot.body.state.Disabled {
			active++
		}
	}
	w.solverStats = SolverStats{Bodies: active, CandidatePairs: candidates, Contacts: len(w.contacts), Duration: time.Since(start)}
}
func (w *World) updateEvents() {
	current := make(map[bodyPair]bool)
	var order []bodyPair
	w.events = nil
	for _, c := range w.contacts {
		key := bodyPair{c.A, c.B}
		if _, exists := current[key]; !exists {
			current[key] = c.Sensor
			order = append(order, key)
			phase := ContactBegin
			if _, exists := w.contactPairs[key]; exists {
				phase = ContactStay
			}
			w.events = append(w.events, ContactEvent{A: c.A, B: c.B, Phase: phase, Sensor: c.Sensor})
		}
	}
	for _, key := range w.pairOrder {
		if _, exists := current[key]; !exists {
			w.events = append(w.events, ContactEvent{A: key.a, B: key.b, Phase: ContactEnd, Sensor: w.contactPairs[key]})
		}
	}
	w.contactPairs = current
	w.pairOrder = order
}

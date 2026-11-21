package physics

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

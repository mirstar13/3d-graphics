package physics

import "time"

// detect uses stable slot ordering; each active slot has exactly one generation.
func (w *World) detect() ([]Contact, int) {
	var pairs []bodyPair
	candidates := 0
	start := time.Now()
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
			w.solverStats.PairTests++
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
			pairs = append(pairs, bodyPair{a.state.ID, b.state.ID})
		}
	}
	w.solverStats.BroadPhaseDuration += time.Since(start)
	start = time.Now()
	var contacts []Contact
	for _, pair := range pairs {
		a, _ := w.lookup(pair.a)
		b, _ := w.lookup(pair.b)
		contacts = append(contacts, collide(a, b)...)
	}
	w.solverStats.NarrowPhaseDuration += time.Since(start)
	w.solverStats.DetectionPasses++
	return contacts, candidates
}

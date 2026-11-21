package physics

import (
	"sort"
	"time"
)

// BroadPhaseMode selects candidate generation; BruteForce is a reference oracle.
type BroadPhaseMode uint8

const (
	SweepAndPrune BroadPhaseMode = iota
	BruteForce
)

func (w *World) candidates() []bodyPair {
	if cap(w.boundsScratch) < len(w.bodies) {
		w.boundsScratch = make([]bounds, len(w.bodies))
	}
	w.boundsScratch = w.boundsScratch[:len(w.bodies)]
	w.orderScratch = w.orderScratch[:0]
	w.planeScratch = w.planeScratch[:0]
	w.pairScratch = w.pairScratch[:0]
	for i, s := range w.bodies {
		b := s.body
		if b == nil || !b.active || b.state.Disabled {
			continue
		}
		w.boundsScratch[i] = bodyBounds(b).expanded(contactSlop)
		if b.state.Collider.Shape.kind == PlaneShape {
			w.planeScratch = append(w.planeScratch, i)
		} else {
			w.orderScratch = append(w.orderScratch, i)
		}
	}
	test := func(i, j int) {
		if i > j {
			i, j = j, i
		}
		a, b := w.bodies[i].body, w.bodies[j].body
		w.solverStats.PairTests++
		ac, bc := a.state.Collider, b.state.Collider
		if ac.Layer&bc.Mask == 0 || bc.Layer&ac.Mask == 0 {
			return
		}
		if a.state.Type != Dynamic && b.state.Type != Dynamic && !ac.Sensor && !bc.Sensor {
			return
		}
		if w.boundsScratch[i].overlaps(w.boundsScratch[j]) {
			w.pairScratch = append(w.pairScratch, bodyPair{a.state.ID, b.state.ID})
		}
	}
	if w.options.BroadPhase == BruteForce {
		for i, slot := range w.bodies {
			if slot.body == nil || !slot.body.active || slot.body.state.Disabled {
				continue
			}
			for j := i + 1; j < len(w.bodies); j++ {
				b := w.bodies[j].body
				if b != nil && b.active && !b.state.Disabled {
					test(i, j)
				}
			}
		}
	} else {
		sort.Slice(w.orderScratch, func(i, j int) bool {
			a, b := w.orderScratch[i], w.orderScratch[j]
			if w.boundsScratch[a].min.X != w.boundsScratch[b].min.X {
				return w.boundsScratch[a].min.X < w.boundsScratch[b].min.X
			}
			return a < b
		})
		for n, i := range w.orderScratch {
			for _, j := range w.orderScratch[n+1:] {
				if w.boundsScratch[j].min.X > w.boundsScratch[i].max.X {
					break
				}
				test(i, j)
			}
		}
		for n, i := range w.planeScratch {
			for _, j := range w.orderScratch {
				test(i, j)
			}
			for _, j := range w.planeScratch[n+1:] {
				test(i, j)
			}
		}
	}
	// Narrow-phase and constraint order do not depend on broad-phase traversal.
	sort.Slice(w.pairScratch, func(i, j int) bool {
		a, b := w.pairScratch[i], w.pairScratch[j]
		if a.a.slot != b.a.slot {
			return a.a.slot < b.a.slot
		}
		if a.a.generation != b.a.generation {
			return a.a.generation < b.a.generation
		}
		if a.b.slot != b.b.slot {
			return a.b.slot < b.b.slot
		}
		return a.b.generation < b.b.generation
	})
	return w.pairScratch
}

// detect returns temporary contacts valid until the next call. Published
// contacts and snapshots own separate storage.
func (w *World) detect() ([]Contact, int) {
	start := time.Now()
	pairs := w.candidates()
	w.solverStats.BroadPhaseDuration += time.Since(start)
	start = time.Now()
	contacts := w.contactScratch[:0]
	for _, pair := range pairs {
		a, _ := w.lookup(pair.a)
		b, _ := w.lookup(pair.b)
		contacts = append(contacts, collide(a, b)...)
	}
	w.contactScratch = contacts
	w.solverStats.NarrowPhaseDuration += time.Since(start)
	w.solverStats.DetectionPasses++
	return contacts, len(pairs)
}

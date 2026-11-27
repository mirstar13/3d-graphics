package physics

import (
	"slices"
	"testing"
	"time"
)

func BenchmarkActiveStacks(b *testing.B) {
	o := DefaultOptions()
	o.Sleeping = false
	w, err := NewWorld(o)
	if err != nil {
		b.Fatal(err)
	}
	floor := NewBodyDesc(Static, Plane(vec{Y: 1}, 0))
	floor.Collider.Restitution = 0
	if _, err = w.AddBody(floor); err != nil {
		b.Fatal(err)
	}
	ids := make([]BodyID, 0, 16)
	for stack := 0; stack < 16; stack++ {
		for level := 0; level < 8; level++ {
			d := NewBodyDesc(Dynamic, Box(vec{X: .5, Y: .5, Z: .5}))
			d.Position = vec{X: float64(stack%4) * 3, Y: float64(level) + .5, Z: float64(stack/4) * 3}
			d.Collider.Restitution = 0
			id, err := w.AddBody(d)
			if err != nil {
				b.Fatal(err)
			}
			if level == 7 {
				ids = append(ids, id)
			}
		}
	}
	w.SetPaused(true)
	durations := make([]time.Duration, 0, 256)
	for tick := 0; b.Loop(); tick++ {
		start := time.Now()
		if tick%8 == 0 {
			if err := w.AddImpulse(ids[tick/8%len(ids)], vec{X: .5, Y: .1}); err != nil {
				b.Fatal(err)
			}
		}
		if err := w.StepOnce(nil); err != nil {
			b.Fatal(err)
		}
		durations = append(durations, time.Since(start))
	}
	slices.Sort(durations)
	b.ReportMetric(float64(durations[(len(durations)-1)*95/100].Microseconds()), "us/tick-p95")
}

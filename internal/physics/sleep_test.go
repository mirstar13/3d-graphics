package physics

import "testing"

func sleepingStack(t *testing.T) (*World, BodyID, []BodyID) {
	t.Helper()
	w := newTestWorld(t)
	f := NewBodyDesc(Static, Plane(vec{Y: 1}, 0))
	f.Collider.Restitution = 0
	floor := addTestBody(t, w, f)
	var ids []BodyID
	for i := 0; i < 3; i++ {
		d := NewBodyDesc(Dynamic, Box(vec{X: .5, Y: .5, Z: .5}))
		d.Position.Y = .5 + float64(i)
		d.Collider.Restitution = 0
		ids = append(ids, addTestBody(t, w, d))
	}
	d := NewBodyDesc(Dynamic, Sphere(.5))
	d.Position = vec{X: 10, Y: .5}
	d.Collider.Restitution = 0
	ids = append(ids, addTestBody(t, w, d))
	stepN(t, w, 600)
	for _, id := range ids {
		if !stateOf(t, w, id).Sleeping {
			t.Fatal("body did not sleep", stateOf(t, w, id))
		}
	}
	return w, floor, ids
}
func TestSleepingIslandsAndWakeTriggers(t *testing.T) {
	for _, trigger := range []string{"force", "impulse", "velocity", "mass", "collider", "teleport", "disable", "remove"} {
		t.Run(trigger, func(t *testing.T) {
			w, _, ids := sleepingStack(t)
			stepN(t, w, 1)
			if w.SolverStats().SolvedConstraints != 0 || w.SolverStats().SleepingBodies != 4 {
				t.Fatal(w.SolverStats())
			}
			for _, e := range w.ContactEvents() {
				if e.Phase != ContactStay {
					t.Fatal("sleep changed events", e)
				}
			}
			id := ids[0]
			s := stateOf(t, w, id)
			var err error
			switch trigger {
			case "force":
				err = w.AddForce(id, vec{X: 1})
			case "impulse":
				err = w.AddImpulse(id, vec{X: 1})
			case "velocity":
				err = w.SetVelocity(id, vec{X: 1}, vec{})
			case "mass":
				err = w.SetMass(id, 2)
			case "collider":
				err = w.SetCollider(id, s.Collider)
			case "teleport":
				err = w.Teleport(id, vec{X: 2, Y: .5}, s.Rotation)
			case "disable":
				err = w.SetEnabled(id, false)
			case "remove":
				err = w.RemoveBody(id)
			}
			if err != nil {
				t.Fatal(err)
			}
			for _, other := range ids[1:3] {
				if stateOf(t, w, other).Sleeping {
					t.Fatal("connected box did not wake")
				}
			}
			if !stateOf(t, w, ids[3]).Sleeping {
				t.Fatal("floor merged separate islands")
			}
		})
	}
}
func TestSleepingSupportRemovalAndImpact(t *testing.T) {
	w, floor, ids := sleepingStack(t)
	w.RemoveBody(floor)
	stepN(t, w, 1)
	for _, id := range ids {
		s := stateOf(t, w, id)
		if s.Sleeping || s.Velocity.Y >= 0 {
			t.Fatal("unsupported body suspended", s)
		}
	}
	w, _, ids = sleepingStack(t)
	d := NewBodyDesc(Dynamic, Sphere(.5))
	d.Position = vec{X: 9.1, Y: .5}
	d.Velocity.X = 2
	addTestBody(t, w, d)
	stepN(t, w, 1)
	if stateOf(t, w, ids[3]).Sleeping {
		t.Fatal("impact did not wake")
	}
}
func TestSleepingKinematicSupportAndPause(t *testing.T) {
	w := newTestWorld(t)
	p := NewBodyDesc(Kinematic, Box(vec{X: 2, Y: .5, Z: 2}))
	p.Collider.Restitution = 0
	platform := addTestBody(t, w, p)
	d := NewBodyDesc(Dynamic, Sphere(.5))
	d.Position.Y = 1
	d.Collider.Restitution = 0
	id := addTestBody(t, w, d)
	stepN(t, w, 120)
	if !stateOf(t, w, id).Sleeping {
		t.Fatal("did not sleep")
	}
	before := stateOf(t, w, id)
	w.SetPaused(true)
	stepN(t, w, 60)
	if stateOf(t, w, id) != before {
		t.Fatal("pause changed sleeping body")
	}
	w.SetKinematicTarget(platform, vec{Y: .1}, p.Rotation)
	if stateOf(t, w, id).Sleeping {
		t.Fatal("target did not wake rider")
	}
	w.StepOnce(nil)
	if stateOf(t, w, id).Velocity.Y <= 0 {
		t.Fatal("platform did not lift rider")
	}
}
func TestSleepOptionsAndZeroForce(t *testing.T) {
	o := DefaultOptions()
	o.SleepDelay = 0
	if _, err := NewWorld(o); err == nil {
		t.Fatal("invalid sleep options accepted")
	}
	w, _, ids := sleepingStack(t)
	w.AddForce(ids[0], vec{})
	if !stateOf(t, w, ids[0]).Sleeping {
		t.Fatal("zero force woke body")
	}
	w = noGravity(t)
	w.options.Sleeping = false
	id := addTestBody(t, w, NewBodyDesc(Dynamic, Sphere(1)))
	stepN(t, w, 120)
	if stateOf(t, w, id).Sleeping {
		t.Fatal("sleep disabled")
	}
}

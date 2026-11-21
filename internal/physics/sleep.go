package physics

// wakeConnected wakes dynamics linked by physical contacts. Static/kinematic
// supports seed their touching dynamics but never join otherwise separate islands.
func (w *World) wakeConnected(id BodyID, contacts []Contact) {
	seen := map[BodyID]bool{id: true}
	queue := []BodyID{id}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if b, err := w.lookup(current); err == nil && b.state.Type == Dynamic {
			b.state.Sleeping = false
			b.quietTime = 0
		}
		for _, c := range contacts {
			if c.Sensor {
				continue
			}
			var next BodyID
			if c.A == current {
				next = c.B
			} else if c.B == current {
				next = c.A
			} else {
				continue
			}
			if seen[next] {
				continue
			}
			b, err := w.lookup(next)
			if err != nil || b.state.Type != Dynamic || b.state.Disabled {
				continue
			}
			seen[next] = true
			queue = append(queue, next)
		}
	}
}

func moving(b *body) bool {
	return length(b.state.Velocity) > 1e-10 || length(b.state.AngularVelocity) > 1e-10
}

func (w *World) wakeImpacts(contacts []Contact) {
	for _, c := range contacts {
		if c.Sensor {
			continue
		}
		a, _ := w.lookup(c.A)
		b, _ := w.lookup(c.B)
		if _, existed := w.contactPairs[bodyPair{c.A, c.B}]; !existed && (a.state.Sleeping || b.state.Sleeping) {
			w.wakeConnected(c.A, contacts)
			w.wakeConnected(c.B, contacts)
		}
		if a.state.Sleeping && ((b.state.Type == Dynamic && !b.state.Sleeping) || (b.state.Type == Kinematic && moving(b))) {
			w.wakeConnected(a.state.ID, contacts)
		}
		if b.state.Sleeping && ((a.state.Type == Dynamic && !a.state.Sleeping) || (a.state.Type == Kinematic && moving(a))) {
			w.wakeConnected(b.state.ID, contacts)
		}
	}
}

func sleepingContact(a, b *body) bool {
	return (a.state.Type != Dynamic || a.state.Sleeping) && (b.state.Type != Dynamic || b.state.Sleeping)
}

func (w *World) updateSleep(dt float64) {
	if !w.options.Sleeping {
		return
	}
	seen := make(map[BodyID]bool)
	for _, slot := range w.bodies {
		root := slot.body
		if root == nil || !root.active || root.state.Disabled || root.state.Type != Dynamic || seen[root.state.ID] {
			continue
		}
		island := []*body{root}
		seen[root.state.ID] = true
		quiet := true
		for i := 0; i < len(island); i++ {
			b := island[i]
			if length(b.state.Velocity) > w.options.SleepLinearThreshold || length(b.state.AngularVelocity) > w.options.SleepAngularThreshold {
				quiet = false
			}
			for _, c := range w.contacts {
				if c.Sensor {
					continue
				}
				var other BodyID
				if c.A == b.state.ID {
					other = c.B
				} else if c.B == b.state.ID {
					other = c.A
				} else {
					continue
				}
				if c.Separation < -sleepPenetrationLimit {
					quiet = false
				}
				neighbor, err := w.lookup(other)
				if err != nil {
					continue
				}
				if neighbor.state.Type == Kinematic && moving(neighbor) {
					quiet = false
				}
				if neighbor.state.Type == Dynamic && !seen[other] {
					seen[other] = true
					island = append(island, neighbor)
				}
			}
		}
		ready := quiet
		for _, b := range island {
			if quiet {
				b.quietTime += dt
			} else {
				b.quietTime = 0
			}
			if b.quietTime < w.options.SleepDelay {
				ready = false
			}
		}
		for _, b := range island {
			b.state.Sleeping = ready
			if ready {
				b.state.Velocity = vec{}
				b.state.AngularVelocity = vec{}
			}
		}
	}
}

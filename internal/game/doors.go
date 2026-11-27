package game

import "github.com/mirstar13/go-3d-graphics/internal/physics"

// updateSignals samples the tick's sensor pairs together. Combining them before
// changing a door handles simultaneous exits and entries and removed bodies.
func (s *Session) updateSignals(tick uint64) {
	if s.doorCount == 0 && s.objective.Outcome != Playing {
		return
	}
	held := map[string]bool{}
	for _, event := range s.World.ContactEvents() {
		if !event.Sensor || event.Phase == physics.ContactEnd {
			continue
		}
		s.holdPlate(held, event.A, event.B)
		s.holdPlate(held, event.B, event.A)
	}
	if s.objective.Outcome == Playing {
		s.objective.Tick = tick
		if held[s.objective.Signal] {
			s.objective.Outcome = Won
		} else if s.objective.TickLimit > 0 && tick >= s.objective.TickLimit {
			s.objective.Outcome = Lost
		}
	}
	for i := range s.entities {
		e := &s.entities[i]
		d := &e.description
		if !e.active || d.DoorSignal == "" {
			continue
		}
		open := held[d.DoorSignal]
		if open == e.doorOpen {
			continue
		}
		position := d.Position
		if open {
			lift := d.DoorLift
			if lift == 0 {
				size := d.CollisionSize()
				lift = 2 * size.Y
				if d.CollisionShape() == "sphere" {
					lift = 2 * size.X
				}
			}
			position.Y += lift
		}
		state, err := s.World.Body(e.body)
		if err == nil {
			err = s.World.Teleport(e.body, position, state.Rotation)
		}
		if err != nil {
			s.lastError = err
			continue
		}
		e.doorOpen = open
	}
}

func (s *Session) holdPlate(held map[string]bool, plate, other physics.BodyID) {
	signal, ok := s.plates[plate]
	if !ok || held[signal] {
		return
	}
	body, err := s.World.Body(other)
	if err == nil && body.Type == physics.Dynamic {
		held[signal] = true
	}
}

package app

import (
	"math"
	"time"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/game"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
)

// switchRoom owns transient controls for the first-person example.
type switchRoom struct {
	yaw, pitch float64
	walk       string
	walkUntil  time.Time
	held       string
	opened     bool
	message    string
}

func (r *switchRoom) sight(s *game.Session) (engine.Point, engine.Point) {
	id, _ := s.Find("player")
	player, _ := s.State(id)
	eye := player.Position
	eye.Y += .85
	direction := engine.Point{X: math.Sin(r.yaw) * math.Cos(r.pitch), Y: math.Sin(r.pitch), Z: math.Cos(r.yaw) * math.Cos(r.pitch)}
	return eye, direction
}

func (r *switchRoom) camera(s *game.Session) {
	eye, d := r.sight(s)
	s.Scene.Camera.SetPosition(eye.X, eye.Y, eye.Z)
	s.Scene.Camera.LookAt(engine.Point{X: eye.X + d.X, Y: eye.Y + d.Y, Z: eye.Z + d.Z})
}

func (r *switchRoom) key(s *game.Session, key string) error {
	switch key {
	case "w", "a", "s", "d":
		r.walk = key
		r.walkUntil = time.Now().Add(180 * time.Millisecond)
	case "left":
		r.yaw -= .12
	case "right":
		r.yaw += .12
	case "up":
		r.pitch = min(1.1, r.pitch+.1)
	case "down":
		r.pitch = max(-1.1, r.pitch-.1)
	case "f", "g", "e":
		if key == "g" && r.held != "" {
			r.held = ""
			r.message = "Released"
			return optionalSound(s, "grab")
		}
		eye, d := r.sight(s)
		hit, ok, err := s.Pick(physics.Ray{Origin: eye, Direction: d, MaxDistance: 3})
		if err != nil {
			return err
		}
		if !ok {
			r.message = "Aim at an object within reach"
			return nil
		}
		if key == "e" && hit.FileID == "button" && !r.opened {
			gate, ok := s.Find("gate")
			if ok {
				if err := s.QueueDestroy(gate); err != nil {
					return err
				}
			}
			r.opened = true
			r.message = "Gate open"
			return optionalSound(s, "gate")
		}
		if hit.FileID != "crate" {
			r.message = "Aim at the crate or green button"
			return nil
		}
		switch key {
		case "g":
			r.held = hit.FileID
			r.message = "Holding crate"
			return optionalSound(s, "grab")
		case "f":
			r.held = ""
			r.message = "Pushed crate"
			return gameImpulse(s, hit.FileID, engine.Point{X: 4 * d.X, Y: 4 * d.Y, Z: 4 * d.Z}, "push")
		}
	}
	r.camera(s)
	return nil
}

func (r *switchRoom) tick(s *game.Session, elapsed time.Duration, now time.Time) error {
	dt := min(.1, max(0., elapsed.Seconds()))
	player, _ := s.Find("player")
	body, err := s.Inspect(player)
	if err != nil {
		return err
	}
	var forward, side float64
	if now.Before(r.walkUntil) {
		switch r.walk {
		case "w":
			forward = 1
		case "s":
			forward = -1
		case "a":
			side = -1
		case "d":
			side = 1
		}
	}
	vx := 3 * (forward*math.Sin(r.yaw) + side*math.Cos(r.yaw))
	vz := 3 * (forward*math.Cos(r.yaw) - side*math.Sin(r.yaw))
	gain := min(1., dt*12)
	if err := s.QueueImpulse(player, engine.Point{X: gain * (vx - body.Body.Velocity.X), Z: gain * (vz - body.Body.Velocity.Z)}); err != nil {
		return err
	}
	if r.held == "" {
		return nil
	}
	id, ok := s.Find(r.held)
	if !ok {
		r.held = ""
		return nil
	}
	crate, err := s.Inspect(id)
	if err != nil {
		return err
	}
	eye, d := r.sight(s)
	pos, vel := crate.Entity.Position, crate.Body.Velocity
	// A damped spring carries the crate while the solver handles contact.
	spring := func(distance, speed float64) float64 { return dt * max(-35., min(35., 32*distance-9*speed)) }
	return s.QueueImpulse(id, engine.Point{X: spring(eye.X+2*d.X-pos.X, vel.X), Y: spring(eye.Y+2*d.Y-pos.Y, vel.Y), Z: spring(eye.Z+2*d.Z-pos.Z, vel.Z)})
}

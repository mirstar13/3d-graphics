package app

import (
	"fmt"

	"github.com/mirstar13/go-3d-graphics/internal/config"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/game"
)

type playerControl func(*game.Session, string) (string, error)

type gameProfile struct {
	player, help string
	firstPerson  bool
	controls     func() playerControl
}

var exampleProfiles = map[string]gameProfile{
	"pulse-dock":  {player: "tug", help: "Arrows move | Space jump"},
	"switch-room": {player: "player", firstPerson: true, help: "WASD walk | Right-drag to look | M free look | Arrows look | E use | F push | G grab"},
	"bank-shot":   {player: "puck", help: "Left/Right aim at a rail | Down aim straight | Space shoot", controls: bankShotControls},
	"lift-run":    {player: "runner", help: "Left/Right steer | Space brake | Ride the blue launch pad", controls: liftRunControls},
}

func RunExample(root, name string) error {
	profile, ok := exampleProfiles[name]
	if !ok {
		return fmt.Errorf("unknown example game %q", name)
	}
	settings := config.Default()
	return runPlayer(root, profile, true, settings.Color, settings.Physics.WorldOptions(), settings.Width, settings.Height)
}

func bankShotControls() playerControl {
	aim := 0.0
	fired := false
	return func(s *game.Session, key string) (string, error) {
		if fired {
			return "", nil
		}
		switch key {
		case "left":
			aim = -7.8
			return "Aiming at left rail", optionalSound(s, "aim")
		case "down":
			aim = 0
			return "Aiming straight", optionalSound(s, "aim")
		case "right":
			aim = 7.8
			return "Aiming at right rail", optionalSound(s, "aim")
		case " ":
			fired = true
			return "Puck launched", gameImpulse(s, "puck", engine.Point{X: aim, Z: 7}, "shoot")
		}
		return "", nil
	}
}

func liftRunControls() playerControl {
	return func(s *game.Session, key string) (string, error) {
		id, ok := s.Find("runner")
		if !ok {
			return "", fmt.Errorf("missing runner")
		}
		state, err := s.Inspect(id)
		if err != nil {
			return "", err
		}
		var speed float64
		switch key {
		case "left":
			speed = -3.5
		case "right":
			speed = 3.5
		case " ":
			speed = 0
		default:
			return "", nil
		}
		return "", s.QueueImpulse(id, engine.Point{X: speed - state.Body.Velocity.X})
	}
}

func gameImpulse(s *game.Session, actor string, impulse engine.Point, cue string) error {
	id, ok := s.Find(actor)
	if !ok {
		return fmt.Errorf("missing actor %q", actor)
	}
	if err := s.QueueImpulse(id, impulse); err != nil {
		return err
	}
	return optionalSound(s, cue)
}

func optionalSound(s *game.Session, cue string) error {
	if s.HasSound(cue) {
		return s.QueueSound(cue)
	}
	return nil
}

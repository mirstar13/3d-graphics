package game

import (
	"fmt"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

// LocalPose returns a copy for editor projection and pose calculations.
func (s *Session) LocalPose(id EntityID) (engine.Transform, error) {
	e, err := s.lookup(id)
	if err != nil {
		return engine.Transform{}, err
	}
	return *e.node.Transform, nil
}

// PreviewPose temporarily changes only the visual pose and restores it after draw.
func (s *Session) PreviewPose(id EntityID, position engine.Point, rotation engine.Quaternion, draw func()) {
	e, err := s.lookup(id)
	if err == nil {
		t := e.node.Transform
		original := *t
		defer func() { *t = original }()
		t.SetPosition(position.X, position.Y, position.Z)
		t.SetRotationQuaternion(rotation)
	}
	draw()
}

// SetPausedPose updates the physics and visual pose together between ticks.
func (s *Session) SetPausedPose(id EntityID, position engine.Point, rotation engine.Quaternion) error {
	if !s.World.Stats().Paused {
		return fmt.Errorf("pause Play before changing a runtime pose")
	}
	e, err := s.lookup(id)
	if err != nil {
		return err
	}
	if e.description.Body != "none" {
		if err := s.World.Teleport(e.body, position, rotation); err != nil {
			return err
		}
	}
	e.node.Transform.SetPosition(position.X, position.Y, position.Z)
	e.node.Transform.SetRotationQuaternion(rotation)
	return nil
}

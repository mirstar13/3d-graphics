// Package game composes scenes and physics independently of demos and terminal UI.
package game

import (
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
)

// BodySpec binds a visual to independently described collision geometry.
type BodySpec struct {
	Node         *engine.SceneNode
	Desc         physics.BodyDesc
	VisualOffset engine.Point
	Target       func(float64) (engine.Point, engine.Quaternion)
}

// JointSpec references body visuals; nil B denotes a world anchor.
type JointSpec struct {
	A, B *engine.SceneNode
	Desc physics.JointDesc
}

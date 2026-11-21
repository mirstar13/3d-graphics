package app

import (
	"fmt"
	"github.com/mirstar13/go-3d-graphics/internal/demo"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
)

type bodyBinding struct {
	id     physics.BodyID
	node   *engine.SceneNode
	target func(float64) (engine.Point, engine.Quaternion)
}
type physicsBindings struct {
	world *physics.World
	scene *engine.Scene
	items []bodyBinding
}

func newPhysicsBindings(world *physics.World, scene *engine.Scene, specs []demo.BodySpec) (*physicsBindings, error) {
	bindings := &physicsBindings{world: world, scene: scene}
	for _, spec := range specs {
		if err := bindings.add(spec); err != nil {
			return nil, err
		}
	}
	return bindings, nil
}
func (b *physicsBindings) add(spec demo.BodySpec) error {
	node := spec.Node
	if node == nil || node.Parent != b.scene.Root || node.Transform.Scale != (engine.Point{X: 1, Y: 1, Z: 1}) || b.scene.Root.Transform.GetWorldMatrix() != engine.IdentityMatrix() {
		return fmt.Errorf("physics binding requires an unscaled visual under the identity scene root")
	}
	if node.OnUpdate != nil {
		return fmt.Errorf("bound nodes must use physics targets instead of OnUpdate")
	}
	for _, binding := range b.items {
		if binding.node == node {
			return fmt.Errorf("node already owns a body")
		}
	}
	id, err := b.world.AddBody(spec.Desc)
	if err != nil {
		return err
	}
	b.items = append(b.items, bodyBinding{id: id, node: node, target: spec.Target})
	return nil
}
func (b *physicsBindings) before(tick physics.Tick) error {
	for _, binding := range b.items {
		if _, err := b.world.Body(binding.id); err != nil {
			continue
		}
		if binding.node.Parent != b.scene.Root || binding.node.Transform.Scale != (engine.Point{X: 1, Y: 1, Z: 1}) || binding.node.OnUpdate != nil || b.scene.Root.Transform.GetWorldMatrix() != engine.IdentityMatrix() {
			return fmt.Errorf("physics-bound node hierarchy, scale, or update callback changed")
		}
		if binding.target != nil {
			p, q := binding.target(tick.Time)
			if err := b.world.SetKinematicTarget(binding.id, p, q); err != nil {
				return err
			}
		}
	}
	return nil
}
func (b *physicsBindings) synchronize() {
	kept := b.items[:0]
	for _, binding := range b.items {
		state, err := b.world.Body(binding.id)
		if err != nil {
			b.scene.RemoveNode(binding.node)
			continue
		}
		binding.node.Transform.SetPosition(state.Position.X, state.Position.Y, state.Position.Z)
		binding.node.Transform.SetRotationQuaternion(state.Rotation)
		binding.node.SetEnabled(!state.Disabled)
		kept = append(kept, binding)
	}
	b.items = kept
}

// setEnabled keeps rendering and physics enabled state under one authority.
func (b *physicsBindings) setEnabled(id physics.BodyID, enabled bool) error {
	if err := b.world.SetEnabled(id, enabled); err != nil {
		return err
	}
	b.synchronize()
	return nil
}

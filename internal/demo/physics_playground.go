package demo

import (
	"fmt"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"math"
)

// PhysicsPlayground preserves the six existing scene numbers and appends number 7.
const PhysicsPlayground = 6

// BodySpec explicitly binds a procedural visual to independent collision geometry.
// The application creates bodies and owns the resulting ID-to-node registry.
type BodySpec struct {
	Node   *engine.SceneNode
	Desc   physics.BodyDesc
	Target func(float64) (engine.Point, engine.Quaternion)
}

func playgroundBox(scene *engine.Scene, name string, p, half engine.Point, color engine.Color) *engine.SceneNode {
	node := box(scene, name, p.X, p.Y, p.Z, 1, 1, 1, color)
	mesh := node.Object.(*engine.Mesh)
	for i, v := range mesh.Vertices {
		mesh.Vertices[i] = engine.Point{X: v.X * half.X, Y: v.Y * half.Y, Z: v.Z * half.Z}
	}
	return node
}
func buildPlayground(scene *engine.Scene) {
	playgroundBox(scene, "PhysicsFloor", engine.Point{Y: -4.5}, engine.Point{X: 16, Y: .25, Z: 12}, engine.Color{R: 80, G: 100, B: 125})
	for i := 0; i < 12; i++ {
		sphere(scene, fmt.Sprintf("Ball%d", i), .8, float64(i%4)*4.5-9, 1.5+float64(i/4)*2.5, float64(i/4)*4-6, palette[i%len(palette)])
	}
	for i, spec := range obstacleSpecs() {
		playgroundBox(scene, fmt.Sprintf("Obstacle%d", i), spec.position, spec.half, palette[1])
	}
	for i := 0; i < 3; i++ {
		playgroundBox(scene, fmt.Sprintf("Crate%d", i), engine.Point{X: 10, Y: -3.35 + float64(i)*1.82, Z: 6}, engine.Point{X: .9, Y: .9, Z: .9}, palette[0])
	}
	playgroundBox(scene, "Platform", engine.Point{X: -3, Y: -2, Z: -6}, engine.Point{X: 2, Y: .4, Z: 2}, palette[3])
	ramp := engine.NewMesh()
	ramp.Vertices = rampVertices()
	ramp.Indices = []int{0, 2, 1, 0, 3, 2}
	ramp.Material = material(palette[2])
	meshNode(scene, "Ramp", ramp, 0, 0, 0)
}

type obstacle struct{ position, half engine.Point }

func obstacleSpecs() []obstacle {
	return []obstacle{
		{engine.Point{X: -9, Y: -3.5, Z: -2}, engine.Point{X: 2, Y: .75, Z: 2}},
		{engine.Point{X: -3, Y: -3.75, Z: 4}, engine.Point{X: 3, Y: .5, Z: 1}},
		{engine.Point{X: 6, Y: -2.25, Z: 4}, engine.Point{X: 2, Y: 2, Z: 1}},
	}
}
func rampVertices() []engine.Point {
	return []engine.Point{{X: 6, Y: -4.2, Z: -8}, {X: 13, Y: -4.2, Z: -8}, {X: 13, Y: -1, Z: -1}, {X: 6, Y: -1, Z: -1}}
}

// PhysicsSpecs describes the playground without mutating scene or world state.
func PhysicsSpecs(scene *engine.Scene, index int) ([]BodySpec, error) {
	if index != PhysicsPlayground {
		return nil, nil
	}
	var specs []BodySpec
	add := func(name string, kind physics.BodyType, shape physics.Shape) *BodySpec {
		node := scene.FindNode(name)
		desc := physics.NewBodyDesc(kind, shape)
		if node != nil {
			desc.Position = node.Transform.Position
			desc.Rotation = node.Transform.Rotation
		}
		specs = append(specs, BodySpec{Node: node, Desc: desc})
		return &specs[len(specs)-1]
	}
	floor := add("PhysicsFloor", physics.Static, physics.Plane(engine.Point{Y: 1}, .25))
	floor.Desc.Collider.Restitution = 0
	for i := 0; i < 12; i++ {
		spec := add(fmt.Sprintf("Ball%d", i), physics.Dynamic, physics.Sphere(.8))
		spec.Desc.Collider.Restitution = .15 + float64(i%3)*.15
	}
	for i, spec := range obstacleSpecs() {
		add(fmt.Sprintf("Obstacle%d", i), physics.Static, physics.Box(spec.half))
	}
	for i := 0; i < 3; i++ {
		spec := add(fmt.Sprintf("Crate%d", i), physics.Dynamic, physics.Box(engine.Point{X: .9, Y: .9, Z: .9}))
		spec.Desc.Collider.Restitution = 0
	}
	platform := add("Platform", physics.Kinematic, physics.Box(engine.Point{X: 2, Y: .4, Z: 2}))
	platform.Target = func(t float64) (engine.Point, engine.Quaternion) {
		return engine.Point{X: -3 + 2*math.Sin(t*.3), Y: -2 + .8*math.Sin(t*.7), Z: -6}, engine.IdentityQuaternion()
	}
	mesh, err := physics.NewTriangleMesh(rampVertices(), []int{0, 2, 1, 0, 3, 2})
	if err != nil {
		return nil, err
	}
	add("Ramp", physics.Static, physics.Mesh(mesh))
	for _, spec := range specs {
		if spec.Node == nil {
			return nil, fmt.Errorf("physics playground visual is missing")
		}
	}
	return specs, nil
}

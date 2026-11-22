package demo

import (
	"fmt"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/game"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"math"
)

// PhysicsPlayground preserves the six existing scene numbers and appends number 7.
const PhysicsPlayground = 6

// BodySpec explicitly binds a procedural visual to independent collision geometry.
// The application creates bodies and owns the resulting ID-to-node registry.
type BodySpec = game.BodySpec

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
	playgroundBox(scene, "Pendulum", engine.Point{X: -11, Y: 1, Z: 8}, engine.Point{X: .5, Y: .5, Z: .5}, palette[4%len(palette)])
	playgroundBox(scene, "PendulumStand", engine.Point{X: -13, Y: 0, Z: 8}, engine.Point{X: .15, Y: 4, Z: .15}, palette[1])
	dumbbell := engine.NewSceneNode("Dumbbell")
	dumbbell.Transform.SetPosition(0, 3, 8)
	scene.AddNode(dumbbell)
	for i, child := range dumbbellChildren() {
		node := playgroundBox(scene, fmt.Sprintf("DumbbellPart%d", i), child.Position, child.Shape.HalfExtents(), palette[2])
		dumbbell.AddChild(node)
	}
	wedge := engine.NewMesh()
	wedge.Vertices, wedge.Indices = wedgeGeometry()
	wedge.Material = material(palette[0])
	meshNode(scene, "Wedge", wedge, 4, 3, 8)
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
	add("Pendulum", physics.Dynamic, physics.Box(engine.Point{X: .5, Y: .5, Z: .5}))
	compound, err := physics.Compound(dumbbellChildren())
	if err != nil {
		return nil, err
	}
	dumbbell := add("Dumbbell", physics.Dynamic, compound)
	center := compound.CenterOfMass()
	dumbbell.Desc.Position.X += center.X
	dumbbell.Desc.Position.Y += center.Y
	dumbbell.Desc.Position.Z += center.Z
	dumbbell.VisualOffset = engine.Point{X: -center.X, Y: -center.Y, Z: -center.Z}
	vertices, indices := wedgeGeometry()
	hull, err := physics.ConvexHull(vertices, indices)
	if err != nil {
		return nil, err
	}
	wedge := add("Wedge", physics.Dynamic, hull)
	center = hull.CenterOfMass()
	wedge.Desc.Position.X += center.X
	wedge.Desc.Position.Y += center.Y
	wedge.Desc.Position.Z += center.Z
	wedge.VisualOffset = engine.Point{X: -center.X, Y: -center.Y, Z: -center.Z}
	for _, spec := range specs {
		if spec.Node == nil {
			return nil, fmt.Errorf("physics playground visual is missing")
		}
	}
	return specs, nil
}

// JointSpec binds visual nodes through the application's body registry. A nil B
// denotes a world anchor; the descriptor's BodyIDs are assigned by the application.
type JointSpec = game.JointSpec

// PhysicsJoints describes constraints between registered body visuals or world anchors.
func PhysicsJoints(scene *engine.Scene, index int) []JointSpec {
	if index != PhysicsPlayground {
		return nil
	}
	return []JointSpec{{A: scene.FindNode("Pendulum"), Desc: physics.JointDesc{Kind: physics.DistanceJoint, AnchorB: engine.Point{X: -13, Y: 4, Z: 8}, Length: math.Sqrt(13)}}}
}

func dumbbellChildren() []physics.CompoundChild {
	return []physics.CompoundChild{
		{Shape: physics.Box(engine.Point{X: .6, Y: .6, Z: .6}), Position: engine.Point{X: -1.2}, Rotation: engine.IdentityQuaternion(), Weight: 1},
		{Shape: physics.Box(engine.Point{X: .6, Y: .6, Z: .6}), Position: engine.Point{X: 1.2, Y: .4}, Rotation: engine.IdentityQuaternion(), Weight: 2},
		{Shape: physics.Box(engine.Point{X: 1.2, Y: .15, Z: .15}), Rotation: engine.IdentityQuaternion(), Weight: .2},
	}
}

func wedgeGeometry() ([]engine.Point, []int) {
	vertices := []engine.Point{{X: -.8, Y: -.7, Z: -.7}, {X: .8, Y: -.7, Z: -.7}, {X: -.8, Y: .4, Z: -.7}, {X: .8, Y: .9, Z: -.7}, {X: -.8, Y: -.7, Z: .7}, {X: .8, Y: -.7, Z: .7}, {X: -.8, Y: .4, Z: .7}, {X: .8, Y: .9, Z: .7}}
	indices := []int{0, 4, 6, 0, 6, 2, 1, 3, 7, 1, 7, 5, 0, 1, 5, 0, 5, 4, 2, 6, 7, 2, 7, 3, 0, 2, 3, 0, 3, 1, 4, 5, 7, 4, 7, 6}
	return vertices, indices
}

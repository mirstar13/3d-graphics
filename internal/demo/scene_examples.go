package demo

import (
	"fmt"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"math"
)

var palette = []engine.Color{{R: 245, G: 145, B: 65}, {R: 65, G: 205, B: 210}, {R: 185, G: 120, B: 240}, {R: 235, G: 205, B: 85}}

func material(color engine.Color) *engine.Material {
	m := engine.NewMaterial()
	m.DiffuseColor = color
	m.AmbientStrength = .6
	m.SpecularStrength = .2
	return &m
}
func meshNode(scene *engine.Scene, name string, mesh *engine.Mesh, x, y, z float64) *engine.SceneNode {
	node := engine.NewSceneNodeWithObject(name, mesh)
	node.Transform.SetPosition(x, y, z)
	scene.AddNode(node)
	return node
}
func box(scene *engine.Scene, name string, x, y, z, sx, sy, sz float64, color engine.Color) *engine.SceneNode {
	node := scene.CreateCube(name, 1, material(color))
	node.Transform.SetPosition(x, y, z)
	node.Transform.SetScale(sx, sy, sz)
	return node
}
func floor(scene *engine.Scene, halfWidth, halfDepth float64) {
	box(scene, "Stage", 0, -4.5, 0, halfWidth, .25, halfDepth, engine.Color{R: 75, G: 90, B: 110})
}
func sphere(scene *engine.Scene, name string, radius, x, y, z float64, color engine.Color) *engine.SceneNode {
	mesh := engine.GenerateSphere(radius, 12, 16)
	mesh.Material = material(color)
	return meshNode(scene, name, mesh, x, y, z)
}

func buildGallery(scene *engine.Scene) {
	floor(scene, 17, 7)
	for i := 0; i < 3; i++ {
		box(scene, fmt.Sprintf("Plinth%d", i), float64(i-1)*10, -3.5, 0, 3.7, .75, 3.7, engine.Color{R: 120, G: 135, B: 150})
	}
	cube := box(scene, "Prism", -10, 0, 0, 2.4, 2.4, 2.4, palette[0])
	mesh := cube.Object.(*engine.Mesh)
	for face := 0; face < 6; face++ {
		m := material(palette[face%len(palette)])
		mesh.FaceMaterials = append(mesh.FaceMaterials, m, m)
	}
	sphere(scene, "Pearl", 3, 0, 0, 0, palette[1])
	ring := engine.GenerateTorus(2.6, .85, 24, 8)
	ring.Material = material(palette[2])
	meshNode(scene, "Ring", ring, 10, 0, 0)
}
func animateGallery(scene *engine.Scene, t float64) {
	scene.FindNode("Prism").Transform.SetRotation(.3+t*.21, t*.35, .15)
	scene.FindNode("Pearl").Transform.SetPosition(0, .5*math.Sin(t), 0)
	scene.FindNode("Ring").Transform.SetRotation(.8+t*.3, t*.2, 0)
}

func buildOrbits(scene *engine.Scene) {
	sphere(scene, "Sun", 3, 0, 0, 0, palette[3])
	for i, radius := range []float64{7, 11, 15} {
		pivot := scene.CreateEmpty(fmt.Sprintf("Orbit%d", i))
		planet := sphere(scene, fmt.Sprintf("Planet%d", i), 1.2+float64(i)*.35, radius, 0, 0, palette[i])
		pivot.AddChild(planet)
		track := engine.GenerateTorus(radius, .09, 48, 4)
		track.Material = material(engine.Color{R: 110, G: 130, B: 155})
		meshNode(scene, fmt.Sprintf("Track%d", i), track, 0, 0, 0)
	}
	moonPivot := scene.CreateEmpty("MoonOrbit")
	scene.FindNode("Planet1").AddChild(moonPivot)
	moon := sphere(scene, "Moon", .6, 2.8, 0, 0, engine.ColorWhite)
	moonPivot.AddChild(moon)
}
func animateOrbits(scene *engine.Scene, t float64) {
	for i, speed := range []float64{.45, .28, .18} {
		scene.FindNode(fmt.Sprintf("Orbit%d", i)).Transform.SetRotation(0, t*speed+float64(i)*2, 0)
	}
	scene.FindNode("MoonOrbit").Transform.SetRotation(.3, t*.9, 0)
}

func textured(texture *engine.Texture) *engine.TexturedMaterial {
	m := engine.NewTexturedMaterial()
	m.DiffuseTexture = texture
	m.UseTextures = true
	m.DiffuseColor = engine.ColorWhite
	m.AmbientStrength = .6
	m.SpecularStrength = .25
	return &m
}
func buildTextures(scene *engine.Scene) {
	floor(scene, 17, 7)
	checker := textured(engine.GenerateCheckerboard(64, 64, 16, engine.ColorWhite, palette[1]))
	gradient := textured(engine.GenerateGradient(64, 64, palette[0], palette[2], true))
	globe := engine.GenerateSphere(3.8, 16, 24)
	globe.Material = checker
	meshNode(scene, "Globe", globe, -10, 0, 0)
	// Independent vertices per face preserve UV seams and allow bold face-sized checks.
	cube := engine.NewMesh()
	cube.Material = checker
	faces := [][4]engine.Point{
		{{X: 3, Y: -3, Z: -3}, {X: -3, Y: -3, Z: -3}, {X: -3, Y: 3, Z: -3}, {X: 3, Y: 3, Z: -3}},
		{{X: -3, Y: -3, Z: 3}, {X: 3, Y: -3, Z: 3}, {X: 3, Y: 3, Z: 3}, {X: -3, Y: 3, Z: 3}},
		{{X: 3, Y: -3, Z: 3}, {X: 3, Y: -3, Z: -3}, {X: 3, Y: 3, Z: -3}, {X: 3, Y: 3, Z: 3}},
		{{X: -3, Y: -3, Z: -3}, {X: -3, Y: -3, Z: 3}, {X: -3, Y: 3, Z: 3}, {X: -3, Y: 3, Z: -3}},
		{{X: -3, Y: 3, Z: 3}, {X: 3, Y: 3, Z: 3}, {X: 3, Y: 3, Z: -3}, {X: -3, Y: 3, Z: -3}},
		{{X: -3, Y: -3, Z: -3}, {X: 3, Y: -3, Z: -3}, {X: 3, Y: -3, Z: 3}, {X: -3, Y: -3, Z: 3}},
	}
	uv := []engine.TextureCoord{{U: 0, V: 0}, {U: 1, V: 0}, {U: 1, V: 1}, {U: 0, V: 1}}
	for _, face := range faces {
		first := len(cube.Vertices)
		for i, p := range face {
			cube.AddVertexWithUV(p.X, p.Y, p.Z, uv[i].U, uv[i].V)
		}
		cube.AddQuadIndices(first, first+1, first+2, first+3)
	}
	meshNode(scene, "Textile", cube, 0, 0, 0)
	ring := engine.GenerateTorus(2.8, 1, 24, 10)
	ring.Material = gradient
	meshNode(scene, "Spectrum", ring, 10, 0, 0)
}
func animateTextures(scene *engine.Scene, t float64) {
	scene.FindNode("Globe").Transform.SetRotation(0, t*.25, 0)
	scene.FindNode("Textile").Transform.SetRotation(.25, t*.3, .12)
	scene.FindNode("Spectrum").Transform.SetRotation(1, t*.2, .2)
}

func buildCourtyard(scene *engine.Scene) {
	floor(scene, 16, 14)
	for i := 0; i < 4; i++ {
		x := float64(i)*7 - 10.5
		box(scene, fmt.Sprintf("Pillar%d", i), x, 0, 5, 1, 4, 1, palette[1])
		if i < 3 {
			box(scene, fmt.Sprintf("Lintel%d", i), x+3.5, 4, 5, 3.5, .7, 1, palette[0])
		}
	}
	sphere(scene, "Sundial", 2.8, 0, -1, -4, palette[2])
	sphere(scene, "Lantern", .7, 0, 22, 0, palette[3])
}
func animateCourtyard(scene *engine.Scene, t float64) {
	p := lanternPosition(t)
	scene.FindNode("Lantern").Transform.SetPosition(p.X, p.Y, p.Z)
}

func buildRunway(scene *engine.Scene) {
	floor(scene, 17, 19)
	for i := 0; i < 3; i++ {
		group := engine.NewLODGroupWithTransitions(engine.LODTransitionCrossFade, .6)
		for level, segments := range [][2]int{{16, 24}, {8, 12}, {4, 6}} {
			mesh := engine.GenerateSphere(3.5, segments[0], segments[1])
			mesh.Material = material(palette[i])
			group.AddLOD(mesh, []float64{52, 66, 200}[level])
		}
		group.UpdateHysteresis = 1
		node := scene.CreateEmpty(fmt.Sprintf("Traveler%d", i))
		node.SetLODGroupWithTransition(group)
	}
}
func animateRunway(scene *engine.Scene, t float64) {
	for i := 0; i < 3; i++ {
		scene.FindNode(fmt.Sprintf("Traveler%d", i)).Transform.SetPosition(float64(i-1)*10, 0, 16*math.Sin(t*.5+float64(i)*2))
	}
}

func buildWaves(scene *engine.Scene) {
	source := box(scene, "Source", 0, 0, 0, 1, 1, 1, palette[0])
	mesh := source.Object.(*engine.Mesh)
	scene.RemoveNode(source)
	instances := engine.NewInstancedMesh(mesh)
	for z := 0; z < 9; z++ {
		for x := 0; x < 9; x++ {
			instances.AddInstance(engine.IdentityMatrix(), engine.Color{R: uint8(65 + x*20), G: uint8(100 + z*16), B: uint8(230 - x*13)})
		}
	}
	scene.AddNode(engine.NewSceneNodeWithObject("Wave", instances))
}
func animateWaves(scene *engine.Scene, t float64) {
	instances := scene.FindNode("Wave").Object.(*engine.InstancedMesh)
	for i := range instances.Instances {
		x, z := float64(i%9-4)*3.2, float64(i/9-4)*3.2
		height := 2 + 1.3*math.Sin(t*1.1+x*.18+z*.2)
		instances.SetInstanceTransform(i, engine.ComposeMatrix(engine.Point{X: x, Y: height - 3, Z: z}, engine.QuaternionFromEuler(0, 0, 0), engine.Point{X: 1.15, Y: height, Z: 1.15}))
	}
}

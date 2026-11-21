package demo

import "github.com/mirstar13/go-3d-graphics/internal/engine"

// Preset defines one curated terminal scene and its initial viewing conditions.
// Geometry is procedural and deterministic, so no external assets are required.
type Preset struct {
	Name, Description string
	Shadows           bool
	radius, height    float64
	build             func(*engine.Scene)
	animate           func(*engine.Scene, float64)
}

// Presets is the ordered scene catalog. YAML and CLI numbers are one-based.
var Presets = []Preset{
	{"Kinetic gallery", "Three moving sculptures: shape, shading, and per-face color.", false, 40, 9, buildGallery, animateGallery},
	{"Orbital clock", "Nested orbits and a moon demonstrate parent-child transforms.", false, 48, 23, buildOrbits, animateOrbits},
	{"Texture workshop", "Bold procedural textures on a sphere, cube, and tilted ring.", false, 40, 12, buildTextures, animateTextures},
	{"Lantern courtyard", "A moving lantern casts shadows across a small colonnade.", true, 55, 28, buildCourtyard, animateCourtyard},
	{"Detail runway", "Three moving spheres cross detail thresholds with dithered fades.", false, 60, 18, buildRunway, animateRunway},
	{"Wave field", "81 instanced columns form a traveling wave; compare CPU modes.", false, 58, 30, buildWaves, animateWaves},
	{"Physics playground", "Falling spheres, crates, a moving platform, and a collision ramp. C shows contacts.", false, 55, 22, buildPlayground, func(*engine.Scene, float64) {}},
}

// Names supplies labels for the scene picker, derived from the catalog.
var Names = func() []string {
	names := make([]string, len(Presets))
	for i, preset := range Presets {
		names[i] = preset.Name
	}
	return names
}()

// PresetAt returns a preset, falling back to the gallery for an invalid index.
func PresetAt(index int) Preset {
	if index < 0 || index >= len(Presets) {
		index = 0
	}
	return Presets[index]
}

// ConfigureCamera frames the whole scene before either manual input or auto orbit.
func ConfigureCamera(camera *engine.Camera, index int, _ engine.OrientationType) {
	preset := PresetAt(index)
	camera.Near, camera.Far = .5, 250
	camera.SetFOV(60, 30)
	camera.SetPosition(0, preset.height, -preset.radius)
	camera.LookAt(engine.Point{})
}

// ConfigureCameraController matches the initial camera, avoiding an orbit-start jump.
func ConfigureCameraController(controller *engine.CameraController, index int) {
	preset := PresetAt(index)
	controller.SetOrbitCenter(0, 0, 0)
	controller.SetOrbitRadius(preset.radius)
	controller.SetOrbitHeight(preset.height)
	controller.OrbitSpeed = .0025
	controller.EnableAutoOrbit(true)
}

// BuildScene replaces scene contents and initializes animation at time zero.
func BuildScene(scene *engine.Scene, index int) {
	scene.Clear()
	preset := PresetAt(index)
	preset.build(scene)
	preset.animate(scene, 0)
}

// AnimateSceneDemo evaluates animation from elapsed seconds, avoiding drift and
// frame-rate-dependent compound rotations. Scene.Update advances LOD fades.
func AnimateSceneDemo(scene *engine.Scene, index int, elapsed, _ float64) {
	PresetAt(index).animate(scene, elapsed)
}

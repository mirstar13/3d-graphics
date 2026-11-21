package demo

import (
	"math"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

// SetupLighting uses a bright studio rig, with a single moving key for shadows.
func SetupLighting(camera *engine.Camera, index int) *engine.LightingSystem {
	lighting := engine.NewLightingSystem(camera)
	lighting.AmbientLight = engine.Color{R: 160, G: 175, B: 200}
	lighting.AmbientIntensity = .45
	lighting.AddLight(engine.NewLight(-18, 28, -24, engine.Color{R: 255, G: 230, B: 195}, 2))
	if !PresetAt(index).Shadows {
		lighting.AddLight(engine.NewLight(20, 12, 18, engine.Color{R: 155, G: 205, B: 255}, 1.1))
	}
	AnimateLighting(lighting, index, 0)
	return lighting
}

// AnimateLighting keeps studio lighting stable and moves only the courtyard lantern.
func AnimateLighting(lighting *engine.LightingSystem, index int, elapsed float64) {
	if lighting == nil || len(lighting.Lights) == 0 || !PresetAt(index).Shadows {
		return
	}
	lighting.Lights[0].Position = lanternPosition(elapsed)
}
func lanternPosition(t float64) engine.Point {
	return engine.Point{X: 16 * math.Cos(t*.3), Y: 14, Z: 16 * math.Sin(t*.3)}
}

package main

import "math"

// ============================================================================
// LIGHTING SCENARIO SYSTEM
// ============================================================================
// Comprehensive lighting setups demonstrating different lighting techniques
// and artistic lighting principles used in 3D rendering
// ============================================================================

// ============================================================================
// SCENARIO 1: AMBIENT ONLY - Flat, even lighting
// ============================================================================
func SetupAmbientLighting(camera *Camera) *LightingSystem {
	ls := NewLightingSystem(camera)

	// High ambient light, no directional lights
	ls.AmbientLight = Color{180, 180, 200}
	ls.AmbientIntensity = 0.8

	return ls
}

// ============================================================================
// SCENARIO 2: SINGLE DIRECTIONAL - Sun/Moon lighting
// ============================================================================
func SetupDirectionalLighting(camera *Camera) *LightingSystem {
	ls := NewLightingSystem(camera)

	// Low ambient
	ls.AmbientLight = Color{30, 40, 50}
	ls.AmbientIntensity = 0.15

	// Single strong directional light (sun)
	sunLight := NewLight(50, 40, -30, Color{255, 245, 230}, 1.5)
	ls.AddLight(sunLight)

	return ls
}

// ============================================================================
// SCENARIO 3: THREE-POINT LIGHTING - Studio/cinematic setup
// ============================================================================
func SetupThreePointLighting(camera *Camera) *LightingSystem {
	ls := NewLightingSystem(camera)

	// Minimal ambient
	ls.AmbientLight = Color{20, 20, 25}
	ls.AmbientIntensity = 0.1

	// Key Light (main light, brightest, from front-side)
	keyLight := NewLight(40, 30, -30, ColorWhite, 1.2)
	ls.AddLight(keyLight)

	// Fill Light (softer, opposite side, fills shadows)
	fillLight := NewLight(-30, 20, -20, Color{200, 210, 220}, 0.4)
	ls.AddLight(fillLight)

	// Rim Light (back light, creates edge highlight)
	rimLight := NewLight(0, 25, 40, Color{230, 240, 255}, 0.7)
	ls.AddLight(rimLight)

	return ls
}

// ============================================================================
// SCENARIO 4: COLORED LIGHTS - RGB mixing demonstration
// ============================================================================
func SetupColoredLighting(camera *Camera) *LightingSystem {
	ls := NewLightingSystem(camera)

	// Low ambient
	ls.AmbientLight = Color{10, 10, 15}
	ls.AmbientIntensity = 0.1

	// Red light from right
	redLight := NewLight(40, 20, 0, ColorRed, 0.9)
	ls.AddLight(redLight)

	// Green light from left
	greenLight := NewLight(-40, 20, 0, ColorGreen, 0.9)
	ls.AddLight(greenLight)

	// Blue light from above
	blueLight := NewLight(0, 50, -20, ColorBlue, 0.7)
	ls.AddLight(blueLight)

	return ls
}

// ============================================================================
// SCENARIO 5: DYNAMIC LIGHTING - Moving/animated lights
// ============================================================================
func SetupDynamicLighting(camera *Camera) *LightingSystem {
	ls := NewLightingSystem(camera)

	// Medium ambient
	ls.AmbientLight = Color{40, 40, 50}
	ls.AmbientIntensity = 0.2

	// Multiple moving lights (will be animated)
	// Light 1 - Orange (will orbit)
	light1 := NewLight(30, 20, 0, ColorOrange, 0.8)
	ls.AddLight(light1)

	// Light 2 - Cyan (will orbit opposite direction)
	light2 := NewLight(-30, 20, 0, ColorCyan, 0.8)
	ls.AddLight(light2)

	// Light 3 - Magenta (will move up/down)
	light3 := NewLight(0, 30, -20, ColorMagenta, 0.6)
	ls.AddLight(light3)

	return ls
}

// ============================================================================
// SCENARIO 6: NIGHT SCENE - Dark, focused lighting
// ============================================================================
func SetupNightLighting(camera *Camera) *LightingSystem {
	ls := NewLightingSystem(camera)

	// Very low, cool ambient
	ls.AmbientLight = Color{10, 15, 25}
	ls.AmbientIntensity = 0.08

	// Moon light (cool, dim)
	moonLight := NewLight(-20, 60, -40, Color{150, 170, 200}, 0.4)
	ls.AddLight(moonLight)

	// Artificial light source (warm spot)
	spotLight := NewLight(15, 10, -15, Color{255, 200, 150}, 0.8)
	ls.AddLight(spotLight)

	return ls
}

// ============================================================================
// SCENARIO 7: OUTDOOR DAY - Natural sunlight
// ============================================================================
func SetupOutdoorLighting(camera *Camera) *LightingSystem {
	ls := NewLightingSystem(camera)

	// High, warm ambient (sky light)
	ls.AmbientLight = Color{120, 150, 200}
	ls.AmbientIntensity = 0.5

	// Strong sun (warm, high intensity)
	sunLight := NewLight(60, 80, -50, Color{255, 250, 240}, 1.8)
	ls.AddLight(sunLight)

	// Subtle sky fill from opposite side
	skyLight := NewLight(-30, 40, 20, Color{180, 200, 230}, 0.3)
	ls.AddLight(skyLight)

	return ls
}

// ============================================================================
// SCENARIO 8: DRAMATIC - High contrast, theater/stage lighting
// ============================================================================
func SetupDramaticLighting(camera *Camera) *LightingSystem {
	ls := NewLightingSystem(camera)

	// Very low ambient (creates drama)
	ls.AmbientLight = Color{5, 5, 8}
	ls.AmbientIntensity = 0.05

	// Strong key from steep angle
	keyLight := NewLight(30, 60, -20, ColorWhite, 2.0)
	ls.AddLight(keyLight)

	// Colored rim for separation
	rimLight := NewLight(-20, 30, 50, Color{255, 150, 100}, 0.8)
	ls.AddLight(rimLight)

	return ls
}

// ============================================================================
// SCENARIO 9: SOFT DIFFUSE - Product photography style
// ============================================================================
func SetupSoftLighting(camera *Camera) *LightingSystem {
	ls := NewLightingSystem(camera)

	// High, neutral ambient
	ls.AmbientLight = Color{180, 180, 180}
	ls.AmbientIntensity = 0.6

	// Multiple soft, evenly distributed lights
	light1 := NewLight(30, 30, -30, Color{240, 240, 240}, 0.5)
	ls.AddLight(light1)

	light2 := NewLight(-30, 30, -30, Color{240, 240, 240}, 0.5)
	ls.AddLight(light2)

	light3 := NewLight(0, 40, 20, Color{240, 240, 240}, 0.3)
	ls.AddLight(light3)

	return ls
}

// ============================================================================
// SCENARIO 10: SUNSET/SUNRISE - Warm gradient lighting
// ============================================================================
func SetupSunsetLighting(camera *Camera) *LightingSystem {
	ls := NewLightingSystem(camera)

	// Warm, orange ambient
	ls.AmbientLight = Color{100, 60, 40}
	ls.AmbientIntensity = 0.4

	// Low-angle sun (very warm)
	sunLight := NewLight(80, 15, -40, Color{255, 150, 80}, 1.5)
	ls.AddLight(sunLight)

	// Cool sky light from opposite (blue hour)
	skyLight := NewLight(-40, 40, 20, Color{100, 120, 180}, 0.3)
	ls.AddLight(skyLight)

	return ls
}

// ============================================================================
// LIGHTING SCENARIO SELECTOR
// ============================================================================

// GetLightingScenario returns the appropriate lighting setup for a demo
func GetLightingScenario(demoIndex int, camera *Camera) *LightingSystem {
	// Map demo indices to lighting scenarios
	switch demoIndex {
	case 0: // Basic Geometry
		return SetupSoftLighting(camera)
	case 1: // Mesh Generators
		return SetupThreePointLighting(camera)
	case 2: // Lighting Showcase - Special handling
		return SetupLightingShowcase(camera)
	case 3: // Material Showcase
		return SetupThreePointLighting(camera)
	case 4: // Transform Hierarchy
		return SetupOutdoorLighting(camera)
	case 5: // LOD System
		return SetupDirectionalLighting(camera)
	case 6: // Spatial Partitioning
		return SetupColoredLighting(camera)
	case 7: // Collision Physics
		return SetupDynamicLighting(camera)
	case 8: // Advanced Rendering
		return SetupDramaticLighting(camera)
	case 9: // Performance Test
		return SetupDirectionalLighting(camera)
	default:
		return SetupThreePointLighting(camera)
	}
}

// SetupLightingShowcase creates multiple isolated lighting setups
// for the lighting demo (each sphere has its own lights)
func SetupLightingShowcase(camera *Camera) *LightingSystem {
	// For the lighting showcase, we'll use a neutral base
	// and rely on the demo logic to show different lighting
	ls := NewLightingSystem(camera)

	// Setup will depend on which sphere the camera is looking at
	// For now, create a versatile setup
	ls.AmbientLight = Color{30, 30, 35}
	ls.AmbientIntensity = 0.15

	// Add lights that will be positioned/modified per sphere
	// These will be animated or repositioned in the demo
	light1 := NewLight(40, 30, -30, ColorWhite, 1.0)
	ls.AddLight(light1)

	light2 := NewLight(-30, 20, -20, Color{200, 210, 220}, 0.5)
	ls.AddLight(light2)

	light3 := NewLight(0, 30, 30, Color{220, 230, 255}, 0.6)
	ls.AddLight(light3)

	return ls
}

// ============================================================================
// LIGHTING ANIMATION HELPERS
// ============================================================================

// AnimateDynamicLights updates moving lights in dynamic scenarios
func AnimateDynamicLights(lightingSystem *LightingSystem, time float64) {
	if lightingSystem == nil || len(lightingSystem.Lights) < 3 {
		return
	}

	// Orbit light 1 around scene
	radius := 40.0
	angle1 := time * 0.5
	lightingSystem.Lights[0].Position.X = radius * math.Cos(angle1)
	lightingSystem.Lights[0].Position.Z = radius * math.Sin(angle1)

	// Orbit light 2 in opposite direction
	angle2 := -time * 0.5
	lightingSystem.Lights[1].Position.X = radius * math.Cos(angle2)
	lightingSystem.Lights[1].Position.Z = radius * math.Sin(angle2)

	// Pulse light 3 up and down
	lightingSystem.Lights[2].Position.Y = 30 + math.Sin(time*1.5)*15
	lightingSystem.Lights[2].Intensity = 0.6 + math.Sin(time*2.0)*0.2
}

// GetLightingScenarioName returns a descriptive name for logging
func GetLightingScenarioName(demoIndex int) string {
	names := []string{
		"Soft Lighting (Product)",
		"Three-Point Lighting (Studio)",
		"Mixed Scenarios (Showcase)",
		"Three-Point Lighting (Studio)",
		"Outdoor Lighting (Day)",
		"Directional Lighting (Sun)",
		"Colored Lighting (RGB Mix)",
		"Dynamic Lighting (Animated)",
		"Dramatic Lighting (High Contrast)",
		"Directional Lighting (Sun)",
	}

	if demoIndex >= 0 && demoIndex < len(names) {
		return names[demoIndex]
	}
	return "Three-Point Lighting (Default)"
}

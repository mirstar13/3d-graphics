package main

import "math"

// Light represents a light source in 3D space
type Light struct {
	Position  Point   // Position in world space
	Color     Color   // Light color
	Intensity float64 // Light intensity (0.0 to 1.0+)
	IsEnabled bool    // Whether this light is active
}

// Material defines surface properties for lighting
type Material struct {
	DiffuseColor     Color
	SpecularColor    Color
	Shininess        float64
	SpecularStrength float64
	AmbientStrength  float64
	Wireframe        bool
	WireframeColor   Color
}

// LightingSystem manages all lights and performs lighting calculations
type LightingSystem struct {
	Lights           []*Light
	AmbientLight     Color   // Global ambient light color
	AmbientIntensity float64 // Global ambient intensity
	Camera           *Camera // Reference to camera for view-dependent calculations
	UseBlinnPhong    bool    // Use Blinn-Phong instead of Phong
}

// NewLight creates a new light source
func NewLight(x, y, z float64, color Color, intensity float64) *Light {
	return &Light{
		Position:  Point{X: x, Y: y, Z: z},
		Color:     color,
		Intensity: intensity,
		IsEnabled: true,
	}
}

// NewMaterial creates a material with default properties
func NewMaterial() Material {
	return Material{
		DiffuseColor:     ColorWhite,
		SpecularColor:    ColorWhite,
		Shininess:        32.0,
		SpecularStrength: 0.5,
		AmbientStrength:  0.1,
		Wireframe:        false,
		WireframeColor:   ColorWhite,
	}
}

func NewWireframeMaterial(color Color) Material {
	m := NewMaterial()
	m.Wireframe = true
	m.WireframeColor = color
	return m
}

// NewLightingSystem creates a new lighting system
func NewLightingSystem(camera *Camera) *LightingSystem {
	return &LightingSystem{
		Lights:           make([]*Light, 0),
		AmbientLight:     Color{30, 30, 40}, // Slight blue ambient
		AmbientIntensity: 0.15,
		Camera:           camera,
		UseBlinnPhong:    true,
	}
}

// AddLight adds a light to the system
func (ls *LightingSystem) AddLight(light *Light) {
	ls.Lights = append(ls.Lights, light)
}

// SetCamera updates the camera reference
func (ls *LightingSystem) SetCamera(camera *Camera) {
	ls.Camera = camera
}

// CalculateLighting computes the final color for a surface point
// using Phong or Blinn-Phong lighting model with multiple lights
func (ls *LightingSystem) CalculateLighting(
	surfacePoint Point,
	normal Point,
	material Material,
	ambientOcclusion float64,
) Color {
	// Clamp AO to valid range
	if ambientOcclusion < 0 {
		ambientOcclusion = 0
	}
	if ambientOcclusion > 1 {
		ambientOcclusion = 1
	}

	// Start with ambient light
	ambientR := float64(ls.AmbientLight.R) * ls.AmbientIntensity * material.AmbientStrength * ambientOcclusion
	ambientG := float64(ls.AmbientLight.G) * ls.AmbientIntensity * material.AmbientStrength * ambientOcclusion
	ambientB := float64(ls.AmbientLight.B) * ls.AmbientIntensity * material.AmbientStrength * ambientOcclusion

	totalR := ambientR
	totalG := ambientG
	totalB := ambientB

	// Normalize the normal vector (should already be normalized, but ensure it)
	nx, ny, nz := normalizeVector(normal.X, normal.Y, normal.Z)

	// Calculate view direction (from surface to camera)
	var viewDirX, viewDirY, viewDirZ float64
	if ls.Camera != nil {
		viewDirX, viewDirY, viewDirZ = ls.Camera.GetViewDirection(surfacePoint)
	} else {
		// Fallback if no camera is set
		viewDirX = 0.0 - surfacePoint.X
		viewDirY = 0.0 - surfacePoint.Y
		viewDirZ = DEFAULT_CAMERA_Z - surfacePoint.Z
		viewDirX, viewDirY, viewDirZ = normalizeVector(viewDirX, viewDirY, viewDirZ)
	}

	// Accumulate contributions from each light
	for _, light := range ls.Lights {
		if !light.IsEnabled {
			continue
		}

		// Light direction (from surface to light)
		lightDirX := light.Position.X - surfacePoint.X
		lightDirY := light.Position.Y - surfacePoint.Y
		lightDirZ := light.Position.Z - surfacePoint.Z

		// Calculate distance for attenuation
		distance := math.Sqrt(lightDirX*lightDirX + lightDirY*lightDirY + lightDirZ*lightDirZ)

		// Prevent division by zero
		if distance < 0.001 {
			distance = 0.001
		}

		lightDirX, lightDirY, lightDirZ = normalizeVector(lightDirX, lightDirY, lightDirZ)

		// --- DIFFUSE COMPONENT (Lambertian) ---
		diffuseIntensity := dotProduct(nx, ny, nz, lightDirX, lightDirY, lightDirZ)

		// Clamp to positive values (surface facing light)
		if diffuseIntensity < 0 {
			diffuseIntensity = 0
		}

		// Apply light attenuation (inverse square law)
		attenuation := 1.0 / (ATTENUATION_CONSTANT + ATTENUATION_LINEAR*distance + ATTENUATION_QUADRATIC*distance*distance)

		// Clamp attenuation to reasonable range
		if attenuation > 1.0 {
			attenuation = 1.0
		}
		if attenuation < 0 {
			attenuation = 0
		}

		diffuseIntensity *= light.Intensity * attenuation

		// Add diffuse contribution
		diffuseR := diffuseIntensity * float64(material.DiffuseColor.R) * float64(light.Color.R) / 255.0
		diffuseG := diffuseIntensity * float64(material.DiffuseColor.G) * float64(light.Color.G) / 255.0
		diffuseB := diffuseIntensity * float64(material.DiffuseColor.B) * float64(light.Color.B) / 255.0

		totalR += diffuseR
		totalG += diffuseG
		totalB += diffuseB

		// --- SPECULAR COMPONENT (Phong or Blinn-Phong) ---
		var specularIntensity float64

		if ls.UseBlinnPhong {
			// Blinn-Phong: use halfway vector
			halfwayX := lightDirX + viewDirX
			halfwayY := lightDirY + viewDirY
			halfwayZ := lightDirZ + viewDirZ
			halfwayX, halfwayY, halfwayZ = normalizeVector(halfwayX, halfwayY, halfwayZ)

			specularIntensity = dotProduct(nx, ny, nz, halfwayX, halfwayY, halfwayZ)
		} else {
			// Phong: use reflection vector
			// R = 2(N·L)N - L
			nDotL := dotProduct(nx, ny, nz, lightDirX, lightDirY, lightDirZ)
			reflectX := 2*nDotL*nx - lightDirX
			reflectY := 2*nDotL*ny - lightDirY
			reflectZ := 2*nDotL*nz - lightDirZ

			specularIntensity = dotProduct(reflectX, reflectY, reflectZ, viewDirX, viewDirY, viewDirZ)
		}

		// Clamp specular to positive values
		if specularIntensity < 0 {
			specularIntensity = 0
		}

		// Apply shininess (specular exponent)
		specularIntensity = math.Pow(specularIntensity, material.Shininess)
		specularIntensity *= material.SpecularStrength * light.Intensity * attenuation

		// Additional clamp to prevent over-bright specular
		if specularIntensity > 1.0 {
			specularIntensity = 1.0
		}

		// Add specular contribution
		specularR := specularIntensity * float64(material.SpecularColor.R) * float64(light.Color.R) / 255.0
		specularG := specularIntensity * float64(material.SpecularColor.G) * float64(light.Color.G) / 255.0
		specularB := specularIntensity * float64(material.SpecularColor.B) * float64(light.Color.B) / 255.0

		totalR += specularR
		totalG += specularG
		totalB += specularB
	}

	// Clamp final values to valid color range [0, 255]
	if totalR < 0 {
		totalR = 0
	}
	if totalR > 255 {
		totalR = 255
	}

	if totalG < 0 {
		totalG = 0
	}
	if totalG > 255 {
		totalG = 255
	}

	if totalB < 0 {
		totalB = 0
	}
	if totalB > 255 {
		totalB = 255
	}

	return Color{
		R: uint8(totalR),
		G: uint8(totalG),
		B: uint8(totalB),
	}
}

// CalculateSimpleAO calculates a simple ambient occlusion term
// based on the angle between the normal and "up" direction
// This is a very simplified approximation - real AO would require ray tracing
func CalculateSimpleAO(normal Point) float64 {
	// Surfaces facing up get more ambient light, surfaces in crevices get less
	nx, ny, nz := normalizeVector(normal.X, normal.Y, normal.Z)

	// Up vector
	upX, upY, upZ := 0.0, 1.0, 0.0

	// How much does this surface face up?
	upFacing := dotProduct(nx, ny, nz, upX, upY, upZ)

	// Map from [-1, 1] to [AO_MIN, AO_MAX] so we don't get completely black
	ao := AO_MIN + (AO_MAX-AO_MIN)*((upFacing+1.0)/2.0)

	return ao
}

// RotateLight rotates a light around an axis (for animated lights)
func (l *Light) Rotate(axis byte, angle float64) {
	l.Position.Rotate(axis, angle)
}

package terminal

import "github.com/mirstar13/go-3d-graphics/internal/engine"

// shadeFragment evaluates lighting once for a covered triangle/cell.
func (r *TerminalRenderer) shadeFragment(t *engine.Triangle, camera *engine.Camera, normal, pixelWorldPos engine.Point, u, v float64) engine.Color {
	material := t.Material
	if r.cheapShading {
		color := material.GetDiffuseColor(u, v)
		factor := .55 + .45*max(0, normal.Y)
		return engine.Color{R: uint8(float64(color.R) * factor), G: uint8(float64(color.G) * factor), B: uint8(float64(color.B) * factor)}
	}
	var pixelColor engine.Color
	if r.LightingSystem != nil {
		if pbrMat, ok := material.(*engine.PBRMaterial); ok {
			shadowCb := func(l *engine.Light, p engine.Point) float64 {
				if r.ShadowRenderer != nil {
					if sm := r.ShadowRenderer.ShadowMaps[l]; sm != nil {
						return sm.CalculateShadow(p)
					}
				}
				return 1.0
			}

			viewDirX, viewDirY, viewDirZ := camera.GetViewDirection(pixelWorldPos)
			viewDir := engine.Point{X: viewDirX, Y: viewDirY, Z: viewDirZ}

			pixelColor = engine.CalculatePBRLightingWithUV(pixelWorldPos, normal, viewDir, pbrMat, r.LightingSystem.Lights, r.LightingSystem.AmbientLight, r.LightingSystem.AmbientIntensity, u, v, shadowCb)
		} else {
			// Standard Lighting with Shadows & Textures
			shadowFactor := 1.0
			if r.ShadowRenderer != nil && len(r.LightingSystem.Lights) > 0 {
				for _, l := range r.LightingSystem.Lights {
					if l.IsEnabled {
						if sm := r.ShadowRenderer.ShadowMaps[l]; sm != nil {
							shadowFactor = sm.CalculateShadow(pixelWorldPos)
							break
						}
					}
				}
			}

			ao := engine.CalculateSimpleAO(normal)

			if texMat, ok := material.(*engine.TexturedMaterial); ok && t.HasUVs && texMat.UseTextures {
				litColor := r.LightingSystem.CalculateLighting(pixelWorldPos, normal, material, ao)
				texColor := texMat.SampleDiffuse(u, v)
				pixelColor = engine.Color{
					R: uint8(float64(litColor.R) * float64(texColor.R) / 255.0),
					G: uint8(float64(litColor.G) * float64(texColor.G) / 255.0),
					B: uint8(float64(litColor.B) * float64(texColor.B) / 255.0),
				}
			} else {
				pixelColor = r.LightingSystem.CalculateLighting(pixelWorldPos, normal, material, ao)
			}

			if shadowFactor < 1.0 {
				pixelColor = engine.Color{
					R: uint8(float64(pixelColor.R) * (0.2 + 0.8*shadowFactor)),
					G: uint8(float64(pixelColor.G) * (0.2 + 0.8*shadowFactor)),
					B: uint8(float64(pixelColor.B) * (0.2 + 0.8*shadowFactor)),
				}
			}
		}
	} else {
		pixelColor = r.simpleLighting(normal, material)
	}
	return pixelColor
}

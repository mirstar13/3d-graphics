package main

import "math"

// PBRMaterial implements Physically Based Rendering material
type PBRMaterial struct {
	Albedo    Color   // Base color
	Metallic  float64 // 0 = dielectric, 1 = metal
	Roughness float64 // 0 = smooth, 1 = rough
	AO        float64 // Ambient occlusion (0-1)

	// Optional textures
	AlbedoMap    *Texture
	MetallicMap  *Texture
	RoughnessMap *Texture
	NormalMap    *Texture
	AOMap        *Texture

	// Texture settings
	UseTextures   bool
	TextureFilter TextureFilter
	TextureWrap   TextureWrap

	// Compatibility with basic material
	SpecularColor  Color
	Wireframe      bool
	WireframeColor Color
}

func NewPBRMaterial() *PBRMaterial {
	return &PBRMaterial{
		Albedo:         ColorWhite,
		Metallic:       0.0,
		Roughness:      0.5,
		AO:             1.0,
		UseTextures:    false,
		TextureFilter:  FilterLinear,
		TextureWrap:    WrapRepeat,
		SpecularColor:  ColorWhite,
		Wireframe:      false,
		WireframeColor: ColorWhite,
	}
}

func (pbr *PBRMaterial) GetType() MaterialType {
	return MaterialTypePBR
}

func (pbr *PBRMaterial) GetDiffuseColor(u, v float64) Color {
	if pbr.UseTextures && pbr.AlbedoMap != nil {
		return pbr.AlbedoMap.Sample(u, v, pbr.TextureFilter, pbr.TextureWrap)
	}
	return pbr.Albedo
}

func (pbr *PBRMaterial) GetSpecularColor() Color {
	return pbr.SpecularColor
}

func (pbr *PBRMaterial) GetShininess() float64 {
	// Convert roughness to shininess approximation
	return (1.0 - pbr.Roughness) * 128.0
}

func (pbr *PBRMaterial) GetSpecularStrength() float64 {
	return 1.0 - pbr.Roughness
}

func (pbr *PBRMaterial) GetAmbientStrength() float64 {
	return pbr.AO
}

func (pbr *PBRMaterial) IsWireframe() bool {
	return pbr.Wireframe
}

func (pbr *PBRMaterial) GetWireframeColor() Color {
	return pbr.WireframeColor
}

func (pbr *PBRMaterial) GetMetallic() float64 {
	return pbr.Metallic
}

func (pbr *PBRMaterial) GetRoughness() float64 {
	return pbr.Roughness
}

func (pbr *PBRMaterial) HasDiffuseTexture() bool {
	return pbr.UseTextures && pbr.AlbedoMap != nil
}

func (pbr *PBRMaterial) HasNormalMap() bool {
	return pbr.UseTextures && pbr.NormalMap != nil
}

func (pbr *PBRMaterial) HasSpecularMap() bool {
	return pbr.UseTextures && pbr.RoughnessMap != nil
}

func (pbr *PBRMaterial) SampleDiffuse(u, v float64) Color {
	return pbr.GetDiffuseColor(u, v)
}

func (pbr *PBRMaterial) SampleNormal(u, v float64) Point {
	if pbr.HasNormalMap() {
		normalColor := pbr.NormalMap.Sample(u, v, pbr.TextureFilter, pbr.TextureWrap)
		return UnpackNormalMap(normalColor)
	}
	return Point{X: 0, Y: 0, Z: 1}
}

func (pbr *PBRMaterial) SampleSpecular(u, v float64) float64 {
	return pbr.GetSpecularStrength()
}

func (pbr *PBRMaterial) SampleMetallic(u, v float64) float64 {
	if pbr.UseTextures && pbr.MetallicMap != nil {
		metalColor := pbr.MetallicMap.Sample(u, v, pbr.TextureFilter, pbr.TextureWrap)
		return float64(metalColor.R) / 255.0
	}
	return pbr.Metallic
}

func (pbr *PBRMaterial) SampleRoughness(u, v float64) float64 {
	if pbr.UseTextures && pbr.RoughnessMap != nil {
		roughColor := pbr.RoughnessMap.Sample(u, v, pbr.TextureFilter, pbr.TextureWrap)
		return float64(roughColor.R) / 255.0
	}
	return pbr.Roughness
}

func (pbr *PBRMaterial) SampleAO(u, v float64) float64 {
	if pbr.UseTextures && pbr.AOMap != nil {
		aoColor := pbr.AOMap.Sample(u, v, pbr.TextureFilter, pbr.TextureWrap)
		return float64(aoColor.R) / 255.0
	}
	return pbr.AO
}

// PBR Lighting Calculations

// DistributionGGX calculates the normal distribution function (NDF)
func DistributionGGX(N, H Point, roughness float64) float64 {
	a := roughness * roughness
	a2 := a * a
	NdotH := math.Max(dotProduct(N.X, N.Y, N.Z, H.X, H.Y, H.Z), 0.0)
	NdotH2 := NdotH * NdotH

	num := a2
	denom := NdotH2*(a2-1.0) + 1.0
	denom = math.Pi * denom * denom

	return num / math.Max(denom, 0.0000001)
}

// GeometrySchlickGGX calculates geometry obstruction
func GeometrySchlickGGX(NdotV, roughness float64) float64 {
	r := roughness + 1.0
	k := (r * r) / 8.0

	num := NdotV
	denom := NdotV*(1.0-k) + k

	return num / math.Max(denom, 0.0000001)
}

// GeometrySmith calculates the geometry function
func GeometrySmith(N, V, L Point, roughness float64) float64 {
	NdotV := math.Max(dotProduct(N.X, N.Y, N.Z, V.X, V.Y, V.Z), 0.0)
	NdotL := math.Max(dotProduct(N.X, N.Y, N.Z, L.X, L.Y, L.Z), 0.0)
	ggx2 := GeometrySchlickGGX(NdotV, roughness)
	ggx1 := GeometrySchlickGGX(NdotL, roughness)

	return ggx1 * ggx2
}

// FresnelSchlick calculates the Fresnel reflection
func FresnelSchlick(cosTheta float64, F0 Point) Point {
	power := math.Pow(1.0-cosTheta, 5.0)
	return Point{
		X: F0.X + (1.0-F0.X)*power,
		Y: F0.Y + (1.0-F0.Y)*power,
		Z: F0.Z + (1.0-F0.Z)*power,
	}
}

// CalculatePBRLighting computes lighting using Cook-Torrance BRDF
func CalculatePBRLighting(
	surfacePoint Point,
	normal Point,
	viewDir Point,
	material *PBRMaterial,
	lights []*Light,
	ambientLight Color,
	ambientIntensity float64,
) Color {
	return CalculatePBRLightingWithUV(surfacePoint, normal, viewDir, material, lights, ambientLight, ambientIntensity, 0, 0, nil)
}

// CalculatePBRLightingWithUV computes lighting with UV support and shadows
func CalculatePBRLightingWithUV(
	surfacePoint Point,
	normal Point,
	viewDir Point,
	material *PBRMaterial,
	lights []*Light,
	ambientLight Color,
	ambientIntensity float64,
	u, v float64,
	shadowCallback func(*Light, Point) float64,
) Color {
	// Sample material properties
	albedo := material.GetDiffuseColor(u, v)
	metallic := material.SampleMetallic(u, v)
	roughness := material.SampleRoughness(u, v)
	ao := material.SampleAO(u, v)

	// Calculate F0 (base reflectivity)
	// Dielectrics have F0 around 0.04, metals use albedo as F0
	F0 := Point{X: 0.04, Y: 0.04, Z: 0.04}
	if metallic > 0 {
		F0.X = float64(albedo.R)/255.0*(metallic) + F0.X*(1.0-metallic)
		F0.Y = float64(albedo.G)/255.0*(metallic) + F0.Y*(1.0-metallic)
		F0.Z = float64(albedo.B)/255.0*(metallic) + F0.Z*(1.0-metallic)
	}

	// Normalize view direction
	viewDir.X, viewDir.Y, viewDir.Z = normalizeVector(viewDir.X, viewDir.Y, viewDir.Z)

	// Accumulate radiance
	Lo := Point{X: 0, Y: 0, Z: 0}

	for _, light := range lights {
		if !light.IsEnabled {
			continue
		}

		// Calculate light direction
		lightDir := Point{
			X: light.Position.X - surfacePoint.X,
			Y: light.Position.Y - surfacePoint.Y,
			Z: light.Position.Z - surfacePoint.Z,
		}
		distance := math.Sqrt(lightDir.X*lightDir.X + lightDir.Y*lightDir.Y + lightDir.Z*lightDir.Z)
		lightDir.X, lightDir.Y, lightDir.Z = normalizeVector(lightDir.X, lightDir.Y, lightDir.Z)

		// Half vector
		H := Point{
			X: (viewDir.X + lightDir.X) / 2.0,
			Y: (viewDir.Y + lightDir.Y) / 2.0,
			Z: (viewDir.Z + lightDir.Z) / 2.0,
		}
		H.X, H.Y, H.Z = normalizeVector(H.X, H.Y, H.Z)

		// Attenuation
		attenuation := 1.0 / (distance * distance)

		// Shadow factor
		shadow := 1.0
		if shadowCallback != nil {
			shadow = shadowCallback(light, surfacePoint)
		}

		radiance := Point{
			X: float64(light.Color.R) / 255.0 * light.Intensity * attenuation * shadow,
			Y: float64(light.Color.G) / 255.0 * light.Intensity * attenuation * shadow,
			Z: float64(light.Color.B) / 255.0 * light.Intensity * attenuation * shadow,
		}

		// Cook-Torrance BRDF
		NDF := DistributionGGX(normal, H, roughness)
		G := GeometrySmith(normal, viewDir, lightDir, roughness)
		F := FresnelSchlick(math.Max(dotProduct(H.X, H.Y, H.Z, viewDir.X, viewDir.Y, viewDir.Z), 0.0), F0)

		NdotL := math.Max(dotProduct(normal.X, normal.Y, normal.Z, lightDir.X, lightDir.Y, lightDir.Z), 0.0)
		NdotV := math.Max(dotProduct(normal.X, normal.Y, normal.Z, viewDir.X, viewDir.Y, viewDir.Z), 0.0)

		numerator := NDF * G
		denominator := 4.0 * NdotV * NdotL
		specular := numerator / math.Max(denominator, 0.0000001)

		// Energy conservation
		kS := F
		kD := Point{
			X: 1.0 - kS.X,
			Y: 1.0 - kS.Y,
			Z: 1.0 - kS.Z,
		}
		kD.X *= 1.0 - metallic
		kD.Y *= 1.0 - metallic
		kD.Z *= 1.0 - metallic

		// Add to outgoing radiance
		albedoNorm := Point{
			X: float64(albedo.R) / 255.0,
			Y: float64(albedo.G) / 255.0,
			Z: float64(albedo.B) / 255.0,
		}

		Lo.X += (kD.X*albedoNorm.X/math.Pi + specular*kS.X) * radiance.X * NdotL
		Lo.Y += (kD.Y*albedoNorm.Y/math.Pi + specular*kS.Y) * radiance.Y * NdotL
		Lo.Z += (kD.Z*albedoNorm.Z/math.Pi + specular*kS.Z) * radiance.Z * NdotL
	}

	// Add ambient
	ambient := Point{
		X: float64(ambientLight.R) / 255.0 * ambientIntensity,
		Y: float64(ambientLight.G) / 255.0 * ambientIntensity,
		Z: float64(ambientLight.B) / 255.0 * ambientIntensity,
	}
	albedoNorm := Point{
		X: float64(albedo.R) / 255.0,
		Y: float64(albedo.G) / 255.0,
		Z: float64(albedo.B) / 255.0,
	}

	finalColor := Point{
		X: (ambient.X*albedoNorm.X*ao + Lo.X),
		Y: (ambient.Y*albedoNorm.Y*ao + Lo.Y),
		Z: (ambient.Z*albedoNorm.Z*ao + Lo.Z),
	}

	// Tone mapping (simple Reinhard)
	finalColor.X = finalColor.X / (finalColor.X + 1.0)
	finalColor.Y = finalColor.Y / (finalColor.Y + 1.0)
	finalColor.Z = finalColor.Z / (finalColor.Z + 1.0)

	// Gamma correction (gamma = 2.2)
	gamma := 1.0 / 2.2
	finalColor.X = math.Pow(finalColor.X, gamma)
	finalColor.Y = math.Pow(finalColor.Y, gamma)
	finalColor.Z = math.Pow(finalColor.Z, gamma)

	// Convert to Color
	return Color{
		R: uint8(clampFloat(finalColor.X*255.0, 0, 255)),
		G: uint8(clampFloat(finalColor.Y*255.0, 0, 255)),
		B: uint8(clampFloat(finalColor.Z*255.0, 0, 255)),
	}
}

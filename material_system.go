package main

// MaterialType specifies the type of material
type MaterialType int

const (
	MaterialTypeBasic MaterialType = iota
	MaterialTypePBR
	MaterialTypeTextured
	MaterialTypeWireframe
)

// IMaterial is the unified interface for all material types
type IMaterial interface {
	GetType() MaterialType
	GetDiffuseColor(u, v float64) Color
	GetSpecularColor() Color
	GetShininess() float64
	GetSpecularStrength() float64
	GetAmbientStrength() float64
	IsWireframe() bool
	GetWireframeColor() Color

	// Extended for PBR
	GetMetallic() float64
	GetRoughness() float64

	// Texture support
	HasDiffuseTexture() bool
	HasNormalMap() bool
	HasSpecularMap() bool
	SampleDiffuse(u, v float64) Color
	SampleNormal(u, v float64) Point
	SampleSpecular(u, v float64) float64
}

// BasicMaterial wraps the existing Material struct to implement IMaterial
type BasicMaterial struct {
	Material
}

func NewBasicMaterial() *BasicMaterial {
	return &BasicMaterial{
		Material: NewMaterial(),
	}
}

func (bm *BasicMaterial) GetType() MaterialType {
	return MaterialTypeBasic
}

func (bm *BasicMaterial) GetDiffuseColor(u, v float64) Color {
	return bm.Material.DiffuseColor
}

func (bm *BasicMaterial) GetSpecularColor() Color {
	return bm.Material.SpecularColor
}

func (bm *BasicMaterial) GetShininess() float64 {
	return bm.Material.Shininess
}

func (bm *BasicMaterial) GetSpecularStrength() float64 {
	return bm.Material.SpecularStrength
}

func (bm *BasicMaterial) GetAmbientStrength() float64 {
	return bm.Material.AmbientStrength
}

func (bm *BasicMaterial) IsWireframe() bool {
	return bm.Material.Wireframe
}

func (bm *BasicMaterial) GetWireframeColor() Color {
	return bm.Material.WireframeColor
}

func (bm *BasicMaterial) GetMetallic() float64 {
	return 0.0
}

func (bm *BasicMaterial) GetRoughness() float64 {
	return 0.5
}

func (bm *BasicMaterial) HasDiffuseTexture() bool {
	return false
}

func (bm *BasicMaterial) HasNormalMap() bool {
	return false
}

func (bm *BasicMaterial) HasSpecularMap() bool {
	return false
}

func (bm *BasicMaterial) SampleDiffuse(u, v float64) Color {
	return bm.Material.DiffuseColor
}

func (bm *BasicMaterial) SampleNormal(u, v float64) Point {
	return Point{X: 0, Y: 0, Z: 1}
}

func (bm *BasicMaterial) SampleSpecular(u, v float64) float64 {
	return bm.Material.SpecularStrength
}

// TexturedMaterialExt extends TexturedMaterial to implement IMaterial
type TexturedMaterialExt struct {
	TexturedMaterial
}

func NewTexturedMaterialExt() *TexturedMaterialExt {
	return &TexturedMaterialExt{
		TexturedMaterial: NewTexturedMaterial(),
	}
}

func (tm *TexturedMaterialExt) GetType() MaterialType {
	return MaterialTypeTextured
}

func (tm *TexturedMaterialExt) GetDiffuseColor(u, v float64) Color {
	if tm.UseTextures && tm.DiffuseTexture != nil {
		return tm.DiffuseTexture.Sample(u, v, tm.TextureFilter, tm.TextureWrap)
	}
	return tm.Material.DiffuseColor
}

func (tm *TexturedMaterialExt) GetSpecularColor() Color {
	return tm.Material.SpecularColor
}

func (tm *TexturedMaterialExt) GetShininess() float64 {
	return tm.Material.Shininess
}

func (tm *TexturedMaterialExt) GetSpecularStrength() float64 {
	return tm.Material.SpecularStrength
}

func (tm *TexturedMaterialExt) GetAmbientStrength() float64 {
	return tm.Material.AmbientStrength
}

func (tm *TexturedMaterialExt) IsWireframe() bool {
	return tm.Material.Wireframe
}

func (tm *TexturedMaterialExt) GetWireframeColor() Color {
	return tm.Material.WireframeColor
}

func (tm *TexturedMaterialExt) GetMetallic() float64 {
	return 0.0
}

func (tm *TexturedMaterialExt) GetRoughness() float64 {
	return 0.5
}

func (tm *TexturedMaterialExt) HasDiffuseTexture() bool {
	return tm.UseTextures && tm.DiffuseTexture != nil
}

func (tm *TexturedMaterialExt) HasNormalMap() bool {
	return tm.UseTextures && tm.NormalMap != nil
}

func (tm *TexturedMaterialExt) HasSpecularMap() bool {
	return tm.UseTextures && tm.SpecularMap != nil
}

func (tm *TexturedMaterialExt) SampleDiffuse(u, v float64) Color {
	if tm.HasDiffuseTexture() {
		return tm.DiffuseTexture.Sample(u, v, tm.TextureFilter, tm.TextureWrap)
	}
	return tm.Material.DiffuseColor
}

func (tm *TexturedMaterialExt) SampleNormal(u, v float64) Point {
	if tm.HasNormalMap() {
		normalColor := tm.NormalMap.Sample(u, v, tm.TextureFilter, tm.TextureWrap)
		return UnpackNormalMap(normalColor)
	}
	return Point{X: 0, Y: 0, Z: 1}
}

func (tm *TexturedMaterialExt) SampleSpecular(u, v float64) float64 {
	if tm.HasSpecularMap() {
		specColor := tm.SpecularMap.Sample(u, v, tm.TextureFilter, tm.TextureWrap)
		return float64(specColor.R) / 255.0
	}
	return tm.Material.SpecularStrength
}

// UnpackNormalMap converts RGB color to tangent space normal
func UnpackNormalMap(color Color) Point {
	// Convert from [0,255] to [-1,1]
	x := (float64(color.R)/255.0)*2.0 - 1.0
	y := (float64(color.G)/255.0)*2.0 - 1.0
	z := (float64(color.B)/255.0)*2.0 - 1.0

	// Normalize
	length := (x*x + y*y + z*z)
	if length > 0 {
		invLength := 1.0 / length
		x *= invLength
		y *= invLength
		z *= invLength
	}

	return Point{X: x, Y: y, Z: z}
}

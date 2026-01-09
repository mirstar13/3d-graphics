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

func (m *Material) GetType() MaterialType {
	return MaterialTypeBasic
}

func (m *Material) GetDiffuseColor(u, v float64) Color {
	return m.DiffuseColor
}

func (m *Material) GetSpecularColor() Color {
	return m.SpecularColor
}

func (m *Material) GetShininess() float64 {
	return m.Shininess
}

func (m *Material) GetSpecularStrength() float64 {
	return m.SpecularStrength
}

func (m *Material) GetAmbientStrength() float64 {
	return m.AmbientStrength
}

func (m *Material) IsWireframe() bool {
	return m.Wireframe
}

func (m *Material) GetWireframeColor() Color {
	return m.WireframeColor
}

func (m *Material) GetMetallic() float64 {
	return 0.0
}

func (m *Material) GetRoughness() float64 {
	return 0.5
}

func (m *Material) HasDiffuseTexture() bool {
	return false
}

func (m *Material) HasNormalMap() bool {
	return false
}

func (m *Material) HasSpecularMap() bool {
	return false
}

func (m *Material) SampleDiffuse(u, v float64) Color {
	return m.DiffuseColor
}

func (m *Material) SampleNormal(u, v float64) Point {
	return Point{X: 0, Y: 0, Z: 1}
}

func (m *Material) SampleSpecular(u, v float64) float64 {
	return m.SpecularStrength
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
	return tm.DiffuseColor
}

func (tm *TexturedMaterialExt) GetSpecularColor() Color {
	return tm.SpecularColor
}

func (tm *TexturedMaterialExt) GetShininess() float64 {
	return tm.Shininess
}

func (tm *TexturedMaterialExt) GetSpecularStrength() float64 {
	return tm.SpecularStrength
}

func (tm *TexturedMaterialExt) GetAmbientStrength() float64 {
	return tm.AmbientStrength
}

func (tm *TexturedMaterialExt) IsWireframe() bool {
	return tm.Wireframe
}

func (tm *TexturedMaterialExt) GetWireframeColor() Color {
	return tm.WireframeColor
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
	return tm.DiffuseColor
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
	return tm.SpecularStrength
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

package main

import (
	"image"
	"image/color"
	"math"
)

// Texture represents a 2D texture map
type Texture struct {
	Width  int
	Height int
	Data   []Color
}

// TextureCoord represents UV coordinates
type TextureCoord struct {
	U, V float64
}

// TextureFilter specifies texture filtering mode
type TextureFilter int

const (
	FilterNearest TextureFilter = iota // Point sampling
	FilterLinear                       // Bilinear interpolation
)

// TextureWrap specifies texture wrapping mode
type TextureWrap int

const (
	WrapRepeat TextureWrap = iota
	WrapClamp
	WrapMirror
)

// NewTexture creates a texture from dimensions
func NewTexture(width, height int) *Texture {
	return &Texture{
		Width:  width,
		Height: height,
		Data:   make([]Color, width*height),
	}
}

// NewTextureFromImage creates a texture from an image
func NewTextureFromImage(img image.Image) *Texture {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	tex := NewTexture(width, height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()
			tex.Data[y*width+x] = Color{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
			}
		}
	}

	return tex
}

// SetPixel sets a pixel color at the given coordinates
func (t *Texture) SetPixel(x, y int, color Color) {
	if x >= 0 && x < t.Width && y >= 0 && y < t.Height {
		t.Data[y*t.Width+x] = color
	}
}

// GetPixel gets a pixel color at the given coordinates
func (t *Texture) GetPixel(x, y int) Color {
	if x >= 0 && x < t.Width && y >= 0 && y < t.Height {
		return t.Data[y*t.Width+x]
	}
	return ColorBlack
}

// Sample samples the texture at UV coordinates with filtering
func (t *Texture) Sample(u, v float64, filter TextureFilter, wrap TextureWrap) Color {
	// Apply wrapping
	u = t.applyWrap(u, wrap)
	v = t.applyWrap(v, wrap)

	switch filter {
	case FilterNearest:
		return t.sampleNearest(u, v)
	case FilterLinear:
		return t.sampleLinear(u, v)
	default:
		return t.sampleNearest(u, v)
	}
}

// sampleNearest performs nearest-neighbor sampling
func (t *Texture) sampleNearest(u, v float64) Color {
	x := int(u * float64(t.Width))
	y := int(v * float64(t.Height))

	// Clamp to valid range
	if x < 0 {
		x = 0
	}
	if x >= t.Width {
		x = t.Width - 1
	}
	if y < 0 {
		y = 0
	}
	if y >= t.Height {
		y = t.Height - 1
	}

	return t.Data[y*t.Width+x]
}

// sampleLinear performs bilinear interpolation
func (t *Texture) sampleLinear(u, v float64) Color {
	// Convert to texture space
	x := u*float64(t.Width) - 0.5
	y := v*float64(t.Height) - 0.5

	// Get integer coordinates
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	x1 := x0 + 1
	y1 := y0 + 1

	// Get fractional part
	fx := x - float64(x0)
	fy := y - float64(y0)

	// Clamp coordinates
	x0 = clampInt(x0, 0, t.Width-1)
	y0 = clampInt(y0, 0, t.Height-1)
	x1 = clampInt(x1, 0, t.Width-1)
	y1 = clampInt(y1, 0, t.Height-1)

	// Sample four nearest texels
	c00 := t.Data[y0*t.Width+x0]
	c10 := t.Data[y0*t.Width+x1]
	c01 := t.Data[y1*t.Width+x0]
	c11 := t.Data[y1*t.Width+x1]

	// Bilinear interpolation
	// First interpolate in X
	cx0 := lerpColor(c00, c10, fx)
	cx1 := lerpColor(c01, c11, fx)

	// Then interpolate in Y
	return lerpColor(cx0, cx1, fy)
}

// applyWrap applies texture wrapping mode
func (t *Texture) applyWrap(coord float64, wrap TextureWrap) float64 {
	switch wrap {
	case WrapRepeat:
		// Wrap to [0, 1]
		coord = math.Mod(coord, 1.0)
		if coord < 0 {
			coord += 1.0
		}
		return coord

	case WrapClamp:
		// Clamp to [0, 1]
		if coord < 0 {
			return 0
		}
		if coord > 1 {
			return 1
		}
		return coord

	case WrapMirror:
		// Mirror repeat
		coord = math.Mod(coord, 2.0)
		if coord < 0 {
			coord += 2.0
		}
		if coord > 1.0 {
			coord = 2.0 - coord
		}
		return coord

	default:
		return coord
	}
}

// lerpColor linearly interpolates between two colors
func lerpColor(c0, c1 Color, t float64) Color {
	return Color{
		R: uint8(float64(c0.R) + t*(float64(c1.R)-float64(c0.R))),
		G: uint8(float64(c0.G) + t*(float64(c1.G)-float64(c0.G))),
		B: uint8(float64(c0.B) + t*(float64(c1.B)-float64(c0.B))),
	}
}

// GenerateCheckerboard generates a procedural checkerboard texture
func GenerateCheckerboard(width, height, checkSize int, color1, color2 Color) *Texture {
	tex := NewTexture(width, height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			checkX := x / checkSize
			checkY := y / checkSize

			if (checkX+checkY)%2 == 0 {
				tex.Data[y*width+x] = color1
			} else {
				tex.Data[y*width+x] = color2
			}
		}
	}

	return tex
}

// GenerateGradient generates a gradient texture
func GenerateGradient(width, height int, color1, color2 Color, horizontal bool) *Texture {
	tex := NewTexture(width, height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var t float64
			if horizontal {
				t = float64(x) / float64(width-1)
			} else {
				t = float64(y) / float64(height-1)
			}

			tex.Data[y*width+x] = lerpColor(color1, color2, t)
		}
	}

	return tex
}

// GenerateNoise generates a simple noise texture
func GenerateNoise(width, height int, seed int64) *Texture {
	tex := NewTexture(width, height)

	// Simple pseudo-random noise
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Simple hash-based noise
			h := seed + int64(x*2654435761) + int64(y*2654435789)
			h = (h ^ (h >> 16)) * 0x45d9f3b
			h = (h ^ (h >> 16)) * 0x45d9f3b
			h = h ^ (h >> 16)

			val := uint8((h & 0xFF))
			tex.Data[y*width+x] = Color{R: val, G: val, B: val}
		}
	}

	return tex
}

// TexturedMaterial extends Material with texture support
type TexturedMaterial struct {
	Material       // Embed base material
	DiffuseTexture *Texture
	NormalMap      *Texture
	SpecularMap    *Texture
	UseTextures    bool
	TextureFilter  TextureFilter
	TextureWrap    TextureWrap
}

// NewTexturedMaterial creates a material with texture support
func NewTexturedMaterial() TexturedMaterial {
	return TexturedMaterial{
		Material:      NewMaterial(),
		UseTextures:   false,
		TextureFilter: FilterLinear,
		TextureWrap:   WrapRepeat,
	}
}

// TexturedTriangle extends Triangle with UV coordinates
type TexturedTriangle struct {
	Triangle
	UV0, UV1, UV2 TextureCoord
}

// NewTexturedTriangle creates a triangle with UV coordinates
func NewTexturedTriangle(p0, p1, p2 Point, uv0, uv1, uv2 TextureCoord) *TexturedTriangle {
	return &TexturedTriangle{
		Triangle: Triangle{
			P0:   p0,
			P1:   p1,
			P2:   p2,
			char: 'x',
		},
		UV0: uv0,
		UV1: uv1,
		UV2: uv2,
	}
}

// InterpolateUV interpolates UV coordinates using barycentric coordinates
func InterpolateUV(uv0, uv1, uv2 TextureCoord, bary0, bary1, bary2 float64) TextureCoord {
	return TextureCoord{
		U: uv0.U*bary0 + uv1.U*bary1 + uv2.U*bary2,
		V: uv0.V*bary0 + uv1.V*bary1 + uv2.V*bary2,
	}
}

// CalculateBarycentricCoords calculates barycentric coordinates for a point in a triangle
func CalculateBarycentricCoords(p, p0, p1, p2 Point) (float64, float64, float64) {
	v0x := p1.X - p0.X
	v0y := p1.Y - p0.Y
	v1x := p2.X - p0.X
	v1y := p2.Y - p0.Y
	v2x := p.X - p0.X
	v2y := p.Y - p0.Y

	d00 := v0x*v0x + v0y*v0y
	d01 := v0x*v1x + v0y*v1y
	d11 := v1x*v1x + v1y*v1y
	d20 := v2x*v0x + v2y*v0y
	d21 := v2x*v1x + v2y*v1y

	denom := d00*d11 - d01*d01
	if math.Abs(denom) < 1e-10 {
		return 1.0, 0.0, 0.0
	}

	v := (d11*d20 - d01*d21) / denom
	w := (d00*d21 - d01*d20) / denom
	u := 1.0 - v - w

	return u, v, w
}

// SampleTextureForPixel samples texture for a pixel coordinate
func SampleTextureForPixel(tri *TexturedTriangle, pixelX, pixelY float64, texture *Texture, filter TextureFilter, wrap TextureWrap) Color {
	// Calculate barycentric coordinates
	p := Point{X: pixelX, Y: pixelY, Z: 0}
	u, v, w := CalculateBarycentricCoords(p, tri.P0, tri.P1, tri.P2)

	// Interpolate UV
	uv := InterpolateUV(tri.UV0, tri.UV1, tri.UV2, u, v, w)

	// Sample texture
	return texture.Sample(uv.U, uv.V, filter, wrap)
}

// ConvertImageToTexture converts a Go image to our texture format
func ConvertImageToTexture(img image.Image) *Texture {
	return NewTextureFromImage(img)
}

// ToImage converts texture back to Go image
func (t *Texture) ToImage() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, t.Width, t.Height))

	for y := 0; y < t.Height; y++ {
		for x := 0; x < t.Width; x++ {
			c := t.Data[y*t.Width+x]
			img.Set(x, y, color.RGBA{
				R: c.R,
				G: c.G,
				B: c.B,
				A: 255,
			})
		}
	}

	return img
}

// GenerateMipmap generates a mipmap level (half size)
func (t *Texture) GenerateMipmap() *Texture {
	if t.Width <= 1 || t.Height <= 1 {
		return t
	}

	newWidth := t.Width / 2
	newHeight := t.Height / 2
	mipmap := NewTexture(newWidth, newHeight)

	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			// Average 2x2 block
			sx := x * 2
			sy := y * 2

			c00 := t.Data[sy*t.Width+sx]
			c10 := t.Data[sy*t.Width+sx+1]
			c01 := t.Data[(sy+1)*t.Width+sx]
			c11 := t.Data[(sy+1)*t.Width+sx+1]

			avgR := (uint16(c00.R) + uint16(c10.R) + uint16(c01.R) + uint16(c11.R)) / 4
			avgG := (uint16(c00.G) + uint16(c10.G) + uint16(c01.G) + uint16(c11.G)) / 4
			avgB := (uint16(c00.B) + uint16(c10.B) + uint16(c01.B) + uint16(c11.B)) / 4

			mipmap.Data[y*newWidth+x] = Color{
				R: uint8(avgR),
				G: uint8(avgG),
				B: uint8(avgB),
			}
		}
	}

	return mipmap
}

// MipmapChain represents a complete mipmap chain
type MipmapChain struct {
	Levels []*Texture
}

// GenerateMipmapChain generates a full mipmap chain
func GenerateMipmapChain(baseTexture *Texture) *MipmapChain {
	chain := &MipmapChain{
		Levels: []*Texture{baseTexture},
	}

	current := baseTexture
	for current.Width > 1 || current.Height > 1 {
		current = current.GenerateMipmap()
		chain.Levels = append(chain.Levels, current)
	}

	return chain
}

// SampleWithMipmap samples using appropriate mip level
func (mc *MipmapChain) Sample(u, v, mipLevel float64, filter TextureFilter, wrap TextureWrap) Color {
	// Clamp mip level
	if mipLevel < 0 {
		mipLevel = 0
	}
	if mipLevel >= float64(len(mc.Levels)) {
		mipLevel = float64(len(mc.Levels) - 1)
	}

	level := int(mipLevel)
	frac := mipLevel - float64(level)

	if level >= len(mc.Levels)-1 || frac < 0.001 {
		// Use single level
		return mc.Levels[level].Sample(u, v, filter, wrap)
	}

	// Trilinear filtering - interpolate between mip levels
	c0 := mc.Levels[level].Sample(u, v, filter, wrap)
	c1 := mc.Levels[level+1].Sample(u, v, filter, wrap)

	return lerpColor(c0, c1, frac)
}

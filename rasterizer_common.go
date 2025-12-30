package main

import "math"

// SurfaceRenderContext holds all the data needed for rendering a surface
type SurfaceRenderContext struct {
	Camera       *Camera
	Normal       Point
	SurfacePoint Point
	Material     Material
	DZ           float64
}

// CalculateSurfaceNormal calculates the normal vector for a surface
// given two edge vectors (u and v should be computed from surface vertices)
func CalculateSurfaceNormal(p0, p1, p2 *Point, setNormal *Point, useSetNormal bool) Point {
	if useSetNormal && setNormal != nil {
		nx, ny, nz := setNormal.X, setNormal.Y, setNormal.Z
		nx, ny, nz = normalizeVector(nx, ny, nz)
		return Point{X: nx, Y: ny, Z: nz}
	}

	// Calculate normal from vertices
	ux, uy, uz := subtract(p1, p0)
	vx, vy, vz := subtract(p2, p0)

	nx, ny, nz := crossProduct(ux, uy, uz, vx, vy, vz)
	nx, ny, nz = normalizeVector(nx, ny, nz)

	return Point{X: nx, Y: ny, Z: nz}
}

// IsBackfacing checks if a surface is facing away from the camera
func IsBackfacing(normal Point, surfacePoint Point, camera *Camera) bool {
	if camera == nil {
		return false
	}

	cameraDirX, cameraDirY, cameraDirZ := camera.GetCameraDirection(surfacePoint)
	facing := dotProduct(normal.X, normal.Y, normal.Z, cameraDirX, cameraDirY, cameraDirZ)

	// Cull if facing away from camera
	// facing < 0 means normal and camera direction align (front-facing) -> don't cull
	// facing > 0 means they point opposite directions (back-facing) -> cull
	return facing > 0
}

// CalculateSurfaceLighting computes the final color and character for a surface
// Returns (color, fillCharacter)
func CalculateSurfaceLighting(ctx SurfaceRenderContext) (Color, rune) {
	ao := CalculateSimpleAO(ctx.Normal)

	var pixelColor Color

	if ctx.Camera != nil {
		// Use camera's lighting system if available
		ls := ctx.Camera.Transform.Parent // This is a hack - we need to pass lighting system properly
		_ = ls                            // Unused for now
	}

	// For now, use simple lighting (we'll fix this in a future refactor)
	lx, ly, lz := -1.0, 1.0, -1.0
	lx, ly, lz = normalizeVector(lx, ly, lz)
	intensity := dotProduct(ctx.Normal.X, ctx.Normal.Y, ctx.Normal.Z, lx, ly, lz)
	if intensity < 0 {
		intensity = 0
	}

	// Apply AO
	intensity *= ao

	// Simple color based on material
	baseR := float64(ctx.Material.DiffuseColor.R)
	baseG := float64(ctx.Material.DiffuseColor.G)
	baseB := float64(ctx.Material.DiffuseColor.B)

	finalR := baseR * intensity
	finalG := baseG * intensity
	finalB := baseB * intensity

	// Clamp before converting to uint8
	if finalR < 0 {
		finalR = 0
	}
	if finalR > 255 {
		finalR = 255
	}
	if finalG < 0 {
		finalG = 0
	}
	if finalG > 255 {
		finalG = 255
	}
	if finalB < 0 {
		finalB = 0
	}
	if finalB > 255 {
		finalB = 255
	}

	pixelColor = Color{
		R: uint8(finalR),
		G: uint8(finalG),
		B: uint8(finalB),
	}

	// Calculate fill character based on brightness
	brightness := (float64(pixelColor.R) + float64(pixelColor.G) + float64(pixelColor.B)) / (3.0 * 255.0)
	index := int(brightness * float64(len(SHADING_RAMP)-1))
	if index < 0 {
		index = 0
	}
	if index >= len(SHADING_RAMP) {
		index = len(SHADING_RAMP) - 1
	}
	fillChar := rune(SHADING_RAMP[index])

	return pixelColor, fillChar
}

// RasterizeScanline fills a horizontal line with perspective-correct depth interpolation
// Uses 1/z for proper perspective interpolation
func RasterizeScanline(renderer *Renderer, y, xStart, xEnd int, zStart, zEnd float64, pixelColor Color, fillChar rune) {
	// Bounds check for y
	if y < 0 || y >= renderer.Height {
		return
	}

	// Early rejection if completely off-screen
	if (xStart < 0 && xEnd < 0) || (xStart >= renderer.Width && xEnd >= renderer.Width) {
		return
	}

	originalXStart := xStart
	originalXEnd := xEnd

	// Guard against zero/negative depths
	if zStart <= 0 {
		zStart = 0.001
	}
	if zEnd <= 0 {
		zEnd = 0.001
	}

	// Perspective-correct interpolation: use 1/z
	invZStart := 1.0 / zStart
	invZEnd := 1.0 / zEnd

	// Clamp to screen bounds
	if xStart < 0 {
		xStart = 0
	}
	if xEnd >= renderer.Width {
		xEnd = renderer.Width - 1
	}

	// Handle degenerate case
	if xStart > xEnd {
		return
	}

	// Draw pixels including the end pixel (<=) to avoid gaps
	for x := xStart; x <= xEnd; x++ {
		// Calculate interpolation parameter accounting for clamping
		t := 0.0
		if originalXEnd != originalXStart {
			t = float64(x-originalXStart) / float64(originalXEnd-originalXStart)
		}

		// Perspective-correct depth interpolation
		invZ := invZStart + t*(invZEnd-invZStart)
		z := 1.0 / invZ

		// Additional safety check
		if z <= 0 || math.IsNaN(z) || math.IsInf(z, 0) {
			continue
		}

		// Z-buffer test
		if z < renderer.ZBuffer[y][x] {
			if renderer.UseColor {
				renderer.Surface[y][x] = FILLED_CHAR
				renderer.ColorBuffer[y][x] = pixelColor
			} else {
				renderer.Surface[y][x] = fillChar
			}
			renderer.ZBuffer[y][x] = z
		}
	}
}

// ClipToScreen checks if a bounding box is completely outside screen bounds
func ClipToScreen(minX, maxX, minY, maxY, width, height int) bool {
	return maxX < 0 || minX >= width || maxY < 0 || minY >= height
}

// ProjectVertices projects multiple vertices using the camera
// Returns false if any projection fails (vertex behind camera)
func ProjectVertices(camera *Camera, points []Point, height, width int) ([]int, []int, []float64, bool) {
	if camera == nil {
		return nil, nil, nil, false
	}

	n := len(points)
	if n == 0 {
		return nil, nil, nil, false
	}

	xs := make([]int, n)
	ys := make([]int, n)
	zs := make([]float64, n)

	for i, p := range points {
		x, y, z := camera.ProjectPoint(p, height, width)
		if x == -1 {
			return nil, nil, nil, false
		}
		xs[i] = x
		ys[i] = y
		zs[i] = z
	}

	return xs, ys, zs, true
}

// SafeArrayAccess safely accesses an array with bounds checking
func SafeArrayAccess(arr []float64, index int, defaultVal float64) float64 {
	if index < 0 || index >= len(arr) {
		return defaultVal
	}
	return arr[index]
}

// SafeIntArrayAccess safely accesses an int array with bounds checking
func SafeIntArrayAccess(arr []int, index int, defaultVal int) int {
	if index < 0 || index >= len(arr) {
		return defaultVal
	}
	return arr[index]
}

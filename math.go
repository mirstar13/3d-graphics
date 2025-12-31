package main

import "math"

func normalize(height, width, x, y int) (int, int) {
	screenX := (width / 2) + (x * ASPECT_RATIO)
	screenY := (height / 2) - y

	return screenX, screenY
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func subtract(p1, p2 *Point) (float64, float64, float64) {
	return p1.X - p2.X, p1.Y - p2.Y, p1.Z - p2.Z
}

func crossProduct(ux, uy, uz, vx, vy, vz float64) (float64, float64, float64) {
	nx := uy*vz - uz*vy
	ny := uz*vx - ux*vz
	nz := ux*vy - uy*vx
	return nx, ny, nz
}

func dotProduct(nx, ny, nz, lx, ly, lz float64) float64 {
	return nx*lx + ny*ly + nz*lz
}

// normalizeVector normalizes a 3D vector with safety checks
func normalizeVector(x, y, z float64) (float64, float64, float64) {
	length := math.Sqrt(x*x + y*y + z*z)

	// Guard against zero-length vectors
	if length < 1e-10 {
		// Return a default "up" vector instead of zero
		return 0, 1, 0
	}

	return x / length, y / length, z / length
}

// Interpolate creates interpolated values between two points
// Returns empty slice for degenerate cases
func Interpolate(i0, d0, i1, d1 float64) []float64 {
	if math.Abs(i1-i0) < 1e-10 {
		return []float64{d0}
	}

	// Guard against very large ranges that could cause memory issues
	steps := int(math.Abs(i1 - i0))
	if steps > 10000 {
		// Clamp to reasonable limit
		steps = 10000
	}

	values := make([]float64, 0, steps+1)
	a := (d1 - d0) / (i1 - i0)
	d := d0

	for i := i0; i <= i1; i++ {
		values = append(values, d)
		d = d + a
	}

	return values
}

// clamp constrains a value between min and max
func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// clampInt constrains an integer value between min and max
func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// InterpolateInt interpolates integer values between two integer endpoints
func InterpolateInt(i0, d0, i1, d1 int) []int {
	if i0 == i1 {
		return []int{d0}
	}

	values := make([]int, 0, abs(i1-i0)+1)
	a := float64(d1-d0) / float64(i1-i0)
	d := float64(d0)

	for i := i0; i <= i1; i++ {
		values = append(values, int(d))
		d += a
	}

	return values
}

// InterpolateFloat interpolates float values between two integer y-coordinates
func InterpolateFloat(i0 int, d0 float64, i1 int, d1 float64) []float64 {
	if i0 == i1 {
		return []float64{d0}
	}

	values := make([]float64, 0, abs(i1-i0)+1)
	a := (d1 - d0) / float64(i1-i0)
	d := d0

	for i := i0; i <= i1; i++ {
		values = append(values, d)
		d += a
	}

	return values
}

// InterpolateFloatAcross interpolates across a horizontal scanline
func InterpolateFloatAcross(x0 int, d0 float64, x1 int, d1 float64) []float64 {
	if x0 == x1 {
		return []float64{d0}
	}

	values := make([]float64, 0, abs(x1-x0)+1)
	a := (d1 - d0) / float64(x1-x0)
	d := d0

	for x := x0; x <= x1; x++ {
		values = append(values, d)
		d += a
	}

	return values
}

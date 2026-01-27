package main

import (
	"testing"
)

func BenchmarkTransformPoint(b *testing.B) {
	// Setup a random matrix and point
	m := ComposeMatrix(
		Point{X: 10, Y: 20, Z: 30},
		QuaternionFromEuler(0.5, 0.5, 0.5),
		Point{X: 2, Y: 2, Z: 2},
	)
	p := Point{X: 100, Y: 200, Z: 300}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use the result to prevent compiler optimization
		res := m.TransformPoint(p)
		if res.X == 0 && res.Y == 0 && res.Z == 0 {
			// Unlikely but prevents unused variable check
			_ = res
		}
	}
}

func BenchmarkTransformPointAffine(b *testing.B) {
	// Setup a random matrix and point
	m := ComposeMatrix(
		Point{X: 10, Y: 20, Z: 30},
		QuaternionFromEuler(0.5, 0.5, 0.5),
		Point{X: 2, Y: 2, Z: 2},
	)
	p := Point{X: 100, Y: 200, Z: 300}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Expecting TransformPointAffine to be available
		res := m.TransformPointAffine(p)
		if res.X == 0 && res.Y == 0 && res.Z == 0 {
			_ = res
		}
	}
}

func TestTransformPointAffineCorrectness(t *testing.T) {
	// Setup a random matrix and point
	m := ComposeMatrix(
		Point{X: 10, Y: 20, Z: 30},
		QuaternionFromEuler(0.5, 0.5, 0.5),
		Point{X: 2, Y: 2, Z: 2},
	)

	// Test multiple points
	points := []Point{
		{X: 0, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
		{X: 0, Y: 1, Z: 0},
		{X: 0, Y: 0, Z: 1},
		{X: -5, Y: 10, Z: -20},
		{X: 100, Y: 200, Z: 300},
	}

	for _, p := range points {
		expected := m.TransformPoint(p)
		actual := m.TransformPointAffine(p)

		if absDiff(expected.X, actual.X) > 1e-9 ||
		   absDiff(expected.Y, actual.Y) > 1e-9 ||
		   absDiff(expected.Z, actual.Z) > 1e-9 {
			t.Errorf("Point %v mismatch: expected %v, got %v", p, expected, actual)
		}
	}
}

func absDiff(a, b float64) float64 {
	if a > b {
		return a - b
	}
	return b - a
}

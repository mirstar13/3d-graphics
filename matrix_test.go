package main

import (
	"testing"
)

func BenchmarkTransformPoint(b *testing.B) {
	m := IdentityMatrix()
	// Make it non-identity to be realistic
	m.M[0] = 2
	m.M[5] = 2
	m.M[10] = 2
	m.M[3] = 10
	m.M[7] = 20
	m.M[11] = 30

	p := Point{X: 1, Y: 2, Z: 3}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.TransformPoint(p)
	}
}

func BenchmarkTransformPointAffine(b *testing.B) {
	m := IdentityMatrix()
	// Make it non-identity to be realistic
	m.M[0] = 2
	m.M[5] = 2
	m.M[10] = 2
	m.M[3] = 10
	m.M[7] = 20
	m.M[11] = 30

	p := Point{X: 1, Y: 2, Z: 3}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.TransformPointAffine(p)
	}
}

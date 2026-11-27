package engine

import (
	"math"
	"testing"
)

func TestSpecularPowerMatchesPow(t *testing.T) {
	for _, exponent := range []float64{-1, 0, .5, 1, 2, 4, 8, 16, 32, 64, 128, 256, 19.5, 512} {
		for i := 0; i <= 1000; i++ {
			x := float64(i) / 1000
			got, want := specularPower(x, exponent), math.Pow(x, exponent)
			if got != want && math.Abs(got-want) > 3e-14*math.Max(math.Abs(want), math.SmallestNonzeroFloat64) {
				t.Fatalf("pow(%g,%g): got %.17g, want %.17g", x, exponent, got, want)
			}
		}
	}
}

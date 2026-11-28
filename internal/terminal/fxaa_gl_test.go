//go:build terminalgl && (linux || windows || darwin) && (amd64 || arm64)

package terminal

import (
	"fmt"
	"math/rand"
	"os"
	"slices"
	"testing"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

func fxaaFixture(width, height int) (image, frame [][]engine.Color) {
	rng := rand.New(rand.NewSource(902))
	image, frame = make([][]engine.Color, height), make([][]engine.Color, height)
	for y := range image {
		image[y], frame[y] = make([]engine.Color, width), make([]engine.Color, width)
		for x := range image[y] {
			if y/3 > x/8 && y%5 != 0 {
				image[y][x] = engine.Color{R: uint8(y * 13), G: 35, B: 214}
			} else {
				image[y][x] = engine.Color{R: uint8(rng.Intn(256)), G: uint8(rng.Intn(256)), B: uint8(rng.Intn(256))}
			}
		}
	}
	return image, frame
}

func resetColorRows(dst, src [][]engine.Color) {
	for i := range dst {
		copy(dst[i], src[i])
	}
}

func TestOpenGLFXAAMatchesCPU(t *testing.T) {
	device, err := openFXAADevice()
	if err != nil {
		t.Skip(err)
	}
	defer device.close()
	t.Logf("OpenGL FXAA device: %s", device.name())
	for _, dim := range [][2]int{{3, 7}, {160, 60}, {320, 120}, {9, 12}, {160, 60}} {
		original, got := fxaaFixture(dim[0], dim[1])
		_, want := fxaaFixture(dim[0], dim[1])
		resetColorRows(got, original)
		resetColorRows(want, original)
		filterFXAA(want, nil)
		if err := device.apply(got); err != nil {
			t.Fatal(err)
		}
		changed := 0
		for y, row := range got {
			if !slices.Equal(row, want[y]) {
				t.Fatalf("%v row %d differs from CPU", dim, y)
			}
			if !slices.Equal(row, original[y]) {
				changed++
			}
		}
		if changed == 0 {
			t.Fatalf("%v filter never changed a row", dim)
		}
	}
}

func BenchmarkFXAADevice(b *testing.B) {
	for _, dim := range [][2]int{{120, 40}, {160, 60}, {320, 120}} {
		for _, method := range []string{"cpu", "gl"} {
			b.Run(fmt.Sprintf("%dx%d/%s", dim[0], dim[1], method), func(b *testing.B) {
				original, frame := fxaaFixture(dim[0], dim[1])
				resetColorRows(frame, original)
				scratch := filterFXAA(frame, nil)
				var device fxaaDevice
				if method == "gl" {
					var err error
					device, err = openFXAADevice()
					if err != nil {
						b.Skip(err)
					}
					defer device.close()
					fmt.Fprintf(os.Stderr, "OpenGL FXAA device: %s\n", device.name())
					if err := device.apply(frame); err != nil {
						b.Fatal(err)
					}
				}
				b.ReportAllocs()
				for b.Loop() {
					resetColorRows(frame, original)
					if method == "cpu" {
						scratch = filterFXAA(frame, scratch)
					} else if err := device.apply(frame); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

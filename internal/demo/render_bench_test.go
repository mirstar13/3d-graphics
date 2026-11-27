package demo_test

import (
	"bufio"
	"fmt"
	"io"
	"testing"

	"github.com/mirstar13/go-3d-graphics/internal/terminal"
)

// Raster only: scene construction, simulation and terminal I/O are excluded.
func BenchmarkSceneRaster(b *testing.B) {
	for _, index := range []int{0, 5, 6} {
		for _, size := range [][2]int{{160, 60}, {320, 120}} {
			b.Run(fmt.Sprintf("scene%d/%dx%d", index+1, size[0], size[1]), func(b *testing.B) {
				scene, lights := presetScene(index, 0, false)
				r := terminal.NewTerminalRenderer(bufio.NewWriter(io.Discard), size[1], size[0])
				r.ShadowRenderer = nil
				r.SetCamera(scene.Camera)
				r.SetLightingSystem(lights)
				b.ReportAllocs()
				for b.Loop() {
					r.RenderScene(scene)
				}
			})
		}
	}
}

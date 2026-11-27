package demo_test

import (
	"bufio"
	"fmt"
	"io"
	"testing"

	"github.com/mirstar13/go-3d-graphics/internal/config"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
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

// BenchmarkCPURaster compares the production renderer compositions on fixed
// camera views. The scaling sweep holds AA off while varying scheduler inputs.
func BenchmarkCPURaster(b *testing.B) {
	for _, index := range []int{0, 5, 6} {
		for _, size := range [][2]int{{120, 40}, {160, 60}, {320, 120}} {
			for mode, name := range config.RenderModes {
				for aa, filter := range config.AAModes {
					workers := []int{4}
					tiles := []int{32}
					tiled := mode == int(engine.RenderModeParallelTiles) || mode == int(engine.RenderModeParallelBatched)
					if mode == int(engine.RenderModeSingle) {
						workers = []int{1}
					} else if aa == 0 {
						workers = []int{1, 2, 4}
						if tiled {
							tiles = []int{16, 32, 64}
						}
					}
					for _, n := range workers {
						for _, tile := range tiles {
							label := fmt.Sprintf("scene%d/%dx%d/%s/%s/w%d/t%d", index+1, size[0], size[1], name, filter, n, tile)
							b.Run(label, func(b *testing.B) {
								scene, lights := presetScene(index, 0, false)
								base := terminal.NewTerminalRenderer(bufio.NewWriter(io.Discard), size[1], size[0])
								base.ShadowRenderer = nil
								r := terminal.ComposeRenderer(base, engine.EngineConfig{RenderMode: engine.RenderMode(mode), AAMode: engine.AAMode(aa), NumWorkers: n, TileSize: tile})
								r.SetCamera(scene.Camera)
								r.SetLightingSystem(lights)
								r.RenderScene(scene)
								b.ReportAllocs()
								for b.Loop() {
									r.RenderScene(scene)
								}
							})
						}
					}
				}
			}
		}
	}
}

// BenchmarkScenePresent isolates terminal frame encoding on a rendered frame.
// The discard writer excludes terminal emulator paint and transport latency.
func BenchmarkScenePresent(b *testing.B) {
	for _, size := range [][2]int{{120, 40}, {160, 60}, {320, 120}} {
		for _, color := range []string{"none", "16", "256", "truecolor"} {
			b.Run(fmt.Sprintf("%dx%d/%s", size[0], size[1], color), func(b *testing.B) {
				scene, lights := presetScene(6, 0, false)
				r := terminal.NewTerminalRenderer(bufio.NewWriter(io.Discard), size[1], size[0])
				r.ShadowRenderer = nil
				r.SetLightingSystem(lights)
				r.SetCamera(scene.Camera)
				r.ColorMode = color
				r.UseColor = color != "none"
				r.RenderScene(scene)
				r.Present()
				b.ReportAllocs()
				for b.Loop() {
					r.Present()
				}
				encoded, _ := r.FrameOutput()
				b.ReportMetric(float64(encoded), "bytes/frame")
			})
		}
	}
}

// BenchmarkCourtyardShadows isolates the cost of the preset's shadow maps.
func BenchmarkCourtyardShadows(b *testing.B) {
	for mode, name := range config.RenderModes {
		for _, shadows := range []bool{false, true} {
			b.Run(fmt.Sprintf("%s/shadows_%t", name, shadows), func(b *testing.B) {
				scene, lights := presetScene(3, 0, false)
				base := terminal.NewTerminalRenderer(bufio.NewWriter(io.Discard), 60, 160)
				if !shadows {
					base.ShadowRenderer = nil
				}
				r := terminal.ComposeRenderer(base, engine.EngineConfig{RenderMode: engine.RenderMode(mode), NumWorkers: 4, TileSize: 32})
				r.SetCamera(scene.Camera)
				r.SetLightingSystem(lights)
				r.RenderScene(scene)
				b.ReportAllocs()
				for b.Loop() {
					r.RenderScene(scene)
				}
			})
		}
	}
}

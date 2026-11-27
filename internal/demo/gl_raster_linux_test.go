//go:build terminalgl && (amd64 || arm64)

package demo_test

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"runtime"
	"testing"

	"github.com/mirstar13/go-3d-graphics/internal/demo"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/terminal"
)

func demoRenderer(index, width, height int, useGL bool, scene *engine.Scene, lighting *engine.LightingSystem) *terminal.TerminalRenderer {
	r := terminal.NewTerminalRenderer(bufio.NewWriter(io.Discard), height, width)
	if !demo.PresetAt(index).Shadows {
		r.ShadowRenderer = nil
	}
	r.UseOpenGLRaster = useGL
	r.SetCamera(scene.Camera)
	r.SetLightingSystem(lighting)
	return r
}

func TestOpenGLRasterDemoParity(t *testing.T) {
	for _, index := range []int{0, 2, 3, 5, 6} {
		t.Run(demo.PresetAt(index).Name, func(t *testing.T) {
			scene, lighting := presetScene(index, 1.25, true)
			cpu := demoRenderer(index, 160, 60, false, scene, lighting)
			gl := demoRenderer(index, 160, 60, true, scene, lighting)
			if err := gl.Initialize(); err != nil {
				t.Fatal(err)
			}
			defer gl.Shutdown()
			for _, size := range [][2]int{{160, 60}, {320, 120}, {120, 40}} {
				cpu.Resize(size[0], size[1])
				gl.Resize(size[0], size[1])
				cpu.RenderScene(scene)
				gl.RenderScene(scene)
				if !gl.UsedOpenGLRaster() {
					t.Skip("OpenGL raster unavailable or scene uses CPU coverage")
				}
				var occupied, coverageDiff, colorDiff, depthDiff, changed int
				for y, row := range cpu.ZBuffer {
					for x, z := range row {
						a, b := math.IsInf(z, 1), math.IsInf(gl.ZBuffer[y][x], 1)
						if a && b {
							continue
						}
						occupied++
						if a != b {
							coverageDiff++
							changed++
							continue
						}
						depthChanged := math.Abs(z-gl.ZBuffer[y][x]) > 1e-5
						colorChanged := cpu.ColorBuffer[y][x] != gl.ColorBuffer[y][x]
						if depthChanged || colorChanged {
							changed++
						}
						if depthChanged {
							depthDiff++
						}
						if colorChanged {
							colorDiff++
						}
					}
				}
				t.Logf("%dx%d: %d occupied cells, %d coverage, %d depth, %d RGB differences, %d distinct changed cells", size[0], size[1], occupied, coverageDiff, depthDiff, colorDiff, changed)
				if occupied == 0 || changed > max(2, occupied/50) {
					t.Fatal("OpenGL differs on more than 2% of occupied cells")
				}
			}
		})
	}
}

func BenchmarkOpenGLSceneRaster(b *testing.B) {
	for _, index := range []int{0, 5, 6} {
		for _, size := range [][2]int{{160, 60}, {320, 120}} {
			for _, method := range []string{"gl_false", "batched", "gl_true"} {
				b.Run(fmt.Sprintf("scene%d/%dx%d/%s", index+1, size[0], size[1], method), func(b *testing.B) {
					useGL := method == "gl_true"
					scene, lighting := presetScene(index, 1.25, true)
					r := demoRenderer(index, size[0], size[1], useGL, scene, lighting)
					if useGL {
						if err := r.Initialize(); err != nil {
							b.Fatal(err)
						}
						defer r.Shutdown()
					}
					var renderer engine.Renderer = r
					if method == "batched" {
						renderer = terminal.ComposeRenderer(r, engine.EngineConfig{
							RenderMode: engine.RenderModeParallelBatched, NumWorkers: runtime.GOMAXPROCS(0), TileSize: 32,
						})
					}
					renderer.RenderScene(scene)
					if useGL && !r.UsedOpenGLRaster() {
						b.Skip("OpenGL raster unavailable or scene uses CPU coverage")
					}
					b.ReportAllocs()
					for b.Loop() {
						renderer.RenderScene(scene)
					}
					if useGL && !r.UsedOpenGLRaster() {
						b.Fatal("OpenGL raster fell back to CPU during the timed run")
					}
				})
			}
		}
	}
}

//go:build terminalgl && (amd64 || arm64)

package terminal

import (
	"bufio"
	"io"
	"math"
	"reflect"
	"testing"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

func TestOpenGLVisibilityAndFallback(t *testing.T) {
	device, err := openRasterDevice()
	if err != nil {
		t.Skip(err)
	}
	gpu := NewTerminalRenderer(bufio.NewWriter(io.Discard), 20, 30)
	gpu.rasterDevice = device
	defer gpu.Shutdown()
	cpu := NewTerminalRenderer(bufio.NewWriter(io.Discard), 20, 30)
	scene := engine.NewScene()
	scene.Camera.SetPosition(0, 0, 0)
	scene.Camera.Near = .5
	scene.Camera.Far = 10
	for _, z := range []float64{4, 2, 20} {
		tri := engine.NewTriangle(engine.Point{X: -1.2, Y: -.8, Z: z}, engine.Point{X: .1, Y: .8, Z: z}, engine.Point{X: 1.3, Y: -.8, Z: z}, 'X')
		scene.AddNode(engine.NewSceneNodeWithObject("triangle", tri))
	}
	for _, size := range [][2]int{{30, 20}, {73, 25}} {
		gpu.Resize(size[0], size[1])
		cpu.Resize(size[0], size[1])
		gpu.RenderScene(scene)
		cpu.RenderScene(scene)
		x, y := size[0]/2, size[1]/2
		if !gpu.UsedOpenGLRaster() || math.Abs(gpu.ZBuffer[y][x]-2) > 1e-12 || gpu.ColorBuffer[y][x] != cpu.ColorBuffer[y][x] {
			t.Fatalf("bad visible center at %v: %.9f", size, gpu.ZBuffer[y][x])
		}
	}
	// A triangle straddling Near must survive clipping and remain visible.
	scene.Clear()
	scene.AddNode(engine.NewSceneNodeWithObject("near crossing", engine.NewTriangle(
		engine.Point{X: -.5, Y: -.3, Z: .3}, engine.Point{X: 0, Y: .6, Z: 2}, engine.Point{X: .5, Y: -.3, Z: 2}, 'X')))
	gpu.RenderScene(scene)
	cpu.RenderScene(scene)
	center := gpu.ZBuffer[gpu.Height/2][gpu.Width/2]
	if !gpu.UsedOpenGLRaster() || math.IsInf(center, 1) || center != cpu.ZBuffer[cpu.Height/2][cpu.Width/2] {
		t.Fatalf("near-clipped center differs: %f", center)
	}
	point := &engine.Point{X: .25, Z: 1}
	scene.AddNode(engine.NewSceneNodeWithObject("point", point))
	gpu.RenderScene(scene)
	cpu.RenderScene(scene)
	x, y, _ := scene.Camera.ProjectPoint(*point, gpu.Height, gpu.Width)
	if !gpu.UsedOpenGLRaster() || gpu.ZBuffer[y][x] != cpu.ZBuffer[y][x] || gpu.Surface[y][x] != cpu.Surface[y][x] {
		t.Fatal("trailing point did not composite over the OpenGL depth buffer")
	}
	scene.AddNode(engine.NewSceneNodeWithObject("interleaved triangle", engine.NewTriangle(
		engine.Point{X: -.5, Y: -.3, Z: 3}, engine.Point{X: 0, Y: .6, Z: 3}, engine.Point{X: .5, Y: -.3, Z: 3}, 'X')))
	gpu.RenderScene(scene)
	cpu.RenderScene(scene)
	if gpu.UsedOpenGLRaster() || !reflect.DeepEqual(gpu.ZBuffer, cpu.ZBuffer) || !reflect.DeepEqual(gpu.Surface, cpu.Surface) {
		t.Fatal("mixed triangle and point scene did not use the CPU result")
	}
}

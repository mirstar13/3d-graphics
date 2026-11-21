package app

import (
	"bufio"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"os"
	"testing"

	"github.com/mirstar13/go-3d-graphics/internal/demo"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"github.com/mirstar13/go-3d-graphics/internal/terminal"
)

// TestPhysicsPlaygroundPreview exports frames at 0, 2, and 5 simulation seconds.
func TestPhysicsPlaygroundPreview(t *testing.T) {
	path := os.Getenv("PHYSICS_PREVIEW")
	if path == "" {
		t.Skip("set PHYSICS_PREVIEW to export frames")
	}
	sim, scene, lighting := testSimulation(t, demo.PhysicsPlayground)
	sheet := image.NewRGBA(image.Rect(0, 0, 720, 160))
	frame := terminal.NewTerminalRenderer(bufio.NewWriter(io.Discard), 40, 120)
	frame.ShadowRenderer = nil
	frame.SetLightingSystem(lighting)
	tick := 0
	for panel, target := range []int{0, 120, 300} {
		for tick < target {
			sendSimulation(t, sim, physics.DefaultOptions().FixedStep)
			tick++
		}
		frame.RenderScene(scene)
		for y, row := range frame.ColorBuffer {
			for x, c := range row {
				draw.Draw(sheet, image.Rect(panel*240+x*2, y*4, panel*240+x*2+2, y*4+4), &image.Uniform{C: color.RGBA{R: c.R, G: c.G, B: c.B, A: 255}}, image.Point{}, draw.Src)
			}
		}
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, sheet); err != nil {
		t.Fatal(err)
	}
}

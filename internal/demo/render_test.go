package demo_test

import (
	"bufio"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/mirstar13/go-3d-graphics/internal/demo"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/terminal"
)

func presetScene(index int, elapsed float64, orbit bool) (*engine.Scene, *engine.LightingSystem) {
	scene := engine.NewScene()
	demo.ConfigureCamera(scene.Camera, index, engine.OrientationTerminal)
	demo.BuildScene(scene, index)
	if orbit {
		controller := engine.NewCameraController(scene.Camera)
		demo.ConfigureCameraController(controller, index)
		controller.Update(engine.InputState{}, engine.OrientationTerminal, elapsed)
	}
	demo.AnimateSceneDemo(scene, index, elapsed, 1.0/30)
	scene.Update(1.0 / 30)
	lighting := demo.SetupLighting(scene.Camera, index)
	demo.AnimateLighting(lighting, index, elapsed)
	return scene, lighting
}
func renderPreset(index int, scene *engine.Scene, lighting *engine.LightingSystem, mode engine.RenderMode, useColor bool) *terminal.TerminalRenderer {
	base := terminal.NewTerminalRenderer(bufio.NewWriter(io.Discard), 40, 120)
	if !demo.PresetAt(index).Shadows {
		base.ShadowRenderer = nil
	}
	base.UseColor = useColor
	renderer := terminal.ComposeRenderer(base, engine.EngineConfig{RenderMode: mode, NumWorkers: 2, TileSize: 16})
	renderer.SetCamera(scene.Camera)
	renderer.SetLightingSystem(lighting)
	renderer.RenderScene(scene)
	return base
}
func TestScenePresetsRender(t *testing.T) {
	for index, name := range demo.Names {
		t.Run(name, func(t *testing.T) {
			for _, orbit := range []bool{false, true} {
				for _, elapsed := range []float64{0, 8, 20} {
					scene, lighting := presetScene(index, elapsed, orbit)
					reference := renderPreset(index, scene, lighting, engine.RenderModeSingle, true)
					count := 0
					colors := map[engine.Color]bool{}
					for _, row := range reference.ColorBuffer {
						for _, c := range row {
							if c.R > 12 || c.G > 12 || c.B > 12 {
								count++
								colors[c] = true
							}
						}
					}
					if count < 120 {
						t.Errorf("orbit=%v t=%g: only %d bright cells", orbit, elapsed, count)
					}
					if len(colors) < 8 {
						t.Errorf("orbit=%v t=%g: only %d colors", orbit, elapsed, len(colors))
					}
					parallel := renderPreset(index, scene, lighting, engine.RenderModeParallelTiles, true)
					if !reflect.DeepEqual(reference.ColorBuffer, parallel.ColorBuffer) {
						t.Errorf("orbit=%v t=%g: CPU modes differ", orbit, elapsed)
					}
				}
			}
			scene, lighting := presetScene(index, 0, false)
			mono := renderPreset(index, scene, lighting, engine.RenderModeSingle, false)
			count := 0
			for _, row := range mono.Surface {
				for _, c := range row {
					if c != ' ' {
						count++
					}
				}
			}
			if count < 120 {
				t.Errorf("monochrome scene too sparse: %d", count)
			}
		})
	}
}

func TestSceneAnimationDeterministic(t *testing.T) {
	for index, name := range demo.Names {
		t.Run(name, func(t *testing.T) {
			a, _ := presetScene(index, 0, false)
			b, _ := presetScene(index, 0, false)
			for frame := 0; frame < 120; frame++ {
				demo.AnimateSceneDemo(a, index, float64(frame)/30, 1.0/30)
			}
			demo.AnimateSceneDemo(a, index, 4, 1.0/30)
			demo.AnimateSceneDemo(b, index, 4, 4)
			for name, node := range a.AllNodes {
				other := b.FindNode(name)
				if other == nil || node.Transform.GetWorldMatrix() != other.Transform.GetWorldMatrix() {
					t.Fatalf("pose depends on frame history: %s", name)
				}
			}
			if wa := a.FindNode("Wave"); wa != nil {
				if !reflect.DeepEqual(wa.Object, b.FindNode("Wave").Object) {
					t.Fatal("instance poses differ")
				}
			}
		})
	}
}

// TestScenePreview exports actual framebuffer colors for visual review on request.
// Each cell is shown at a 1:2 aspect ratio; rows follow the picker order.
func TestScenePreview(t *testing.T) {
	path := os.Getenv("DEMO_PREVIEW")
	if path == "" {
		t.Skip("set DEMO_PREVIEW to export a contact sheet")
	}
	const width, height = 240, 160
	sheet := image.NewRGBA(image.Rect(0, 0, width*3, height*((len(demo.Names)+2)/3)))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{C: color.Black}, image.Point{}, draw.Src)
	for index, name := range demo.Names {
		scene, lighting := presetScene(index, 0, false)
		frame := renderPreset(index, scene, lighting, engine.RenderModeSingle, true)
		for y, row := range frame.ColorBuffer {
			for x, c := range row {
				rect := image.Rect(index%3*width+x*2, index/3*height+y*4, index%3*width+x*2+2, index/3*height+y*4+4)
				draw.Draw(sheet, rect, &image.Uniform{C: color.RGBA{R: c.R, G: c.G, B: c.B, A: 255}}, image.Point{}, draw.Src)
			}
		}
		t.Logf("%d: %s", index+1, name)
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

func TestPresetGeometryBudgets(t *testing.T) {
	for index, name := range demo.Names {
		t.Run(name, func(t *testing.T) {
			scene, _ := presetScene(index, 0, false)
			triangles := 0
			for _, node := range scene.GetRenderableNodes() {
				switch object := node.Object.(type) {
				case *engine.Mesh:
					triangles += len(object.Indices) / 3
				case *engine.LODGroupWithTransitions:
					for _, level := range object.Levels {
						triangles += len(level.Mesh.Indices) / 3
					}
				case *engine.InstancedMesh:
					triangles += len(object.BaseMesh.Indices) / 3 * len(object.Instances)
				}
			}
			if triangles > 5000 {
				t.Fatalf("%d triangles exceed the terminal preset budget", triangles)
			}
		})
	}
}

func TestCourtyardCastsShadows(t *testing.T) {
	index := 3
	scene, lighting := presetScene(index, 0, false)
	shadowed := renderPreset(index, scene, lighting, engine.RenderModeSingle, true)
	plain := terminal.NewTerminalRenderer(bufio.NewWriter(io.Discard), 40, 120)
	plain.ShadowRenderer = nil
	plain.SetLightingSystem(lighting)
	plain.RenderScene(scene)
	changed := 0
	for y, row := range shadowed.ColorBuffer {
		for x, c := range row {
			if c != plain.ColorBuffer[y][x] {
				changed++
			}
		}
	}
	if changed < 10 {
		t.Fatalf("courtyard has only %d shadowed cells", changed)
	}
}

func TestRunwayVisitsAllDetailLevels(t *testing.T) {
	scene, _ := presetScene(4, 0, false)
	visited := map[int]bool{}
	for frame := 0; frame < 450; frame++ {
		demo.AnimateSceneDemo(scene, 4, float64(frame)/30, 1.0/30)
		scene.Update(1.0 / 30)
		for _, node := range scene.GetRenderableNodes() {
			if lod, ok := node.Object.(*engine.LODGroupWithTransitions); ok {
				visited[lod.CurrentLOD] = true
			}
		}
	}
	if len(visited) != 3 {
		t.Fatalf("runway only visits %v", visited)
	}
}

func TestInstancedTileParity(t *testing.T) {
	for _, elapsed := range []float64{0, 1.75} {
		scene, lights := presetScene(5, elapsed, false)
		want := renderPreset(5, scene, lights, engine.RenderModeSingle, true)
		for _, mode := range []engine.RenderMode{engine.RenderModeParallelTiles, engine.RenderModeParallelBatched} {
			got := renderPreset(5, scene, lights, mode, true)
			if !reflect.DeepEqual(got.Surface, want.Surface) || !reflect.DeepEqual(got.ColorBuffer, want.ColorBuffer) || !reflect.DeepEqual(got.ZBuffer, want.ZBuffer) {
				t.Fatalf("tile parity failed for mode %v at %.2fs", mode, elapsed)
			}
		}
	}
}

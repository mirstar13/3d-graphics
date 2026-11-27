package terminal

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"reflect"
	"testing"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

// A floor spanning the near plane must stay solid while walking and turning.
// Compare raster coverage to independent ray/plane intersections at cell centres.
func TestNearPlaneFloorCoverage(t *testing.T) {
	camera := engine.NewCameraAt(2.3, 1.3, -4.2)
	camera.Near, camera.Far = .1, 1000
	camera.SetFOV(90, 60)
	floor := engine.NewQuad(
		engine.Point{X: -100, Z: -100}, engine.Point{X: -100, Z: 100},
		engine.Point{X: 100, Z: 100}, engine.Point{X: 100, Z: -100})
	r := NewTerminalRenderer(bufio.NewWriter(io.Discard), 40, 100)
	for turn := 0; turn < 30; turn++ {
		camera.SetRotation(.17, float64(turn)*.13, 0)
		r.BeginFrame()
		r.RenderQuad(floor, engine.IdentityMatrix(), camera)
		covered := 0
		for y := 0; y < r.Height; y++ {
			for x := 0; x < r.Width; x++ {
				ray := engine.CameraScreenPointToRay(camera, x, y, r.Width, r.Height)
				if ray.Direction.Y >= 0 {
					continue
				}
				p := ray.GetPoint(-ray.Origin.Y / ray.Direction.Y)
				depth := camera.TransformToViewSpace(p).Z
				if math.Abs(p.X) >= 99 || math.Abs(p.Z) >= 99 || depth <= camera.Near || depth >= camera.Far {
					continue
				}
				covered++
				if math.IsInf(r.ZBuffer[y][x], 1) || math.Abs(r.ZBuffer[y][x]-depth) > 1e-6 {
					t.Fatalf("turn %d: floor missing or wrong depth at (%d,%d): got %g, want %g", turn, x, y, r.ZBuffer[y][x], depth)
				}
			}
		}
		if covered < 500 {
			t.Fatalf("turn %d: insufficient floor coverage (%d cells)", turn, covered)
		}
	}
}

func TestParallelPrimitiveParity(t *testing.T) {
	scene := engine.NewScene()
	scene.Camera = engine.NewCameraAt(0, 0, 0)
	scene.Camera.SetFOV(90, 90)
	mesh := engine.NewMesh()
	mesh.Vertices = []engine.Point{{X: -1, Y: -1}, {X: 0, Y: 1}, {X: 1, Y: -1}}
	mesh.Indices = []int{0, 1, 2}
	mesh.Position = engine.Point{X: 2, Z: 12}
	instances := engine.NewInstancedMesh(mesh)
	instances.AddInstance(engine.IdentityMatrix(), engine.ColorRed)
	objects := []any{
		engine.NewPoint(-4, 3, 10),
		engine.NewLine(engine.Point{X: -6, Y: -3, Z: 10}, engine.Point{X: 6, Y: -3, Z: 10}),
		engine.NewCircle(-3, 0, 12, 1, 16),
		engine.NewQuad(engine.Point{X: 1, Y: 1, Z: 14}, engine.Point{X: 3, Y: 1, Z: 14}, engine.Point{X: 3, Y: 3, Z: 14}, engine.Point{X: 1, Y: 3, Z: 14}),
		instances,
	}
	for i, object := range objects {
		scene.AddNode(engine.NewSceneNodeWithObject(fmt.Sprint(i), object))
	}
	var reference *TerminalRenderer
	for mode := engine.RenderModeSingle; mode <= engine.RenderModeParallelBatched; mode++ {
		base := NewTerminalRenderer(bufio.NewWriter(io.Discard), 40, 80)
		base.ShadowRenderer = nil
		renderer := ComposeRenderer(base, engine.EngineConfig{RenderMode: mode, NumWorkers: 3, TileSize: 8})
		renderer.SetCamera(scene.Camera)
		renderer.RenderScene(scene)
		if reference == nil {
			reference = base
			continue
		}
		if !reflect.DeepEqual(base.Surface, reference.Surface) || !reflect.DeepEqual(base.ColorBuffer, reference.ColorBuffer) || !reflect.DeepEqual(base.ZBuffer, reference.ZBuffer) {
			t.Errorf("mode %d differs from single-threaded primitives", mode)
		}
	}
}

func TestMeshOffsetMatchesTransformedVertices(t *testing.T) {
	camera := engine.NewCameraAt(0, 0, 0)
	mesh := engine.NewMesh()
	mesh.Vertices = []engine.Point{{X: -1, Y: -1}, {X: 0, Y: 1}, {X: 1, Y: -1}}
	mesh.Indices = []int{0, 1, 2}
	mesh.Position = engine.Point{X: 2, Z: 10}
	transform := engine.NewTransform()
	transform.SetScale(2, 1, 1)
	manual := engine.NewMesh()
	for _, point := range mesh.Vertices {
		point.X += mesh.Position.X
		point.Z += mesh.Position.Z
		manual.Vertices = append(manual.Vertices, transform.TransformPoint(point))
	}
	manual.Indices = mesh.Indices
	actual := NewTerminalRenderer(bufio.NewWriter(io.Discard), 40, 80)
	expected := NewTerminalRenderer(bufio.NewWriter(io.Discard), 40, 80)
	actual.RenderMesh(mesh, transform.GetWorldMatrix(), camera)
	expected.RenderMesh(manual, engine.IdentityMatrix(), camera)
	if !reflect.DeepEqual(actual.ZBuffer, expected.ZBuffer) {
		t.Fatal("mesh offset applied in world space")
	}
	mesh.Indices = []int{-1, 1, 2}
	actual.RenderMesh(mesh, transform.GetWorldMatrix(), camera)
}

func TestPointVisibilityAndDegenerateLine(t *testing.T) {
	camera := engine.NewCameraAt(0, 0, 0)
	camera.Far = 10
	r := NewTerminalRenderer(bufio.NewWriter(io.Discard), 20, 40)
	r.RenderPoint(engine.NewPoint(0, 0, 5), engine.IdentityMatrix(), camera)
	if r.ColorBuffer[10][20] != engine.ColorWhite {
		t.Fatal("colored point is invisible against black")
	}
	r.BeginFrame()
	r.RenderPoint(engine.NewPoint(0, 0, 20), engine.IdentityMatrix(), camera)
	if r.Surface[10][20] != ' ' {
		t.Fatal("point beyond far plane rendered")
	}
	point := engine.Point{Z: 5}
	r.RenderLine(engine.NewLine(point, point), engine.IdentityMatrix(), camera)
	if r.Surface[10][20] == ' ' {
		t.Fatal("zero-length visible line disappeared")
	}
}

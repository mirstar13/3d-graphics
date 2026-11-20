package terminal

import (
	"bufio"
	"fmt"
	"io"
	"reflect"
	"testing"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

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

package terminal

import (
	"bufio"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"io"
	"math"
	"reflect"
	"testing"
)

func qualityScene() *engine.Scene {
	scene := engine.NewScene()
	scene.Camera = engine.NewCameraAt(0, 0, 0)
	scene.Camera.SetFOV(90, 90)
	mesh := engine.NewMesh()
	// A fractional diagonal exercises coverage that integer projection would lose.
	mesh.Vertices = []engine.Point{{X: -3.7, Y: 3.7, Z: 5}, {X: 3.7, Y: -3.7, Z: 5}, {X: -3.7, Y: -3.7, Z: 5}}
	mesh.Indices = []int{0, 1, 2}
	mat := engine.NewMaterial()
	mat.DiffuseColor = engine.ColorRed
	mesh.Material = &mat
	scene.AddNode(engine.NewSceneNodeWithObject("mesh", mesh))
	return scene
}
func renderQuality(scene *engine.Scene, mode engine.RenderMode, aa engine.AAMode) *TerminalRenderer {
	r := NewTerminalRenderer(bufio.NewWriter(io.Discard), 16, 16)
	r.ShadowRenderer = nil
	ComposeRenderer(r, engine.EngineConfig{RenderMode: mode, AAMode: aa, NumWorkers: 2, TileSize: 5}).RenderScene(scene)
	return r
}
func TestSampledAAEdgesAndModeParity(t *testing.T) {
	scene := qualityScene()
	plain := renderQuality(scene, engine.RenderModeSingle, engine.AANone)
	for _, aa := range []engine.AAMode{engine.AAMSAA2x, engine.AAMSAA4x, engine.AASSAA} {
		reference := renderQuality(scene, engine.RenderModeSingle, aa)
		if reflect.DeepEqual(reference.ColorBuffer, plain.ColorBuffer) {
			t.Fatalf("AA %d leaves edges unchanged", aa)
		}
		partial := 0
		full := plain.ColorBuffer[10][4].R
		if full == 0 {
			t.Fatal("fixture has no interior")
		}
		for y, row := range reference.ColorBuffer {
			for x, c := range row {
				if c.R > 0 && c.R < full {
					partial++
				}
				if c.R > 0 && (math.IsInf(reference.ZBuffer[y][x], 1) || reference.Surface[y][x] == ' ') {
					t.Fatal("resolved edge lost depth or glyph")
				}
			}
		}
		if partial == 0 {
			t.Fatalf("AA %d has no fractional coverage", aa)
		}
		if reference.Width != 16 || reference.Height != 16 {
			t.Fatal("SSAA dimensions leaked")
		}
		for mode := engine.RenderModeParallelTiles; mode <= engine.RenderModeParallelBatched; mode++ {
			actual := renderQuality(scene, mode, aa)
			if !reflect.DeepEqual(reference.ColorBuffer, actual.ColorBuffer) || !reflect.DeepEqual(reference.ZBuffer, actual.ZBuffer) {
				t.Fatalf("mode %d AA %d differs", mode, aa)
			}
		}
	}
}
func TestMSAADepthOrderIndependent(t *testing.T) {
	scene := qualityScene()
	near := scene.GetRenderableNodes()[0]
	farMesh := *near.Object.(*engine.Mesh)
	farMesh.Vertices = append([]engine.Point(nil), farMesh.Vertices...)
	for i := range farMesh.Vertices {
		farMesh.Vertices[i].X *= 2
		farMesh.Vertices[i].Y *= 2
		farMesh.Vertices[i].Z *= 2
	}
	mat := engine.NewMaterial()
	mat.DiffuseColor = engine.ColorBlue
	farMesh.Material = &mat
	far := engine.NewSceneNodeWithObject("far", &farMesh)
	scene.AddNode(far)
	first := renderQuality(scene, engine.RenderModeSingle, engine.AAMSAA4x)
	scene.Root.Children[0], scene.Root.Children[1] = scene.Root.Children[1], scene.Root.Children[0]
	second := renderQuality(scene, engine.RenderModeSingle, engine.AAMSAA4x)
	if !reflect.DeepEqual(first.ColorBuffer, second.ColorBuffer) {
		t.Fatal("far geometry overwrites near samples")
	}
}
func TestLODTransitionEndpointsAndMaterials(t *testing.T) {
	scene := qualityScene()
	node := scene.GetRenderableNodes()[0]
	from := node.Object.(*engine.Mesh)
	to := *from
	mat := engine.NewMaterial()
	mat.DiffuseColor = engine.ColorBlue
	to.Material = &mat
	group := engine.NewLODGroupWithTransitions(engine.LODTransitionCrossFade, 1)
	group.AddLOD(from, 10)
	group.AddLOD(&to, 100)
	node.SetLODGroupWithTransition(group)
	group.TransitionState.StartTransition(0, 1)
	start := renderQuality(scene, engine.RenderModeSingle, engine.AANone)
	group.TransitionState.Progress = .5
	mid := renderQuality(scene, engine.RenderModeSingle, engine.AANone)
	red, blue := 0, 0
	for _, row := range mid.ColorBuffer {
		for _, c := range row {
			if c.R > 0 {
				red++
			}
			if c.B > 0 {
				blue++
			}
		}
	}
	if red == 0 || blue == 0 {
		t.Fatal("transition does not contain both materials")
	}
	for mode := engine.RenderModeParallelTiles; mode <= engine.RenderModeParallelBatched; mode++ {
		actual := renderQuality(scene, mode, engine.AANone)
		if !reflect.DeepEqual(mid.ColorBuffer, actual.ColorBuffer) {
			t.Fatal("transition differs by CPU mode")
		}
	}
	group.TransitionState.Progress = 1
	end := renderQuality(scene, engine.RenderModeSingle, engine.AANone)
	node.Object = from
	if !reflect.DeepEqual(start.ColorBuffer, renderQuality(scene, 0, 0).ColorBuffer) {
		t.Fatal("incorrect fade start")
	}
	node.Object = &to
	if !reflect.DeepEqual(end.ColorBuffer, renderQuality(scene, 0, 0).ColorBuffer) {
		t.Fatal("incorrect fade end")
	}
}
func TestMeshFaceMaterialsRender(t *testing.T) {
	scene := qualityScene()
	mesh := scene.GetRenderableNodes()[0].Object.(*engine.Mesh)
	mesh.Vertices = append(mesh.Vertices, engine.Point{X: 3.7, Y: 3.7, Z: 5})
	mesh.Indices = append(mesh.Indices, 0, 3, 1)
	blue := engine.NewMaterial()
	blue.DiffuseColor = engine.ColorBlue
	mesh.FaceMaterials = []engine.IMaterial{nil, &blue}
	r := renderQuality(scene, 0, 0)
	redCount, blueCount := 0, 0
	for _, row := range r.ColorBuffer {
		for _, c := range row {
			if c.R > 0 {
				redCount++
			}
			if c.B > 0 {
				blueCount++
			}
		}
	}
	if redCount == 0 || blueCount == 0 {
		t.Fatal("face materials not rendered")
	}
}
func BenchmarkAntiAliasing(b *testing.B) {
	for _, entry := range []struct {
		name string
		aa   engine.AAMode
	}{{"none", engine.AANone}, {"fxaa", engine.AAFXAA}, {"msaa2", engine.AAMSAA2x}, {"msaa4", engine.AAMSAA4x}, {"ssaa", engine.AASSAA}} {
		b.Run(entry.name, func(b *testing.B) {
			scene := qualityScene()
			r := NewTerminalRenderer(bufio.NewWriter(io.Discard), 40, 100)
			r.ShadowRenderer = nil
			pipeline := ComposeRenderer(r, engine.EngineConfig{AAMode: entry.aa})
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				pipeline.RenderScene(scene)
			}
		})
	}
}

func TestSampledAAMonochrome(t *testing.T) {
	for _, aa := range []engine.AAMode{engine.AAMSAA2x, engine.AAMSAA4x, engine.AASSAA} {
		r := NewTerminalRenderer(bufio.NewWriter(io.Discard), 16, 16)
		r.UseColor = false
		r.ShadowRenderer = nil
		ComposeRenderer(r, engine.EngineConfig{AAMode: aa}).RenderScene(qualityScene())
		visible := 0
		for _, row := range r.Surface {
			for _, char := range row {
				if char != ' ' {
					visible++
				}
			}
		}
		if visible == 0 {
			t.Fatalf("AA %d loses monochrome geometry", aa)
		}
	}
}

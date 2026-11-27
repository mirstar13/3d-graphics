package app

import (
	"bytes"
	"github.com/qmuntal/gltf"
	"github.com/qmuntal/gltf/modeler"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/game"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"github.com/mirstar13/go-3d-graphics/internal/project"
)

func TestAssetInspectorToPlay(t *testing.T) {
	m := inputEditor(t)
	for name, data := range map[string]string{"model.obj": "v 0 0 0\nv 2 0 0\nv 0 2 0\nf 1 2 3\n", "palette.mtl": "newmtl Red\nKd 1 0 0\n"} {
		if err := os.WriteFile(filepath.Join(m.project.Root, "assets", name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	m.selectID("box")
	edit := func(key, value string) {
		t.Helper()
		for i, p := range project.Properties {
			if p.Key == key {
				m.property = i
				m, _ = editorRune(m, "e")
				m.draft = value
				m = resourceKey(m, tea.KeyEnter)
				return
			}
		}
		t.Fatal("unknown inspector field", key)
	}
	edit("mesh_asset", "assets/missing.obj")
	if m.modal != "property" || m.document.Dirty() {
		t.Fatal("missing asset accepted", m.message)
	}
	m = resourceKey(m, tea.KeyEsc)
	for _, setting := range [][2]string{{"mesh_asset", "assets/model.obj"}, {"material_asset", "assets/palette.mtl#Red"}, {"collider_shape", "sphere"}, {"collider_size.x", "2"}} {
		edit(setting[0], setting[1])
		if m.modal != "" {
			t.Fatal(setting, m.message)
		}
	}
	check := func(s *game.Session) {
		t.Helper()
		mesh := s.Scene.FindNode("entity:box").Object.(*engine.Mesh)
		if len(mesh.Indices) != 3 || mesh.Material.GetDiffuseColor(0, 0) != (engine.Color{R: 255}) {
			t.Fatal("wrong mesh or material")
		}
		id, _ := s.Find("box")
		state, err := s.Inspect(id)
		if err != nil {
			t.Fatal(err)
		}
		shape := state.Body.Collider.Shape
		if shape.Kind() != physics.SphereShape || shape.Radius() != 2 {
			t.Fatal("wrong independent collider", shape)
		}
	}
	check(m.preview)
	m.save()
	p, err := project.Open(m.project.Root)
	if err != nil {
		t.Fatal(err)
	}
	if err = m.install(p); err != nil {
		t.Fatal(err)
	}
	m.startPlay()
	check(m.play)
	m.resetPlay()
	check(m.play)
}

func TestImportTexturedGLBIntoEditor(t *testing.T) {
	m := inputEditor(t)
	doc := gltf.NewDocument()
	position := modeler.WritePosition(doc, [][3]float32{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}})
	uv := modeler.WriteTextureCoord(doc, [][2]float32{{0, 0}, {1, 0}, {0, 1}})
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, img); err != nil {
		t.Fatal(err)
	}
	imageIndex, err := modeler.WriteImage(doc, "red", "image/png", &pngData)
	if err != nil {
		t.Fatal(err)
	}
	doc.Textures = []*gltf.Texture{{Source: gltf.Index(imageIndex)}}
	doc.Materials = []*gltf.Material{{PBRMetallicRoughness: &gltf.PBRMetallicRoughness{BaseColorTexture: &gltf.TextureInfo{Index: 0}}}}
	doc.Meshes = []*gltf.Mesh{{Primitives: []*gltf.Primitive{{Attributes: gltf.PrimitiveAttributes{"POSITION": position, "TEXCOORD_0": uv}, Material: gltf.Index(0)}}}}
	doc.Nodes = []*gltf.Node{{Translation: [3]float64{2, 0, -1}, Children: []int{1}}, {Mesh: gltf.Index(0), Translation: [3]float64{0, 3, 0}}}
	doc.Scenes[0].Nodes = []int{0}
	var model bytes.Buffer
	if err = gltf.NewEncoder(&model).Encode(doc); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "crate.glb")
	if err = os.WriteFile(source, model.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	m.selectID("box")
	m = resourceKey(m, tea.KeyCtrlX)
	m, _ = editorRune(m, "i")
	m.draft = source
	m = resourceKey(m, tea.KeyEnter)
	if m.modal != "mesh_assets" || len(m.meshPaths) != 1 {
		t.Fatal(m.modal, m.message)
	}
	m = resourceKey(m, tea.KeyEnter)
	if m.modal != "" {
		t.Fatal(m.message)
	}
	m.save()
	if _, err = project.Open(m.project.Root); err != nil {
		t.Fatal(err)
	}
	m.startPlay()
	mesh := m.play.Scene.FindNode("entity:box").Object.(*engine.Mesh)
	if len(mesh.Indices) != 3 || mesh.Indices[1] != 2 || mesh.MaterialForFace(0).GetDiffuseColor(.2, .2) != (engine.Color{R: 255}) {
		t.Fatal("GLB lost winding or texture", mesh.Indices)
	}
	entity, _ := m.selectedEntity()
	if mesh.Vertices[0] != (engine.Point{X: 2 * entity.Size.X, Y: 3 * entity.Size.Y, Z: entity.Size.Z}) {
		t.Fatal("GLB lost the node transforms", mesh.Vertices[0])
	}
	if _, err = m.project.ImportGLB(source); err == nil {
		t.Fatal("import overwrote existing asset")
	}
}

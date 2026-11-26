package app

import (
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

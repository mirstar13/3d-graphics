package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinkedMeshAndMaterialOverride(t *testing.T) {
	p, ed, resource := resourceFixture(t)
	assets := filepath.Join(p.Root, "assets")
	for name, data := range map[string]string{"palette.mtl": "newmtl Green\nKd 0 1 0\nnewmtl Red\nKd 1 0 0\n", "model.obj": "mtllib palette.mtl\nusemtl Green\nv 0 0 0\nv 1 0 0\nv 0 1 0\nf 1 2 3\n"} {
		if err := os.WriteFile(filepath.Join(assets, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	source, err := p.LoadResource(resource)
	if err != nil {
		t.Fatal(err)
	}
	for i := range source.Scene.Entities {
		source.Scene.Entities[i].MeshAsset = "assets/model.obj"
	}
	if err = p.SaveResource(resource, source); err != nil {
		t.Fatal(err)
	}
	root, err := ed.Instantiate(p, resource)
	if err != nil {
		t.Fatal(err)
	}
	if err = ed.Set(root, "material_asset", "assets/palette.mtl#Red"); err != nil {
		t.Fatal(err)
	}
	if err = p.Save(ed.Document()); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(p.Root)
	if err != nil {
		t.Fatal(err)
	}
	e := entityByID(t, reopened.Scene, root)
	mesh, material, err := e.LoadVisualAssets(p.Root)
	if err != nil {
		t.Fatal(err)
	}
	if material.DiffuseColor.R != 255 || mesh.FaceMaterials[0].GetDiffuseColor(0, 0).G != 255 {
		t.Fatal("lost assigned or embedded material")
	}
	if err = ed.ResetOverride(p, root, "material_asset"); err != nil {
		t.Fatal(err)
	}
	if entityByID(t, ed.Document(), root).MaterialAsset != "" {
		t.Fatal("material override survived reset")
	}
}

package project

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestExportIncludesLinkedMeshDependencies(t *testing.T) {
	p, ed, path := resourceFixture(t)
	files := map[string]string{
		"renderer.yaml":     "color: none\n",
		"main.go":           "package main\nfunc main(){}\n",
		"go.mod":            "module example.org/export\n\ngo 1.24.6\n",
		"assets/model.obj":  "mtllib colors.mtl\nusemtl Green\nv 0 0 0\nv 1 0 0\nv 0 1 0\nf 1 2 3\n",
		"assets/colors.mtl": "newmtl Green\nKd 0 1 0\n",
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(p.Root, name), []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	r, err := p.LoadResource(path)
	if err != nil {
		t.Fatal(err)
	}
	r.Scene.Entities[0].MeshAsset = "assets/model.obj"
	if err = p.SaveResource(path, r); err != nil {
		t.Fatal(err)
	}
	if _, err = ed.Instantiate(p, path); err != nil {
		t.Fatal(err)
	}
	if err = p.Save(ed.Document()); err != nil {
		t.Fatal(err)
	}
	archive, err := p.Export(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	z, err := zip.OpenReader(filepath.Join(p.Root, archive))
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	names := map[string]bool{}
	for _, f := range z.File {
		names[f.Name] = true
	}
	for _, name := range []string{"project.yaml", "renderer.yaml", filepath.ToSlash(path), p.Manifest.EntryScene, "assets/model.obj", "assets/colors.mtl", gameExecutable()} {
		if !names[name] {
			t.Error("missing", name)
		}
	}
	for _, name := range []string{"go.mod", "go.sum", "main.go", "build/"} {
		if names[name] {
			t.Error("unexpected", name)
		}
	}
}

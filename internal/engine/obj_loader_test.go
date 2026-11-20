package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOBJPreservesUVsAndRelativeIndices(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mesh.obj")
	source := "v 0 0 2\nv 1 0 2\nv 0 1 2\nvt 0 0\nvt 1 0\nvt 0 1\nf -3/-3 -2/-2 -1/-1 # face\n"
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	mesh, err := LoadOBJ(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(mesh.Indices) != 3 || len(mesh.UVs) != 3 || mesh.UVs[1].U != 1 || mesh.UVs[2].V != 1 {
		t.Fatalf("lost UVs: %+v", mesh)
	}
	for _, face := range []string{"1/99 2/2 3/3", "1/nope 2/2 3/3", "0 2 3"} {
		if err := os.WriteFile(path, []byte("v 0 0 2\nv 1 0 2\nv 0 1 2\nf "+face+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadOBJ(path); err == nil {
			t.Fatalf("invalid face accepted: %s", face)
		}
	}
}

func TestOBJFaceMaterials(t *testing.T) {
	dir := t.TempDir()
	write := func(name, data string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("colors.mtl", "newmtl red\nKd 1 0 0\nnewmtl blue\nKd 0 0 1\n")
	write("mesh.obj", "mtllib colors.mtl\nv 0 0 4\nv 1 0 4\nv 1 1 4\nv 0 1 4\nf 1 2 3\nusemtl red\nf 1 2 3 4\nusemtl blue\nf 1 3 4\n")
	asset, err := LoadOBJWithMaterials(filepath.Join(dir, "mesh.obj"))
	if err != nil {
		t.Fatal(err)
	}
	mesh := asset.Mesh
	if len(mesh.FaceMaterials) != 4 {
		t.Fatal(mesh.FaceMaterials)
	}
	want := []IMaterial{mesh.Material, asset.Materials.Materials["red"], asset.Materials.Materials["red"], asset.Materials.Materials["blue"]}
	for i, m := range want {
		if mesh.MaterialForFace(i) != m {
			t.Fatalf("face %d lost assignment", i)
		}
	}
	reduced := SimplifyMesh(mesh, .75)
	for i, source := range []int{0, 1, 2} {
		if reduced.MaterialForFace(i) != mesh.MaterialForFace(source) {
			t.Fatal("LOD lost material")
		}
	}
	write("bad.obj", "mtllib missing.mtl\nv 0 0 1\n")
	if _, err := LoadOBJ(filepath.Join(dir, "bad.obj")); err == nil {
		t.Fatal("missing library accepted")
	}
	write("bad.obj", "mtllib colors.mtl\nusemtl missing\n")
	if _, err := LoadOBJ(filepath.Join(dir, "bad.obj")); err == nil {
		t.Fatal("unknown material accepted")
	}
}

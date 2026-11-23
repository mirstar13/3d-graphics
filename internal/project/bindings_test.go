package project

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mirstar13/go-3d-graphics/internal/input"
)

func TestBindingsPersistWithoutChangingScene(t *testing.T) {
	p, err := Create(filepath.Join(t.TempDir(), "Game"))
	if err != nil {
		t.Fatal(err)
	}
	scene, err := os.ReadFile(filepath.Join(p.Root, p.Manifest.EntryScene))
	if err != nil {
		t.Fatal(err)
	}
	b := input.Bindings{input.Play: {"v"}}
	if err = p.SaveBindings(b); err != nil {
		t.Fatal(err)
	}
	b[input.Play][0] = "x"
	loaded, err := Open(p.Root)
	if err != nil || !reflect.DeepEqual(loaded.Manifest.Bindings, input.Bindings{input.Play: {"v"}}) {
		t.Fatal(loaded, err)
	}
	before, err := os.ReadFile(filepath.Join(p.Root, "project.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if p.SaveBindings(input.Bindings{input.Play: {"q"}}) == nil {
		t.Fatal("invalid binding accepted")
	}
	after, _ := os.ReadFile(filepath.Join(p.Root, "project.yaml"))
	if string(before) != string(after) {
		t.Fatal("invalid save changed manifest")
	}
	after, _ = os.ReadFile(filepath.Join(p.Root, p.Manifest.EntryScene))
	if string(scene) != string(after) {
		t.Fatal("binding save changed scene")
	}
	if err = p.SaveBindings(nil); err != nil {
		t.Fatal(err)
	}
	loaded, err = Open(p.Root)
	if err != nil || len(loaded.Manifest.Bindings) != 0 {
		t.Fatal("reset defaults", err)
	}
}

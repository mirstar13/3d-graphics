package project

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestProjectRoundTripAndFailedSave(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Example")
	p, err := Create(root)
	if err != nil {
		t.Fatal(err)
	}
	editor, _ := NewEditor(p.Scene)
	id, err := editor.Add("sphere")
	if err != nil {
		t.Fatal(err)
	}
	if err = editor.Set(id, "name", "Player"); err != nil {
		t.Fatal(err)
	}
	if err = editor.Set(id, "position.y", "8"); err != nil {
		t.Fatal(err)
	}
	if err = p.Save(editor.Document()); err != nil {
		t.Fatal(err)
	}
	loaded, err := Open(root)
	if err != nil || !reflect.DeepEqual(loaded.Scene, editor.Document()) {
		t.Fatal(loaded, err)
	}
	path := filepath.Join(root, p.Manifest.EntryScene)
	before, _ := os.ReadFile(path)
	bad := editor.Document()
	bad.Entities[0].Mass = -1
	if p.Save(bad) == nil {
		t.Fatal("invalid save accepted")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("failed save changed file")
	}
	if _, err = Create(root); err == nil {
		t.Fatal("existing project overwritten")
	}
	after, _ = os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("create overwrote existing data")
	}
}
func TestEditorHistoryAndSnapshotOwnership(t *testing.T) {
	e, _ := NewEditor(NewDocument())
	original := e.Document()
	if err := e.Set("box", "mass", "bad"); err == nil {
		t.Fatal("invalid numeric property accepted")
	}
	if e.Dirty() {
		t.Fatal("failed edit marked dirty")
	}
	duplicate, err := e.Duplicate("box")
	if err != nil {
		t.Fatal(err)
	}
	if duplicate == "box" {
		t.Fatal("duplicate reused ID")
	}
	if !e.Undo() || !reflect.DeepEqual(e.Document(), original) {
		t.Fatal("undo")
	}
	if !e.Redo() {
		t.Fatal("redo")
	}
	snapshot := e.Document()
	snapshot.Entities[0].Name = "MUTATED"
	if e.Document().Entities[0].Name == "MUTATED" {
		t.Fatal("snapshot aliases authoring state")
	}
	e.MarkSaved()
	if e.Dirty() {
		t.Fatal("save point lost")
	}
	e.Delete(duplicate)
	if !e.Dirty() {
		t.Fatal("delete not dirty")
	}
	e.Undo()
	if e.Dirty() {
		t.Fatal("undo did not restore save point")
	}
}
func TestOpenRejectsMalformedAndEscapingDocuments(t *testing.T) {
	for _, manifest := range []string{"version: 99\nname: test\nentry_scene: scenes/main.yaml", "version: 1\nname: test\nentry_scene: ../outside.yaml", "version: 1\nname: test\nentry_scene: scenes/main.yaml\nunknown: x"} {
		root := filepath.Join(t.TempDir(), "Project")
		_, err := Create(root)
		if err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(root, "project.yaml"), []byte(manifest), 0600)
		if _, err = Open(root); err == nil {
			t.Fatal("invalid manifest accepted")
		}
	}
	d := NewDocument()
	d.Entities = append(d.Entities, d.Entities[0])
	if d.Validate() == nil {
		t.Fatal("duplicate identity accepted")
	}
}

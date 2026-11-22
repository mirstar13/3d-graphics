package project

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func resourceFixture(t *testing.T) (*Project, *Editor, string) {
	t.Helper()
	p, err := Create(filepath.Join(t.TempDir(), "Game"))
	if err != nil {
		t.Fatal(err)
	}
	path, err := p.Extract(nestedDocument(), "root", "Assembly")
	if err != nil {
		t.Fatal(err)
	}
	ed, err := NewEditor(p.Scene)
	if err != nil {
		t.Fatal(err)
	}
	return p, ed, path
}
func entityByID(t *testing.T, d Document, id string) Entity {
	t.Helper()
	for _, e := range d.Entities {
		if e.ID == id {
			return e
		}
	}
	t.Fatal("missing", id)
	return Entity{}
}
func TestResourceInstancesRoundTripOverridesAndRefresh(t *testing.T) {
	p, ed, path := resourceFixture(t)
	first, err := ed.Instantiate(p, path)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ed.Instantiate(p, path)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("instance identity shared")
	}
	child := ed.Document().Instances[0].Members["child"]
	if err = ed.Set(child, "red", "33"); err != nil {
		t.Fatal(err)
	}
	r, err := p.LoadResource(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := range r.Scene.Entities {
		r.Scene.Entities[i].Color.R = 99
	}
	if err = p.SaveResource(path, r); err != nil {
		t.Fatal(err)
	}
	before := ed.Document()
	if err = ed.RefreshLinks(p); err != nil {
		t.Fatal(err)
	}
	after := ed.Document()
	if entityByID(t, after, child).Color.R != 33 || entityByID(t, after, second).Color.R != 99 {
		t.Fatal("source override propagation")
	}
	if !ed.Undo() || !reflect.DeepEqual(before, ed.Document()) || !ed.Redo() {
		t.Fatal("refresh history")
	}
	if err = p.Save(ed.Document()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(p.Root, p.Manifest.EntryScene))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), "shape:") != 2 || !strings.Contains(string(data), "instances:") {
		t.Fatal("serialized expanded entities", string(data))
	}
	reopened, err := Open(p.Root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(after, reopened.Scene) {
		t.Fatal("semantic round trip", after, reopened.Scene)
	}
	snapshot := ed.Document()
	snapshot.Instances[0].Members["child"] = "tampered"
	snapshot.Instances[0].Overrides["child"]["red"] = "44"
	if ed.Document().Instances[0].Members["child"] != child || entityByID(t, ed.Document(), child).Color.R != 33 {
		t.Fatal("snapshot aliases")
	}
	if err = ed.ResetOverride(p, child, "red"); err != nil {
		t.Fatal(err)
	}
	if entityByID(t, ed.Document(), child).Color.R != 99 {
		t.Fatal("reset did not restore source")
	}
	if !ed.Undo() || entityByID(t, ed.Document(), child).Color.R != 33 {
		t.Fatal("reset undo")
	}
}
func TestResourceStructuralChangesAndAtomicConflicts(t *testing.T) {
	p, ed, path := resourceFixture(t)
	root, err := ed.Instantiate(p, path)
	if err != nil {
		t.Fatal(err)
	}
	child := ed.Document().Instances[0].Members["child"]
	for _, action := range []func() error{func() error { return ed.Delete(child) }, func() error { _, err := ed.Duplicate(child); return err }, func() error { return ed.Reparent(child, "") }, func() error { return ed.Reparent("box", root) }} {
		before := ed.Document()
		if action() == nil || !reflect.DeepEqual(before, ed.Document()) {
			t.Fatal("structural command accepted")
		}
	}
	if err = ed.Set(child, "name", "Custom"); err != nil {
		t.Fatal(err)
	}
	r, _ := p.LoadResource(path)
	r.Scene.Entities = []Entity{entityByID(t, r.Scene, "root")}
	if err = p.SaveResource(path, r); err != nil {
		t.Fatal(err)
	}
	before := ed.Document()
	if err = ed.RefreshLinks(p); err == nil || !strings.Contains(err.Error(), "overridden") {
		t.Fatal("missing conflict", err)
	}
	if !reflect.DeepEqual(before, ed.Document()) {
		t.Fatal("failed refresh changed state")
	}
	saved, _ := os.ReadFile(filepath.Join(p.Root, p.Manifest.EntryScene))
	if p.Save(ed.Document()) == nil {
		t.Fatal("saved invalid references")
	}
	still, _ := os.ReadFile(filepath.Join(p.Root, p.Manifest.EntryScene))
	if string(saved) != string(still) {
		t.Fatal("failed save changed file")
	}
	if err = ed.Unpack(child); err != nil {
		t.Fatal(err)
	}
	if len(ed.Document().Instances) != 0 || entityByID(t, ed.Document(), child).Name != "Custom" {
		t.Fatal("unpack lost content")
	}
	if err = p.Save(ed.Document()); err != nil {
		t.Fatal(err)
	}
	if !ed.Undo() || len(ed.Document().Instances) != 1 {
		t.Fatal("unpack undo")
	}
}
func TestResourceDuplicateDeleteAndSourceAddRemove(t *testing.T) {
	p, ed, path := resourceFixture(t)
	root, err := ed.Instantiate(p, path)
	if err != nil {
		t.Fatal(err)
	}
	copyID, err := ed.Duplicate(root)
	if err != nil {
		t.Fatal(err)
	}
	d := ed.Document()
	if len(d.Instances) != 2 || d.Instances[0].Members["child"] == d.Instances[1].Members["child"] {
		t.Fatal("copy identity")
	}
	if err = ed.RefreshLinks(p); err != nil {
		t.Fatal(err)
	}
	if entityByID(t, ed.Document(), copyID).Position.X != entityByID(t, ed.Document(), root).Position.X+1 {
		t.Fatal("copy placement override lost")
	}
	if err = ed.Delete(copyID); err != nil {
		t.Fatal(err)
	}
	if len(ed.Document().Instances) != 1 {
		t.Fatal("delete link leaked")
	}
	if !ed.Undo() || len(ed.Document().Instances) != 2 {
		t.Fatal("delete undo")
	}
	if err = ed.Unpack(copyID); err != nil {
		t.Fatal(err)
	}
	mixed := ed.Document()
	if err = p.Save(mixed); err != nil {
		t.Fatal(err)
	}
	loaded, err := Open(p.Root)
	if err != nil || !reflect.DeepEqual(mixed, loaded.Scene) {
		t.Fatal("mixed root order round trip", err)
	}
	ed.Undo()

	r, _ := p.LoadResource(path)
	r.Scene.Entities = r.Scene.Entities[1:] // remove leaf, serialized before its parents
	extra := NewEntity("new", "New")
	extra.Body = "none"
	extra.Parent = "root"
	r.Scene.Entities = append(r.Scene.Entities, extra)
	if err = p.SaveResource(path, r); err != nil {
		t.Fatal(err)
	}
	if err = ed.RefreshLinks(p); err != nil {
		t.Fatal(err)
	}
	d = ed.Document()
	if d.Instances[0].Members["leaf"] != "" || d.Instances[0].Members["new"] == "" {
		t.Fatal("add/remove mapping")
	}
	if err = p.Save(d); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(p.Root)
	if err != nil || !reflect.DeepEqual(d, reopened.Scene) {
		t.Fatal("new member round trip", err)
	}
}
func TestResourceInvalidSourcesAndPaths(t *testing.T) {
	p, ed, path := resourceFixture(t)
	_, err := ed.Instantiate(p, path)
	if err != nil {
		t.Fatal(err)
	}
	if err = p.Save(ed.Document()); err != nil {
		t.Fatal(err)
	}
	r, _ := p.LoadResource(path)
	bad := r
	bad.ID = "different"
	if p.SaveResource(path, bad) == nil {
		t.Fatal("identity replacement")
	}
	bad = r
	bad.Scene = ed.Document()
	if bad.Validate() == nil {
		t.Fatal("nested dependencies accepted")
	}
	full := filepath.Join(p.Root, path)
	if err = os.Remove(full); err != nil {
		t.Fatal(err)
	}
	if _, err = Open(p.Root); err == nil {
		t.Fatal("missing source accepted")
	}
	for _, path := range []string{"../outside.yaml", "resources/../../outside.yaml", "/tmp/outside.yaml"} {
		if _, err = p.LoadResource(path); err == nil {
			t.Fatal("bad path", path)
		}
	}
	// A nonexistent file under a symlinked resource directory cannot escape root.
	if err = os.Remove(filepath.Join(p.Root, "resources")); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err = os.Symlink(outside, filepath.Join(p.Root, "resources")); err != nil {
		t.Skip("symlink unavailable", err)
	}
	if _, err = p.Extract(nestedDocument(), "root", "Escape"); err == nil {
		t.Fatal("write escaped through directory symlink")
	}
	entries, _ := os.ReadDir(outside)
	if len(entries) != 0 {
		t.Fatal("outside write")
	}
}

func TestResourceReferenceValidationAndRefreshTransaction(t *testing.T) {
	p, ed, path := resourceFixture(t)
	_, err := ed.Instantiate(p, path)
	if err != nil {
		t.Fatal(err)
	}
	raw := ed.Document().compact()
	cases := []struct {
		name   string
		change func(*Document)
	}{
		{"identity", func(d *Document) { d.Instances[0].Resource = "replaced" }},
		{"root", func(d *Document) { d.Instances[0].Root = "missing" }},
		{"overlapping IDs", func(d *Document) { d.Instances[0].Members["child"] = d.Instances[0].Root }},
		{"unknown override", func(d *Document) { d.Instances[0].Overrides = map[string]map[string]string{"root": {"arbitrary": "1"}} }},
		{"invalid override", func(d *Document) { d.Instances[0].Overrides = map[string]map[string]string{"root": {"mass": "NaN"}} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bad := raw.Clone()
			tc.change(&bad)
			if _, err := p.Resolve(bad); err == nil {
				t.Fatal("invalid reference accepted")
			}
		})
	}
	// The first source is valid and changed; a later missing source must keep both
	// the expanded document and existing undo history intact.
	path2, err := p.Extract(nestedDocument(), "root", "Second")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ed.Instantiate(p, path2); err != nil {
		t.Fatal(err)
	}
	ed.MarkSaved()
	before := ed.Document()
	history := len(ed.past)
	r, _ := p.LoadResource(path)
	r.Scene.Entities[1].Mass = 2
	if err = p.SaveResource(path, r); err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(filepath.Join(p.Root, path2)); err != nil {
		t.Fatal(err)
	}
	if ed.RefreshLinks(p) == nil || !reflect.DeepEqual(before, ed.Document()) || len(ed.past) != history || ed.Dirty() {
		t.Fatal("failed multi-source refresh mutated state")
	}
	// Recursive resources fail locally, before any dependency expansion.
	r.Scene = raw
	if err = writeAtomic(filepath.Join(p.Root, path), r); err != nil {
		t.Fatal(err)
	}
	if _, err = p.LoadResource(path); err == nil || !strings.Contains(err.Error(), "nested") {
		t.Fatal("recursive resource accepted", err)
	}
	infos, err := p.Resources()
	if err != nil || len(infos) != 1 || infos[0].Error == "" {
		t.Fatal("invalid resource hidden", infos, err)
	}
}

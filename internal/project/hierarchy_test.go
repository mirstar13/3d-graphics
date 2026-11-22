package project

import (
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func nestedDocument() Document {
	root := NewEntity("root", "Assembly")
	child := NewEntity("child", "Detail")
	child.Body = "none"
	child.Parent = "root"
	leaf := NewEntity("leaf", "Detail")
	leaf.Body = "none"
	leaf.Parent = "child"
	return Document{Version: Version, Name: "Nested", Entities: []Entity{leaf, root, child}}
}
func TestHierarchyValidationAndOrder(t *testing.T) {
	d := nestedDocument()
	if err := d.Validate(); err != nil {
		t.Fatal(err)
	}
	tree := d.Hierarchy()
	for i, id := range []string{"root", "child", "leaf"} {
		if tree[i].ID != id || tree[i].Depth != i {
			t.Fatal(tree)
		}
	}
	for _, tc := range []struct {
		name   string
		change func(*Document)
	}{
		{"missing", func(d *Document) { d.Entities[0].Parent = "absent" }},
		{"self", func(d *Document) { d.Entities[0].Parent = "leaf" }},
		{"cycle", func(d *Document) { d.Entities[2].Parent = "leaf" }},
		{"nested body", func(d *Document) { d.Entities[0].Body = "static" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := d.Clone()
			tc.change(&bad)
			if bad.Validate() == nil {
				t.Fatal("accepted invalid hierarchy")
			}
		})
	}
	// Omitting parent retains compatibility with the original scene schema.
	if err := NewDocument().Validate(); err != nil {
		t.Fatal(err)
	}
}
func TestHierarchyCommandsAtomicAndUndoable(t *testing.T) {
	d := nestedDocument()
	ed, _ := NewEditor(d)
	for _, parent := range []string{"absent", "leaf", "child"} {
		if err := ed.Reparent("child", parent); err == nil {
			t.Fatal("bad reparent accepted", parent)
		}
		if ed.Dirty() || ed.Undo() {
			t.Fatal("failed command changed history")
		}
	}
	if err := ed.Reparent("root", "child"); err == nil {
		t.Fatal("parented body")
	}
	if err := ed.Set("child", "body", "dynamic"); err == nil {
		t.Fatal("nested physics property accepted")
	}
	if err := ed.Reparent("child", ""); err != nil {
		t.Fatal(err)
	}
	if got := ed.Document().Entities[2]; got.Position != d.Entities[2].Position || got.Parent != "" {
		t.Fatal(got)
	}
	if !ed.Undo() || !reflect.DeepEqual(d, ed.Document()) {
		t.Fatal("reparent undo")
	}
	copyID, err := ed.Duplicate("child")
	if err != nil {
		t.Fatal(err)
	}
	ids := ed.Document().SubtreeIDs(copyID)
	if len(ids) != 2 || ids[1] == "leaf" {
		t.Fatal("copy identities", ids)
	}
	byID := map[string]Entity{}
	for _, e := range ed.Document().Entities {
		byID[e.ID] = e
	}
	if byID[copyID].Parent != "root" || byID[ids[1]].Parent != copyID || byID[copyID].Position.X != d.Entities[2].Position.X+1 {
		t.Fatal("copy links/offset", byID)
	}
	if byID[ids[1]].Position != d.Entities[0].Position {
		t.Fatal("descendant offset changed")
	}
	ed.MarkSaved()
	before := ed.Document()
	if err := ed.Delete(copyID); err != nil {
		t.Fatal(err)
	}
	if len(ed.Document().Entities) != 3 {
		t.Fatal("subtree leaked")
	}
	if !ed.Undo() || ed.Dirty() || !reflect.DeepEqual(before, ed.Document()) {
		t.Fatal("delete undo")
	}
	if !ed.Redo() || len(ed.Document().Entities) != 3 {
		t.Fatal("delete redo")
	}
	choices := d.ParentChoices("child")
	if len(choices) != 1 || choices[0].ID != "root" || len(d.ParentChoices("root")) != 0 {
		t.Fatal("parent choices", choices)
	}
}
func TestHierarchyRoundTripAndDuplicateLimits(t *testing.T) {
	p, err := Create(filepath.Join(t.TempDir(), "Nested"))
	if err != nil {
		t.Fatal(err)
	}
	d := nestedDocument()
	if err = p.Save(d); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(p.Root)
	if err != nil || !reflect.DeepEqual(d, reopened.Scene) {
		t.Fatal("round trip", err)
	}
	d.Entities[1].Name = strings.Repeat("界", 80)
	ed, _ := NewEditor(d)
	if _, err = ed.Duplicate("root"); err != nil {
		t.Fatal("long name copy", err)
	}
	// An otherwise valid subtree copy exceeding the entity budget is atomic.
	for len(d.Entities) < MaxEntities {
		e := NewEntity("extra-"+strconv.Itoa(len(d.Entities)), "Extra")
		d.Entities = append(d.Entities, e)
	}
	ed, _ = NewEditor(d)
	if _, err = ed.Duplicate("child"); err == nil || ed.Dirty() || ed.Undo() {
		t.Fatal("budget copy changed state")
	}
}

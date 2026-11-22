package project

import "fmt"

// TreeEntry pairs an entity with its depth in a stable, parent-first traversal.
type TreeEntry struct {
	Entity
	Depth int
}

// Hierarchy returns a depth-first view of a validated document. Sibling order
// follows document order; the serialized entity order need not be parent-first.
func (d Document) Hierarchy() []TreeEntry {
	children := make(map[string][]Entity)
	for _, e := range d.Entities {
		children[e.Parent] = append(children[e.Parent], e)
	}
	var result []TreeEntry
	var visit func(string, int)
	visit = func(parent string, depth int) {
		for _, e := range children[parent] {
			result = append(result, TreeEntry{e, depth})
			visit(e.ID, depth+1)
		}
	}
	visit("", 0)
	return result
}

func (d Document) validateHierarchy() error {
	byID := make(map[string]Entity, len(d.Entities))
	for _, e := range d.Entities {
		byID[e.ID] = e
	}
	for _, e := range d.Entities {
		if e.Parent != "" && e.Body != "none" {
			return fmt.Errorf("entity %s: physics bodies must be roots; set Body to none before parenting", e.ID)
		}
		seen := map[string]bool{e.ID: true}
		for parent := e.Parent; parent != ""; {
			p, ok := byID[parent]
			if !ok {
				return fmt.Errorf("entity %s: missing parent %q", e.ID, parent)
			}
			if seen[parent] {
				return fmt.Errorf("entity %s: hierarchy cycle", e.ID)
			}
			seen[parent] = true
			parent = p.Parent
		}
	}
	return nil
}

// SubtreeIDs returns the selected entity and descendants in parent-first order.
// An unknown ID returns an empty slice.
func (d Document) SubtreeIDs(id string) []string {
	var ids []string
	included := map[string]bool{}
	for _, e := range d.Hierarchy() {
		if e.ID == id || included[e.Parent] {
			included[e.ID] = true
			ids = append(ids, e.ID)
		}
	}
	return ids
}

// Reparent changes only the parent link, preserving local position and rotation.
// Use an empty parent to detach to the root. Validation is atomic with undo history.
func (e *Editor) Reparent(id, parent string) error {
	next := e.Document()
	for i := range next.Entities {
		if next.Entities[i].ID == id {
			next.Entities[i].Parent = parent
			return e.change(next)
		}
	}
	return fmt.Errorf("entity no longer exists")
}

// ParentChoices returns valid parents in tree order, excluding the entity and
// its descendants. The root choice is represented by the UI separately.
func (d Document) ParentChoices(id string) []TreeEntry {
	excluded := map[string]bool{}
	for _, child := range d.SubtreeIDs(id) {
		excluded[child] = true
	}
	var choices []TreeEntry
	for _, entry := range d.Hierarchy() {
		if entry.ID == id && entry.Body != "none" {
			return nil
		}
		if !excluded[entry.ID] {
			choices = append(choices, entry)
		}
	}
	return choices
}

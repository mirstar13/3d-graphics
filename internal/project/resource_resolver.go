package project

import (
	"fmt"
	"path/filepath"
	"sort"
)

// Link locates the instance and source identity owning an expanded entity.
func (d Document) Link(id string) (int, string, bool) {
	for i, in := range d.Instances {
		for source, target := range in.Members {
			if target == id {
				return i, source, true
			}
		}
	}
	return 0, "", false
}
func property(key string) (Property, bool) {
	for _, p := range Properties {
		if p.Key == key {
			return p, true
		}
	}
	return Property{}, false
}

// compact removes expanded instance members; only references and overrides go on disk.
func (d Document) compact() Document {
	out := d.Clone()
	out.Entities = nil
	out.RootOrder = nil
	if len(d.Instances) > 0 {
		for _, entry := range d.Hierarchy() {
			if entry.Parent == "" {
				out.RootOrder = append(out.RootOrder, entry.ID)
			}
		}
	}
	for _, e := range d.Entities {
		if _, _, ok := d.Link(e.ID); !ok {
			out.Entities = append(out.Entities, e)
		}
	}
	return out
}

// Resolve expands an on-disk scene transactionally. Existing mappings keep their
// IDs; new source members get fresh IDs. Overrides on removed members are conflicts.
func (p *Project) Resolve(raw Document) (Document, error) {
	result := raw.Clone()
	result.Instances = nil
	result.RootOrder = nil
	if err := result.Validate(); err != nil {
		return Document{}, err
	}
	if len(raw.Instances) > MaxEntities {
		return Document{}, fmt.Errorf("too many instances")
	}
	resources := map[string]string{}
	loaded := map[string]Resource{}
	for _, link := range raw.Instances {
		r, ok := loaded[link.Path]
		if !ok {
			var err error
			r, err = p.LoadResource(link.Path)
			if err != nil {
				return Document{}, err
			}
			loaded[link.Path] = r
		}
		if r.ID != link.Resource {
			return Document{}, fmt.Errorf("resource %s identity mismatch", link.Path)
		}
		if prior, ok := resources[r.ID]; ok && prior != link.Path {
			return Document{}, fmt.Errorf("duplicate resource identity in %s and %s", prior, link.Path)
		}
		resources[r.ID] = link.Path
		in := cloneInstance(link)
		if in.Members == nil {
			in.Members = map[string]string{}
		}
		source := map[string]bool{}
		root := r.Scene.Hierarchy()[0].ID
		if in.Root == "" || in.Members[root] != in.Root {
			return Document{}, fmt.Errorf("instance root mapping missing or source root changed")
		}
		for _, e := range r.Scene.Entities {
			source[e.ID] = true
		}
		for id, values := range in.Overrides {
			if !source[id] && len(values) > 0 {
				return Document{}, fmt.Errorf("resource %s removed overridden entity %s; restore source or unpack instance", link.Path, id)
			}
		}
		for id := range in.Members {
			if !source[id] {
				delete(in.Members, id)
				delete(in.Overrides, id)
			}
		}
		for _, e := range r.Scene.Hierarchy() {
			if _, ok := in.Members[e.ID]; !ok {
				fresh, err := newID()
				if err != nil {
					return Document{}, err
				}
				in.Members[e.ID] = fresh
			}
		}
		for _, entry := range r.Scene.Hierarchy() {
			e := entry.Entity
			for key, value := range in.Overrides[e.ID] {
				prop, ok := property(key)
				if !ok {
					return Document{}, fmt.Errorf("unknown override %s", key)
				}
				if err := prop.Set(&e, value); err != nil {
					return Document{}, fmt.Errorf("override %s: %w", key, err)
				}
			}
			e.ID = in.Members[entry.ID]
			if e.Parent != "" {
				e.Parent = in.Members[e.Parent]
			}
			result.Entities = append(result.Entities, e)
			if len(result.Entities) > MaxEntities {
				return Document{}, fmt.Errorf("expanded scene exceeds %d entities", MaxEntities)
			}
		}
		result.Instances = append(result.Instances, in)
	}
	if err := result.Validate(); err != nil {
		return Document{}, err
	}
	if len(raw.RootOrder) > 0 {
		roots := map[string]string{}
		for _, entry := range result.Hierarchy() {
			if entry.Parent == "" {
				roots[entry.ID] = entry.ID
			} else {
				roots[entry.ID] = roots[entry.Parent]
			}
		}
		rank := map[string]int{}
		for i, id := range raw.RootOrder {
			if _, duplicate := rank[id]; duplicate || roots[id] != id {
				return Document{}, fmt.Errorf("invalid root order identity %s", id)
			}
			rank[id] = i
		}
		for id, root := range roots {
			if id == root {
				if _, ok := rank[id]; !ok {
					return Document{}, fmt.Errorf("root order omits %s", id)
				}
			}
		}
		sort.SliceStable(result.Entities, func(i, j int) bool { return rank[roots[result.Entities[i].ID]] < rank[roots[result.Entities[j].ID]] })
	}
	return result, nil
}

func cloneInstance(in Instance) Instance {
	members := make(map[string]string, len(in.Members))
	for k, v := range in.Members {
		members[k] = v
	}
	in.Members = members
	if in.Overrides != nil {
		overrides := make(map[string]map[string]string, len(in.Overrides))
		for id, values := range in.Overrides {
			copy := map[string]string{}
			for k, v := range values {
				copy[k] = v
			}
			overrides[id] = copy
		}
		in.Overrides = overrides
	}
	return in
}

func (d Document) validateInstances() error {
	if len(d.RootOrder) > 0 {
		return fmt.Errorf("resolve serialized root order before editing")
	}
	entities := map[string]Entity{}
	for _, e := range d.Entities {
		entities[e.ID] = e
	}
	claimed := map[string]bool{}
	for _, in := range d.Instances {
		if !ValidText(in.Resource, 80) {
			return fmt.Errorf("invalid resource identity")
		}
		if filepath.ToSlash(filepath.Dir(in.Path)) != "resources" || filepath.Ext(in.Path) != ".yaml" {
			return fmt.Errorf("invalid resource path")
		}
		root, ok := entities[in.Root]
		if !ok || root.Parent != "" {
			return fmt.Errorf("instance root missing or parented")
		}
		owned := map[string]bool{}
		for source, id := range in.Members {
			if !ValidText(source, 80) || claimed[id] {
				return fmt.Errorf("invalid or overlapping instance member %s", id)
			}
			if _, ok := entities[id]; !ok {
				return fmt.Errorf("instance member %s missing", id)
			}
			claimed[id] = true
			owned[id] = true
		}
		if !owned[in.Root] {
			return fmt.Errorf("instance root is not a member")
		}
		subtree := d.SubtreeIDs(in.Root)
		if len(subtree) != len(owned) {
			return fmt.Errorf("unpack instance before editing its structure")
		}
		for _, id := range subtree {
			if !owned[id] {
				return fmt.Errorf("unpack instance before parenting into it")
			}
		}
		for source, values := range in.Overrides {
			id, ok := in.Members[source]
			if !ok {
				return fmt.Errorf("override member missing")
			}
			for key, value := range values {
				prop, ok := property(key)
				if !ok {
					return fmt.Errorf("unknown override %s", key)
				}
				copy := entities[id]
				if err := prop.Set(&copy, value); err != nil {
					return err
				}
				if prop.Get(copy) != prop.Get(entities[id]) {
					return fmt.Errorf("override %s disagrees with expanded entity", key)
				}
			}
		}
	}
	return nil
}

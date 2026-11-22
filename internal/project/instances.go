package project

import "fmt"

func (d *Document) recordOverride(id, key, value string) {
	if i, source, ok := d.Link(id); ok {
		if d.Instances[i].Overrides == nil {
			d.Instances[i].Overrides = map[string]map[string]string{}
		}
		if d.Instances[i].Overrides[source] == nil {
			d.Instances[i].Overrides[source] = map[string]string{}
		}
		d.Instances[i].Overrides[source][key] = value
	}
}

// Instantiate loads a resource and adds a linked instance in one undoable command.
func (e *Editor) Instantiate(p *Project, path string) (string, error) {
	r, err := p.LoadResource(path)
	if err != nil {
		return "", err
	}
	in := Instance{Resource: r.ID, Path: path, Members: map[string]string{}}
	for _, entry := range r.Scene.Hierarchy() {
		id, err := newID()
		if err != nil {
			return "", err
		}
		in.Members[entry.ID] = id
		if entry.Parent == "" {
			in.Root = id
		}
	}
	raw := Document{Version: Version, Name: "Instance", Instances: []Instance{in}}
	expanded, err := p.Resolve(raw)
	if err != nil {
		return "", err
	}
	next := e.Document()
	next.Entities = append(next.Entities, expanded.Entities...)
	next.Instances = append(next.Instances, expanded.Instances...)
	if err = e.change(next); err != nil {
		return "", err
	}
	return in.Root, nil
}

// RefreshLinks reloads every source transactionally, preserving explicit overrides.
func (e *Editor) RefreshLinks(p *Project) error {
	next, err := p.Resolve(e.document.compact())
	if err != nil {
		return err
	}
	return e.change(next)
}

// ResetOverride restores one property from the current source. Other objects,
// properties and source changes are untouched until an explicit refresh.
func (e *Editor) ResetOverride(p *Project, id, key string) error {
	next := e.Document()
	i, source, ok := next.Link(id)
	if !ok {
		return fmt.Errorf("select a linked instance member")
	}
	in := &next.Instances[i]
	prop, ok := property(key)
	if !ok {
		return fmt.Errorf("unknown property")
	}
	r, err := p.LoadResource(in.Path)
	if err != nil {
		return err
	}
	if r.ID != in.Resource {
		return fmt.Errorf("resource identity mismatch")
	}
	for _, original := range r.Scene.Entities {
		if original.ID == source {
			for j := range next.Entities {
				if next.Entities[j].ID == id {
					if err = prop.Set(&next.Entities[j], prop.Get(original)); err != nil {
						return err
					}
				}
			}
			delete(in.Overrides[source], key)
			if len(in.Overrides[source]) == 0 {
				delete(in.Overrides, source)
			}
			if len(in.Overrides) == 0 {
				in.Overrides = nil
			}
			return e.change(next)
		}
	}
	return fmt.Errorf("source member missing; restore source or unpack instance")
}

// Unpack retains expanded values and IDs, removing the whole instance's link.
func (e *Editor) Unpack(id string) error {
	next := e.Document()
	i, _, ok := next.Link(id)
	if !ok {
		return fmt.Errorf("select a linked instance member")
	}
	next.Instances = append(next.Instances[:i], next.Instances[i+1:]...)
	if len(next.Instances) == 0 {
		next.Instances = nil
	}
	return e.change(next)
}

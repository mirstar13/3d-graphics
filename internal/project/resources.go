package project

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Resource is a reusable ordinary subtree with stable identity, independent of its filename.
// Nested linked resources are deliberately unsupported in this schema.
type Resource struct {
	Version int      `yaml:"version"`
	ID      string   `yaml:"id"`
	Scene   Document `yaml:"scene"`
}

// Instance links a resource to independent live document identities. Overrides
// address source IDs and inspector property keys, never display names.
type Instance struct {
	Root      string                       `yaml:"root"`
	Resource  string                       `yaml:"resource"`
	Path      string                       `yaml:"path"`
	Members   map[string]string            `yaml:"members"`
	Overrides map[string]map[string]string `yaml:"overrides,omitempty"`
}

func (r Resource) Validate() error {
	if r.Scene.SceneSettings != (SceneSettings{}) {
		return fmt.Errorf("global settings are only valid in scenes")
	}
	if r.Version != Version || !ValidText(r.ID, 80) {
		return fmt.Errorf("invalid resource version or ID")
	}
	if len(r.Scene.Instances) != 0 {
		return fmt.Errorf("nested linked resources are unsupported (including recursive links)")
	}
	if err := r.Scene.Validate(); err != nil {
		return err
	}
	roots := 0
	for _, e := range r.Scene.Entities {
		if e.Parent == "" {
			roots++
		}
	}
	if roots != 1 {
		return fmt.Errorf("resource must contain exactly one root")
	}
	return nil
}

func resourcePath(root, path string) (string, error) {
	if filepath.ToSlash(filepath.Dir(path)) != "resources" || filepath.Ext(path) != ".yaml" {
		return "", fmt.Errorf("resource must be resources/<file>.yaml")
	}
	return scenePath(root, path)
}

// LoadResource validates the complete source before it can replace editor state.
func (p *Project) LoadResource(path string) (Resource, error) {
	full, err := resourcePath(p.Root, path)
	if err != nil {
		return Resource{}, err
	}
	var r Resource
	if err = decode(full, &r); err == nil {
		err = r.Validate()
	}
	if err != nil {
		return Resource{}, fmt.Errorf("resource %s: %w", path, err)
	}
	return r, nil
}

// SaveResource preserves identity and atomically replaces an existing source.
func (p *Project) SaveResource(path string, r Resource) error {
	if err := r.Validate(); err != nil {
		return err
	}
	old, err := p.LoadResource(path)
	if err != nil {
		return err
	}
	if old.ID != r.ID {
		return fmt.Errorf("resource identity changed")
	}
	full, err := resourcePath(p.Root, path)
	if err != nil {
		return err
	}
	return writeAtomic(full, r)
}

// Extract writes a new resource without changing the source document. Linked
// selections must first be unpacked, making the loss of linkage explicit.
func (p *Project) Extract(d Document, id, name string) (string, error) {
	if err := d.Validate(); err != nil {
		return "", err
	}
	if !ValidText(name, 80) {
		return "", fmt.Errorf("resource name must contain 1..80 printable characters")
	}
	ids := d.SubtreeIDs(id)
	if len(ids) == 0 {
		return "", fmt.Errorf("entity no longer exists")
	}
	selected := map[string]bool{}
	for _, id := range ids {
		selected[id] = true
		if _, _, ok := d.Link(id); ok {
			return "", fmt.Errorf("unpack linked instances before extracting")
		}
	}
	scene := Document{Version: Version, Name: name}
	for _, e := range d.Entities {
		if selected[e.ID] {
			if e.ID == id {
				e.Parent = ""
			}
			scene.Entities = append(scene.Entities, e)
		}
	}
	rid, err := newID()
	if err != nil {
		return "", err
	}
	r := Resource{Version: Version, ID: rid, Scene: scene}
	if err = r.Validate(); err != nil {
		return "", err
	}
	path := "resources/" + rid + ".yaml"
	full, err := resourcePath(p.Root, path)
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return "", err
	}
	// Reserve only this new file; existing assets can never be overwritten by extraction.
	f, err := os.OpenFile(full, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
	}
	f.Close()
	if err = writeAtomic(full, r); err != nil {
		os.Remove(full)
		return "", err
	}
	return path, nil
}

// ResourceInfo is browser metadata. Invalid resources remain visible with an error.
type ResourceInfo struct{ Path, Name, ID, Error string }

// Resources lists up to 256 source files, retaining invalid entries for diagnosis.
func (p *Project) Resources() ([]ResourceInfo, error) {
	// Use a child path to check the resources directory even before it exists.
	checked, err := resourcePath(p.Root, "resources/probe.yaml")
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Dir(checked))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var result []ResourceInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if len(result) >= 256 {
			return nil, fmt.Errorf("resource browser limit is 256 files")
		}
		info := ResourceInfo{Path: "resources/" + entry.Name(), Name: fmt.Sprintf("%q", entry.Name())}
		r, err := p.LoadResource(info.Path)
		if err != nil {
			info.Error = err.Error()
		} else {
			info.Name = r.Scene.Name
			info.ID = r.ID
		}
		result = append(result, info)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Name == result[j].Name {
			return result[i].Path < result[j].Path
		}
		return result[i].Name < result[j].Name
	})
	return result, nil
}

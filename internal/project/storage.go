package project

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mirstar13/go-3d-graphics/internal/input"
	"gopkg.in/yaml.v3"
)

// Project is a loaded manifest and authored scene, separate from editor history.
type Project struct {
	Root     string
	Manifest Manifest
	Scene    Document
}

func decode(path string, out any) error {
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	data, e := io.ReadAll(io.LimitReader(f, 1<<20+1))
	if e != nil {
		return e
	}
	if len(data) > 1<<20 {
		return fmt.Errorf("document exceeds 1 MiB")
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if e = decoder.Decode(out); e != nil {
		return e
	}
	var extra any
	if e = decoder.Decode(&extra); e != io.EOF {
		return fmt.Errorf("expected exactly one YAML document")
	}
	return nil
}
func scenePath(root, entry string) (string, error) {
	if entry == "" || filepath.IsAbs(entry) || filepath.VolumeName(entry) != "" {
		return "", fmt.Errorf("entry_scene must be project-relative")
	}
	path := filepath.Join(root, entry)
	rel, e := filepath.Rel(root, path)
	if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("entry_scene escapes project")
	}
	// Resolve the nearest existing ancestor, not just the final file. A new
	// resource beneath a symlinked directory must obey the same containment rule.
	base, e := filepath.EvalSymlinks(root)
	if e != nil {
		return "", e
	}
	probe := path
	for {
		resolved, err := filepath.EvalSymlinks(probe)
		if err == nil {
			rel, err = filepath.Rel(base, resolved)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return "", fmt.Errorf("project path resolves outside project")
			}
			break
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		// A dangling symlink is an error, not an absent ordinary destination.
		if info, err := os.Lstat(probe); err == nil && info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("dangling project symlink")
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			return "", fmt.Errorf("cannot resolve project path")
		}
		probe = parent
	}

	return path, nil
}

// Open validates every document before returning a replacement for the current project.
func Open(root string) (*Project, error) {
	root, e := filepath.Abs(root)
	if e != nil {
		return nil, e
	}
	p := &Project{Root: root}
	if e = decode(filepath.Join(root, "project.yaml"), &p.Manifest); e != nil {
		return nil, fmt.Errorf("project: %w", e)
	}
	if p.Manifest.Version != Version || !ValidText(p.Manifest.Name, 80) {
		return nil, fmt.Errorf("unsupported project version or invalid name")
	}
	if e = input.Validate(p.Manifest.Bindings); e != nil {
		return nil, e
	}
	for key, action := range p.Manifest.GameplayBindings {
		if !ValidText(key, 40) || !ValidText(action, 64) ||
			strings.TrimSpace(key) != key || strings.TrimSpace(action) != action {
			return nil, fmt.Errorf("invalid gameplay binding %q", key)
		}
	}
	p.Scene, e = p.LoadScene(p.Manifest.EntryScene)
	if e != nil {
		return nil, e
	}
	return p, nil
}

// LoadScene validates and resolves a project-relative scene without changing the project.
func (p *Project) LoadScene(entry string) (Document, error) {
	path, err := scenePath(p.Root, entry)
	if err != nil {
		return Document{}, err
	}
	var raw Document
	if err := decode(path, &raw); err != nil {
		return Document{}, fmt.Errorf("scene: %w", err)
	}
	resolved, err := p.Resolve(raw)
	if err != nil {
		return Document{}, err
	}
	if err := resolved.validateVisualFiles(p.Root); err != nil {
		return Document{}, err
	}
	return resolved, nil
}
func writeAtomic(path string, value any) error {
	data, e := yaml.Marshal(value)
	if e != nil {
		return e
	}
	if len(data) > 1<<20 {
		return fmt.Errorf("document exceeds 1 MiB")
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".scene-*.yaml")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if _, e = f.Write(data); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(f.Name(), path)
}

// Create creates a new project directory, refusing to overwrite an existing one.
func Create(root string) (*Project, error) {
	root, e := filepath.Abs(root)
	if e != nil {
		return nil, e
	}
	if e = os.Mkdir(root, 0755); e != nil {
		return nil, e
	}
	successful := false
	defer func() {
		if !successful {
			os.RemoveAll(root)
		}
	}()
	if e = os.Mkdir(filepath.Join(root, "scenes"), 0755); e != nil {
		return nil, e
	}
	if e = os.Mkdir(filepath.Join(root, "assets"), 0755); e != nil {
		return nil, e
	}
	name := filepath.Base(root)
	if !ValidText(name, 80) {
		return nil, fmt.Errorf("invalid project directory name")
	}
	p := &Project{Root: root, Manifest: Manifest{Version: Version, Name: name, EntryScene: "scenes/main.yaml"}, Scene: NewDocument()}
	if e = writeAtomic(filepath.Join(root, "project.yaml"), p.Manifest); e != nil {
		return nil, e
	}
	for file, contents := range map[string]string{"go.mod": starterModule, "go.sum": starterSums, "main.go": starterMain} {
		if e = os.WriteFile(filepath.Join(root, file), []byte(contents), 0644); e != nil {
			return nil, e
		}
	}
	if e = p.Save(p.Scene); e != nil {
		return nil, e
	}
	successful = true
	return p, nil
}

// Save atomically replaces the entry scene only after complete validation.
func (p *Project) Save(d Document) error {
	if e := d.Validate(); e != nil {
		return e
	}
	path, e := scenePath(p.Root, p.Manifest.EntryScene)
	if e != nil {
		return e
	}
	serialized := d.compact()
	resolved, e := p.Resolve(serialized)
	if e != nil {
		return e
	}
	if e = resolved.validateVisualFiles(p.Root); e != nil {
		return e
	}
	if e = writeAtomic(path, serialized); e != nil {
		return e
	}
	p.Scene = d.Clone()
	return nil
}

// SaveBindings atomically persists project input preferences independently of scene undo.
func (p *Project) SaveBindings(bindings input.Bindings) error {
	if err := input.Validate(bindings); err != nil {
		return err
	}
	next := p.Manifest
	next.Bindings = input.Clone(bindings)
	if err := writeAtomic(filepath.Join(p.Root, "project.yaml"), next); err != nil {
		return err
	}
	p.Manifest = next
	return nil
}

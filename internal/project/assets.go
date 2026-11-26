package project

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

// CollisionShape returns the effective box or sphere collider primitive.
func (e Entity) CollisionShape() string {
	if e.ColliderShape != "" {
		return e.ColliderShape
	}
	return e.Shape
}

// CollisionSize returns the half extents, or the sphere radius in X.
func (e Entity) CollisionSize() engine.Point {
	s := e.ColliderSize
	if s.X == 0 {
		s.X = e.Size.X
	}
	if s.Y == 0 {
		s.Y = e.Size.Y
	}
	if s.Z == 0 {
		s.Z = e.Size.Z
	}
	return s
}

func validAsset(name, ext string) bool {
	return fs.ValidPath(name) && strings.HasPrefix(name, "assets/") && strings.EqualFold(path.Ext(name), ext) && !strings.ContainsAny(name, "\\:#") && ValidText(name, 512)
}

func (e Entity) validateAssets() error {
	if e.MeshAsset != "" && !validAsset(e.MeshAsset, ".obj") {
		return fmt.Errorf("mesh must be assets/path.obj")
	}
	if e.MaterialAsset != "" {
		file, name, ok := strings.Cut(e.MaterialAsset, "#")
		if !ok || !validAsset(file, ".mtl") || !ValidText(name, 80) {
			return fmt.Errorf("material must be assets/path.mtl#name")
		}
	}
	return nil
}

// LoadVisualAssets reads an independent mesh and material for this entity.
// Nil results select the primitive mesh or its existing material assignments.
func (e Entity) LoadVisualAssets(root string) (*engine.Mesh, *engine.Material, error) {
	if err := e.validateAssets(); err != nil {
		return nil, nil, err
	}
	if root == "" && (e.MeshAsset != "" || e.MaterialAsset != "") {
		return nil, nil, fmt.Errorf("visual assets require a project root")
	}
	var mesh *engine.Mesh
	if e.MeshAsset != "" {
		var err error
		mesh, err = engine.LoadProjectOBJ(root, e.MeshAsset)
		if err != nil {
			return nil, nil, err
		}
		if len(mesh.Indices) == 0 {
			return nil, nil, fmt.Errorf("mesh %s contains no faces", e.MeshAsset)
		}
	}
	if e.MaterialAsset == "" {
		return mesh, nil, nil
	}
	file, name, _ := strings.Cut(e.MaterialAsset, "#")
	lib, err := engine.LoadProjectMTL(root, file)
	if err != nil {
		return nil, nil, err
	}
	m := lib.Materials[name]
	if m == nil {
		return nil, nil, fmt.Errorf("unknown material %q in %s", name, file)
	}
	return mesh, m, nil
}

func (d Document) validateVisualFiles(root string) error {
	for _, e := range d.Entities {
		if _, _, err := e.LoadVisualAssets(root); err != nil {
			return fmt.Errorf("entity %s: %w", e.ID, err)
		}
	}
	return nil
}

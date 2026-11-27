package project

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

// MeshAssets returns project-relative mesh paths in lexical order.
func (p *Project) MeshAssets() ([]string, error) {
	root, err := os.OpenRoot(p.Root)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	var paths []string
	err = fs.WalkDir(root.FS(), "assets", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type().IsRegular() && validMeshAsset(name) {
			paths = append(paths, name)
		}
		return nil
	})
	if os.IsNotExist(err) {
		return nil, nil
	}
	return paths, err
}

func validMeshAsset(name string) bool { return validAsset(name, ".obj") || validAsset(name, ".glb") }

// ImportGLB validates a self-contained asset before copying it into assets/.
func (p *Project) ImportGLB(source string) (string, error) {
	name := path.Join("assets", filepath.Base(source))
	if !validAsset(name, ".glb") {
		return "", fmt.Errorf("choose an exported .glb file")
	}
	file, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if info, err := file.Stat(); err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("import requires a regular GLB file")
	}
	data, err := io.ReadAll(io.LimitReader(file, engine.MaxGLBBytes+1))
	if err != nil {
		return "", err
	}
	if _, err = engine.ReadGLB(bytes.NewReader(data)); err != nil {
		return "", err
	}

	root, err := os.OpenRoot(p.Root)
	if err != nil {
		return "", err
	}
	defer root.Close()
	if err := writeImportedAsset(root, name, data); err != nil {
		return "", err
	}
	return name, nil
}

func writeImportedAsset(root *os.Root, name string, data []byte) error {
	if err := root.Mkdir("assets", 0755); err != nil && !os.IsExist(err) {
		return err
	}
	dest, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, writeErr := dest.Write(data)
	closeErr := dest.Close()
	if writeErr != nil || closeErr != nil {
		_ = root.Remove(name)
		if writeErr != nil {
			return writeErr
		}
		return closeErr
	}
	return nil
}

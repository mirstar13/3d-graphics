package project

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

// Export rebuilds the saved project and atomically publishes a host-platform
// ZIP under build/. Runtime data belongs in assets/, resources/ and scenes/.
// The launcher starts the executable with the extracted directory as its cwd.
func (p *Project) Export(ctx context.Context) (string, error) {
	saved, err := Open(p.Root)
	if err != nil {
		return "", err
	}
	if saved.Manifest.BehaviorWorker != "" {
		if _, err = saved.WorkerExecutable(); err != nil {
			return "", fmt.Errorf("behavior worker: %w", err)
		}
	}
	build, err := scenePath(saved.Root, "build")
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(build, 0755); err != nil {
		return "", err
	}
	stage, err := os.MkdirTemp(build, ".export-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stage)
	tree := filepath.Join(stage, "package")
	if err = os.Mkdir(tree, 0755); err != nil {
		return "", err
	}
	copyFile := func(src, dst string, mode os.FileMode) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		srcFile, err := os.OpenInRoot(saved.Root, src)
		if err != nil {
			return err
		}
		defer srcFile.Close()
		info, err := srcFile.Stat()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported export file %s", src)
		}
		path := filepath.Join(tree, dst)
		if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		dest, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
		if err != nil {
			return err
		}
		_, err = io.Copy(exportWriter{ctx, dest}, srcFile)
		closeErr := dest.Close()
		if err != nil {
			return err
		}
		return closeErr
	}
	entry := filepath.Clean(saved.Manifest.EntryScene)
	files := []string{"project.yaml", entry}
	if info, err := os.Lstat(filepath.Join(saved.Root, "renderer.yaml")); err == nil && entry != "renderer.yaml" {
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("export requires a regular renderer.yaml")
		}
		files = append(files, "renderer.yaml")
	} else if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	for _, path := range files {
		if err = copyFile(path, path, 0644); err != nil {
			return "", err
		}
	}
	for _, dir := range []string{"scenes", "resources", "assets"} {
		base := filepath.Join(saved.Root, dir)
		if _, err = os.Lstat(base); os.IsNotExist(err) {
			continue
		}
		err = filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			rel, err := filepath.Rel(saved.Root, path)
			if err != nil {
				return err
			}
			if d.IsDir() {
				return os.MkdirAll(filepath.Join(tree, rel), 0755)
			}
			if !d.Type().IsRegular() {
				return fmt.Errorf("export requires regular files: %s", rel)
			}
			if rel == entry {
				return nil
			}
			mode := os.FileMode(0644)
			if filepath.ToSlash(rel) == saved.Manifest.BehaviorWorker {
				mode = 0755
			}
			return copyFile(rel, rel, mode)
		})
		if err != nil {
			return "", err
		}
	}
	if _, err = Open(tree); err != nil {
		return "", fmt.Errorf("exported scene: %w", err)
	}

	launcher, script := "play.sh", "#!/bin/sh\ncd -- \"$(dirname -- \"$0\")\" || exit 1\nexec ./game \"$@\"\n"
	if runtime.GOOS == "windows" {
		launcher, script = "play.cmd", "@echo off\r\npushd \"%~dp0\" || exit /b 1\r\ngame.exe %*\r\nset rc=%errorlevel%\r\npopd\r\nexit /b %rc%\r\n"
	}
	for _, reserved := range []string{gameExecutable(), launcher} {
		if _, err := os.Lstat(filepath.Join(tree, reserved)); err == nil {
			return "", fmt.Errorf("package path %s already in use", reserved)
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}
	if err = saved.compile(ctx, filepath.Join(tree, gameExecutable())); err != nil {
		return "", err
	}
	if err = os.WriteFile(filepath.Join(tree, launcher), []byte(script), 0755); err != nil {
		return "", err
	}
	archive := filepath.Join(stage, "package.zip")
	out, err := os.Create(archive)
	if err != nil {
		return "", err
	}
	writer := zip.NewWriter(exportWriter{ctx, out})
	err = writer.AddFS(os.DirFS(tree))
	zipErr := writer.Close()
	fileErr := out.Close()
	for _, failure := range []error{err, zipErr, fileErr, ctx.Err()} {
		if failure != nil {
			return "", failure
		}
	}
	name := fmt.Sprintf("game-%s-%s.zip", runtime.GOOS, runtime.GOARCH)
	if err = os.Rename(archive, filepath.Join(build, name)); err != nil {
		return "", err
	}
	return filepath.Join("build", name), nil
}

// exportWriter checks cancellation between filesystem writes during a large export.
type exportWriter struct {
	ctx context.Context
	dst io.Writer
}

func (w exportWriter) Write(data []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	return w.dst.Write(data)
}

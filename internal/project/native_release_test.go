package project

import (
	"archive/zip"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestNativePulseDockRelease runs the shipped sample via the native launcher
// after relocation. The child gets scripted input, allowing headless CI use.
func TestNativePulseDockRelease(t *testing.T) {
	output := runNativeRelease(t, "pulse-dock", strings.Repeat("d\n", 12)+"r\nw\nw\nd\nd\nd\nd\ns\ns\nq\n", nil)
	lost, won := strings.Index(output, "Lost | R restart"), strings.Index(output, "Won | R restart")
	if lost < 0 || won <= lost {
		t.Fatalf("missing loss, restart or victory:\n%s", output)
	}
}

func TestNativeRelayRoomsRelease(t *testing.T) {
	output := runNativeRelease(t, "relay-rooms", "u\nv\nv\nr\nv\nq\n", func(root string) {
		manifest := filepath.Join(root, "project.yaml")
		data, err := os.ReadFile(manifest)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(manifest, []byte(strings.Replace(string(data), "  u: use", "  v: use", 1)), 0644); err != nil {
			t.Fatal(err)
		}
	})
	if strings.Count(output, "scenes/relay.yaml\n") != 2 || strings.Count(output, "Exit beacon activated") != 2 {
		t.Fatalf("expected rebound action, scene change, and restart in the exit room:\n%s", output)
	}
}

func runNativeRelease(t *testing.T, example, input string, prepare func(string)) string {
	t.Helper()
	parent := t.TempDir()
	root := filepath.Join(parent, "Sample source")
	if err := os.CopyFS(root, os.DirFS(filepath.Join("../../examples", example))); err != nil {
		t.Fatal(err)
	}
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	modCmd := exec.Command("go", "mod", "edit", "-replace=github.com/mirstar13/go-3d-graphics="+repo)
	modCmd.Dir = root
	if output, err := modCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod edit: %v\n%s", err, output)
	}
	if prepare != nil {
		prepare(root)
	}
	p, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	path, err := p.Export(ctx)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.OpenReader(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	release := filepath.Join(parent, "Release Game")
	for _, file := range archive.File {
		dst := filepath.Join(release, filepath.FromSlash(file.Name))
		if file.FileInfo().IsDir() {
			if err = os.MkdirAll(dst, 0755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err = os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			t.Fatal(err)
		}
		src, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, file.Mode().Perm())
		if err != nil {
			t.Fatal(err)
		}
		_, copyErr := io.Copy(out, src)
		err = out.Close()
		src.Close()
		if copyErr != nil || err != nil {
			t.Fatal(copyErr, err)
		}
	}
	if err = archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(root, root+"-unavailable"); err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, filepath.Join(release, "play.sh"))
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd.exe", "/d", "/c", "call", `Release Game\play.cmd`)
	}
	cmd.Dir = parent
	cmd.Stdin = strings.NewReader(input)
	data, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("launcher: %v\n%s", err, data)
	}
	return string(data)
}

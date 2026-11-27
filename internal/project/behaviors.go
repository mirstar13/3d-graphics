package project

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func (e Entity) validateBehavior() error {
	if e.Behavior != "" && (!ValidText(e.Behavior, 80) || strings.TrimSpace(e.Behavior) != e.Behavior) {
		return fmt.Errorf("invalid behavior name")
	}
	if e.BehaviorArgs != "" {
		var data map[string]json.RawMessage
		if e.Behavior == "" || !ValidText(e.BehaviorArgs, 512) || json.Unmarshal([]byte(e.BehaviorArgs), &data) != nil || data == nil {
			return fmt.Errorf("behavior parameters must be a JSON object of up to 512 characters")
		}
	}
	return nil
}

func validWorkerPath(name string) bool {
	return fs.ValidPath(name) && strings.HasPrefix(name, "assets/") && !strings.ContainsAny(name, "\\:#") && ValidText(name, 512)
}

// WorkerExecutable resolves the configured program inside the project.
func (p *Project) WorkerExecutable() (string, error) {
	name := p.Manifest.BehaviorWorker
	if !validWorkerPath(name) {
		return "", fmt.Errorf("set behavior_worker to an executable under assets/")
	}
	root, err := filepath.EvalSymlinks(p.Root)
	if err != nil {
		return "", err
	}
	file, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(name)))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, file)
	if err != nil || !filepath.IsLocal(rel) || !strings.HasPrefix(filepath.ToSlash(rel), "assets/") {
		return "", fmt.Errorf("behavior worker must resolve inside assets/")
	}
	info, err := os.Stat(file)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("behavior worker must be a regular file")
	}
	return file, nil
}

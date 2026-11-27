package project

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// New projects use the public API revision that introduced fixed-tick behaviors.
// The repository URL and its declared Go module path differ.
const starterModule = `module example.org/terminalgame

go 1.24.6

require github.com/mirstar13/go-3d-graphics v0.0.0

replace github.com/mirstar13/go-3d-graphics => github.com/mirstar13/3d-graphics v0.0.0-20261126211025-9200f63b835a
`

const starterSums = `gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405/go.mod h1:Co6ibVJAznAaIkqp8huTwlJQCZ016jof/cbN4VW5Yz0=
gopkg.in/yaml.v3 v3.0.1 h1:fxVm/GzAzEWqLHuvctI91KS9hhNmmWOoWu0XTYJS7CA=
gopkg.in/yaml.v3 v3.0.1/go.mod h1:K4uyk7z7BCEPqu6E+C64Yfv1cQ7kz7rIZviUmN+EgEM=
github.com/mirstar13/3d-graphics v0.0.0-20261126211025-9200f63b835a h1:98nmwvWYXM18c3yqXNs6kiaax1Efpp6CVtiFe8SIRjE=
github.com/mirstar13/3d-graphics v0.0.0-20261126211025-9200f63b835a/go.mod h1:kpY0umyhzWQIfi848UjSYMqs8wE2+hM/q6IdmQpTW7o=
`

const starterMain = `package main

import (
	"log"

	"github.com/mirstar13/go-3d-graphics/creator"
)

// update runs before each fixed physics tick.
func update(world *creator.World, tick creator.Tick) error {
	return nil
}

func main() {
	game, err := creator.Open(".", update)
	if err != nil {
		log.Fatal(err)
	}
	if err := game.Step(); err != nil {
		log.Fatal(err)
	}
}
`

type compilerOutput struct {
	data      bytes.Buffer
	truncated bool
}

func (o *compilerOutput) Write(p []byte) (int, error) {
	n := len(p)
	space := 65536 - o.data.Len()
	if n > space {
		o.truncated = true
		p = p[:space]
	}
	_, err := o.data.Write(p)
	return n, err
}

// Build compiles the project's root main package into build/game (game.exe on
// Windows). It publishes a completed executable only after a successful build.
// Compiler errors include their file and line locations.
func (p *Project) Build(ctx context.Context) (string, error) {
	module, err := scenePath(p.Root, "go.mod")
	if err != nil {
		return "", err
	}
	if _, err = os.Stat(module); err != nil {
		return "", fmt.Errorf("project requires go.mod and a root main package: %w", err)
	}
	dir, err := scenePath(p.Root, "build")
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	stage, err := os.MkdirTemp(dir, ".compile-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stage)
	name := gameExecutable()
	tmp := filepath.Join(stage, name)
	cmd := exec.CommandContext(ctx, "go", "build", "-mod=mod", "-buildmode=exe", "-buildvcs=false", "-o", tmp, ".")
	cmd.Dir = p.Root
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOOS="+runtime.GOOS, "GOARCH="+runtime.GOARCH)
	var output compilerOutput
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err = cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		msg := strings.TrimSpace(output.data.String())
		if output.truncated {
			msg += "\n(compiler output truncated)"
		}
		if msg != "" {
			return "", fmt.Errorf("%w\n%s", err, msg)
		}
		return "", err
	}
	dest := filepath.Join(dir, name)
	if err = os.Rename(tmp, dest); err != nil {
		return "", err
	}
	return filepath.Join("build", name), nil
}

func gameExecutable() string {
	if runtime.GOOS == "windows" {
		return "game.exe"
	}
	return "game"
}

// RunBuilt starts the last successful host build in the project directory. It
// waits for exit, returning bounded stdout/stderr and the process error. Cancel
// the context to stop the game. The editor retains ownership of terminal input.
func (p *Project) RunBuilt(ctx context.Context) (string, error) {
	path, err := scenePath(p.Root, filepath.Join("build", gameExecutable()))
	if err != nil {
		return "", err
	}
	if _, err = os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("build the project with Ctrl+B first")
	}
	cmd := exec.CommandContext(ctx, path)
	cmd.Dir = p.Root
	cmd.WaitDelay = time.Second
	var output compilerOutput
	cmd.Stdout = &output
	cmd.Stderr = &output
	err = cmd.Run()
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	text := strings.TrimSpace(output.data.String())
	if output.truncated {
		text += "\n(game output truncated)"
	}
	return text, err
}

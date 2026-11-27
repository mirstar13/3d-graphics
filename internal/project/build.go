package project

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mirstar13/go-3d-graphics/internal/audio"
)

// The starter pins the creator API by its repository URL.
//
//go:embed starter/go.mod.tmpl
var starterModule string

//go:embed starter/go.sum.tmpl
var starterSums string

const starterMain = `package main

import (
	"log"
	"os"

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
	defer game.Close()
	if len(os.Args) == 2 && os.Args[1] == "-view" {
		if err := game.RunTerminal("Q quit"); err != nil {
			log.Fatal(err)
		}
		return
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
	if err = p.compile(ctx, tmp); err != nil {
		return "", err
	}
	dest := filepath.Join(dir, name)
	if err = os.Rename(tmp, dest); err != nil {
		return "", err
	}
	return filepath.Join("build", name), nil
}

func (p *Project) compile(ctx context.Context, tmp string) error {
	module, err := scenePath(p.Root, "go.mod")
	if err != nil {
		return err
	}
	if _, err = os.Stat(module); err != nil {
		return fmt.Errorf("project requires go.mod and a root main package: %w", err)
	}
	args := []string{"build", "-mod=mod", "-buildmode=exe", "-buildvcs=false", "-o", tmp}
	if audio.BuildTag != "" {
		args = append(args, "-tags", audio.BuildTag)
	}
	cmd := exec.CommandContext(ctx, "go", append(args, ".")...)
	cmd.Dir = p.Root
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOOS="+runtime.GOOS, "GOARCH="+runtime.GOARCH)
	var output compilerOutput
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err = cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		msg := strings.TrimSpace(output.data.String())
		if output.truncated {
			msg += "\n(compiler output truncated)"
		}
		if msg != "" {
			return fmt.Errorf("%w\n%s", err, msg)
		}
		return err
	}
	return nil
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
	cmd, err := p.BuiltCommand(ctx)
	if err != nil {
		return "", err
	}
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

// BuiltCommand prepares a child for a host that provides its own terminal.
func (p *Project) BuiltCommand(ctx context.Context, args ...string) (*exec.Cmd, error) {
	path, err := scenePath(p.Root, filepath.Join("build", gameExecutable()))
	if err != nil {
		return nil, err
	}
	if _, err = os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("build the project with Ctrl+B first")
	}
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Dir = p.Root
	cmd.WaitDelay = time.Second
	return cmd, nil
}

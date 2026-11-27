package game

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"github.com/mirstar13/go-3d-graphics/internal/project"
)

func TestBehaviorLifecycleAndRejectedTick(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test fixture uses POSIX sh")
	}
	projectRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(projectRoot, "assets"), 0755); err != nil {
		t.Fatal(err)
	}
	program := `#!/bin/sh
IFS= read -r first || exit 1
case "$first" in
  *'"tick":1'*'"start":true'*'"actions":["launch"]'*'"parameters":{"strength":3}'*) ;;
  *) exit 1 ;;
esac
printf '%s\n' '{"version":1,"tick":1,"impulses":[{"entity":"crate","vector":[0,3,0]}]}'
IFS= read -r second || exit 1
case "$second" in *'"tick":2'*'"start":false'*'"actions":[]'*) ;; *) exit 1 ;; esac
printf '%s\n' '{"version":1,"tick":2,"impulses":[{"entity":"crate","vector":[0,3,0]},{"entity":"crate","vector":[0,7,0]}],"sounds":["undeclared"]}'
`
	if err := os.WriteFile(filepath.Join(projectRoot, "assets", "worker"), []byte(program), 0755); err != nil {
		t.Fatal(err)
	}
	p := &project.Project{Root: projectRoot, Manifest: project.Manifest{BehaviorWorker: "assets/worker", GameplayBindings: map[string]string{"e": "launch"}}}
	crate := project.NewEntity("crate", "Crate")
	crate.Behavior, crate.BehaviorArgs = "launch", `{"strength":3}`
	doc := project.Document{Version: 1, Name: "Behavior test", Entities: []project.Entity{crate}}
	session, err := NewProjectSession(projectRoot, doc, physics.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	session.EnableBehaviors(context.Background(), p)
	defer session.CloseBehaviors()
	session.World.SetPaused(true)
	session.BehaviorInput("e")
	if err = session.Step(); err != nil {
		t.Fatal(err)
	}
	id, _ := session.Find("crate")
	before, err := session.Inspect(id)
	if err != nil || before.Body.Velocity.Y <= 2 {
		t.Fatal("worker impulse did not run", err)
	}
	if err = session.Step(); err == nil || session.World.Stats().Tick != 1 || session.PendingCommands() != 0 {
		t.Fatal("invalid response partially applied", err, session.World.Stats().Tick, session.PendingCommands())
	}
}

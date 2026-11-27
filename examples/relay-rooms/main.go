// Relay Rooms demonstrates a rebindable action, a scene change, and restarting
// the current room in a standalone project using the public creator API.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mirstar13/go-3d-graphics/creator"
)

func relayBehavior(out io.Writer) creator.Behavior {
	return func(w *creator.World, _ creator.Tick) error {
		if !w.ActionPressed("use") {
			return nil
		}
		hit, found, err := w.Pick(creator.Vec3{}, creator.Vec3{Z: 1}, 4)
		if err != nil || !found {
			return err
		}
		state, err := w.Inspect(hit)
		if err != nil {
			return err
		}
		switch state.ID {
		case "relay":
			if err := w.ChangeScene("scenes/exit.yaml"); err != nil {
				fmt.Fprintf(out, "Relay unavailable: %v\n", err)
			}
		case "exit":
			fmt.Fprintln(out, "Exit beacon activated")
		}
		return nil
	}
}

func play(in io.Reader, out io.Writer) error {
	game, err := creator.Open(".", relayBehavior(out))
	if err != nil {
		return err
	}
	fmt.Fprintln(out, "Submit an action key from project.yaml (default U), R restart, Q quit")
	keys := bufio.NewScanner(in)
	for {
		fmt.Fprintln(out, game.Scene())
		if !keys.Scan() {
			return keys.Err()
		}
		switch key := strings.ToLower(strings.TrimSpace(keys.Text())); key {
		case "q":
			return nil
		case "r":
			err = game.Restart(relayBehavior(out))
		default:
			err = game.StepWithKeys(key)
		}
		if err != nil {
			return err
		}
	}
}

// runView shows the authored room with script-controlled camera input.
func runView() error {
	g, err := creator.Open(".", func(w *creator.World, _ creator.Tick) error {
		if w.ActionPressed("mouse_look") {
			w.CaptureMouse(true)
		}
		x, y := w.MouseDelta()
		right, up := float64(x)*.035, -float64(y)*.06
		if w.ActionPressed("look_left") {
			right -= .12
		}
		if w.ActionPressed("look_right") {
			right += .12
		}
		if w.ActionPressed("look_up") {
			up += .1
		}
		if w.ActionPressed("look_down") {
			up -= .1
		}
		w.TurnCamera(right, up)
		if w.ActionPressed("use") {
			eye, direction := w.CameraRay()
			entity, ok, err := w.Pick(eye, direction, 100)
			if err != nil || !ok {
				return err
			}
			state, err := w.Inspect(entity)
			if err != nil {
				return err
			}
			if state.ID == "relay" {
				return w.ChangeScene("scenes/exit.yaml")
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return g.RunTerminal("M mouse look | Arrows look | U use")
}

func main() {
	var err error
	if len(os.Args) == 2 && os.Args[1] == "-view" {
		err = runView()
	} else {
		err = play(os.Stdin, os.Stdout)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Relay Rooms demonstrates a rebindable action, a scene change, and restarting
// the current room in a standalone project using the public creator API.
package main

import (
	"bufio"
	"fmt"
	"io"
	"math"
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

func runView() error {
	status := ""
	restart := func() creator.Behavior {
		status = "Find the relay"
		finished := false
		return func(w *creator.World, _ creator.Tick) error {
			if finished {
				return nil
			}
			if w.ActionPressed("mouse_look") {
				w.CaptureMouse(true)
			}
			w.MouseLook()
			var right, up float64
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
			if err := walk(w); err != nil {
				return err
			}
			if !w.ActionPressed("use") {
				return nil
			}
			eye, direction := w.CameraRay()
			entity, ok, err := w.Pick(eye, direction, 3)
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}
			state, err := w.Inspect(entity)
			if err != nil {
				return err
			}
			switch state.ID {
			case "relay":
				if err := w.ChangeScene("scenes/exit.yaml"); err != nil {
					return err
				}
				status = "Find the exit beacon"
			case "exit":
				status = "Rooms complete"
				finished = true
				w.CaptureMouse(false)
			}
			return nil
		}
	}
	g, err := creator.Open(".", restart())
	if err != nil {
		return err
	}
	return g.RunTerminalWithOptions(creator.TerminalOptions{
		Help:   "WASD walk | Arrows look | M mouse | U use",
		Status: func() string { return status }, Restart: restart, Reticle: true,
	})
}

func walk(w *creator.World) error {
	var forward, right float64
	if w.ActionPressed("move_forward") {
		forward++
	}
	if w.ActionPressed("move_back") {
		forward--
	}
	if w.ActionPressed("move_left") {
		right--
	}
	if w.ActionPressed("move_right") {
		right++
	}
	if forward == 0 && right == 0 {
		return nil
	}
	eye, view := w.CameraRay()
	yaw := math.Atan2(view.X, view.Z)
	advance, strafe := math.Sincos(yaw)
	speed := .22 / math.Hypot(forward, right)
	dir := creator.Vec3{X: speed * (forward*advance + right*strafe), Z: speed * (forward*strafe - right*advance)}
	_, blocked, err := w.Pick(eye, dir, .38)
	if err == nil && !blocked {
		eye.X += dir.X
		eye.Z += dir.Z
		w.SetCameraPosition(eye)
	}
	return err
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

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

func main() {
	if err := play(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

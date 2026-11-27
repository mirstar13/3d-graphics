// Pulse Dock is a turn-based zero-gravity docking game. The tug must steer
// around a pylon and touch the beacon before its tick budget expires.
//
// From the engine checkout, open the sample with:
//
//	go run . -project examples/pulse-dock
//
// For a real-time 3D run, use go run . -play-project examples/pulse-dock
// -player tug from the engine checkout (both flags on one command line).
// The editable Go entry point opens the game menu in an interactive terminal.
// Pass -text to use the turn-based radar cockpit.
//
// Copy the entire sample directory before experimenting. Move the pylon, change
// the beacon collider or edit the objective deadline in Ctrl+E. Keep the three
// entity IDs tug, pylon and beacon so the cockpit can identify their markers.
// Save and use F9 to export. Extract the ZIP and start play.sh or play.cmd.
// From the sample directory, go run . runs the same game during development.
package main

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"github.com/mirstar13/go-3d-graphics/creator"
	"golang.org/x/term"
)

const ticksPerBurn = 15

// drawRadar plots entity centres on the X/Z plane in half-metre cells.
func drawRadar(out io.Writer, g *creator.Game) error {
	var grid [9][17]byte
	for i := range grid {
		for j := range grid[i] {
			grid[i][j] = '.'
		}
	}
	for _, marker := range []struct {
		id    string
		glyph byte
	}{{"pylon", '#'}, {"beacon", 'O'}, {"tug", '@'}} {
		id, ok := g.World().Find(marker.id)
		if !ok {
			return fmt.Errorf("missing scene entity %q", marker.id)
		}
		state, err := g.World().Inspect(id)
		if err != nil {
			return err
		}
		x, z := 4+int(math.Round(2*state.Position.X)), 4-int(math.Round(2*state.Position.Z))
		if x >= 0 && x < len(grid[0]) && z >= 0 && z < len(grid) {
			grid[z][x] = marker.glyph
		}
		if marker.id == "tug" {
			fmt.Fprintf(out, "Tug X %.1f  Z %.1f\n", state.Position.X, state.Position.Z)
		}
	}
	for _, row := range grid {
		fmt.Fprintln(out, string(row[:]))
	}
	return nil
}

func play(in io.Reader, out io.Writer) error {
	reader := bufio.NewScanner(in)
	fmt.Fprintln(out, "Pulse Dock\n@ tug   # pylon   O docking beacon")
	for {
		g, err := creator.Open(".", nil)
		if err != nil {
			return err
		}
		tug, ok := g.World().Find("tug")
		if !ok {
			return fmt.Errorf("scene needs a tug")
		}
		for {
			o := g.Objective()
			if o.Outcome == "" {
				return fmt.Errorf("configure a scene objective in Ctrl+E")
			}
			fmt.Fprintln(out, o.Text)
			if err := drawRadar(out, g); err != nil {
				return err
			}
			if o.Outcome == "Playing" {
				if o.TickLimit > 0 {
					fmt.Fprintf(out, "Ticks left: %d\n", o.TickLimit-o.Tick)
				}
				fmt.Fprintln(out, "W north / S south / A west / D east / . brake / R restart / Q quit, then Enter")
			} else {
				fmt.Fprintf(out, "%s | R restart / Q quit\n", o.Outcome)
			}
			if !reader.Scan() {
				return reader.Err()
			}
			key := strings.ToLower(strings.TrimSpace(reader.Text()))
			if key == "q" {
				return nil
			}
			if key == "r" {
				break
			}
			if o.Outcome != "Playing" {
				continue
			}
			var velocity creator.Vec3
			switch key {
			case "w":
				velocity.Z = 4
			case "s":
				velocity.Z = -4
			case "a":
				velocity.X = -4
			case "d":
				velocity.X = 4
			case ".":
			default:
				continue
			}
			state, err := g.World().Inspect(tug)
			if err != nil {
				return err
			}
			impulse := creator.Vec3{X: velocity.X - state.Velocity.X, Z: velocity.Z - state.Velocity.Z}
			if err = g.World().Impulse(tug, impulse); err != nil {
				return err
			}
			for i := 0; i < ticksPerBurn; i++ {
				if err = g.Step(); err != nil {
					return err
				}
			}
		}
	}
}
func fly(w *creator.World, _ creator.Tick) error {
	var impulse creator.Vec3
	switch {
	case w.ActionPressed("west"):
		impulse.X = -1
	case w.ActionPressed("east"):
		impulse.X = 1
	case w.ActionPressed("north"):
		impulse.Z = 1
	case w.ActionPressed("south"):
		impulse.Z = -1
	default:
		return nil
	}
	tug, ok := w.Find("tug")
	if !ok {
		return fmt.Errorf("scene needs a tug")
	}
	if err := w.Impulse(tug, impulse); err != nil {
		return err
	}
	return w.PlaySound("thruster")
}

func view() error {
	g, err := creator.Open(".", fly)
	if err != nil {
		return err
	}
	return g.RunTerminalWithOptions(creator.TerminalOptions{
		Title: "Pulse Dock", Instructions: "Guide the tug around the pylon and into the beacon.",
		Help: "Arrows fire thrusters. Inertia carries the tug between burns.",
	})
}

func main() {
	var err error
	if len(os.Args) == 2 && os.Args[1] == "-view" ||
		len(os.Args) == 1 && term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())) {
		err = view()
	} else {
		err = play(os.Stdin, os.Stdout)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

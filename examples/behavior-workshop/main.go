package main

import (
	"fmt"
	"log"

	"github.com/mirstar13/go-3d-graphics/creator"
)

func round(world *creator.World, tick creator.Tick) error {
	blue, ok := world.Find("right")
	if !ok {
		return fmt.Errorf("scene needs the blue crate")
	}
	state, err := world.Inspect(blue)
	if err != nil {
		return err
	}
	if state.Position.Y > 2 {
		return world.FinishRound(true)
	}
	if tick.Index >= 600 {
		return world.FinishRound(false)
	}
	return nil
}

func main() {
	game, err := creator.Open(".", round)
	if err != nil {
		log.Fatal(err)
	}
	if err := game.RunTerminalWithOptions(creator.TerminalOptions{
		Title: "Behavior Workshop", Instructions: "Launch the blue crate above two metres within ten seconds.",
		Help: "E launches both crates. Build a C# or C++ worker before playing.",
	}); err != nil {
		log.Fatal(err)
	}
}

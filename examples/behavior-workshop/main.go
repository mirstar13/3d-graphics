package main

import (
	"log"

	"github.com/mirstar13/go-3d-graphics/creator"
)

func main() {
	game, err := creator.Open(".", nil)
	if err != nil {
		log.Fatal(err)
	}
	if err := game.RunTerminal("E launch both crates | Q quit"); err != nil {
		log.Fatal(err)
	}
}

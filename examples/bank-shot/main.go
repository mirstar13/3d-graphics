package main

import (
	"log"

	"github.com/mirstar13/go-3d-graphics/internal/app"
)

func main() {
	if err := app.RunExample(".", "bank-shot"); err != nil {
		log.Fatal(err)
	}
}

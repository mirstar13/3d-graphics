// terminal-benchmark runs deterministic renderer and gameplay workloads.
package main

import (
	"fmt"
	"os"

	"github.com/mirstar13/go-3d-graphics/internal/app"
)

func main() {
	if err := app.RunTerminalBenchmark(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

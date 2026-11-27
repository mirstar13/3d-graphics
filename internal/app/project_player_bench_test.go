package app

import (
	"fmt"
	"testing"

	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"github.com/mirstar13/go-3d-graphics/internal/project"
)

// BenchmarkPulseDockView measures one monochrome player frame with the HUD.
// Disk loading, simulation and terminal I/O are outside the measured loop.
func BenchmarkPulseDockView(b *testing.B) {
	p, err := project.Open("../../examples/pulse-dock")
	if err != nil {
		b.Fatal(err)
	}
	for _, size := range [][2]int{{80, 24}, {120, 40}} {
		b.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(b *testing.B) {
			m := projectPlayer{project: p, playerID: "tug", options: physics.DefaultOptions(), width: size[0], height: size[1]}
			if err := m.reset(); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				m.View()
			}
		})
	}
}

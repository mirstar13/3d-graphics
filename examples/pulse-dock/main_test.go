package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestDockingRound(t *testing.T) {
	// Burn straight into the pylon until timeout, restart, then detour north of the solid pylon.
	input := strings.Repeat("d\n", 12) + "r\nw\nw\nd\nd\nd\nd\ns\ns\nq\n"
	var out bytes.Buffer
	if err := play(strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	lost, won := strings.Index(out.String(), "Lost | R restart"), strings.Index(out.String(), "Won | R restart")
	if lost < 0 || won <= lost {
		t.Fatalf("expected timeout followed by successful docking:\n%s", out.String())
	}
}

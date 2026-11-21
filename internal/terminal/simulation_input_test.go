package terminal

import (
	"bufio"
	"bytes"
	"strings"
	"sync"
	"testing"

	"github.com/eiannone/keyboard"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

func enqueueEvent(sim *SilentInputManager, event keyboard.KeyEvent) {
	sim.mutex.Lock()
	defer sim.mutex.Unlock()
	sim.enqueueSimulationEventLocked(event)
}
func TestSimulationEventsSurviveClearAndDrain(t *testing.T) {
	sim := NewSilentInputManager()
	for _, event := range []keyboard.KeyEvent{{Key: keyboard.KeySpace}, {Rune: 'n'}, {Rune: 'N'}, {Key: keyboard.KeyEnter}} {
		enqueueEvent(sim, event)
	}
	sim.ClearKeys()
	batch := sim.DrainSimulationCommands()
	want := []engine.SimulationCommand{engine.PauseSimulation, engine.StepSimulation, engine.StepSimulation, engine.ResumeSimulation}
	if len(batch.Commands) != len(want) {
		t.Fatal(batch)
	}
	for i, c := range want {
		if batch.Commands[i] != c {
			t.Fatal(batch)
		}
	}
	enqueueEvent(sim, keyboard.KeyEvent{Rune: 'n'})
	sim.ClearKeys()
	if len(sim.DrainSimulationCommands().Commands) != 1 || len(sim.DrainSimulationCommands().Commands) != 0 {
		t.Fatal("drain lost or repeated events")
	}
	for i := 0; i < engine.SimulationCommandCapacity+3; i++ {
		enqueueEvent(sim, keyboard.KeyEvent{Rune: 'n'})
	}
	batch = sim.DrainSimulationCommands()
	if len(batch.Commands) != engine.SimulationCommandCapacity || batch.Dropped != 3 {
		t.Fatal(batch)
	}
	if sim.DrainSimulationCommands().Dropped != 0 {
		t.Fatal("drop counter not drained")
	}
}
func TestConcurrentSimulationDrain(t *testing.T) {
	sim := NewSilentInputManager()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			enqueueEvent(sim, keyboard.KeyEvent{Rune: 'n'})
		}
	}()
	received := uint64(0)
	for i := 0; i < 1000; i++ {
		batch := sim.DrainSimulationCommands()
		received += uint64(len(batch.Commands)) + batch.Dropped
		sim.ClearKeys()
	}
	wg.Wait()
	batch := sim.DrainSimulationCommands()
	received += uint64(len(batch.Commands)) + batch.Dropped
	if received != 1000 {
		t.Fatalf("accounted for %d/1000 input events", received)
	}
}
func TestPausedStatusWithoutDebugAndClearOnResume(t *testing.T) {
	var out bytes.Buffer
	renderer := NewTerminalRenderer(bufio.NewWriter(&out), 5, 80)
	renderer.ShowDebugInfo = false
	renderer.SetSimulationStatus(engine.SimulationStatus{Paused: true, Hz: 60, Tick: 4})
	renderer.showDebugLine()
	if !strings.Contains(out.String(), "PAUSED 60Hz Tick:4 | N step | Enter resume") {
		t.Fatal(out.String())
	}
	out.Reset()
	renderer.SetSimulationStatus(engine.SimulationStatus{Hz: 60, Tick: 4})
	renderer.showDebugLine()
	if !strings.Contains(out.String(), "\033[K") || renderer.lastDebugLine != "" {
		t.Fatal("stale paused status")
	}
	renderer.ShowDebugInfo = true
	renderer.SetSimulationStatus(engine.SimulationStatus{Hz: 60, DroppedSeconds: 1.5, DroppedCommands: 2})
	if !strings.Contains(renderer.statusLine(), "Drop:1.500s Cmd:2") {
		t.Fatal(renderer.statusLine())
	}
}

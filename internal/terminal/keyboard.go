package terminal

import (
	"sync"

	"github.com/eiannone/keyboard"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

// SilentInputManager - Input system that reads keyboard without interfering with rendering
type SilentInputManager struct {
	commands        []engine.SimulationCommand
	droppedCommands uint64
	keys            map[rune]bool
	mutex           sync.RWMutex
	running         bool
	stopChan        chan struct{}
	done            chan struct{}
	quit            bool
	settings        bool
	lifecycle       sync.Mutex
}

// NewSilentInputManager creates a new silent input manager
func NewSilentInputManager() *SilentInputManager {
	return &SilentInputManager{
		keys:    make(map[rune]bool),
		running: false,
	}
}

// Start begins reading keyboard input in a separate goroutine
func (sim *SilentInputManager) Start() error {
	sim.lifecycle.Lock()
	defer sim.lifecycle.Unlock()
	if sim.running {
		return nil
	}
	events, err := keyboard.GetKeys(10)
	if err != nil {
		return err
	}
	sim.mutex.Lock()
	sim.keys = make(map[rune]bool)
	sim.quit = false
	sim.settings = false
	sim.commands = nil
	sim.droppedCommands = 0
	sim.mutex.Unlock()
	sim.stopChan = make(chan struct{})
	sim.done = make(chan struct{})
	sim.running = true
	go sim.readEvents(events)
	return nil
}

func (sim *SilentInputManager) readEvents(events <-chan keyboard.KeyEvent) {
	defer close(sim.done)
	for {
		select {
		case <-sim.stopChan:
			return
		case event, ok := <-events:
			sim.mutex.Lock()
			if !ok || event.Err != nil {
				sim.quit = true
				sim.mutex.Unlock()
				return
			}
			sim.enqueueSimulationEventLocked(event)
			sim.keys[event.Rune] = true
			switch event.Key {
			case keyboard.KeyEsc, keyboard.KeyCtrlC:
				sim.quit = true
			case keyboard.KeyArrowUp:
				sim.keys['i'] = true
			case keyboard.KeyArrowDown:
				sim.keys['k'] = true
			case keyboard.KeyArrowLeft:
				sim.keys['j'] = true
			case keyboard.KeyArrowRight:
				sim.keys['l'] = true
			}
			if event.Rune == 'p' || event.Rune == 'P' {
				sim.settings = true
			}
			if event.Rune == 'x' || event.Rune == 'X' {
				sim.quit = true
			}
			sim.mutex.Unlock()
		}
	}
}

// Stop cancels the event consumer before restoring the terminal.
func (sim *SilentInputManager) Stop() {
	sim.lifecycle.Lock()
	defer sim.lifecycle.Unlock()
	if !sim.running {
		return
	}
	close(sim.stopChan)
	<-sim.done
	keyboard.Close()
	sim.running = false
}

// SetKey manually sets a key state
func (sim *SilentInputManager) SetKey(key rune, pressed bool) {
	sim.mutex.Lock()
	defer sim.mutex.Unlock()
	sim.keys[key] = pressed
}

// GetInputState returns the current input state
func (sim *SilentInputManager) GetInputState() engine.InputState {
	sim.mutex.RLock()
	defer sim.mutex.RUnlock()

	return engine.InputState{
		Forward:  sim.keys['w'] || sim.keys['W'],
		Backward: sim.keys['s'] || sim.keys['S'],
		Left:     sim.keys['a'] || sim.keys['A'],
		Right:    sim.keys['d'] || sim.keys['D'],
		Up:       sim.keys['e'] || sim.keys['E'], // E = move up
		Down:     sim.keys['q'] || sim.keys['Q'], // Q = move down
		RotLeft:  sim.keys['j'] || sim.keys['J'], // J = rotate left
		RotRight: sim.keys['l'] || sim.keys['L'], // L = rotate right
		RotUp:    sim.keys['i'] || sim.keys['I'], // I = rotate up
		RotDown:  sim.keys['k'] || sim.keys['K'], // K = rotate down
		SpeedUp:  sim.keys['+'] || sim.keys['='],
		SlowDown: sim.keys['-'] || sim.keys['_'],
		Reset:    sim.keys['r'] || sim.keys['R'],
		Settings: sim.settings,
		Quit:     sim.quit || sim.keys['x'] || sim.keys['X'],
	}
}

// ClearKeys clears all key states
func (sim *SilentInputManager) ClearKeys() {
	sim.mutex.Lock()
	defer sim.mutex.Unlock()
	sim.keys = make(map[rune]bool)
}

// enqueueSimulationEventLocked records one terminal event. Caller holds mutex.
func (sim *SilentInputManager) enqueueSimulationEventLocked(event keyboard.KeyEvent) {
	var command engine.SimulationCommand
	switch {
	case event.Key == keyboard.KeySpace || event.Rune == ' ':
		command = engine.PauseSimulation
	case event.Key == keyboard.KeyEnter || event.Rune == '\r' || event.Rune == '\n':
		command = engine.ResumeSimulation
	case event.Rune == 'c' || event.Rune == 'C':
		command = engine.TogglePhysicsDebug
	case event.Rune == 't' || event.Rune == 'T':
		command = engine.ToggleFrameTiming
	case event.Rune == 'n' || event.Rune == 'N':
		command = engine.StepSimulation
	default:
		return
	}
	if len(sim.commands) >= engine.SimulationCommandCapacity {
		sim.droppedCommands++
		return
	}
	sim.commands = append(sim.commands, command)
}

// DrainSimulationCommands atomically consumes queued commands. Events arriving
// after the drain belong to the next batch, even if ClearKeys runs in between.
func (sim *SilentInputManager) DrainSimulationCommands() engine.SimulationCommands {
	sim.mutex.Lock()
	defer sim.mutex.Unlock()
	batch := engine.SimulationCommands{Commands: sim.commands, Dropped: sim.droppedCommands}
	sim.commands = nil
	sim.droppedCommands = 0
	return batch
}

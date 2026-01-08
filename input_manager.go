package main

import (
	"fmt"

	"github.com/go-gl/glfw/v3.3/glfw"
)

// InputManager interface for different input backends
type InputManager interface {
	Start() error
	Stop()
	GetInputState() InputState
	ClearKeys()
	ShouldClose() bool
}

// TerminalInputManager wraps SilentInputManager for terminal backend
type TerminalInputManager struct {
	*SilentInputManager
}

// NewTerminalInputManager creates a terminal input manager
func NewTerminalInputManager() *TerminalInputManager {
	return &TerminalInputManager{
		SilentInputManager: NewSilentInputManager(),
	}
}

// Start initializes terminal input
func (tim *TerminalInputManager) Start() error {
	fmt.Println("[Input] Starting terminal input manager...")
	tim.SilentInputManager.Start()
	fmt.Println("[Input] Terminal input manager started successfully")
	return nil
}

// ShouldClose checks if terminal should close (never for terminal)
func (tim *TerminalInputManager) ShouldClose() bool {
	state := tim.GetInputState()
	return state.Quit
}

// GLFWInputManager handles input for GLFW-based renderers (OpenGL/Vulkan)
type GLFWInputManager struct {
	window *glfw.Window
	state  InputState
}

// NewGLFWInputManager creates a GLFW input manager
func NewGLFWInputManager(window *glfw.Window) *GLFWInputManager {
	if window == nil {
		panic("NewGLFWInputManager: window parameter is nil. Ensure renderer.Initialize() is called before creating input manager.")
	}

	manager := &GLFWInputManager{
		window: window,
	}

	// Set up key callbacks
	window.SetKeyCallback(manager.keyCallback)

	return manager
}

// Start initializes GLFW input (no-op, callbacks already set)
func (gim *GLFWInputManager) Start() error {
	return nil
}

// Stop cleans up GLFW input
func (gim *GLFWInputManager) Stop() {
	// Clear callbacks
	if gim.window != nil {
		gim.window.SetKeyCallback(nil)
	}
}

// GetInputState returns current input state
func (gim *GLFWInputManager) GetInputState() InputState {
	if gim.window == nil {
		return InputState{}
	}

	// Poll for current key states
	state := InputState{}

	// Movement keys
	state.Forward = gim.window.GetKey(glfw.KeyW) == glfw.Press
	state.Backward = gim.window.GetKey(glfw.KeyS) == glfw.Press
	state.Left = gim.window.GetKey(glfw.KeyA) == glfw.Press
	state.Right = gim.window.GetKey(glfw.KeyD) == glfw.Press

	// Vertical movement
	state.Up = gim.window.GetKey(glfw.KeyE) == glfw.Press
	state.Down = gim.window.GetKey(glfw.KeyQ) == glfw.Press

	// Rotation keys
	state.RotLeft = gim.window.GetKey(glfw.KeyJ) == glfw.Press ||
		gim.window.GetKey(glfw.KeyLeft) == glfw.Press
	state.RotRight = gim.window.GetKey(glfw.KeyL) == glfw.Press ||
		gim.window.GetKey(glfw.KeyRight) == glfw.Press
	state.RotUp = gim.window.GetKey(glfw.KeyI) == glfw.Press ||
		gim.window.GetKey(glfw.KeyUp) == glfw.Press
	state.RotDown = gim.window.GetKey(glfw.KeyK) == glfw.Press ||
		gim.window.GetKey(glfw.KeyDown) == glfw.Press

	// Speed control
	state.SpeedUp = gim.window.GetKey(glfw.KeyEqual) == glfw.Press
	state.SlowDown = gim.window.GetKey(glfw.KeyMinus) == glfw.Press

	// Reset
	state.Reset = gim.window.GetKey(glfw.KeyR) == glfw.Press

	// Quit
	state.Quit = gim.window.GetKey(glfw.KeyX) == glfw.Press ||
		gim.window.GetKey(glfw.KeyEscape) == glfw.Press

	return state
}

// ClearKeys clears key state (no-op for GLFW)
func (gim *GLFWInputManager) ClearKeys() {
	// GLFW handles key state internally
}

// ShouldClose checks if window should close
func (gim *GLFWInputManager) ShouldClose() bool {
	if gim.window == nil {
		return true // If window is gone, we should close
	}
	return gim.window.ShouldClose()
}

// keyCallback handles key events
func (gim *GLFWInputManager) keyCallback(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	// Handle quit keys
	if action == glfw.Press {
		if key == glfw.KeyEscape || key == glfw.KeyX {
			w.SetShouldClose(true)
		}
	}
}

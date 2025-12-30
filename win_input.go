package main

import (
	"math"
	"sync"

	"github.com/eiannone/keyboard"
)

// SilentInputManager - Input system that reads keyboard without interfering with rendering
type SilentInputManager struct {
	keys     map[rune]bool
	mutex    sync.RWMutex
	running  bool
	stopChan chan bool
}

// InputState represents the current state of all inputs
type InputState struct {
	Forward  bool
	Backward bool
	Left     bool
	Right    bool
	Up       bool
	Down     bool
	RotLeft  bool
	RotRight bool
	RotUp    bool
	RotDown  bool
	SpeedUp  bool
	SlowDown bool
	Reset    bool
	Quit     bool
}

// NewSilentInputManager creates a new silent input manager
func NewSilentInputManager() *SilentInputManager {
	return &SilentInputManager{
		keys:     make(map[rune]bool),
		running:  false,
		stopChan: make(chan bool),
	}
}

// Start begins reading keyboard input in a separate goroutine
func (sim *SilentInputManager) Start() {
	if sim.running {
		return
	}

	if err := keyboard.Open(); err != nil {
		panic(err)
	}

	sim.running = true

	go func() {
		for {
			select {
			case <-sim.stopChan:
				return
			default:
				char, key, err := keyboard.GetKey()
				if err != nil {
					continue
				}

				sim.mutex.Lock()

				// Handle character keys
				if char != 0 {
					sim.keys[char] = true
				}

				// Handle special keys
				switch key {
				case keyboard.KeyEsc:
					sim.keys['x'] = true // Quit
				case keyboard.KeyArrowUp:
					sim.keys['i'] = true
				case keyboard.KeyArrowDown:
					sim.keys['k'] = true
				case keyboard.KeyArrowLeft:
					sim.keys['j'] = true
				case keyboard.KeyArrowRight:
					sim.keys['l'] = true
				}

				sim.mutex.Unlock()
			}
		}
	}()
}

// Stop stops reading keyboard input
func (sim *SilentInputManager) Stop() {
	if !sim.running {
		return
	}

	sim.running = false
	sim.stopChan <- true
	keyboard.Close()
}

// SetKey manually sets a key state (for programmatic control)
func (sim *SilentInputManager) SetKey(key rune, pressed bool) {
	sim.mutex.Lock()
	defer sim.mutex.Unlock()
	sim.keys[key] = pressed
}

// GetInputState returns the current input state
func (sim *SilentInputManager) GetInputState() InputState {
	sim.mutex.RLock()
	defer sim.mutex.RUnlock()

	return InputState{
		Forward:  sim.keys['w'] || sim.keys['W'],
		Backward: sim.keys['s'] || sim.keys['S'],
		Left:     sim.keys['a'] || sim.keys['A'],
		Right:    sim.keys['d'] || sim.keys['D'],
		Up:       sim.keys['e'] || sim.keys['E'],
		Down:     sim.keys['q'] || sim.keys['Q'],
		RotLeft:  sim.keys['j'] || sim.keys['J'],
		RotRight: sim.keys['l'] || sim.keys['L'],
		RotUp:    sim.keys['i'] || sim.keys['I'],
		RotDown:  sim.keys['k'] || sim.keys['K'],
		SpeedUp:  sim.keys['+'] || sim.keys['='],
		SlowDown: sim.keys['-'] || sim.keys['_'],
		Reset:    sim.keys['r'] || sim.keys['R'],
		Quit:     sim.keys['x'] || sim.keys['X'],
	}
}

// ClearKeys clears all key states
func (sim *SilentInputManager) ClearKeys() {
	sim.mutex.Lock()
	defer sim.mutex.Unlock()
	sim.keys = make(map[rune]bool)
}

// CameraController - Transform-based camera controller
type CameraController struct {
	Camera          *Camera
	MoveSpeed       float64
	RotationSpeed   float64
	InitialPosition Point
	InitialRotation Point
	InitialDZ       float64
	AutoOrbit       bool
	OrbitRadius     float64
	OrbitSpeed      float64
	OrbitAngle      float64
	OrbitCenter     Point
}

// NewCameraController creates a camera controller with transform-based movement
func NewCameraController(camera *Camera) *CameraController {
	pos := camera.GetPosition()
	pitch, yaw, roll := camera.GetRotation()

	return &CameraController{
		Camera:          camera,
		MoveSpeed:       2.0,
		RotationSpeed:   0.05,
		InitialPosition: pos,
		InitialRotation: Point{X: pitch, Y: yaw, Z: roll},
		InitialDZ:       camera.DZ,
		AutoOrbit:       true,
		OrbitRadius:     50.0,
		OrbitSpeed:      0.01,
		OrbitAngle:      0.0,
		OrbitCenter:     Point{X: 0, Y: 0, Z: 0},
	}
}

// Update processes input and updates camera using transform-based movement
func (cc *CameraController) Update(input InputState) {
	// Check if user pressed any movement key - disable auto-orbit
	if input.Forward || input.Backward || input.Left || input.Right ||
		input.Up || input.Down || input.RotLeft || input.RotRight ||
		input.RotUp || input.RotDown {
		cc.AutoOrbit = false
	}

	// Auto-orbit mode
	if cc.AutoOrbit {
		cc.OrbitAngle += cc.OrbitSpeed

		// Calculate orbit position
		x := cc.OrbitCenter.X + cc.OrbitRadius*math.Cos(cc.OrbitAngle)
		y := cc.OrbitCenter.Y + 20*math.Sin(cc.OrbitAngle*0.5)
		z := cc.OrbitCenter.Z + cc.OrbitRadius*math.Sin(cc.OrbitAngle)

		cc.Camera.SetPosition(x, y, z)
		cc.Camera.LookAt(cc.OrbitCenter)
		return
	}

	// Manual control mode - use transform-based movement

	// WASD Movement (relative to camera orientation)
	if input.Forward {
		cc.Camera.MoveForward(cc.MoveSpeed)
	}
	if input.Backward {
		cc.Camera.MoveForward(-cc.MoveSpeed)
	}
	if input.Right {
		cc.Camera.MoveRight(cc.MoveSpeed)
	}
	if input.Left {
		cc.Camera.MoveRight(-cc.MoveSpeed)
	}

	// Q/E - Vertical movement (world space up/down)
	if input.Up {
		cc.Camera.MoveUp(cc.MoveSpeed)
	}
	if input.Down {
		cc.Camera.MoveUp(-cc.MoveSpeed)
	}

	// Camera rotation (IJKL or arrow keys)
	if input.RotLeft {
		cc.Camera.RotateYaw(-cc.RotationSpeed)
	}
	if input.RotRight {
		cc.Camera.RotateYaw(cc.RotationSpeed)
	}
	if input.RotUp {
		cc.Camera.RotatePitch(cc.RotationSpeed)
		// Clamp pitch to prevent gimbal lock
		pitch, _, _ := cc.Camera.GetRotation()
		if pitch > math.Pi/2-0.1 {
			cc.Camera.Transform.Rotation.X = math.Pi/2 - 0.1
		}
	}
	if input.RotDown {
		cc.Camera.RotatePitch(-cc.RotationSpeed)
		// Clamp pitch to prevent gimbal lock
		pitch, _, _ := cc.Camera.GetRotation()
		if pitch < -math.Pi/2+0.1 {
			cc.Camera.Transform.Rotation.X = -math.Pi/2 + 0.1
		}
	}

	// Speed adjustment
	if input.SpeedUp {
		cc.MoveSpeed += 0.5
		if cc.MoveSpeed > 20.0 {
			cc.MoveSpeed = 20.0
		}
	}
	if input.SlowDown {
		cc.MoveSpeed -= 0.5
		if cc.MoveSpeed < 0.5 {
			cc.MoveSpeed = 0.5
		}
	}

	// Reset
	if input.Reset {
		cc.Camera.SetPosition(cc.InitialPosition.X, cc.InitialPosition.Y, cc.InitialPosition.Z)
		cc.Camera.SetRotation(cc.InitialRotation.Y, cc.InitialRotation.X)
		cc.Camera.DZ = cc.InitialDZ
		cc.OrbitAngle = 0.0
		cc.AutoOrbit = true
		cc.MoveSpeed = 2.0
	}
}

// SetOrbitCenter sets the center point for auto-orbit
func (cc *CameraController) SetOrbitCenter(x, y, z float64) {
	cc.OrbitCenter = Point{X: x, Y: y, Z: z}
}

// SetOrbitRadius sets the radius for auto-orbit
func (cc *CameraController) SetOrbitRadius(radius float64) {
	cc.OrbitRadius = radius
}

// EnableAutoOrbit enables or disables auto-orbit mode
func (cc *CameraController) EnableAutoOrbit(enable bool) {
	cc.AutoOrbit = enable
}

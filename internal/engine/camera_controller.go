package engine

import (
	"math"
)

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
	OrbitHeight     float64
	OrbitCenter     Point
}

// NewCameraController creates a camera controller
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
		OrbitRadius:     80.0,
		OrbitSpeed:      0.01,
		OrbitAngle:      0.0,
		OrbitHeight:     20.0,
		OrbitCenter:     Point{X: 0, Y: 0, Z: 0},
	}
}

// Update processes input using elapsed seconds. Omitting delta preserves a
// single 1/60-second update for existing callers. Speed fields use 60 Hz units.
func (cc *CameraController) Update(input InputState, orientation OrientationType, delta ...float64) {
	dt := 1.0 / 60
	if len(delta) > 0 {
		dt = delta[0]
	}
	scale := dt * 60
	// Check if user pressed any control key - disable auto-orbit
	if input.Forward || input.Backward || input.Left || input.Right ||
		input.Up || input.Down || input.RotLeft || input.RotRight ||
		input.RotUp || input.RotDown {
		cc.AutoOrbit = false
	}

	// Auto-orbit mode - circle around center while looking at it
	if cc.AutoOrbit {
		cc.OrbitAngle += cc.OrbitSpeed * scale

		// Calculate orbit position
		x := cc.OrbitCenter.X + cc.OrbitRadius*math.Cos(cc.OrbitAngle)
		z := cc.OrbitCenter.Z + cc.OrbitRadius*math.Sin(cc.OrbitAngle)
		y := cc.OrbitCenter.Y + cc.OrbitHeight

		// Set camera position
		cc.Camera.SetPosition(x, y, z)

		// Always look at the center
		cc.Camera.LookAt(cc.OrbitCenter)

		return
	}

	// Manual control mode

	// WASD Movement (relative to camera orientation)
	if input.Forward {
		cc.Camera.MoveForward((cc.MoveSpeed * float64(orientation)) * scale)
	}
	if input.Backward {
		cc.Camera.MoveForward((-cc.MoveSpeed * float64(orientation)) * scale)
	}
	if input.Right {
		cc.Camera.MoveRight((cc.MoveSpeed) * scale)
	}
	if input.Left {
		cc.Camera.MoveRight((-cc.MoveSpeed) * scale)
	}

	// Q/E - Vertical movement (WORLD space up/down)
	if input.Up {
		cc.Camera.MoveUp((cc.MoveSpeed) * scale)
	}
	if input.Down {
		cc.Camera.MoveUp((-cc.MoveSpeed) * scale)
	}

	// IJKL - Camera rotation
	// J = rotate left (yaw left)
	// L = rotate right (yaw right)
	// I = rotate up (pitch up)
	// K = rotate down (pitch down)

	if input.RotLeft {
		cc.Camera.RotateYaw((-cc.RotationSpeed * float64(orientation)) * scale)
	}
	if input.RotRight {
		cc.Camera.RotateYaw((cc.RotationSpeed * float64(orientation)) * scale)
	}
	if input.RotUp {
		cc.Camera.RotatePitch((-cc.RotationSpeed * float64(orientation)) * scale)
	}
	if input.RotDown {
		cc.Camera.RotatePitch((cc.RotationSpeed * float64(orientation)) * scale)
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
		cc.Camera.SetRotation(cc.InitialRotation.X, cc.InitialRotation.Y, cc.InitialRotation.Z)
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

// SetOrbitHeight sets the height offset for auto-orbit
func (cc *CameraController) SetOrbitHeight(height float64) {
	cc.OrbitHeight = height
}

// EnableAutoOrbit enables or disables auto-orbit mode
func (cc *CameraController) EnableAutoOrbit(enable bool) {
	cc.AutoOrbit = enable

	// If enabling, reset angle to current position and look at center
	if enable {
		pos := cc.Camera.GetPosition()
		dx := pos.X - cc.OrbitCenter.X
		dz := pos.Z - cc.OrbitCenter.Z
		cc.OrbitAngle = math.Atan2(dz, dx)
	}
}

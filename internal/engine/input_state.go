package engine

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
	Settings bool
}

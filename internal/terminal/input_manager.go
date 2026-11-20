package terminal

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
	return tim.SilentInputManager.Start()
}

// ShouldClose checks if terminal should close (never for terminal)
func (tim *TerminalInputManager) ShouldClose() bool {
	state := tim.GetInputState()
	return state.Quit
}

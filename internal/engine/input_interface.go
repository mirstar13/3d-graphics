package engine

// InputManager interface for different input backends
type InputManager interface {
	Start() error
	Stop()
	GetInputState() InputState
	ClearKeys()
	ShouldClose() bool
}

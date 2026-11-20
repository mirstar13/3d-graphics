package engine

// MainThreadRenderer is implemented by renderers whose graphics context
// requires drawing on the calling thread.
type MainThreadRenderer interface {
	RequiresMainThread() bool
}

func RequiresMainThread(renderer Renderer) bool {
	requirement, ok := renderer.(MainThreadRenderer)
	return ok && requirement.RequiresMainThread()
}

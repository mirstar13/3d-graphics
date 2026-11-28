//go:build !darwin || !terminalgl || (!amd64 && !arm64)

package terminal

// OpenGLRendererIDs is used on platforms exposing a native renderer-ID list.
func OpenGLRendererIDs() ([]string, error) { return nil, nil }

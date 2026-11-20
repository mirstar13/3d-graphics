//go:build !windows

package terminal

func enableTerminalOutput() (func(), error) {
	return func() {}, nil
}

//go:build !terminalgl || (!linux && !windows && !darwin) || (!amd64 && !arm64)

package terminal

import "errors"

func openFXAADevice() (fxaaDevice, error) {
	return nil, errors.New("build with -tags terminalgl on Linux, Windows, or macOS (amd64 or arm64)")
}

func openRasterDevice() (rasterDevice, error) {
	return nil, errors.New("build with -tags terminalgl on Linux, Windows, or macOS (amd64 or arm64)")
}

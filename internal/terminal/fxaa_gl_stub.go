//go:build !terminalgl || !linux || (!amd64 && !arm64)

package terminal

import "errors"

func openFXAADevice() (fxaaDevice, error) {
	return nil, errors.New("build with -tags terminalgl on Linux amd64 or arm64")
}

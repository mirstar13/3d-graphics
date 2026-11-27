//go:build audio_disabled || (!windows && !darwin && !(linux && audio && cgo))

package audio

import (
	"context"
	"fmt"
	"io"
	"runtime"
)

type nativeOutput struct{}

func (nativeOutput) Run(context.Context, io.Reader) error {
	if runtime.GOOS == "linux" {
		return fmt.Errorf("audio output requires a Linux build with -tags audio and ALSA development files")
	}
	return fmt.Errorf("audio output is disabled in this build")
}

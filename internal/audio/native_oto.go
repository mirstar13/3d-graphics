//go:build !audio_disabled && (windows || darwin || (linux && audio && cgo))

package audio

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
)

type nativeOutput struct{}

var device struct {
	once    sync.Once
	context *oto.Context
	ready   chan struct{}
	err     error
}

func (nativeOutput) Run(ctx context.Context, pcm io.Reader) error {
	device.once.Do(func() {
		device.context, device.ready, device.err = oto.NewContext(&oto.NewContextOptions{
			SampleRate: Rate, ChannelCount: 2, Format: oto.FormatSignedInt16LE, BufferSize: deviceBuffer,
		})
	})
	if device.err != nil {
		return fmt.Errorf("audio device: %w", device.err)
	}
	select {
	case <-ctx.Done():
		return nil
	case <-device.ready:
	}
	if err := device.context.Err(); err != nil {
		return fmt.Errorf("audio device: %w", err)
	}
	player := device.context.NewPlayer(pcm)
	defer player.Close()
	player.SetBufferSize(BlockFrames * 4)
	player.Play()
	poll := time.NewTicker(100 * time.Millisecond)
	defer poll.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-poll.C:
			if err := device.context.Err(); err != nil {
				return fmt.Errorf("audio device: %w", err)
			}
			if err := player.Err(); err != nil {
				return fmt.Errorf("audio playback: %w", err)
			}
		}
	}
}

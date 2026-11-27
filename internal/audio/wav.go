package audio

import (
	"encoding/binary"
	"fmt"
)

const Rate = 48000
const MaxWAVBytes = 8 << 20
const maxClipSeconds = 30

// Clip holds immutable stereo samples at the mixer rate.
type Clip struct{ samples []int16 }

func (c *Clip) Bytes() int { return 2 * len(c.samples) }

// DecodeWAV accepts RIFF PCM16 WAV with one or two channels at 44.1 or 48 kHz.
func DecodeWAV(wav []byte) (*Clip, error) {
	if len(wav) < 12 || len(wav) > MaxWAVBytes || string(wav[:4]) != "RIFF" ||
		string(wav[8:12]) != "WAVE" || uint64(binary.LittleEndian.Uint32(wav[4:]))+8 != uint64(len(wav)) {
		return nil, fmt.Errorf("invalid RIFF WAV header or file size")
	}
	le := binary.LittleEndian
	var format, data []byte
	for rest := wav[12:]; len(rest) > 0; {
		if len(rest) < 8 {
			return nil, fmt.Errorf("truncated WAV chunk header")
		}
		size := uint64(le.Uint32(rest[4:]))
		padded := size + size%2
		if padded > uint64(len(rest)-8) {
			return nil, fmt.Errorf("truncated WAV chunk")
		}
		chunk := rest[8 : 8+int(size)]
		switch string(rest[:4]) {
		case "fmt ":
			if format != nil {
				return nil, fmt.Errorf("duplicate WAV format")
			}
			format = chunk
		case "data":
			if data != nil {
				return nil, fmt.Errorf("duplicate WAV data")
			}
			data = chunk
		}
		rest = rest[8+int(padded):]
	}
	if len(format) < 16 || le.Uint16(format) != 1 || le.Uint16(format[14:]) != 16 {
		return nil, fmt.Errorf("WAV requires uncompressed 16-bit PCM")
	}
	channels, rate := int(le.Uint16(format[2:])), int(le.Uint32(format[4:]))
	stride := channels * 2
	if (channels != 1 && channels != 2) || (rate != 44100 && rate != Rate) ||
		int(le.Uint16(format[12:])) != stride || int(le.Uint32(format[8:])) != rate*stride {
		return nil, fmt.Errorf("WAV requires mono or stereo at 44.1 or 48 kHz")
	}
	if len(data) == 0 || len(data)%stride != 0 || len(data)/stride > maxClipSeconds*rate {
		return nil, fmt.Errorf("WAV must contain complete frames and last at most %d seconds", maxClipSeconds)
	}
	frames := len(data) / stride
	outputFrames := (frames*160 + rate/300 - 1) / (rate / 300)
	pcm := make([]int16, outputFrames*2)
	for i := range outputFrames {
		position := i * (rate / 300)
		frame, fraction := position/160, position%160
		next := min(frame+1, frames-1)
		for channel := range 2 {
			offset := 2 * min(channel, channels-1)
			a := int(int16(le.Uint16(data[frame*stride+offset:])))
			b := int(int16(le.Uint16(data[next*stride+offset:])))
			pcm[2*i+channel] = int16((a*(160-fraction) + b*fraction) / 160)
		}
	}
	return &Clip{samples: pcm}, nil
}

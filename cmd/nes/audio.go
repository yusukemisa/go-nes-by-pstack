package main

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/veandco/go-sdl2/sdl"

	"github.com/misawa/go-nes-by-pstack/internal/apu"
)

// audioOut queues the APU's mono SampleRate stream to an SDL device.
// SDL stays in cmd/nes; internal packages do not import it.
type audioOut struct {
	dev      sdl.AudioDeviceID
	rate     int
	channels int
}

func openAudio() (*audioOut, error) {
	desired := &sdl.AudioSpec{
		Freq:     apu.SampleRate,
		Format:   sdl.AUDIO_F32SYS,
		Channels: 1,
		Samples:  1024,
	}
	obtained := &sdl.AudioSpec{}
	dev, err := sdl.OpenAudioDevice("", false, desired, obtained,
		sdl.AUDIO_ALLOW_FREQUENCY_CHANGE|sdl.AUDIO_ALLOW_CHANNELS_CHANGE)
	if err != nil {
		return nil, err
	}
	if obtained.Format != sdl.AUDIO_F32SYS || (obtained.Channels != 1 && obtained.Channels != 2) || obtained.Freq <= 0 {
		sdl.CloseAudioDevice(dev)
		return nil, fmt.Errorf("sdl audio: freq %d channels %d format %x", obtained.Freq, obtained.Channels, obtained.Format)
	}
	sdl.PauseAudioDevice(dev, false)
	return &audioOut{dev: dev, rate: int(obtained.Freq), channels: int(obtained.Channels)}, nil
}

func (o *audioOut) Close() {
	if o == nil || o.dev == 0 {
		return
	}
	sdl.CloseAudioDevice(o.dev)
	o.dev = 0
}

func (o *audioOut) queue(samples []float32) error {
	if o == nil || len(samples) == 0 {
		return nil
	}
	// A slow frame must not pile up latency. Drop this block past ~200 ms.
	if sdl.GetQueuedAudioSize(o.dev) > uint32(o.rate/5*4*o.channels) {
		return nil
	}
	samples = resample(samples, apu.SampleRate, o.rate)
	if o.channels == 2 {
		stereo := make([]float32, len(samples)*2)
		for i, s := range samples {
			stereo[i*2] = s
			stereo[i*2+1] = s
		}
		samples = stereo
	}
	buf := make([]byte, len(samples)*4)
	for i, s := range samples {
		binary.NativeEndian.PutUint32(buf[i*4:], math.Float32bits(s))
	}
	return sdl.QueueAudio(o.dev, buf)
}

func resample(in []float32, from, to int) []float32 {
	if from == to || len(in) == 0 || to <= 0 {
		return in
	}
	n := int(float64(len(in)) * float64(to) / float64(from))
	if n < 1 {
		n = 1
	}
	out := make([]float32, n)
	last := len(in) - 1
	for i := range out {
		pos := float64(i) * float64(from) / float64(to)
		j := int(pos)
		if j >= last {
			out[i] = in[last]
			continue
		}
		frac := float32(pos - float64(j))
		out[i] = in[j]*(1-frac) + in[j+1]*frac
	}
	return out
}

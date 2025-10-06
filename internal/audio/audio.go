// internal/audio/audio.go
// Minimal WAV playback via SDL2, preferring ALSA. No mixers, no FFT, no extras.
package audio

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/veandco/go-sdl2/sdl"
)

// AudioData holds a loaded WAV buffer and its SDL spec.
type AudioData struct {
	RawBuffer []byte
	Spec      *sdl.AudioSpec
	sdlOwned  bool // true if buffer must be freed with sdl.FreeWAV
}

// Init initializes SDL audio and prefers the ALSA backend.
// Safe to call multiple times (SDL ignores duplicate inits).
func Init() error {
	// Prefer ALSA on minimal systems.
	// Do both the env var and the SDL hint to maximize the chance SDL picks ALSA.
	_ = os.Setenv("SDL_AUDIODRIVER", "alsa")

	if err := sdl.InitSubSystem(sdl.INIT_AUDIO); err != nil {
		return fmt.Errorf("SDL audio init failed: %w", err)
	}
	return nil
}

// Quit tears down the SDL audio subsystem.
func Quit() { sdl.QuitSubSystem(sdl.INIT_AUDIO) }

// ListOutputDevices returns the current SDL output device names (as shown by the ALSA backend).
func ListOutputDevices() ([]string, error) {
	n := sdl.GetNumAudioDevices(false)
	if n < 0 {
		return nil, fmt.Errorf("GetNumAudioDevices failed: %s", sdl.GetError())
	}
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		if name := sdl.GetAudioDeviceName(i, false); name != "" {
			out = append(out, name)
		}
	}
	return out, nil
}

// LoadWAV loads a WAV file. No decoding beyond SDL_LoadWAV. Playback converts on-the-fly if needed.
func LoadWAV(path string) (*AudioData, error) {
	buf, spec := sdl.LoadWAV(path)
	if spec == nil {
		return nil, fmt.Errorf("LoadWAV(%q) failed: %s", path, sdl.GetError())
	}
	return &AudioData{RawBuffer: buf, Spec: spec, sdlOwned: true}, nil
}

// Free releases the WAV buffer if SDL owns it.
func (a *AudioData) Free() {
	if a == nil {
		return
	}
	if a.sdlOwned && a.RawBuffer != nil {
		sdl.FreeWAV(a.RawBuffer)
	}
	a.RawBuffer = nil
	a.Spec = nil
	a.sdlOwned = false
}

// PlayLoop opens an output device (exact name or "" for default) and loops the WAV indefinitely.
// Returns a stop channel you can close, a WaitGroup to join, and the device ID.
func PlayLoop(deviceName string, data *AudioData) (stop chan struct{}, wg *sync.WaitGroup, dev sdl.AudioDeviceID, err error) {
	if data == nil || data.Spec == nil || len(data.RawBuffer) == 0 {
		return nil, nil, 0, fmt.Errorf("invalid audio data")
	}

	// Let device pick the closest format; we’ll convert via AudioStream if needed.
	const allow = sdl.AUDIO_ALLOW_FREQUENCY_CHANGE |
		sdl.AUDIO_ALLOW_FORMAT_CHANGE |
		sdl.AUDIO_ALLOW_CHANNELS_CHANGE

	var obtained sdl.AudioSpec
	dev, err = sdl.OpenAudioDevice(deviceName, false, data.Spec, &obtained, allow)
	if err != nil {
		// If an explicit device failed and wasn't "", try the default "" as a last resort.
		if deviceName != "" {
			dev, err = sdl.OpenAudioDevice("", false, data.Spec, &obtained, allow)
		}
		if err != nil {
			return nil, nil, 0, fmt.Errorf("OpenAudioDevice(%q) failed: %w", deviceName, err)
		}
	}

	// Converter only if device spec differs from source.
	var stream *sdl.AudioStream
	if obtained.Freq != data.Spec.Freq ||
		obtained.Format != data.Spec.Format ||
		obtained.Channels != data.Spec.Channels {
		st, e := sdl.NewAudioStream(
			data.Spec.Format, data.Spec.Channels, int(data.Spec.Freq),
			obtained.Format, obtained.Channels, int(obtained.Freq),
		)
		if e != nil {
			sdl.CloseAudioDevice(dev)
			return nil, nil, 0, fmt.Errorf("create AudioStream failed: %w", e)
		}
		stream = st
	}

	// Local helper to queue PCM to device, converting if needed.
	queue := func(src []byte) error {
		if stream == nil {
			return sdl.QueueAudio(dev, src)
		}
		if err := stream.Put(src); err != nil {
			return err
		}
		if avail := stream.Available(); avail > 0 {
			tmp := make([]byte, avail)
			got, err := stream.Get(tmp)
			if err != nil {
				return err
			}
			if got > 0 {
				return sdl.QueueAudio(dev, tmp[:got])
			}
		}
		return nil
	}

	// Prefill ~100ms to avoid initial underrun (approx; the loop maintains it).
	// We don’t need exact math here; keep it simple.
	targetQueued := uint32(float64(int(obtained.Freq)) * 4 * 0.10) // rough bytes for 16-/32-bit * channels
	src := data.RawBuffer
	readPos := 0
	prefilled := 0

	for uint32(prefilled) < targetQueued && len(src) > 0 {
		chunk := 4096
		if readPos >= len(src) {
			readPos = 0 // loop around
		}
		if end := readPos + chunk; end > len(src) {
			chunk = len(src) - readPos
		}
		if chunk == 0 {
			break
		}
		if err := queue(src[readPos : readPos+chunk]); err != nil {
			if stream != nil {
				stream.Clear()
				stream.Free()
			}
			sdl.CloseAudioDevice(dev)
			return nil, nil, 0, fmt.Errorf("prefill queue failed: %w", err)
		}
		readPos += chunk
		prefilled += chunk
	}

	sdl.PauseAudioDevice(dev, false)

	stop = make(chan struct{})
	wg = &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		defer sdl.ClearQueuedAudio(dev)
		defer func() {
			if stream != nil {
				stream.Clear()
				stream.Free()
			}
			sdl.CloseAudioDevice(dev)
		}()

		ticker := time.NewTicker(15 * time.Millisecond)
		defer ticker.Stop()

		const chunk = 4096
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				// Keep some audio queued; if below target, top it up.
				if sdl.GetQueuedAudioSize(dev) > targetQueued {
					continue
				}
				if readPos >= len(src) {
					readPos = 0
				}
				end := readPos + chunk
				if end > len(src) {
					end = len(src)
				}
				if end <= readPos {
					continue
				}
				_ = queue(src[readPos:end]) // on error: exit loop (silent); keep minimal
				readPos = end
			}
		}
	}()

	return stop, wg, dev, nil
}

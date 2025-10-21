// internal/audio/mixer/mixer.go
package mixer

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/veandco/go-sdl2/mix"
	"github.com/veandco/go-sdl2/sdl"
	"radiantwavetech.com/radiant_wave/internal/config"
	"radiantwavetech.com/radiant_wave/internal/logger"
)

var (
	mu         sync.Mutex
	inited     bool
	deviceOpen bool
	deviceName string

	currentMusic  *mix.Music
	currentPath   string
	currentLoops  int // 0 = once, -1 = forever
	currentVolume int // 0..128, defaults to MAX_VOLUME

	// Config knobs (could be made public or read from your config)
	defaultFreq      = 48000
	defaultFormat    = uint16(sdl.AUDIO_F32)
	defaultChannels  = 2
	defaultChunkSize = 1024
	defaultAllow     = sdl.AUDIO_ALLOW_ANY_CHANGE
)

// Init initializes SDL audio + SDL_mixer. Safe to call multiple times.
func Init(wantedDevice string) error {
	mu.Lock()
	defer mu.Unlock()
	if inited {
		return nil
	}

	// Optional: prefer ALSA; do this before SDL init.
	_ = os.Setenv("SDL_AUDIODRIVER", "alsa")

	if err := sdl.InitSubSystem(sdl.INIT_AUDIO); err != nil {
		return fmt.Errorf("SDL audio init failed: %w", err)
	}

	// WAV-only does not require mix.Init(); add when you support OGG/MP3/etc.
	if err := openDeviceLocked(wantedDevice); err != nil {
		// Clean up the subsystem on failure so callers can retry later.
		sdl.QuitSubSystem(sdl.INIT_AUDIO)
		return err
	}

	// Check if the volume is stored in the config; if not, default to max.
	storedVolume := config.Get().LastVolume
	if storedVolume < 0 {
		currentVolume = mix.MAX_VOLUME
		config.Get().LastVolume = currentVolume
		if err := config.Get().Save(); err != nil {
			logger.LogErrorF("saving default volume to config: %v", err)
		}
	} else {
		currentVolume = storedVolume
	}

	inited = true
	return nil
}

// openDeviceLocked tries to open the named device; if it fails and the name wasn’t empty, falls back to default.
func openDeviceLocked(name string) error {
	if err := mix.OpenAudioDevice(defaultFreq, defaultFormat, defaultChannels, defaultChunkSize, name, defaultAllow); err != nil {
		if name != "" {
			// Fallback to default device
			if err2 := mix.OpenAudioDevice(defaultFreq, defaultFormat, defaultChannels, defaultChunkSize, "", defaultAllow); err2 != nil {
				return fmt.Errorf("failed to open device %q and default device: %w", name, err2)
			}
			deviceName = "" // actually opened default
		} else {
			return fmt.Errorf("failed to open default audio device: %w", err)
		}
	} else {
		deviceName = name
	}
	deviceOpen = true
	return nil
}

// Play loads a WAV (as music) and plays it. If loopForever is true, loops forever.
// If something is already playing, it stops and replaces it.
func Play(path string, loopForever bool) error {
	mu.Lock()
	defer mu.Unlock()

	if !inited || !deviceOpen {
		return errors.New("audio mixer not initialized")
	}
	if path == "" {
		return errors.New("empty audio path")
	}

	// Stop and free any prior music
	mix.HaltMusic()
	if currentMusic != nil {
		currentMusic.Free()
		currentMusic = nil
	}

	m, err := mix.LoadMUS(path)
	if err != nil {
		return fmt.Errorf("LoadMUS(%q): %w", path, err)
	}
	currentMusic = m
	currentPath = path
	if loopForever {
		currentLoops = -1
	} else {
		currentLoops = 0
	}

	// Apply remembered volume
	mix.VolumeMusic(currentVolume)

	if err := currentMusic.Play(currentLoops); err != nil {
		currentMusic.Free()
		currentMusic = nil
		return fmt.Errorf("music.Play: %w", err)
	}
	return nil
}

// Stop stops playback. If fadeMs > 0, fades out asynchronously; otherwise halts immediately.
// It also frees the currently loaded music (so a new Play will reload).
func Stop(fadeMs int) {
	mu.Lock()
	defer mu.Unlock()
	if !inited {
		return
	}

	if fadeMs > 0 && mix.PlayingMusic() {
		_ = mix.FadeOutMusic(fadeMs) // async fade; we’ll still free below
	}

	mix.HaltMusic()
	if currentMusic != nil {
		currentMusic.Free()
		currentMusic = nil
	}
	currentPath = ""
	currentLoops = 0
}

// SetVolume128 sets music volume using SDL_mixer’s 0..128 scale.
func SetVolume128(increment int) {
	mu.Lock()
	defer mu.Unlock()

	currentVolume += increment

	if currentVolume < 0 {
		currentVolume = 0
	} else if currentVolume > mix.MAX_VOLUME {
		currentVolume = mix.MAX_VOLUME
	}

	// Store in config
	config.Get().LastVolume = currentVolume
	if err := config.Get().Save(); err != nil {
		logger.LogErrorF("saving volume to config: %v", err)
	}

	// Apply to SDL_mixer
	if inited && deviceOpen {
		mix.VolumeMusic(currentVolume)
	}
}

// IsPlaying reports whether music is currently playing (or fading).
func IsPlaying() bool {
	mu.Lock()
	defer mu.Unlock()
	if !inited {
		return false
	}
	return mix.PlayingMusic()
}

// ListDevices returns the list of available output device names (from core SDL).
func ListDevices() []string {
	mu.Lock()
	defer mu.Unlock()
	n := sdl.GetNumAudioDevices(false)
	if n <= 0 {
		return nil
	}
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		if name := sdl.GetAudioDeviceName(i, false); name != "" {
			out = append(out, name)
		}
	}
	return out
}

// CurrentDevice returns the last requested/opened device name ("" means default).
func CurrentDevice() string {
	mu.Lock()
	defer mu.Unlock()
	return deviceName
}

// SwitchDevice switches output devices, preserving volume and resuming playback of the current track/loop state if any.
func SwitchDevice(newName string) error {
	mu.Lock()
	defer mu.Unlock()

	if !inited {
		return errors.New("audio mixer not initialized")
	}

	// Capture current state
	wasPlaying := mix.PlayingMusic()
	savedPath := currentPath
	savedLoops := currentLoops
	savedVolume := currentVolume

	// Stop and release current music before reopening device
	mix.HaltMusic()
	if currentMusic != nil {
		currentMusic.Free()
		currentMusic = nil
	}
	currentPath = ""
	currentLoops = 0

	// Close and reopen device
	if deviceOpen {
		mix.CloseAudio()
		deviceOpen = false
	}
	if err := openDeviceLocked(newName); err != nil {
		// Best effort: try to reopen the previous device (or default) so we don’t leave audio dead
		_ = openDeviceLocked(deviceName)
		return err
	}

	// Restore volume
	mix.VolumeMusic(savedVolume)

	// Optionally resume previous track if there was one
	if wasPlaying && savedPath != "" {
		m, err := mix.LoadMUS(savedPath)
		if err != nil {
			// Don’t fail the device switch just because reload failed
			return fmt.Errorf("device switched, but reloading %q failed: %w", savedPath, err)
		}
		currentMusic = m
		currentPath = savedPath
		currentLoops = savedLoops
		if err := currentMusic.Play(currentLoops); err != nil {
			currentMusic.Free()
			currentMusic = nil
			currentPath = ""
			currentLoops = 0
			return fmt.Errorf("device switched, but restarting playback failed: %w", err)
		}
	}

	return nil
}

func GetVolume128() int {
	mu.Lock()
	defer mu.Unlock()
	return currentVolume
}

// Shutdown fully tears everything down. Call at app exit.
func Shutdown() {
	mu.Lock()
	defer mu.Unlock()

	mix.HaltMusic()
	if currentMusic != nil {
		currentMusic.Free()
		currentMusic = nil
	}
	if deviceOpen {
		mix.CloseAudio()
		deviceOpen = false
	}
	// If you add mix.Init(decoderFlags) in the future, pair it with mix.Quit() here.
	if inited {
		sdl.QuitSubSystem(sdl.INIT_AUDIO)
		inited = false
	}
}

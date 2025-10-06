package page

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/veandco/go-sdl2/sdl"
	"radiantwavetech.com/radiant_wave/internal/audio"
	"radiantwavetech.com/radiant_wave/internal/colors"
	"radiantwavetech.com/radiant_wave/internal/config"
	"radiantwavetech.com/radiant_wave/internal/fontManager"
	"radiantwavetech.com/radiant_wave/internal/logger"
	colorPattern "radiantwavetech.com/radiant_wave/internal/pattern/color"
	linePattern "radiantwavetech.com/radiant_wave/internal/pattern/line"
	"radiantwavetech.com/radiant_wave/internal/shaderManager"
)

type Line struct {
	words []StringItem
	width int32
}

type ScrollerPage struct {
	Base

	audioData     *audio.AudioData
	words         []string
	wordIndex     int
	lines         []Line
	lineHeight    int32
	fontScale     float32
	shader        *shaderManager.Shader
	activePattern linePattern.LinePattern
	font          *fontManager.FontEntry
	spaceWidth    int32
	lineCtx       *linePattern.LineRenderContext
	colorPattern  *colorPattern.ColorPattern
	scrollOffset  float32
	scrollY       float32

	increment float32
	velocity  float32

	// Time-based velocity control
	velocityFloor     float32 // Minimum velocity (negative)
	velocityCeiling   float32 // Maximum velocity (positive)
	velocityStep      float32 // Rate of change per second
	velocityDelay     float32 // Delay at ceiling/floor in seconds
	velocityDirection int     // 1 for increasing, -1 for decreasing
	velocityTimer     float32 // Timer for delay
	velocityState     int     // 0=changing, 1=waiting at ceiling, 2=waiting at floor
	autoVelocity      bool    // Enable/disable automatic velocity
	hold              float32 // Hold time at ceiling/floor in seconds

	audioStopChan chan struct{}
	audioWg       *sync.WaitGroup
	audioDeviceID sdl.AudioDeviceID
}

func (p *ScrollerPage) Init(app ApplicationInterface) error {
	p.increment = 100.0
	p.velocity = 800.0
	p.SetAutoVelocity(true, -4000.0, 4000.0, 100.0, 2.5, 6.0)

	sm := shaderManager.Get()
	fm := fontManager.Get()
	cfg := config.Get()
	font, ok := fm.GetFont("Scrambled")
	if !ok {
		return fmt.Errorf("unable to fetch font 'Scrambled'")
	}
	p.font = font

	p.Base.Init(app)

	textShader, ok := sm.Get("text")
	if !ok {
		return fmt.Errorf("text shader not found")
	}
	p.shader = textShader

	err := p.readFiles()
	if err != nil {
		logger.LogErrorF("readFiles error: %v", err)
		// Don’t stop execution, just show fallback words
		p.words = strings.Fields("No Files Selected!")
	}
	p.wordIndex = 0

	p.activePattern, err = linePattern.Get(cfg.LinePattern)
	if err != nil {
		return fmt.Errorf("failed to load line pattern '%s': %w", cfg.LinePattern, err)
	}

	var baseColors = []sdl.Color{
		colors.Tan,
		colors.PeachPuff,
		colors.LightCyan,
		colors.LightSeaGreen,
		colors.MediumTurquoise,
		colors.DarkCyan,
		colors.MediumPurple,
	}

	pattern := colorPattern.NewColorPattern(baseColors)
	p.colorPattern = pattern

	// Font setup
	p.fontScale = float32(cfg.StandardFontSize) / float32(cfg.BaseFontSize)
	nLines := int(p.ScreenHeight / int32(cfg.StandardFontSize))
	p.lines = make([]Line, nLines)
	spaceGlyph, _ := fm.GetGlyph(p.font.Name, ' ')
	tmpStrID, _, H, _ := fm.NewCreateStringTexture(p.font.Name, "Mg", p.shader)
	spaceWidth := int32(float32(spaceGlyph.Advance) * p.fontScale)
	p.spaceWidth = spaceWidth
	p.lineHeight = int32(float32(H) * p.fontScale)
	gl.DeleteTextures(1, &tmpStrID)

	p.lineCtx = &linePattern.LineRenderContext{
		ScreenWidth: p.ScreenWidth,
		Font:        p.font,
		FontScale:   p.fontScale,
		SpaceWidth:  p.spaceWidth,
	}

	// Create initial lines
	for i := range p.lines {
		words := p.activePattern.GetNextLine(p.words, &p.wordIndex, p.lineCtx)
		line := Line{words: []StringItem{}}

		for _, word := range words {
			if word == "" {
				continue
			}
			item, err := NewStringItem(word, font, p.colorPattern.Color())
			if err != nil {
				logger.LogErrorF("unable to create string item for word '%s': %v", word, err)
				continue
			}
			item.W = int32(float32(item.W) * p.fontScale)
			item.H = int32(float32(item.H) * p.fontScale)

			if len(line.words) > 0 {
				line.width += spaceWidth
			}
			line.words = append(line.words, item)
			line.width += item.W
		}
		p.lines[i] = line

		// Center horizontally
		startX := p.ScreenCenterX - (line.width / 2)
		for j := range p.lines[i].words {
			if j == 0 {
				p.lines[i].words[j].X = startX
			} else {
				p.lines[i].words[j].X = p.lines[i].words[j-1].X + p.lines[i].words[j-1].W + spaceWidth
			}
			p.lines[i].words[j].Y = int32(p.scrollY) - int32(p.lineHeight)*int32(i)
		}
	}

	// Try to start audio
	audioData, err := audio.LoadWAV(cfg.SelectedAudioFilePath)
	if err != nil {
		logger.LogWarningF("Audio load failed: %v (continuing without audio)", err)
		p.audioData = nil
		return nil // Not fatal
	}
	p.audioData = audioData

	// stopChan, wg, deviceID, err := audio.PlayAudioLoop(cfg.AudioDeviceName, audioData)
	// if err != nil {
	// 	logger.LogWarningF("Audio playback failed: %v (continuing without audio)", err)
	// 	p.audioData = nil
	// 	return nil
	// }
	// p.audioStopChan = stopChan
	// p.audioWg = wg
	// p.audioDeviceID = deviceID
	// assumes audio.Init() was called earlier in app startup

	stopChan, wg, deviceID, err := audio.PlayLoop(cfg.AudioDeviceName, audioData)
	if err != nil {
		logger.LogWarningF("Audio playback failed: %v (continuing without audio)", err)
		p.audioData = nil
		return nil
	}

	chosen := cfg.AudioDeviceName
	if chosen == "" {
		chosen = "default"
	}
	logger.LogInfoF("Playing audio on device: %q (id=%d)", chosen, deviceID)

	p.audioStopChan = stopChan
	p.audioWg = wg
	p.audioDeviceID = deviceID

	return nil
}

func (p *ScrollerPage) readFiles() error {
	config := config.Get()
	p.words = []string{}
	filepaths := config.SelectedFilePaths
	if len(filepaths) == 0 {
		p.words = strings.Fields("No Files Selected!")
		return fmt.Errorf("no files selected to load")
	}

	for _, path := range filepaths {
		if !filepath.IsAbs(path) && config.AssetsDir != "" {
			path = filepath.Join(config.AssetsDir, "affirmations", path)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		p.words = append(p.words, strings.Fields(string(data))...)

		if len(p.words) == 0 {
			p.words = strings.Fields("Files may be empty or unreadable")
		}
	}

	return nil
}

func (p *ScrollerPage) HandleEvent(event *sdl.Event) error {
	switch e := (*event).(type) {
	case *sdl.KeyboardEvent:
		if e.Type != sdl.KEYDOWN {
			return nil
		}
		switch e.Keysym.Sym {
		case sdl.K_UP:
			if !p.autoVelocity { // Only allow manual control when auto is disabled
				p.velocity += p.increment
			}
		case sdl.K_DOWN:
			if !p.autoVelocity {
				p.velocity -= p.increment
			}
		case sdl.K_a: // Toggle auto velocity (example key)
			p.autoVelocity = !p.autoVelocity
			if p.autoVelocity {
				// Reset to initial state when enabling
				p.velocity = p.velocityFloor
				p.velocityDirection = 1
				p.velocityState = 0
				p.velocityTimer = 0.0
			}
		}
		return nil
	}
	return nil
}

func (p *ScrollerPage) Update(dt float32) error {
	p.updateAutoVelocity(dt)

	p.scrollOffset += dt * p.velocity
	move := int32(p.scrollOffset)
	if move == 0 {
		return nil
	}
	p.scrollOffset = float32(math.Mod(float64(p.scrollOffset), 1.0))

	// Move all lines vertically
	for i := range p.lines {
		for j := range p.lines[i].words {
			p.lines[i].words[j].Y += move
		}
	}

	// Handle multiple line replacements in a single frame
	if p.velocity > 0 {
		// Moving up, check if top lines are off-screen
		for len(p.lines) > 0 && len(p.lines[0].words) > 0 && p.lines[0].words[0].Y > p.ScreenHeight {
			// Delete textures for the line going off-screen
			for _, word := range p.lines[0].words {
				gl.DeleteTextures(1, &word.ID)
			}
			p.lines = p.lines[1:]

			// Create new line at the bottom
			newWords := p.activePattern.GetNextLine(p.words, &p.wordIndex, p.lineCtx)
			line := Line{words: []StringItem{}}
			for _, word := range newWords {
				if word == "" {
					continue
				}
				item, err := NewStringItem(word, p.font, p.colorPattern.Color())
				if err != nil {
					logger.LogErrorF("unable to create string item for word '%s': %v", word, err)
					return err
				}
				item.W = int32(float32(item.W) * p.fontScale)
				item.H = int32(float32(item.H) * p.fontScale)
				if len(line.words) > 0 {
					line.width += p.spaceWidth
				}
				line.words = append(line.words, item)
				line.width += item.W
			}

			startX := p.ScreenCenterX - (line.width / 2)
			// Position relative to the last existing line
			var baseY int32
			if len(p.lines) > 0 && len(p.lines[len(p.lines)-1].words) > 0 {
				baseY = p.lines[len(p.lines)-1].words[0].Y - int32(p.lineHeight)
			} else {
				baseY = int32(0) - int32(p.lineHeight)
			}

			for j := range line.words {
				if j == 0 {
					line.words[j].X = startX
				} else {
					line.words[j].X = line.words[j-1].X + line.words[j-1].W + p.spaceWidth
				}
				line.words[j].Y = baseY
			}

			p.lines = append(p.lines, line)
		}
	} else if p.velocity < 0 {
		// Moving down, check if bottom lines are off-screen
		for len(p.lines) > 0 {
			lastIdx := len(p.lines) - 1
			lastLine := p.lines[lastIdx]
			if len(lastLine.words) == 0 || lastLine.words[0].Y+lastLine.words[0].H >= 0 {
				break
			}

			// Delete textures for the line going off-screen
			for _, word := range lastLine.words {
				gl.DeleteTextures(1, &word.ID)
			}
			p.lines = p.lines[:lastIdx]

			// Create new line at the top
			newWords := p.activePattern.GetNextLine(p.words, &p.wordIndex, p.lineCtx)
			line := Line{words: []StringItem{}}
			for _, word := range newWords {
				if word == "" {
					continue
				}
				item, err := NewStringItem(word, p.font, p.colorPattern.Color())
				if err != nil {
					logger.LogErrorF("unable to create string item for word '%s': %v", word, err)
					return err
				}
				item.W = int32(float32(item.W) * p.fontScale)
				item.H = int32(float32(item.H) * p.fontScale)
				if len(line.words) > 0 {
					line.width += p.spaceWidth
				}
				line.words = append(line.words, item)
				line.width += item.W
			}

			startX := p.ScreenCenterX - (line.width / 2)
			// Position relative to the first existing line
			var baseY int32
			if len(p.lines) > 0 && len(p.lines[0].words) > 0 {
				baseY = p.lines[0].words[0].Y + int32(p.lineHeight)
			} else {
				baseY = p.ScreenHeight
			}

			for j := range line.words {
				if j == 0 {
					line.words[j].X = startX
				} else {
					line.words[j].X = line.words[j-1].X + line.words[j-1].W + p.spaceWidth
				}
				line.words[j].Y = baseY
			}

			p.lines = append([]Line{line}, p.lines...)
		}
	}

	return nil
}

// Render draws elements on the screen with their intended positions.
// To update positions, see Update(dt float32)
func (p *ScrollerPage) Render() error {
	for _, line := range p.lines {
		for _, word := range line.words {
			if word.ID == 0 {
				continue
			}
			err := p.RenderTexture(p.shader, word.ID, word.Position(), word.RawDimensions(), word.Color)
			if err != nil {
				logger.LogErrorF("failed to render texture for word '%s': %v", word.content, err)
				return err
			}

		}
	}

	return nil
}

func (p *ScrollerPage) Destroy() error {
	// Make this safe to call multiple times
	// (early-out if already destroyed)
	if p == nil {
		return nil
	}

	// 1) Stop audio loop first so it stops touching buffers
	if p.audioStopChan != nil {
		// close only once
		close(p.audioStopChan)
		p.audioStopChan = nil
	}
	if p.audioWg != nil {
		p.audioWg.Wait()
		p.audioWg = nil
	}

	// 2) Close audio device and clear queued data
	if p.audioDeviceID != 0 {
		sdl.PauseAudioDevice(p.audioDeviceID, true)
		sdl.ClearQueuedAudio(p.audioDeviceID)
		sdl.CloseAudioDevice(p.audioDeviceID)
		p.audioDeviceID = 0
	}

	// 3) Free WAV/converted buffer (our AudioData.Free() knows whether it’s SDL-owned)
	if p.audioData != nil {
		p.audioData.Free()
		p.audioData = nil
	}

	// 4) GL cleanup — delete any remaining text textures
	for i := range p.lines {
		for j := range p.lines[i].words {
			if id := p.lines[i].words[j].ID; id != 0 {
				gl.DeleteTextures(1, &p.lines[i].words[j].ID)
				p.lines[i].words[j].ID = 0
			}
		}
	}

	// (Optional) Ensure driver flushes deletes promptly.
	// Not strictly required, but can help on some drivers.
	gl.Finish()

	// 5) Drop references so GC can reclaim memory
	p.lines = nil
	p.words = nil
	p.lineCtx = nil
	p.font = nil
	p.shader = nil
	p.colorPattern = nil

	return nil
}

// Add this method to configure the automatic velocity system
func (p *ScrollerPage) SetAutoVelocity(enabled bool, floor, ceiling, step, delay, hold float32) {
	p.autoVelocity = enabled
	p.velocityFloor = -float32(math.Abs(float64(floor)))    // Ensure floor is negative
	p.velocityCeiling = float32(math.Abs(float64(ceiling))) // Ensure ceiling is positive
	p.velocityStep = float32(math.Abs(float64(step)))       // Ensure step is positive
	p.velocityDelay = delay

	// Initialize state
	p.velocityDirection = 1 // Start increasing
	p.velocityState = 0     // Start changing
	p.velocityTimer = 0.0

	// Set initial velocity to start from floor
	p.velocity = p.velocityFloor

	p.hold = hold
}

// Add this method to update velocity automatically
func (p *ScrollerPage) updateAutoVelocity(dt float32) {
	if !p.autoVelocity {
		return
	}

	switch p.velocityState {
	case 0: // Changing velocity
		if p.velocityDirection == 1 {
			// Increasing velocity
			p.velocity += p.velocityStep * dt
			if p.velocity >= p.velocityCeiling {
				p.velocity = p.velocityCeiling
				p.velocityState = 1 // Wait at ceiling
				p.velocityTimer = 0.0
			}
		} else {
			// Decreasing velocity
			p.velocity -= p.velocityStep * dt
			if p.velocity <= p.velocityFloor {
				p.velocity = p.velocityFloor
				p.velocityState = 2 // Wait at floor
				p.velocityTimer = 0.0
			}
		}

	case 1: // Waiting at ceiling
		p.velocityTimer += dt
		if p.velocityTimer >= p.velocityDelay {
			p.velocityDirection = -1 // Start decreasing
			p.velocityState = 0      // Start changing
		}

	case 2: // Waiting at floor
		p.velocityTimer += dt
		if p.velocityTimer >= p.velocityDelay {
			p.velocityDirection = 1 // Start increasing
			p.velocityState = 0     // Start changing
		}
	}
}

// ATTENTION: TED
//
// # Integrated Multi-Domain Radiant Wave Generation System
//
// This module implements the complete Radiant Wave generation system by synchronizing multiple
// subsystems that operate simultaneously across spatial, temporal, and chromatic
// domains to produce a unified frequency-structured visual output for subliminal affirmation display.
//
// The technology addresses the fundamental challenge of presenting textual affirmation content
// in a format optimized for subconscious processing rather than conscious reading. Conventional
// subliminal text systems use isolated techniques (simple masking, single-frequency flashing, or
// uniform scrolling) that operate in only one domain. Our system integrates four mathematically-
// synchronized subsystems, each generating Fibonacci-based patterns in different domains, creating
// a compound frequency signature designed to enhance subconscious absorption while maintaining
// conscious illegibility.
//
// The system architecture integrates:
//
//  1. Spatial Character Transformation - Each rendered character undergoes Fibonacci-based
//     block scrambling (described separately), creating local spatial frequency patterns at
//     the character level.
//
//  2. Spatial Line Distribution - Text lines contain varying word counts following an oscillating
//     Fibonacci sequence (1, 1, 2, 3, 5, 8, 13... then reversing), creating a macro-level
//     two-dimensional wave pattern in the display geometry and ensuring balanced temporal
//     exposure of affirmation content segments.
//
//  3. Temporal Color Modulation - Sequential words receive colors from an intent-based palette,
//     with color duration determined by an oscillating Fibonacci sequence, creating rhythmic
//     chromatic variation synchronized with natural mathematical progression.
//
//  4. Temporal Velocity Modulation - Display scroll velocity oscillates between negative and
//     positive extremes with acceleration/deceleration rates and hold periods, creating a
//     "breathing" temporal rhythm for the entire visual field.
//
// The key innovation lies in the temporal and mathematical synchronization of these subsystems.
// All four operate on Fibonacci-derived parameters, creating harmonic relationships between the
// spatial frequencies (character scrambling, line wave patterns), chromatic frequencies (color
// duration oscillation), and temporal frequencies (scroll velocity modulation). This multi-domain
// integration produces what we term the "Radiant Wave" - a compound frequency signature where
// spatial, temporal, and chromatic patterns reinforce each other through mathematical harmony.
//
// Key differentiators from existing solutions:
//   - Integrates four distinct Fibonacci-based frequency generators in a single system
//   - Synchronizes pattern generation across spatial, temporal, and chromatic domains
//   - Creates compound frequency signatures rather than isolated single-domain effects
//   - Maintains character-specific and content-specific mathematical structure throughout
//   - Uses bidirectional oscillation across multiple subsystems for rhythmic coherence
//   - Combines macro-level (line patterns, velocity) and micro-level (character, color) modulation
//
// The system loads affirmation text content, processes it through all four subsystems simultaneously,
// and generates a continuously scrolling display where every element - from individual character
// pixels to overall scroll rhythm - contributes to the unified Radiant Wave output. The mathematical
// relationships between subsystems create interference patterns and harmonic resonances designed
// to enhance subconscious processing of the affirmation content while maintaining conscious
// illegibility through the character scrambling subsystem.
package page

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/veandco/go-sdl2/sdl"
	"radiantwavetech.com/radiantwave/internal/colors"
	"radiantwavetech.com/radiantwave/internal/db"
	"radiantwavetech.com/radiantwave/internal/fontManager"
	"radiantwavetech.com/radiantwave/internal/logger"
	"radiantwavetech.com/radiantwave/internal/mixer"
	colorPattern "radiantwavetech.com/radiantwave/internal/pattern/color"
	linePattern "radiantwavetech.com/radiantwave/internal/pattern/line"
	"radiantwavetech.com/radiantwave/internal/shaderManager"
)

type Line struct {
	words []StringItem
	width int32
}

type ScrollerPage struct {
	Base

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
	velocityFloor     float32
	velocityCeiling   float32
	velocityStep      float32
	velocityDelay     float32
	velocityDirection int
	velocityTimer     float32
	velocityState     int
	autoVelocity      bool
	hold              float32
}

func (p *ScrollerPage) Init(app ApplicationInterface) error {
	logger.InfoF("Initializing ScrollerPage")
	p.increment = 100.0
	p.velocity = 800.0
	p.SetAutoVelocity(true, -4000.0, 4000.0, 100.0, 2.5, 6.0)

	sm := shaderManager.Get()
	fm := fontManager.Get()

	// Get font
	font, ok := fm.GetFont("Scrambled")
	if !ok {
		return fmt.Errorf("unable to fetch font 'Scrambled'")
	}
	p.font = font

	p.Base.Init(app)

	// Get text shader
	textShader, ok := sm.Get("text")
	if !ok {
		return fmt.Errorf("text shader not found")
	}
	p.shader = textShader

	// Load affirmation words from database
	if err := p.loadAffirmations(); err != nil {
		logger.ErrorF("loadAffirmations error: %v", err)
		// Don't stop execution, just show fallback words
		p.words = strings.Fields("No Affirmations Selected!")
	}
	p.wordIndex = 0

	// Get line pattern from database
	linePatternName, err := db.GetConfigValue("line_pattern")
	if err != nil {
		return fmt.Errorf("retrieving line_pattern from db: %w", err)
	}

	p.activePattern, err = linePattern.Get(linePatternName)
	if err != nil {
		return fmt.Errorf("failed to load line pattern '%s': %w", linePatternName, err)
	}

	// Initialize color pattern
	var baseColors = []sdl.Color{
		colors.Tan,
		colors.PeachPuff,
		colors.LightCyan,
		colors.LightSeaGreen,
		colors.MediumTurquoise,
		colors.DarkCyan,
		colors.MediumPurple,
	}
	p.colorPattern = colorPattern.NewColorPattern(baseColors)

	// Get font sizes from database (optimize: load once, cache as int32)
	standardFontSizeStr, err := db.GetConfigValue("standard_font_size")
	if err != nil {
		return fmt.Errorf("retrieving standard_font_size from db: %w", err)
	}
	standardFontSize, err := strconv.Atoi(standardFontSizeStr)
	if err != nil {
		return fmt.Errorf("parsing standard_font_size: %w", err)
	}

	baseFontSizeStr, err := db.GetConfigValue("init_font_size")
	if err != nil {
		return fmt.Errorf("retrieving init_font_size from db: %w", err)
	}
	baseFontSize, err := strconv.Atoi(baseFontSizeStr)
	if err != nil {
		return fmt.Errorf("parsing init_font_size: %w", err)
	}

	// Font setup (optimized: calculate once, reuse)
	p.fontScale = float32(standardFontSize) / float32(baseFontSize)
	nLines := int(p.ScreenHeight / int32(standardFontSize))
	p.lines = make([]Line, nLines)

	// Calculate space width once
	spaceGlyph, ok := fm.GetGlyph(p.font.Name, ' ')
	if !ok {
		return fmt.Errorf("failed to get space glyph")
	}
	p.spaceWidth = int32(float32(spaceGlyph.Advance) * p.fontScale)

	// Get line height from a sample string
	tmpStrID, _, H, err := fm.NewCreateStringTexture(p.font.Name, "Mg", p.shader)
	if err != nil {
		return fmt.Errorf("failed to create sample texture: %w", err)
	}
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
		if err := p.createLine(i, true); err != nil {
			return fmt.Errorf("failed to create initial line %d: %w", i, err)
		}
	}

	// Audio setup
	audioDeviceName, err := db.GetConfigValue("audio_device_name")
	if err != nil {
		return fmt.Errorf("retrieving audio_device_name from db: %w", err)
	}

	// Switch audio devices if needed
	if mixer.CurrentDevice() != audioDeviceName {
		if err := mixer.SwitchDevice(audioDeviceName); err != nil {
			logger.ErrorF("Failed to switch to audio device %q: %v", audioDeviceName, err)
		} else {
			logger.InfoF("Switched audio device to %q", audioDeviceName)
		}
	}

	// Start audio playback
	selectedAudio, err := db.GetConfigValue("selected_audio")
	if err != nil {
		return fmt.Errorf("retrieving selected_audio from db: %w", err)
	}

	logger.InfoF("Starting audio playback")
	if selectedAudio == "" {
		logger.WarningF("No audio file selected, skipping audio playback")
	} else {
		if err := mixer.Play(selectedAudio, true); err != nil {
			logger.ErrorF("mixer.Play error: %v", err)
		}
	}

	currentDevice := mixer.CurrentDevice()
	if currentDevice == "" {
		currentDevice = "default"
	}
	logger.InfoF("Playing music on device %q", currentDevice)

	return nil
}

// loadAffirmations loads selected affirmations from the database
func (p *ScrollerPage) loadAffirmations() error {
	p.words = []string{}

	// Get all selected affirmations from database
	affirmations, err := db.GetAffirmations()
	if err != nil {
		return fmt.Errorf("retrieving affirmations from db: %w", err)
	}

	// Build word list from selected, available affirmations
	for _, affirmation := range affirmations {
		if affirmation.Selected && affirmation.Available {
			// Split content into words and add to list
			words := strings.Fields(affirmation.Content)
			p.words = append(p.words, words...)
		}
	}

	if len(p.words) == 0 {
		p.words = strings.Fields("No Affirmations Selected!")
		return fmt.Errorf("no affirmations selected or available")
	}

	return nil
}

// createLine is a helper to generate a new line at a specific index
// isInitial determines if this is during initialization (positions from top)
// or dynamic creation (positions relative to existing lines)
func (p *ScrollerPage) createLine(index int, isInitial bool) error {
	words := p.activePattern.GetNextLine(p.words, &p.wordIndex, p.lineCtx)
	line := Line{words: []StringItem{}}

	for _, word := range words {
		if word == "" {
			continue
		}
		item, err := NewStringItem(word, p.font, p.colorPattern.Color())
		if err != nil {
			return fmt.Errorf("unable to create string item for word '%s': %w", word, err)
		}
		item.W = int32(float32(item.W) * p.fontScale)
		item.H = int32(float32(item.H) * p.fontScale)

		if len(line.words) > 0 {
			line.width += p.spaceWidth
		}
		line.words = append(line.words, item)
		line.width += item.W
	}

	// Calculate horizontal position (centered)
	startX := p.ScreenCenterX - (line.width / 2)

	// Calculate vertical position
	var baseY int32
	if isInitial {
		baseY = int32(p.scrollY) - int32(p.lineHeight)*int32(index)
	} else {
		baseY = 0 // Will be set by caller based on context
	}

	// Position all words in the line
	for j := range line.words {
		if j == 0 {
			line.words[j].X = startX
		} else {
			line.words[j].X = line.words[j-1].X + line.words[j-1].W + p.spaceWidth
		}
		line.words[j].Y = baseY
	}

	if isInitial && index < len(p.lines) {
		p.lines[index] = line
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
			if !p.autoVelocity {
				p.velocity += p.increment
			}
		case sdl.K_DOWN:
			if !p.autoVelocity {
				p.velocity -= p.increment
			}
		case sdl.K_a:
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

	// Handle line replacement based on scroll direction
	if p.velocity > 0 {
		// Moving up: replace lines that go off top
		for len(p.lines) > 0 && len(p.lines[0].words) > 0 && p.lines[0].words[0].Y > p.ScreenHeight {
			// Cleanup old line
			for _, word := range p.lines[0].words {
				gl.DeleteTextures(1, &word.ID)
			}
			p.lines = p.lines[1:]

			// Create new line at bottom
			if err := p.addLineAtBottom(); err != nil {
				return err
			}
		}
	} else if p.velocity < 0 {
		// Moving down: replace lines that go off bottom
		for len(p.lines) > 0 {
			lastIdx := len(p.lines) - 1
			lastLine := p.lines[lastIdx]
			if len(lastLine.words) == 0 || lastLine.words[0].Y+lastLine.words[0].H >= 0 {
				break
			}

			// Cleanup old line
			for _, word := range lastLine.words {
				gl.DeleteTextures(1, &word.ID)
			}
			p.lines = p.lines[:lastIdx]

			// Create new line at top
			if err := p.addLineAtTop(); err != nil {
				return err
			}
		}
	}

	return nil
}

// addLineAtBottom creates and adds a new line at the bottom of the display
func (p *ScrollerPage) addLineAtBottom() error {
	newWords := p.activePattern.GetNextLine(p.words, &p.wordIndex, p.lineCtx)
	line := Line{words: []StringItem{}}

	for _, word := range newWords {
		if word == "" {
			continue
		}
		item, err := NewStringItem(word, p.font, p.colorPattern.Color())
		if err != nil {
			return fmt.Errorf("unable to create string item for word '%s': %w", word, err)
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

	// Position relative to last existing line
	var baseY int32
	if len(p.lines) > 0 && len(p.lines[len(p.lines)-1].words) > 0 {
		baseY = p.lines[len(p.lines)-1].words[0].Y - int32(p.lineHeight)
	} else {
		baseY = -int32(p.lineHeight)
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
	return nil
}

// addLineAtTop creates and adds a new line at the top of the display
func (p *ScrollerPage) addLineAtTop() error {
	newWords := p.activePattern.GetNextLine(p.words, &p.wordIndex, p.lineCtx)
	line := Line{words: []StringItem{}}

	for _, word := range newWords {
		if word == "" {
			continue
		}
		item, err := NewStringItem(word, p.font, p.colorPattern.Color())
		if err != nil {
			return fmt.Errorf("unable to create string item for word '%s': %w", word, err)
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

	// Position relative to first existing line
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
	return nil
}

func (p *ScrollerPage) Render() error {
	for _, line := range p.lines {
		for _, word := range line.words {
			if word.ID == 0 {
				continue
			}
			if err := p.RenderTexture(p.shader, word.ID, word.Position(), word.RawDimensions(), word.Color); err != nil {
				return fmt.Errorf("failed to render texture for word '%s': %w", word.content, err)
			}
		}
	}
	return nil
}

func (p *ScrollerPage) Destroy() error {
	if p == nil {
		return nil
	}

	mixer.Stop(1000)

	// GL cleanup — delete all text textures
	for i := range p.lines {
		for j := range p.lines[i].words {
			if id := p.lines[i].words[j].ID; id != 0 {
				gl.DeleteTextures(1, &p.lines[i].words[j].ID)
				p.lines[i].words[j].ID = 0
			}
		}
	}

	gl.Finish()

	// Drop references for GC
	p.lines = nil
	p.words = nil
	p.lineCtx = nil
	p.font = nil
	p.shader = nil
	p.colorPattern = nil

	return nil
}

// SetAutoVelocity configures the automatic velocity system for the ScrollerPage.
func (p *ScrollerPage) SetAutoVelocity(enabled bool, floor, ceiling, step, delay, hold float32) {
	p.autoVelocity = enabled
	p.velocityFloor = -float32(math.Abs(float64(floor)))
	p.velocityCeiling = float32(math.Abs(float64(ceiling)))
	p.velocityStep = float32(math.Abs(float64(step)))
	p.velocityDelay = delay
	p.hold = hold

	// Initialize state
	p.velocityDirection = 1
	p.velocityState = 0
	p.velocityTimer = 0.0
	p.velocity = p.velocityFloor
}

// updateAutoVelocity handles automatic velocity changes
func (p *ScrollerPage) updateAutoVelocity(dt float32) {
	if !p.autoVelocity {
		return
	}

	switch p.velocityState {
	case 0: // Changing velocity
		if p.velocityDirection == 1 {
			p.velocity += p.velocityStep * dt
			if p.velocity >= p.velocityCeiling {
				p.velocity = p.velocityCeiling
				p.velocityState = 1
				p.velocityTimer = 0.0
			}
		} else {
			p.velocity -= p.velocityStep * dt
			if p.velocity <= p.velocityFloor {
				p.velocity = p.velocityFloor
				p.velocityState = 2
				p.velocityTimer = 0.0
			}
		}

	case 1: // Waiting at ceiling
		p.velocityTimer += dt
		if p.velocityTimer >= p.velocityDelay {
			p.velocityDirection = -1
			p.velocityState = 0
		}

	case 2: // Waiting at floor
		p.velocityTimer += dt
		if p.velocityTimer >= p.velocityDelay {
			p.velocityDirection = 1
			p.velocityState = 0
		}
	}
}

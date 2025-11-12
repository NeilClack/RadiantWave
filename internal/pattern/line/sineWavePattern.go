package pattern

import (
	"math"

	"radiantwavetech.com/radiantwave/internal/fontManager"
)

type SineWavePattern struct {
	waveStepCounter int
}

func init() {
	Register("sine", func() LinePattern { return &SineWavePattern{} })
}

func (s *SineWavePattern) Name() string { return "Sine Wave" }
func (s *SineWavePattern) GetNextLine(words []string, wordIndex *int, ctx *LineRenderContext) []string {
	const (
		totalLines = 67
		frequency  = 5.0  // cycles across all lines
		amplitude  = 0.85 // how much wave can scale
		baseRatio  = 0.03 // minimum line width as % of screen
		sharpness  = 3.0  // sharpness of the curve
	)

	screenWidth := int32(800)
	spaceWidth := int32(10)
	font := (*fontManager.FontEntry)(nil)
	fontScale := float32(1.0)

	if ctx != nil {
		screenWidth = ctx.ScreenWidth
		spaceWidth = ctx.SpaceWidth
		font = ctx.Font
		fontScale = ctx.FontScale
	}

	// Sharpen the wave (skinnier troughs)
	// raw := (math.Sin(2*math.Pi*frequency*float64(s.waveStepCounter)/totalLines) + 1.0) / 2.0
	// waveVal := math.Pow(raw, sharpness)
	angle := 2 * math.Pi * frequency * float64(s.waveStepCounter) / totalLines
	waveVal := (math.Sin(angle) + 1.0) / 2.0

	// Target width in pixels
	targetWidth := int32(float64(screenWidth) * (baseRatio + amplitude*waveVal))

	var line []string
	var currentWidth int32

	for len(words) > 0 {
		word := words[*wordIndex%len(words)]
		*wordIndex++

		if word == "" {
			continue
		}

		wordWidth := estimateWordWidth(word, font, fontScale)
		if len(line) > 0 {
			wordWidth += spaceWidth
		}

		if len(line) == 0 {
			// Always include the first word
			line = append(line, word)
			currentWidth += wordWidth
		} else {
			// Only add next word if it fits
			if currentWidth+wordWidth > targetWidth {
				break
			}
			line = append(line, word)
			currentWidth += wordWidth
		}
	}

	s.waveStepCounter++
	return line
}

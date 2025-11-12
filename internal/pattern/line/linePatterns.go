package pattern

import (
	"fmt"

	"radiantwavetech.com/radiantwave/internal/fontManager"
)

type LinePattern interface {
	GetNextLine(words []string, wordIndex *int, ctx *LineRenderContext) []string // Generates the next line in the pattern
	Name() string                                                                // Used to display the name of a LinePattern
}

// PatternInfo holds the public information about a registered pattern.
type PatternInfo struct {
	Key         string // The machine-readable key (e.g., "sine")
	DisplayName string // The human-readable name for the UI (e.g., "Sine Wave")
}

type LineRenderContext struct {
	ScreenWidth int32
	Font        *fontManager.FontEntry
	FontScale   float32
	SpaceWidth  int32
}

var Registry = make(map[string]func() LinePattern)

func Register(name string, constructor func() LinePattern) {
	Registry[name] = constructor
}

func Get(name string) (LinePattern, error) {
	constructor, ok := Registry[name]
	if !ok {
		return nil, fmt.Errorf("failed to find line pattern %s in Registry", name)
	}
	return constructor(), nil
}

// ListAvailable returns a slice of info for all registered patterns.
// This is used to build a settings menu.
func ListAvailable() []PatternInfo {
	var list []PatternInfo
	for key, constructor := range Registry {
		pattern := constructor() // Create a temporary instance to get its name
		list = append(list, PatternInfo{
			Key:         key,
			DisplayName: pattern.Name(),
		})
	}
	return list
}

// estimateWordWidth provides a basic estimate of the width of a word based on the font and scale.
func estimateWordWidth(word string, font *fontManager.FontEntry, scale float32) int32 {
	if font == nil {
		// Basic fallback estimate
		return int32(float32(len(word)) * 10 * scale)
	}

	total := int32(0)
	for _, ch := range word {
		glyph, ok := fontManager.Get().GetGlyph(font.Name, ch)
		if !ok {
			continue
		}
		total += int32(float32(glyph.Advance) * scale)
	}
	return total
}

package pattern

import (
	"github.com/veandco/go-sdl2/sdl"
	"radiantwavetech.com/radiantwave/internal/colors"
)

type ColorPattern struct {
	Colors      []sdl.Color
	colorIndex  int
	repeatsLeft int
	a, b        int
	increasing  bool
}

func NewColorPattern(colors []sdl.Color) *ColorPattern {
	if len(colors) < 2 {
		panic("ColorPattern requires at least 2 colors")
	}
	return &ColorPattern{
		Colors:      colors,
		a:           0,
		b:           1,
		increasing:  true,
		repeatsLeft: 1, // Start with b (1)
	}
}

// ATTENTION: TED
//
// # Fibonacci-Oscillating Temporal Color Distribution for Sequential Text Elements
//
// This method distributes intent-based colors across sequential text elements (words) using an
// oscillating Fibonacci sequence to control temporal color duration, creating rhythmic color
// patterns that maintain visual balance while varying exposure times.
//
// The technology addresses the challenge of applying multiple psychologically-selected colors
// to scrolling text in a way that maintains even distribution over time while creating natural
// rhythm. Conventional approaches either apply colors randomly (creating unbalanced temporal
// distribution) or use fixed intervals (producing mechanical, unnatural patterns that lack
// harmonic variation). Our method ensures each color in the palette appears throughout the
// entire display duration while varying its temporal presence according to natural mathematical
// progression.
//
// The approach uses a bidirectional Fibonacci sequence to determine how many consecutive words
// receive each color. The sequence increases (1, 1, 2, 3, 5, 8, 13, 21, 34, 55 words) then
// decreases back down (55, 34, 21, 13, 8, 5, 3, 2, 1, 1 words), creating an oscillating pattern.
// Colors from the intent-based palette are applied in sequence, with each color's duration
// determined by the current Fibonacci value. This creates a "breathing" rhythm where colors
// appear briefly, expand to longer durations, then contract back to brief appearances before
// cycling to the next color.
//
// Key differentiators from existing solutions:
//   - Uses bidirectional Fibonacci sequence (ascending then descending) rather than fixed intervals
//   - Creates natural rhythmic variation in color exposure duration
//   - Ensures all palette colors appear throughout entire display timeline, not in isolated sections
//   - Combines intent-based color selection with mathematical temporal distribution
//   - Produces harmonic color rhythm synchronized with natural numerical progression
//
// Within our Radiant Wave generation system, this color distribution works in conjunction with
// the spatially-scrambled character patterns. While the scrambling creates spatial frequency
// signatures, the Fibonacci-oscillating color pattern creates temporal frequency modulation.
// The colors themselves are selected according to color theory principles based on the affirmation's
// psychological intent, and the oscillating Fibonacci distribution ensures these therapeutic colors
// maintain presence throughout the viewing session while creating harmonic temporal rhythm that
// reinforces the overall Radiant Wave effect.
func (c *ColorPattern) Color() sdl.Color {
	// 1, 2, 3, 5, 8, 13, 21, 34, 55, 89, 144, 233, 377, 610, 987, 1597, 2584, 4181, 6765, 10946, 17711, 28657, 46368, 75025, 121393, 196418
	// This is a failsafe in case this pattern is used but colors are not supplied. Without this check, no text will display due to not having a color
	// and becoming translucent.
	if len(c.Colors) == 0 {
		return colors.White // fallback to white
	}

	color := c.Colors[c.colorIndex%len(c.Colors)]
	c.repeatsLeft--

	if c.repeatsLeft <= 0 {
		// Move to next color in the palette
		c.colorIndex = (c.colorIndex + 1) % len(c.Colors)

		// Update Fibonacci sequence for repeat count
		if c.increasing {
			c.a, c.b = c.b, c.a+c.b
			if c.b > 55 { // Update this threahold to adjust the number of repeats
				c.increasing = false
			}
		} else {
			c.a, c.b = c.b-c.a, c.a
			if c.a <= 0 {
				c.a, c.b = 0, 1
				c.increasing = true
			}
		}

		c.repeatsLeft = c.b
	}

	return color
}

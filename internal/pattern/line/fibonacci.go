package pattern

import "syscall"

type FibonacciPattern struct {
	masterSequence []int
	sequence       []int
	increasing     bool
}

func init() {
	Register("fibonacci", func() LinePattern {
		return &FibonacciPattern{
			masterSequence: []int{1, 2, 3, 5, 8, 13, 21},
			sequence:       []int{1},
			increasing:     true,
		}
	})
}

func (f *FibonacciPattern) Name() string { return "Fibonacci" }

// ATTENTION: TED
//
// # Fibonacci-Oscillating Spatial Distribution for Sequential Text Display
//
// This method arranges sequential text elements (words from affirmations) into display lines
// where the word count per line follows an oscillating Fibonacci sequence, creating both a
// visual wave pattern in two-dimensional space and ensuring even temporal distribution of
// content segments.
//
// The technology addresses the challenge of displaying repeated affirmation content in a way
// that maintains balanced exposure of all content portions while creating visually dynamic
// spatial patterns. Conventional scrolling text approaches either use fixed line lengths
// (creating uniform, non-dynamic displays) or random word counts (producing unbalanced content
// distribution where some portions appear more frequently than others). Our method ensures
// affirmation content is consumed in progressively sized segments that balance over time while
// creating a natural wave-like visual rhythm.
//
// The approach uses a bidirectional Fibonacci sequence to determine words per line. Starting
// with minimal word count (1, 1 words), the sequence expands (2, 3, 5, 8, 13, 21... words per
// line) until reaching a threshold, then contracts back down (21, 13, 8, 5, 3, 2, 1, 1 words),
// creating an oscillating wave. As lines scroll vertically, this varying horizontal word count
// creates a lateral wave pattern in the two-dimensional display space. Words are drawn sequentially
// from the affirmation text, and the varying segment sizes (1 word, then 2, then 3, then 5, etc.)
// ensure that over multiple oscillation cycles, the entire affirmation content receives balanced
// temporal exposure.
//
// Key differentiators from existing solutions:
//   - Uses oscillating Fibonacci sequence for line word counts rather than fixed or random lengths
//   - Creates two-dimensional wave pattern from one-dimensional text sequence
//   - Ensures mathematical content distribution balance over time through progressive segmentation
//   - Produces natural expansion-contraction rhythm synchronized with Fibonacci progression
//   - Combines spatial wave formation with temporal content distribution in single mechanism
//
// Within our Radiant Wave generation system, this spatial distribution pattern works in concert
// with the character-level scrambling and color distribution methods. While character scrambling
// creates local spatial frequency patterns and color oscillation creates temporal color rhythm,
// the Fibonacci line length pattern generates a macro-level spatial wave that modulates the
// overall display geometry. The mathematical relationship between line lengths creates harmonic
// spatial progression, and the oscillating nature ensures affirmation content cycles through
// varying exposure durations, contributing to the layered frequency composition of the complete
// Radiant Wave output.
func (f *FibonacciPattern) GetNextLine(words []string, wordIndex *int, ctx *LineRenderContext) []string {
	if ctx == nil || len(words) == 0 {
		return nil
	}

	numWords := f.sequence[len(f.sequence)-1]
	targetWidth := int32(ctx.ScreenWidth)

	var line []string
	var currentWidth int32
	wordsAdded := 0

	for wordsAdded < numWords && len(words) > 0 {
		if int(*wordIndex) > len(words) {
			*wordIndex = 0
		}
		word := words[*wordIndex%len(words)]
		*wordIndex++

		if word == "" {
			continue
		}

		wordWidth := estimateWordWidth(word, ctx.Font, ctx.FontScale)
		if len(line) > 0 {
			wordWidth += ctx.SpaceWidth
		}

		if len(line) == 0 || currentWidth+wordWidth <= int32(float32(targetWidth)*float32(.9)) {
			line = append(line, word)
			currentWidth += wordWidth
			wordsAdded++
		} else {
			break
		}
	}

	if currentWidth >= ctx.ScreenWidth {
		syscall.Kill(syscall.Getpid(), syscall.SIGTRAP)

	}

	// Update the sequence based on direction
	if f.increasing {
		if len(f.sequence) < len(f.masterSequence) {
			// Grow sequence by appending next Fibonacci number
			f.sequence = append(f.sequence, f.masterSequence[len(f.sequence)])
		} else {
			// Reached the top, now start shrinking
			f.increasing = false
			f.sequence = f.sequence[:len(f.sequence)-1]
		}
	} else {
		if len(f.sequence) > 1 {
			// Shrink the sequence
			f.sequence = f.sequence[:len(f.sequence)-1]
		} else {
			// Hit bottom, start growing again
			f.increasing = true
			f.sequence = append(f.sequence, f.masterSequence[1]) // index 1 = second element = 2
		}
	}

	return line
}

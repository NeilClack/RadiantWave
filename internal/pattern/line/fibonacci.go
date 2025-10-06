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

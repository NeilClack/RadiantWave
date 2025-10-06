package pattern

import (
	"github.com/veandco/go-sdl2/sdl"
	"radiantwavetech.com/radiant_wave/internal/colors"
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

func (c *ColorPattern) Color() sdl.Color {
	// 1, 2, 3, 5, 8, 13, 21, 34, 55, 89, 144, 233, 377, 610, 987, 1597, 2584, 4181, 6765, 10946, 17711, 28657, 46368, 75025, 121393, 196418
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

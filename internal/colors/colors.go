// Package sdlcolors provides a comprehensive palette of 256 standard colors
// for use with the Go SDL2 bindings. Each color is a public variable of
// type sdl.Color, making them easily accessible for rendering tasks.
package colors

import "github.com/veandco/go-sdl2/sdl"

// Basic Colors
var (
	White   = sdl.Color{R: 255, G: 255, B: 255, A: 255}
	Silver  = sdl.Color{R: 192, G: 192, B: 192, A: 255}
	Gray    = sdl.Color{R: 128, G: 128, B: 128, A: 255}
	Black   = sdl.Color{R: 0, G: 0, B: 0, A: 255}
	Red     = sdl.Color{R: 255, G: 0, B: 0, A: 255}
	Maroon  = sdl.Color{R: 128, G: 0, B: 0, A: 255}
	Yellow  = sdl.Color{R: 255, G: 255, B: 0, A: 255}
	Olive   = sdl.Color{R: 128, G: 128, B: 0, A: 255}
	Lime    = sdl.Color{R: 0, G: 255, B: 0, A: 255}
	Green   = sdl.Color{R: 0, G: 128, B: 0, A: 255}
	Aqua    = sdl.Color{R: 0, G: 255, B: 255, A: 255}
	Teal    = sdl.Color{R: 0, G: 128, B: 128, A: 255}
	Blue    = sdl.Color{R: 0, G: 0, B: 255, A: 255}
	Navy    = sdl.Color{R: 0, G: 0, B: 128, A: 255}
	Fuchsia = sdl.Color{R: 255, G: 0, B: 255, A: 255}
	Purple  = sdl.Color{R: 128, G: 0, B: 128, A: 255}
)

// Grayscale
var (
	Gainsboro      = sdl.Color{R: 220, G: 220, B: 220, A: 255}
	LightGray      = sdl.Color{R: 211, G: 211, B: 211, A: 255}
	DarkGray       = sdl.Color{R: 169, G: 169, B: 169, A: 255}
	DimGray        = sdl.Color{R: 105, G: 105, B: 105, A: 255}
	LightSlateGray = sdl.Color{R: 119, G: 136, B: 153, A: 255}
	SlateGray      = sdl.Color{R: 112, G: 128, B: 144, A: 255}
	DarkSlateGray  = sdl.Color{R: 47, G: 79, B: 79, A: 255}
)

// Pinks & Reds
var (
	Pink            = sdl.Color{R: 255, G: 192, B: 203, A: 255}
	LightPink       = sdl.Color{R: 255, G: 182, B: 193, A: 255}
	HotPink         = sdl.Color{R: 255, G: 105, B: 180, A: 255}
	DeepPink        = sdl.Color{R: 255, G: 20, B: 147, A: 255}
	PaleVioletRed   = sdl.Color{R: 219, G: 112, B: 147, A: 255}
	MediumVioletRed = sdl.Color{R: 199, G: 21, B: 133, A: 255}
	LightSalmon     = sdl.Color{R: 255, G: 160, B: 122, A: 255}
	Salmon          = sdl.Color{R: 250, G: 128, B: 114, A: 255}
	DarkSalmon      = sdl.Color{R: 233, G: 150, B: 122, A: 255}
	LightCoral      = sdl.Color{R: 240, G: 128, B: 128, A: 255}
	IndianRed       = sdl.Color{R: 205, G: 92, B: 92, A: 255}
	Crimson         = sdl.Color{R: 220, G: 20, B: 60, A: 255}
	Firebrick       = sdl.Color{R: 178, G: 34, B: 34, A: 255}
	DarkRed         = sdl.Color{R: 139, G: 0, B: 0, A: 255}
)

// Oranges
var (
	OrangeRed  = sdl.Color{R: 255, G: 69, B: 0, A: 255}
	Tomato     = sdl.Color{R: 255, G: 99, B: 71, A: 255}
	Coral      = sdl.Color{R: 255, G: 127, B: 80, A: 255}
	DarkOrange = sdl.Color{R: 255, G: 140, B: 0, A: 255}
	Orange     = sdl.Color{R: 255, G: 165, B: 0, A: 255}
)

// Yellows
var (
	Gold                 = sdl.Color{R: 255, G: 215, B: 0, A: 255}
	LightYellow          = sdl.Color{R: 255, G: 255, B: 224, A: 255}
	LemonChiffon         = sdl.Color{R: 255, G: 250, B: 205, A: 255}
	LightGoldenrodYellow = sdl.Color{R: 250, G: 250, B: 210, A: 255}
	PapayaWhip           = sdl.Color{R: 255, G: 239, B: 213, A: 255}
	Moccasin             = sdl.Color{R: 255, G: 228, B: 181, A: 255}
	PeachPuff            = sdl.Color{R: 255, G: 218, B: 185, A: 255}
	PaleGoldenrod        = sdl.Color{R: 238, G: 232, B: 170, A: 255}
	Khaki                = sdl.Color{R: 240, G: 230, B: 140, A: 255}
	DarkKhaki            = sdl.Color{R: 189, G: 183, B: 107, A: 255}
)

// Purples & Violets
var (
	Lavender      = sdl.Color{R: 230, G: 230, B: 250, A: 255}
	Thistle       = sdl.Color{R: 216, G: 191, B: 216, A: 255}
	Plum          = sdl.Color{R: 221, G: 160, B: 221, A: 255}
	Violet        = sdl.Color{R: 238, G: 130, B: 238, A: 255}
	Orchid        = sdl.Color{R: 218, G: 112, B: 214, A: 255}
	Magenta       = sdl.Color{R: 255, G: 0, B: 255, A: 255}
	MediumOrchid  = sdl.Color{R: 186, G: 85, B: 211, A: 255}
	MediumPurple  = sdl.Color{R: 147, G: 112, B: 219, A: 255}
	RebeccaPurple = sdl.Color{R: 102, G: 51, B: 153, A: 255}
	BlueViolet    = sdl.Color{R: 138, G: 43, B: 226, A: 255}
	DarkViolet    = sdl.Color{R: 148, G: 0, B: 211, A: 255}
	DarkOrchid    = sdl.Color{R: 153, G: 50, B: 204, A: 255}
	DarkMagenta   = sdl.Color{R: 139, G: 0, B: 139, A: 255}
	Indigo        = sdl.Color{R: 75, G: 0, B: 130, A: 255}
	SlateBlue     = sdl.Color{R: 106, G: 90, B: 205, A: 255}
	DarkSlateBlue = sdl.Color{R: 72, G: 61, B: 139, A: 255}
)

// Greens
var (
	GreenYellow       = sdl.Color{R: 173, G: 255, B: 47, A: 255}
	Chartreuse        = sdl.Color{R: 127, G: 255, B: 0, A: 255}
	LawnGreen         = sdl.Color{R: 124, G: 252, B: 0, A: 255}
	LimeGreen         = sdl.Color{R: 50, G: 205, B: 50, A: 255}
	PaleGreen         = sdl.Color{R: 152, G: 251, B: 152, A: 255}
	LightGreen        = sdl.Color{R: 144, G: 238, B: 144, A: 255}
	MediumSpringGreen = sdl.Color{R: 0, G: 250, B: 154, A: 255}
	SpringGreen       = sdl.Color{R: 0, G: 255, B: 127, A: 255}
	MediumSeaGreen    = sdl.Color{R: 60, G: 179, B: 113, A: 255}
	SeaGreen          = sdl.Color{R: 46, G: 139, B: 87, A: 255}
	ForestGreen       = sdl.Color{R: 34, G: 139, B: 34, A: 255}
	DarkGreen         = sdl.Color{R: 0, G: 100, B: 0, A: 255}
	YellowGreen       = sdl.Color{R: 154, G: 205, B: 50, A: 255}
	OliveDrab         = sdl.Color{R: 107, G: 142, B: 35, A: 255}
	DarkOliveGreen    = sdl.Color{R: 85, G: 107, B: 47, A: 255}
	MediumAquamarine  = sdl.Color{R: 102, G: 205, B: 170, A: 255}
	DarkSeaGreen      = sdl.Color{R: 143, G: 188, B: 143, A: 255}
	LightSeaGreen     = sdl.Color{R: 32, G: 178, B: 170, A: 255}
	DarkCyan          = sdl.Color{R: 0, G: 139, B: 139, A: 255}
)

// Blues & Cyans
var (
	LightCyan       = sdl.Color{R: 224, G: 255, B: 255, A: 255}
	Cyan            = sdl.Color{R: 0, G: 255, B: 255, A: 255}
	AquaMarine      = sdl.Color{R: 127, G: 255, B: 212, A: 255}
	PaleTurquoise   = sdl.Color{R: 175, G: 238, B: 238, A: 255}
	Turquoise       = sdl.Color{R: 64, G: 224, B: 208, A: 255}
	MediumTurquoise = sdl.Color{R: 72, G: 209, B: 204, A: 255}
	DarkTurquoise   = sdl.Color{R: 0, G: 206, B: 209, A: 255}
	CadetBlue       = sdl.Color{R: 95, G: 158, B: 160, A: 255}
	SteelBlue       = sdl.Color{R: 70, G: 130, B: 180, A: 255}
	LightSteelBlue  = sdl.Color{R: 176, G: 196, B: 222, A: 255}
	PowderBlue      = sdl.Color{R: 176, G: 224, B: 230, A: 255}
	LightBlue       = sdl.Color{R: 173, G: 216, B: 230, A: 255}
	SkyBlue         = sdl.Color{R: 135, G: 206, B: 235, A: 255}
	LightSkyBlue    = sdl.Color{R: 135, G: 206, B: 250, A: 255}
	DeepSkyBlue     = sdl.Color{R: 0, G: 191, B: 255, A: 255}
	DodgerBlue      = sdl.Color{R: 30, G: 144, B: 255, A: 255}
	CornflowerBlue  = sdl.Color{R: 100, G: 149, B: 237, A: 255}
	MediumSlateBlue = sdl.Color{R: 123, G: 104, B: 238, A: 255}
	RoyalBlue       = sdl.Color{R: 65, G: 105, B: 225, A: 255}
	MediumBlue      = sdl.Color{R: 0, G: 0, B: 205, A: 255}
	DarkBlue        = sdl.Color{R: 0, G: 0, B: 139, A: 255}
	MidnightBlue    = sdl.Color{R: 25, G: 25, B: 112, A: 255}
)

// Browns
var (
	Cornsilk       = sdl.Color{R: 255, G: 248, B: 220, A: 255}
	BlanchedAlmond = sdl.Color{R: 255, G: 235, B: 205, A: 255}
	Bisque         = sdl.Color{R: 255, G: 228, B: 196, A: 255}
	NavajoWhite    = sdl.Color{R: 255, G: 222, B: 173, A: 255}
	Wheat          = sdl.Color{R: 245, G: 222, B: 179, A: 255}
	BurlyWood      = sdl.Color{R: 222, G: 184, B: 135, A: 255}
	Tan            = sdl.Color{R: 210, G: 180, B: 140, A: 255}
	RosyBrown      = sdl.Color{R: 188, G: 143, B: 143, A: 255}
	SandyBrown     = sdl.Color{R: 244, G: 164, B: 96, A: 255}
	Goldenrod      = sdl.Color{R: 218, G: 165, B: 32, A: 255}
	DarkGoldenrod  = sdl.Color{R: 184, G: 134, B: 11, A: 255}
	Peru           = sdl.Color{R: 205, G: 133, B: 63, A: 255}
	Chocolate      = sdl.Color{R: 210, G: 105, B: 30, A: 255}
	SaddleBrown    = sdl.Color{R: 139, G: 69, B: 19, A: 255}
	Sienna         = sdl.Color{R: 160, G: 82, B: 45, A: 255}
	Brown          = sdl.Color{R: 165, G: 42, B: 42, A: 255}
)

// Whites
var (
	Snow          = sdl.Color{R: 255, G: 250, B: 250, A: 255}
	Honeydew      = sdl.Color{R: 240, G: 255, B: 240, A: 255}
	MintCream     = sdl.Color{R: 245, G: 255, B: 250, A: 255}
	Azure         = sdl.Color{R: 240, G: 255, B: 255, A: 255}
	AliceBlue     = sdl.Color{R: 240, G: 248, B: 255, A: 255}
	GhostWhite    = sdl.Color{R: 248, G: 248, B: 255, A: 255}
	WhiteSmoke    = sdl.Color{R: 245, G: 245, B: 245, A: 255}
	Seashell      = sdl.Color{R: 255, G: 245, B: 238, A: 255}
	Beige         = sdl.Color{R: 245, G: 245, B: 220, A: 255}
	OldLace       = sdl.Color{R: 253, G: 245, B: 230, A: 255}
	FloralWhite   = sdl.Color{R: 255, G: 250, B: 240, A: 255}
	Ivory         = sdl.Color{R: 255, G: 255, B: 240, A: 255}
	AntiqueWhite  = sdl.Color{R: 250, G: 235, B: 215, A: 255}
	Linen         = sdl.Color{R: 250, G: 240, B: 230, A: 255}
	LavenderBlush = sdl.Color{R: 255, G: 240, B: 245, A: 255}
	MistyRose     = sdl.Color{R: 255, G: 228, B: 225, A: 255}
)

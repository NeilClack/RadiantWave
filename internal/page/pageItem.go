package page

import (
	"fmt"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/veandco/go-sdl2/sdl"
	"radiantwavetech.com/radiantwave/internal/fontManager"
	"radiantwavetech.com/radiantwave/internal/shaderManager"
)

// PageItem defines the basic features of any item on any page
// This coupling makes creating, updating and managing pages easier
type PageItem struct {
	ID    uint32
	W     int32
	H     int32
	X     int32 // Horizontal position
	Y     int32 // Vertical position
	Color sdl.Color
	// index int
}

// Scale updates the W and H of an item based on a scale factor (usually determined by base font size / desired font size)
func (i *PageItem) Scale(scale float32) {
	i.W = int32(float32(i.W) * scale)
	i.H = int32(float32(i.H) * scale)
}

// Dimensions returns an mgl32.Vec2 containing the W and H of an item, respectively
func (i *PageItem) Dimensions(scale float32) mgl32.Vec2 {
	return mgl32.Vec2{float32(float32(i.W) * scale), float32(float32(i.H) * scale)}
}

// RawDimensions returns a mgl32.Vec2 containing the W and H of an item, without scaling
// This method will replace Dimensions in the future. Future PageItems should first call Scale(), then RawDimensions()
func (i *PageItem) RawDimensions() mgl32.Vec2 {
	return mgl32.Vec2{float32(i.W), float32(i.H)}
}

// Position returns an mgl32.Vec2 contianing the X and Y positions of an item, respectively
func (i *PageItem) Position() mgl32.Vec2 {
	return mgl32.Vec2{float32(i.X), float32(i.Y)}
}

// StringItem is an extended PageItem that contains a string content and string specific methods
type StringItem struct {
	PageItem
	content string
}

// NewStringItem creates a new StringItem with the given content, font, and color.
// It initializes the texture ID, width, and height by creating a string texture using the font
// and shader manager.
func NewStringItem(content string, font *fontManager.FontEntry, color sdl.Color) (StringItem, error) {
	var err error

	fm := fontManager.Get()
	sm := shaderManager.Get()

	shader, ok := sm.Get("text")
	if !ok {
		return StringItem{}, fmt.Errorf("failed to fetch shader")
	}

	id, width, height, err := fm.CreateStringTexture(font.Name, content, shader)
	if err != nil {
		return StringItem{}, err
	}

	newStringItem := StringItem{
		PageItem: PageItem{
			ID:    id,
			W:     width,
			H:     height,
			Color: color,
		},
		content: content,
	}

	return newStringItem, err

}

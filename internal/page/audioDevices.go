package page

import (
	"fmt"
	"strconv"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/veandco/go-sdl2/sdl"
	"radiantwavetech.com/radiantwave/internal/audio"
	"radiantwavetech.com/radiantwave/internal/colors"
	"radiantwavetech.com/radiantwave/internal/db"
	"radiantwavetech.com/radiantwave/internal/fontManager"
	"radiantwavetech.com/radiantwave/internal/shaderManager"
)

type AudioDevices struct {
	Base
	font           *fontManager.FontEntry
	textShader     *shaderManager.Shader
	solidShader    *shaderManager.Shader
	title          StringItem
	prompt         StringItem
	devices        []StringItem
	highlightIndex int
	selectedIndex  int
	verticalMargin int32
}

func (p *AudioDevices) Init(app ApplicationInterface) error {
	var err error
	p.verticalMargin = int32(20)

	// Get font
	fm := fontManager.Get()
	font, ok := fm.GetFont("Roboto-Regular")
	if !ok {
		return fmt.Errorf("failed to get font: %w", sdl.GetError())
	}
	p.font = font

	// Get shaders
	sm := shaderManager.Get()
	p.textShader, ok = sm.Get("text")
	if !ok {
		return fmt.Errorf("failed to get text shader: %w", sdl.GetError())
	}
	p.solidShader, ok = sm.Get("solid_color")
	if !ok {
		return fmt.Errorf("failed to get solid_color shader: %w", sdl.GetError())
	}

	// Initialize base
	if err := p.Base.Init(app); err != nil {
		return fmt.Errorf("base init failed: %w", err)
	}

	// Get font size from database
	baseFontSizeStr, err := db.GetConfigValue("init_font_size")
	if err != nil {
		return fmt.Errorf("retrieving init_font_size from db: %w", err)
	}
	baseFontSize, err := strconv.Atoi(baseFontSizeStr)
	if err != nil {
		return fmt.Errorf("parsing init_font_size %q: %w", baseFontSizeStr, err)
	}

	// Calculate scales
	titleScale := float32(64) / float32(baseFontSize)
	itemScale := float32(32) / float32(baseFontSize)

	// Create title
	p.title, err = NewStringItem("Select an Audio Output Device", p.font, colors.White)
	if err != nil {
		return fmt.Errorf("creating title: %w", err)
	}
	p.title.Scale(titleScale)

	// Create prompt
	p.prompt, err = NewStringItem("Use Arrow Keys to select, Enter to confirm", p.font, colors.White)
	if err != nil {
		return fmt.Errorf("creating prompt: %w", err)
	}
	p.prompt.Scale(itemScale)

	// Get list of audio output devices
	devices, err := audio.ListOutputDevices()
	if err != nil {
		return fmt.Errorf("failed to get audio devices: %w", err)
	}

	// Get currently selected device from database
	currentDevice, err := db.GetConfigValue("audio_device_name")
	if err != nil {
		return fmt.Errorf("retrieving audio_device_name from db: %w", err)
	}

	// Create UI items for each device
	for i, name := range devices {
		item, err := NewStringItem(name, p.font, colors.White)
		if err != nil {
			return fmt.Errorf("creating device item: %w", err)
		}
		item.Scale(itemScale)
		p.devices = append(p.devices, item)

		// Mark currently selected device
		if name == currentDevice {
			p.selectedIndex = i
			p.highlightIndex = i // Start with cursor on current device
		}
	}

	p.layout()
	return nil
}

func (p *AudioDevices) layout() {
	// Position title at bottom
	p.title.X = p.ScreenCenterX - p.title.W/2
	p.title.Y = p.ScreenHeight - p.title.H - p.verticalMargin*2

	// Position prompt above title
	p.prompt.X = p.ScreenCenterX - p.prompt.W/2
	p.prompt.Y = p.title.Y - p.verticalMargin*2 - p.prompt.H

	// Position device items stacked upward from prompt
	for i := range p.devices {
		p.devices[i].X = p.ScreenCenterX - p.devices[i].W/2
		if i == 0 {
			p.devices[i].Y = p.prompt.Y - p.verticalMargin*2 - p.devices[i].H
		} else {
			p.devices[i].Y = p.devices[i-1].Y - p.devices[i].H - 10
		}
	}
}

func (p *AudioDevices) HandleEvent(event *sdl.Event) error {
	switch e := (*event).(type) {
	case *sdl.KeyboardEvent:
		if e.Type != sdl.KEYDOWN {
			return nil
		}
		switch e.Keysym.Sym {
		case sdl.K_UP:
			if p.highlightIndex > 0 {
				p.highlightIndex--
			}
		case sdl.K_DOWN:
			if p.highlightIndex < len(p.devices)-1 {
				p.highlightIndex++
			}
		case sdl.K_RETURN:
			// Save selected device to database
			deviceName := p.devices[p.highlightIndex].content
			if err := db.SetConfigValue("audio_device_name", deviceName); err != nil {
				return fmt.Errorf("failed to save audio device: %w", err)
			}
			p.App.UnwindToPage(&Settings{})
		case sdl.K_BACKSPACE:
			p.App.UnwindToPage(&Settings{})
		}
	}
	return nil
}

func (p *AudioDevices) Render() error {
	// Always render title and prompt
	p.RenderTexture(p.textShader, p.title.ID, p.title.Position(), p.title.RawDimensions(), p.title.Color)
	p.RenderTexture(p.textShader, p.prompt.ID, p.prompt.Position(), p.prompt.RawDimensions(), p.prompt.Color)

	// No devices to display
	if len(p.devices) == 0 {
		return nil
	}

	// Render highlight box behind highlighted item
	p.RenderSolidColorQuad(
		p.solidShader,
		p.devices[p.highlightIndex].Position(),
		p.devices[p.highlightIndex].RawDimensions(),
		colors.White,
	)

	// Render device items with appropriate colors
	for i, item := range p.devices {
		if i == p.highlightIndex && i == p.selectedIndex {
			// Currently highlighted AND selected device - dark green
			item.Color = colors.DarkGreen
		} else if i == p.highlightIndex {
			// Just highlighted - black (shows on white background)
			item.Color = colors.Black
		} else if i == p.selectedIndex {
			// Just selected - green
			item.Color = colors.Green
		} else {
			// Normal item - white
			item.Color = colors.White
		}
		p.RenderTexture(p.textShader, item.ID, item.Position(), item.RawDimensions(), item.Color)
	}

	return nil
}

func (p *AudioDevices) Destroy() error {
	gl.DeleteTextures(1, &p.title.ID)
	gl.DeleteTextures(1, &p.prompt.ID)
	for _, item := range p.devices {
		gl.DeleteTextures(1, &item.ID)
	}
	return nil
}

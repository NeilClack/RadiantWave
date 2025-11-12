package page

import (
	"fmt"
	"strconv"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/veandco/go-sdl2/sdl"
	"radiantwavetech.com/radiantwave/internal/colors"
	"radiantwavetech.com/radiantwave/internal/db"
	"radiantwavetech.com/radiantwave/internal/fontManager"
	"radiantwavetech.com/radiantwave/internal/shaderManager"
)

type Settings struct {
	Base
	font           *fontManager.FontEntry
	textShader     *shaderManager.Shader
	solidShader    *shaderManager.Shader
	title          StringItem
	prompt         StringItem
	options        []Option
	verticalMargin int32
	bgHighlight    PageItem
	highlightIndex int
}

type Option struct {
	page  Page
	label StringItem
}

func (p *Settings) Init(app ApplicationInterface) error {
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
		return fmt.Errorf("failed base page initialization: %w", err)
	}

	// Create title
	p.title, err = NewStringItem("Settings", font, colors.White)
	if err != nil {
		return fmt.Errorf("creating title: %w", err)
	}

	// Create prompt
	p.prompt, err = NewStringItem("Use Arrow Keys to Navigate, Enter to Select, or F1 to return to the Main Menu", font, colors.White)
	if err != nil {
		return fmt.Errorf("creating prompt: %w", err)
	}

	// Create options
	affirmationsLabel, err := NewStringItem("Affirmations", font, colors.White)
	if err != nil {
		return fmt.Errorf("creating affirmations label: %w", err)
	}
	p.options = append(p.options, Option{
		page:  &AffirmationOptions{},
		label: affirmationsLabel,
	})

	wifiLabel, err := NewStringItem("WiFi Setup", font, colors.White)
	if err != nil {
		return fmt.Errorf("creating wifi label: %w", err)
	}
	p.options = append(p.options, Option{
		page:  &WiFiSetupPage{},
		label: wifiLabel,
	})

	audioLabel, err := NewStringItem("Audio Device", font, colors.White)
	if err != nil {
		return fmt.Errorf("creating audio label: %w", err)
	}
	p.options = append(p.options, Option{
		page:  &AudioDevices{},
		label: audioLabel,
	})

	// Get base font size from database
	baseFontSizeStr, err := db.GetConfigValue("init_font_size")
	if err != nil {
		return fmt.Errorf("retrieving init_font_size from db: %w", err)
	}
	baseFontSize, err := strconv.Atoi(baseFontSizeStr)
	if err != nil {
		return fmt.Errorf("parsing init_font_size: %w", err)
	}

	// Calculate scales
	titleScale := float32(64) / float32(baseFontSize)
	scale := float32(32) / float32(baseFontSize)

	// Apply scales
	p.title.Scale(titleScale)
	p.prompt.Scale(scale)

	for i := range p.options {
		p.options[i].label.Scale(scale)
	}

	// Create background highlight using first option's dimensions
	p.bgHighlight = PageItem{
		ID:    0,
		W:     p.options[0].label.W,
		H:     p.options[0].label.H,
		Color: colors.White,
	}
	p.highlightIndex = 0

	// Position title at bottom
	p.title.X = p.Base.ScreenCenterX - p.title.W/2
	p.title.Y = p.ScreenHeight - p.title.H - p.verticalMargin*2

	// Position prompt above title
	p.prompt.X = p.ScreenCenterX - p.prompt.W/2
	p.prompt.Y = p.title.Y - p.verticalMargin*2 - p.prompt.H

	// Position options stacked upward from prompt
	for i, option := range p.options {
		p.options[i].label.X = p.ScreenCenterX - option.label.W/2
		if i == 0 {
			p.options[i].label.Y = int32(float32(p.prompt.Y)-(float32(p.verticalMargin)*4)) - option.label.H
		} else {
			p.options[i].label.Y = p.options[i-1].label.Y - option.label.H - 10
		}
	}

	return nil
}

func (p *Settings) Update(dt float32) error {
	return nil
}

func (p *Settings) HandleEvent(event *sdl.Event) error {
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
			if p.highlightIndex < len(p.options)-1 {
				p.highlightIndex++
			}
		case sdl.K_RETURN:
			if p.highlightIndex >= 0 && p.highlightIndex < len(p.options) {
				selectedOption := p.options[p.highlightIndex]
				p.App.PushPage(selectedOption.page)
			}
		case sdl.K_BACKSPACE:
			p.App.UnwindToPage(&Welcome{})
		}
	}
	return nil
}

func (p *Settings) Render() error {
	// Render title and prompt
	p.RenderTexture(p.textShader, p.title.ID, p.title.Position(), p.title.RawDimensions(), p.title.Color)
	p.RenderTexture(p.textShader, p.prompt.ID, p.prompt.Position(), p.prompt.RawDimensions(), p.prompt.Color)

	// Render highlight behind selected option
	p.RenderSolidColorQuad(
		p.solidShader,
		p.options[p.highlightIndex].label.Position(),
		p.options[p.highlightIndex].label.RawDimensions(),
		p.bgHighlight.Color,
	)

	// Render all options with appropriate colors
	for i, option := range p.options {
		if p.highlightIndex == i {
			option.label.Color = colors.Black
		} else {
			option.label.Color = colors.White
		}
		p.RenderTexture(p.textShader, option.label.ID, option.label.Position(), option.label.RawDimensions(), option.label.Color)
	}

	return nil
}

func (p *Settings) Destroy() error {
	gl.DeleteTextures(1, &p.title.ID)
	gl.DeleteTextures(1, &p.prompt.ID)
	for i := range p.options {
		gl.DeleteTextures(1, &p.options[i].label.ID)
	}
	return nil
}

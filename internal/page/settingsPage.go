package page

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/veandco/go-sdl2/sdl"
	"radiantwavetech.com/radiant_wave/internal/colors"
	"radiantwavetech.com/radiant_wave/internal/config"
	"radiantwavetech.com/radiant_wave/internal/fontManager"
	"radiantwavetech.com/radiant_wave/internal/shaderManager"
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

	fm := fontManager.Get()
	font, ok := fm.GetFont("Roboto-Regular")
	if !ok {
		return sdl.GetError()
	}
	p.font = font

	sm := shaderManager.Get()
	p.textShader, ok = sm.Get("text")
	if !ok {
		return sdl.GetError()
	}
	p.solidShader, ok = sm.Get("solid_color")
	if !ok {
		return sdl.GetError()
	}

	if err := p.Base.Init(app); err != nil {
		return fmt.Errorf("failed base page initialization: %w", err)
	}

	// Create title
	p.title, err = NewStringItem("Settings", font, colors.White)
	if err != nil {
		return err
	}

	// Create prompt
	p.prompt, err = NewStringItem("Use Arrow Keys to Navigate, Enter to Select, or F1 to return to the Main Menu", font, colors.White)
	if err != nil {
		return err
	}

	// Create options
	affirmationsLabel, err := NewStringItem("Affirmations", font, colors.White)
	if err != nil {
		return err
	}
	affirmationsOption := Option{
		page:  &AffirmationOptions{},
		label: affirmationsLabel,
	}

	p.options = append(p.options, affirmationsOption)

	wifiLabel, err := NewStringItem("WiFi Setup", font, colors.White)
	if err != nil {
		return err
	}
	wifiOption := Option{
		page:  &WiFiSetupPage{},
		label: wifiLabel,
	}
	p.options = append(p.options, wifiOption)

	audioLabel, err := NewStringItem("Audio Device", font, colors.White)
	if err != nil {
		return err
	}
	audioOption := Option{
		page:  &AudioDevices{},
		label: audioLabel,
	}
	p.options = append(p.options, audioOption)

	// Size everything
	config := config.Get()
	titleScale := float32(64) / float32(config.BaseFontSize)
	scale := float32(32) / float32(config.BaseFontSize)

	p.title.Scale(titleScale)
	p.prompt.Scale(scale)

	for i := range p.options {
		p.options[i].label.Scale(scale)
	}

	// Grab the [0] option's size to use for the background highlight
	p.bgHighlight = PageItem{
		ID:    0, // ID gets created later when the texture is created.
		W:     p.options[0].label.W,
		H:     p.options[0].label.H,
		Color: colors.White,
	}
	p.highlightIndex = 0

	// Positions
	p.title.X = p.Base.ScreenCenterX - p.title.W/2              // Center horizontally
	p.title.Y = p.ScreenHeight - p.title.H - p.verticalMargin*2 // Position at the top 1 vertical Margin down

	p.prompt.X = p.ScreenCenterX - p.prompt.W/2              // Center horizontally
	p.prompt.Y = p.title.Y - p.verticalMargin*2 - p.prompt.H // Position below the title

	for i, option := range p.options {
		p.options[i].label.X = p.ScreenCenterX - option.label.W/2 // Center horizontally
		if i == 0 {                                               // This is the first element
			p.options[i].label.Y = int32(float32(p.prompt.Y)-(float32(p.verticalMargin)*4)) - option.label.H
		} else {
			p.options[i].label.Y = p.options[i-1].label.Y - option.label.H - 10 // Position below the previous option
		}
	}

	return nil
}

func (p *Settings) Update(dt float32) error { return nil }

func (p *Settings) HandleEvent(event *sdl.Event) error {
	switch e := (*event).(type) {
	case *sdl.KeyboardEvent:
		if e.Type != sdl.KEYDOWN {
			return nil // Only handle key down events
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
			fmt.Println("Exiting settings page")
			return nil
		}
	}
	return nil
}

func (p *Settings) Render() error {
	p.RenderTexture(p.textShader, p.title.ID, p.title.Position(), p.title.RawDimensions(), p.title.Color)
	p.RenderTexture(p.textShader, p.prompt.ID, p.prompt.Position(), p.prompt.RawDimensions(), p.prompt.Color)
	p.RenderSolidColorQuad(p.solidShader, p.options[p.highlightIndex].label.Position(), p.options[p.highlightIndex].label.RawDimensions(), p.bgHighlight.Color)

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
	// for _, option := range p.options {
	// gl.DeleteTextures(1, &option.ID)
	// }
	return nil
}

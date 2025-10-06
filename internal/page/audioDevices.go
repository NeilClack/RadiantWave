package page

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/veandco/go-sdl2/sdl"
	"radiantwavetech.com/radiant_wave/internal/audio"
	"radiantwavetech.com/radiant_wave/internal/colors"
	"radiantwavetech.com/radiant_wave/internal/config"
	"radiantwavetech.com/radiant_wave/internal/fontManager"
	"radiantwavetech.com/radiant_wave/internal/shaderManager"
)

type AudioDevices struct {
	Base
	font           *fontManager.FontEntry
	textShader     *shaderManager.Shader
	title          StringItem
	prompt         StringItem
	devices        []StringItem
	highlightIndex int
	verticalMargin int32
	solidShader    *shaderManager.Shader
	selectedIndex  int
}

func (p *AudioDevices) Init(app ApplicationInterface) error {
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
		return fmt.Errorf("base init failed: %w", err)
	}

	p.title, err = NewStringItem("Select an Audio Output Device", p.font, colors.White)
	if err != nil {
		return err
	}
	p.prompt, err = NewStringItem("Use Arrow Keys to select, Enter to confirm", p.font, colors.White)
	if err != nil {
		return err
	}

	cfg := config.Get()
	titleScale := float32(64) / float32(cfg.BaseFontSize)
	itemScale := float32(32) / float32(cfg.BaseFontSize)
	p.title.Scale(titleScale)
	p.prompt.Scale(itemScale)

	devices, err := audio.ListOutputDevices()
	if err != nil {
		return fmt.Errorf("failed to get audio devices: %w", err)
	}

	for i, name := range devices {
		item, err := NewStringItem(name, p.font, colors.White)
		if err != nil {
			return err
		}
		item.Scale(itemScale)
		p.devices = append(p.devices, item)
		if name == cfg.AudioDeviceName {
			p.selectedIndex = i
		}
	}

	p.layout()
	return nil
}

func (p *AudioDevices) layout() {
	p.title.X = p.ScreenCenterX - p.title.W/2
	p.title.Y = p.ScreenHeight - p.title.H - p.verticalMargin*2

	p.prompt.X = p.ScreenCenterX - p.prompt.W/2
	p.prompt.Y = p.title.Y - p.verticalMargin*2 - p.prompt.H

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
			cfg := config.Get()
			cfg.AudioDeviceName = p.devices[p.highlightIndex].content
			if err := cfg.Save(); err != nil {
				fmt.Println("failed to save config:", err)
			}
			p.App.UnwindToPage(&Settings{})
		case sdl.K_BACKSPACE:
			p.App.UnwindToPage(&Settings{})
		}
	}
	return nil
}

func (p *AudioDevices) Render() error {
	p.RenderTexture(p.textShader, p.title.ID, p.title.Position(), p.title.RawDimensions(), p.title.Color)
	p.RenderTexture(p.textShader, p.prompt.ID, p.prompt.Position(), p.prompt.RawDimensions(), p.prompt.Color)

	p.RenderSolidColorQuad(
		p.solidShader,
		p.devices[p.highlightIndex].Position(),
		p.devices[p.highlightIndex].RawDimensions(),
		colors.White,
	)

	for i, item := range p.devices {
		if i == p.highlightIndex && i == p.selectedIndex {
			item.Color = colors.DarkGreen
		} else if i == p.highlightIndex {
			item.Color = colors.Black
		} else if i == p.selectedIndex {
			item.Color = colors.Green
		} else {
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

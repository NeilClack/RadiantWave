package page

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/veandco/go-sdl2/sdl"
	"radiantwavetech.com/radiantwave/internal/colors"
	"radiantwavetech.com/radiantwave/internal/config"
	"radiantwavetech.com/radiantwave/internal/fontManager"
	"radiantwavetech.com/radiantwave/internal/shaderManager"
)

type AffirmationOptions struct {
	Base
	font           *fontManager.FontEntry
	textShader     *shaderManager.Shader
	title          StringItem
	prompt         StringItem
	affirmations   []StringItem
	activeMap      map[int]bool
	highlightIndex int
	verticalMargin int32
	solidShader    *shaderManager.Shader
}

func (p *AffirmationOptions) Init(app ApplicationInterface) error {
	var err error
	p.verticalMargin = int32(20)
	p.activeMap = make(map[int]bool)

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

	p.title, err = NewStringItem("Select Affirmations", p.font, colors.White)
	if err != nil {
		return err
	}
	p.prompt, err = NewStringItem("Use Arrow Keys to select, Space to toggle, Enter to confirm", p.font, colors.White)
	if err != nil {
		return err
	}

	config := config.Get()
	titleScale := float32(64) / float32(config.BaseFontSize)
	itemScale := float32(32) / float32(config.BaseFontSize)

	p.title.Scale(titleScale)
	p.prompt.Scale(itemScale)

	dir := filepath.Join(config.AssetsDir, "affirmations", config.SystemType)
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("could not read affirmations dir: %w", err)
	}

	selected := make(map[string]bool)
	for _, path := range config.SelectedFilePaths {
		if strings.HasPrefix(path, dir) {
			base := filepath.Base(path)
			selected[strings.TrimSuffix(base, ".txt")] = true
		}
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(strings.ToLower(file.Name()), ".txt") {
			continue
		}
		name := strings.TrimSuffix(file.Name(), ".txt")
		label, err := NewStringItem(name, p.font, colors.White)
		if err != nil {
			return err
		}

		label.Scale(itemScale)
		p.affirmations = append(p.affirmations, label)
		if selected[name] {
			p.activeMap[len(p.affirmations)-1] = true
		}
	}

	p.layout()
	return nil
}

func (p *AffirmationOptions) layout() {
	p.title.X = p.ScreenCenterX - p.title.W/2
	p.title.Y = p.ScreenHeight - p.title.H - p.verticalMargin*2

	p.prompt.X = p.ScreenCenterX - p.prompt.W/2
	p.prompt.Y = p.title.Y - p.verticalMargin*2 - p.prompt.H

	for i := range p.affirmations {
		p.affirmations[i].X = p.ScreenCenterX - p.affirmations[i].W/2
		if i == 0 {
			p.affirmations[i].Y = p.prompt.Y - p.verticalMargin*2 - p.affirmations[i].H
		} else {
			p.affirmations[i].Y = p.affirmations[i-1].Y - p.affirmations[i].H - 10
		}
	}
}

func (p *AffirmationOptions) HandleEvent(event *sdl.Event) error {
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
			if p.highlightIndex < len(p.affirmations)-1 {
				p.highlightIndex++
			}
		case sdl.K_BACKSPACE:
			p.App.UnwindToPage(&Settings{})
		case sdl.K_SPACE:
			if p.activeMap[p.highlightIndex] {
				delete(p.activeMap, p.highlightIndex)
			} else {
				p.activeMap[p.highlightIndex] = true
			}
		case sdl.K_RETURN:
			config := config.Get()
			dir := filepath.Join(config.AssetsDir, "affirmations", config.SystemType)
			var selected []string
			for i := range p.activeMap {
				name := p.affirmations[i].content + ".txt"
				path := filepath.Join(dir, name)
				selected = append(selected, path)
			}
			config.SelectedFilePaths = selected
			if err := config.Save(); err != nil {
				fmt.Println("failed to save config:", err)
			}
			p.App.UnwindToPage(&Settings{})
		}
	}
	return nil
}

func (p *AffirmationOptions) Render() error {
	// Always render title/prompt
	p.RenderTexture(p.textShader, p.title.ID, p.title.Position(), p.title.RawDimensions(), p.title.Color)
	p.RenderTexture(p.textShader, p.prompt.ID, p.prompt.Position(), p.prompt.RawDimensions(), p.prompt.Color)

	// Nothing to list? Just don't draw a highlight or items.
	if len(p.affirmations) == 0 {
		return nil
	}

	// Safe to draw highlight now
	p.RenderSolidColorQuad(
		p.solidShader,
		p.affirmations[p.highlightIndex].Position(),
		p.affirmations[p.highlightIndex].RawDimensions(),
		colors.White,
	)

	for i, item := range p.affirmations {
		if p.activeMap[i] {
			item.Color = colors.Green
		} else if i == p.highlightIndex {
			item.Color = colors.Black
		} else {
			item.Color = colors.White
		}
		p.RenderTexture(p.textShader, item.ID, item.Position(), item.RawDimensions(), item.Color)
	}
	return nil
}

func (p *AffirmationOptions) Destroy() error {
	gl.DeleteTextures(1, &p.title.ID)
	gl.DeleteTextures(1, &p.prompt.ID)
	for _, item := range p.affirmations {
		gl.DeleteTextures(1, &item.ID)
	}
	return nil
}

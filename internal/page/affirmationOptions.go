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

type AffirmationOptions struct {
	Base
	font           *fontManager.FontEntry
	textShader     *shaderManager.Shader
	solidShader    *shaderManager.Shader
	title          StringItem
	prompt         StringItem
	affirmations   []StringItem
	affirmationIDs []uint       // Database IDs corresponding to each affirmation
	activeMap      map[int]bool // Maps display index to selected state
	highlightIndex int
	verticalMargin int32
}

func (p *AffirmationOptions) Init(app ApplicationInterface) error {
	var err error
	p.verticalMargin = int32(20)
	p.activeMap = make(map[int]bool)
	p.affirmationIDs = make([]uint, 0)

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
	p.title, err = NewStringItem("Select Affirmations", p.font, colors.White)
	if err != nil {
		return fmt.Errorf("creating title: %w", err)
	}
	p.title.Scale(titleScale)

	// Create prompt
	p.prompt, err = NewStringItem("Use Arrow Keys to select, Space to toggle, Enter to confirm", p.font, colors.White)
	if err != nil {
		return fmt.Errorf("creating prompt: %w", err)
	}
	p.prompt.Scale(itemScale)

	// Load affirmations from database
	affirmations, err := db.GetAffirmations()
	if err != nil {
		return fmt.Errorf("retrieving affirmations from db: %w", err)
	}

	// Create UI items for each available affirmation
	for _, affirmation := range affirmations {
		if !affirmation.Available {
			continue
		}

		label, err := NewStringItem(affirmation.Title, p.font, colors.White)
		if err != nil {
			return fmt.Errorf("creating affirmation label: %w", err)
		}

		label.Scale(itemScale)
		p.affirmations = append(p.affirmations, label)
		p.affirmationIDs = append(p.affirmationIDs, affirmation.ID)

		// Set initial selection state
		if affirmation.Selected {
			p.activeMap[len(p.affirmations)-1] = true
		}
	}

	p.layout()
	return nil
}

func (p *AffirmationOptions) layout() {
	// Position title at bottom
	p.title.X = p.ScreenCenterX - p.title.W/2
	p.title.Y = p.ScreenHeight - p.title.H - p.verticalMargin*2

	// Position prompt above title
	p.prompt.X = p.ScreenCenterX - p.prompt.W/2
	p.prompt.Y = p.title.Y - p.verticalMargin*2 - p.prompt.H

	// Position affirmation items stacked upward from prompt
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
			// Toggle selection for highlighted item
			if p.activeMap[p.highlightIndex] {
				delete(p.activeMap, p.highlightIndex)
			} else {
				p.activeMap[p.highlightIndex] = true
			}
		case sdl.K_RETURN:
			// Save selections to database and return to settings
			if err := p.saveSelections(); err != nil {
				return fmt.Errorf("failed to save affirmation selections: %w", err)
			}
			p.App.UnwindToPage(&Settings{})
		}
	}
	return nil
}

// saveSelections updates the Selected field for all affirmations in the database
func (p *AffirmationOptions) saveSelections() error {
	// Build a map of which database IDs should be selected
	selectedIDs := make(map[uint]bool)
	for displayIndex := range p.activeMap {
		if displayIndex < len(p.affirmationIDs) {
			dbID := p.affirmationIDs[displayIndex]
			selectedIDs[dbID] = true
		}
	}

	// Get all affirmations from database
	affirmations, err := db.GetAffirmations()
	if err != nil {
		return fmt.Errorf("retrieving affirmations from db: %w", err)
	}

	// Update each affirmation's Selected field if it changed
	for _, affirmation := range affirmations {
		shouldBeSelected := selectedIDs[affirmation.ID]
		if affirmation.Selected != shouldBeSelected {
			affirmation.Selected = shouldBeSelected
			if err := db.DB.Save(&affirmation).Error; err != nil {
				return fmt.Errorf("updating affirmation %d: %w", affirmation.ID, err)
			}
		}
	}

	return nil
}

func (p *AffirmationOptions) Render() error {
	// Always render title and prompt
	p.RenderTexture(p.textShader, p.title.ID, p.title.Position(), p.title.RawDimensions(), p.title.Color)
	p.RenderTexture(p.textShader, p.prompt.ID, p.prompt.Position(), p.prompt.RawDimensions(), p.prompt.Color)

	// No affirmations to display
	if len(p.affirmations) == 0 {
		return nil
	}

	// Render highlight box behind selected item
	p.RenderSolidColorQuad(
		p.solidShader,
		p.affirmations[p.highlightIndex].Position(),
		p.affirmations[p.highlightIndex].RawDimensions(),
		colors.White,
	)

	// Render affirmation items with appropriate colors
	for i, item := range p.affirmations {
		if p.activeMap[i] {
			// Selected items are green
			item.Color = colors.Green
		} else if i == p.highlightIndex {
			// Highlighted item is black (shows on white background)
			item.Color = colors.Black
		} else {
			// Normal items are white
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

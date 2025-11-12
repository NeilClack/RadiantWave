package page

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/veandco/go-sdl2/sdl"
	"radiantwavetech.com/radiantwave/internal/colors"
	"radiantwavetech.com/radiantwave/internal/config"
	"radiantwavetech.com/radiantwave/internal/fontManager"
	"radiantwavetech.com/radiantwave/internal/logger"
	"radiantwavetech.com/radiantwave/internal/shaderManager"
)

var GitVersion = "dev"

type Welcome struct {
	Base
	title   StringItem
	version StringItem
	prompt  StringItem
	note    StringItem
}

// Init performs basic page initialization and creation of static assets.
func (p *Welcome) Init(app ApplicationInterface) error {
	var err error
	// Setup
	logger.LogInfoF("Initializing Welcome page")
	title := "Radiant Wave"
	version := GitVersion
	prompt := "Press F2 to Start, or F3 for Settings"
	note := "You can always press F1 to return here"

	if err := p.Base.Init(app); err != nil {
		return fmt.Errorf("failed base page initialization: %w", err)
	}

	fm := fontManager.Get()
	font, ok := fm.GetFont("Roboto-Regular")
	if !ok {
		return fmt.Errorf("unable to fetch font")
	}

	// Create Textures //
	p.title, err = NewStringItem(title, font, colors.White)
	if err != nil {
		logger.LogErrorF("unable to create title string texture: %v", err)
	}
	logger.LogInfoF("Created title texture")

	// Version
	p.version, err = NewStringItem(version, font, colors.White)
	if err != nil {
		logger.LogErrorF("unable to create version string texture: %v", err)
	}

	// Prompt
	p.prompt, err = NewStringItem(prompt, font, colors.White)
	if err != nil {
		logger.LogErrorF("unable to create prompt string texture: %v", err)
	}

	p.note, err = NewStringItem(note, font, colors.White)
	if err != nil {
		logger.LogErrorF("unable to create note string texture: %v", err)
	}
	fontSize := int32(32)
	scale := float32(fontSize) / float32(config.Get().BaseFontSize)
	p.note.Scale((scale * .75))

	return nil
}

// HandleEvent processes SDL events and handles them appropriately for the SerialNumberPage.
func (p *Welcome) HandleEvent(event *sdl.Event) error { return nil }

// Update updates animations and changing textures and other changes using `dt` which represents a time delta in float32.
func (p *Welcome) Update(dt float32) error { return nil }

func (p *Welcome) Render() error {
	cfg := config.Get()
	sm := shaderManager.Get()

	verticalMargin := int32(20)

	// 1) Scales
	titleFontSize := int32(64)
	bodyFontSize := int32(32)

	titleScale := float32(titleFontSize) / float32(cfg.BaseFontSize)
	bodyScale := float32(bodyFontSize) / float32(cfg.BaseFontSize)

	// 2) Dimensions (scaled)
	titleW := float32(p.title.W) * titleScale
	titleH := float32(p.title.H) * titleScale
	titleDim := mgl32.Vec2{titleW, titleH}

	versionW := float32(p.version.W) * bodyScale
	versionH := float32(p.version.H) * bodyScale
	versionDim := mgl32.Vec2{versionW, versionH}

	promptW := float32(p.prompt.W) * bodyScale
	promptH := float32(p.prompt.H) * bodyScale
	promptDim := mgl32.Vec2{promptW, promptH}

	// p.note already scaled in Init via p.note.Scale(scale*0.75)
	noteDim := p.note.RawDimensions()
	noteW, noteH := noteDim[0], noteDim[1]

	// 3) Total height of the vertical stack
	totalH := int32(titleH) + verticalMargin +
		int32(versionH) + verticalMargin +
		int32(promptH) + verticalMargin +
		int32(noteH)

	// 4) Y-up layout: start from the TOP of the centered stack and step DOWN
	top := p.Base.ScreenCenterY + totalH/2

	titleY := top - int32(titleH) // Title sits at the top
	versionY := titleY - verticalMargin - int32(versionH)
	p.prompt.Y = versionY - verticalMargin - int32(promptH)
	p.note.Y = p.prompt.Y - verticalMargin - int32(noteH)

	// 5) Center X for each item
	titleX := p.Base.ScreenCenterX - int32(titleW/2)
	versionX := p.Base.ScreenCenterX - int32(versionW/2)
	p.prompt.X = p.Base.ScreenCenterX - int32(promptW/2)
	p.note.X = p.Base.ScreenCenterX - int32(noteW/2)

	titlePos := mgl32.Vec2{float32(titleX), float32(titleY)}
	versionPos := mgl32.Vec2{float32(versionX), float32(versionY)}

	// 6) Render
	textShader, ok := sm.Get("text")
	if !ok {
		return fmt.Errorf("failed to fetch text shader")
	}
	color := colors.White

	p.Base.RenderTexture(textShader, p.title.ID, titlePos, titleDim, color)
	p.Base.RenderTexture(textShader, p.version.ID, versionPos, versionDim, color)
	p.Base.RenderTexture(textShader, p.prompt.ID, p.prompt.Position(), promptDim, color)
	p.Base.RenderTexture(textShader, p.note.ID, p.note.Position(), p.note.RawDimensions(), color)

	return nil
}

// Destroy cleans up page-specific resources.
func (p *Welcome) Destroy() error {
	logger.LogInfoF("Destroying LicenseKeyPage...")
	gl.DeleteTextures(1, &p.title.ID)
	gl.DeleteTextures(1, &p.version.ID)
	gl.DeleteTextures(1, &p.prompt.ID)
	return p.Base.Destroy()
}

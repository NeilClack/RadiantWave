package page

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/veandco/go-sdl2/sdl"
	"radiantwavetech.com/radiantwave/internal/config"
	"radiantwavetech.com/radiantwave/internal/fontManager"
	"radiantwavetech.com/radiantwave/internal/logger"
	"radiantwavetech.com/radiantwave/internal/shaderManager"
)

type EmailAddressPage struct {
	Base
	logger *logger.Logger
	config *config.Config

	title              string
	titleTextureID     uint32
	titleTextureWidth  int32
	titleTextureHeight int32

	inputText              string
	inputTextTextureID     uint32
	inputTextTextureWidth  int32
	inputTextTextureHeight int32
}

// Init performs basic page initialization and creation of static assets.
func (p *EmailAddressPage) Init(app ApplicationInterface) error {
	p.config = config.Get()
	p.logger = logger.Get()
	p.logger.Info("Initializing EmailAddressPage")

	if err := p.Base.Init(app); err != nil {
		return fmt.Errorf("failed base page initialization: %w", err)
	}
	p.title = "Your Email Address"

	fm := fontManager.Get()
	textShader, ok := shaderManager.Get().Get("text")
	if !ok {
		return fmt.Errorf("unable to fetch text shader")
	}

	titleTextureID, titleWidth, titleHeight, err := fm.CreateStringTexture(p.config.ApplicationFont, p.title, textShader)
	if err != nil {
		return fmt.Errorf("failed to create title texture: %w", err)
	}
	if titleTextureID == 0 {
		return fmt.Errorf("CreateStringTexture returned a nil texture ID")
	}

	p.logger.Infof("Created title texture %d", titleTextureID)
	p.titleTextureID = titleTextureID
	p.titleTextureWidth = titleWidth
	p.titleTextureHeight = titleHeight

	emailAddress := config.Get().EmailAddress // Renamed from SerialNumber
	if emailAddress != "" {
		p.inputText = emailAddress
	} else {
		p.inputText = ""
	}

	sdl.StartTextInput()

	return nil
}

// HandleEvent processes SDL events and handles them appropriately for the EmailAddressPage.
func (p *EmailAddressPage) HandleEvent(event *sdl.Event) error {
	actualEvent := *event
	if !sdl.IsTextInputActive() {
		sdl.StartTextInput()
	}

	switch e := actualEvent.(type) {
	case *sdl.TextInputEvent:
		p.inputText += e.GetText()
		p.logger.Infof("Input text updated: %s", p.inputText)
	case *sdl.KeyboardEvent:
		if e.Type == sdl.KEYDOWN {
			switch e.Keysym.Sym {
			case sdl.K_RETURN:
				sdl.StopTextInput()
				config.Get().EmailAddress = p.inputText // Renamed from SerialNumber
				config.Get().Update()
				logger.LogInfoF("Email Submitted: %s", p.inputText)
				if config.Get().EmailAddress != "" {
					p.Base.App.PopPage() // Remove this page from the pageStack
				}
			case sdl.K_BACKSPACE:
				if len(p.inputText) > 0 {
					p.inputText = p.inputText[:len(p.inputText)-1]
					p.logger.Infof("Input text updated: %s", p.inputText)
				}
			}
		}
	}
	return nil
}

// Update updates animations and changing textures and other changes using `dt` which represents a time delta in float32.
func (p *EmailAddressPage) Update(dt float32) error {
	if len(p.inputText) > 0 {
		fm := fontManager.Get()
		sm := shaderManager.Get()
		shader, ok := sm.Get("text")
		if !ok {
			logger.LogErrorF("unable to fetch shader")
			return fmt.Errorf("unable to fetch shader")
		}
		// Create the new inputText texture
		inputTextTextureID, inputTextTextureWidth, inputTextTextureHeight, err := fm.CreateStringTexture(
			config.Get().ApplicationFont,
			p.inputText,
			shader,
		)
		if err != nil {
			return fmt.Errorf("unable to create new input text texture")
		}

		p.inputTextTextureID = inputTextTextureID
		p.inputTextTextureWidth = inputTextTextureWidth
		p.inputTextTextureHeight = inputTextTextureHeight

	}
	return nil
}

// Render performs page drawing logic.
func (p *EmailAddressPage) Render() error {
	config := config.Get()
	sm := shaderManager.Get()

	// 1. Calculate Scale
	titleFontSize := int32(64)
	titleScale := float32(titleFontSize) / float32(config.BaseFontSize)
	padding := int32(4)
	verticalMargin := int32(20)

	// 2. Define dimensions
	// TODO: Increase the size of the input box
	targetFontSize := int32(36)
	scale := float32(targetFontSize) / float32(config.BaseFontSize)
	border := int32(3)
	boxH := targetFontSize + (padding * 2)
	boxW := int32(600) + (padding * 2)
	boxDimensions := mgl32.Vec2{float32(boxW), float32(boxH)}
	borderBoxH := boxH + (border * 2)
	borderBoxW := boxW + (border * 2)
	borderBoxDimensions := mgl32.Vec2{float32(borderBoxW), float32(borderBoxH)}
	titleH := float32(p.titleTextureHeight) * titleScale
	titleW := float32(p.titleTextureWidth) * titleScale
	titleDimensions := mgl32.Vec2{titleW, titleH}
	inputTextH := float32(p.inputTextTextureHeight) * scale
	inputTextW := float32(p.inputTextTextureWidth) * scale
	inputTextDimensions := mgl32.Vec2{inputTextW, inputTextH}

	// 3. Define positions
	borderBoxY := p.Base.ScreenCenterY - (borderBoxH / 2)
	borderBoxX := p.Base.ScreenCenterX - (borderBoxW / 2)
	borderBoxPosition := mgl32.Vec2{float32(borderBoxX), float32(borderBoxY)}
	boxY := p.Base.ScreenCenterY - (boxH / 2)
	boxX := p.Base.ScreenCenterX - (boxW / 2)
	boxPosition := mgl32.Vec2{float32(boxX), float32(boxY)}
	titleY := borderBoxY + borderBoxH + verticalMargin
	titleX := p.Base.ScreenCenterX - int32(titleW/2)
	titlePosition := mgl32.Vec2{float32(titleX), float32(titleY)}
	inputTextY := boxY + padding
	inputTextX := boxX + padding
	inputTextPosition := mgl32.Vec2{float32(inputTextX), float32(inputTextY)}

	// 4. Define Colors
	white := sdl.Color{R: 255, G: 255, B: 255, A: 255}
	grey := sdl.Color{R: 120, G: 120, B: 120, A: 255}
	darkGrey := sdl.Color{R: 30, G: 30, B: 30, A: 255}

	// 5. Get Shaders
	textShader, _ := sm.Get("text")
	solidShader, _ := sm.Get("solid_color")

	// 6. Draw everything
	p.Base.RenderSolidColorQuad(solidShader, borderBoxPosition, borderBoxDimensions, grey)
	p.Base.RenderSolidColorQuad(solidShader, boxPosition, boxDimensions, darkGrey)
	p.Base.RenderTexture(textShader, p.titleTextureID, titlePosition, titleDimensions, white)
	if len(p.inputText) > 0 {
		logger.LogInfoF("inputTextTextureID: %d", p.inputTextTextureID)
		p.Base.RenderTexture(textShader, p.inputTextTextureID, inputTextPosition, inputTextDimensions, white)
	}
	return nil
}

// Destroy cleans up page-specific resources.
func (p *EmailAddressPage) Destroy() error {
	p.logger.Info("Destroying EmailAddressPage...")
	// titleTextureID is now managed by Base.Destroy, so we remove its explicit deletion here.
	// gl.DeleteTextures(1, &p.titleTextureID) // Removed as Base.Destroy handles it

	// inputTextTextureID is still managed explicitly due to its dynamic nature
	if p.inputTextTextureID != 0 {
		gl.DeleteTextures(1, &p.inputTextTextureID)
		p.inputTextTextureID = 0 // Reset ID after deletion
	}
	sdl.StopTextInput()     // Stop text input when the page is destroyed
	return p.Base.Destroy() // Call the Base page's Destroy method to clean up common resources and managedTextureIDs
}

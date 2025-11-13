package page

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/veandco/go-sdl2/sdl"
	"radiantwavetech.com/radiantwave/internal/colors"
	"radiantwavetech.com/radiantwave/internal/db"
	"radiantwavetech.com/radiantwave/internal/fontManager"
	"radiantwavetech.com/radiantwave/internal/logger"
	"radiantwavetech.com/radiantwave/internal/shaderManager"
)

type LicenseKeyPage struct {
	Base

	title              string
	titleTextureID     uint32
	titleTextureWidth  int32
	titleTextureHeight int32

	currentLicenseDisplayTextureID     uint32
	currentLicenseDisplayTextureWidth  int32
	currentLicenseDisplayTextureHeight int32

	inputText              string
	inputTextTextureID     uint32
	inputTextTextureWidth  int32
	inputTextTextureHeight int32
}

// Init performs basic page initialization and creation of static assets.
func (p *LicenseKeyPage) Init(app ApplicationInterface) error {
	logger.InfoF("Initializing LicenseKeyPage")

	if err := p.Base.Init(app); err != nil {
		return fmt.Errorf("failed base page initialization: %w", err)
	}

	p.title = "License Key"

	// Get application font from database
	applicationFont, err := db.GetConfigValue("application_font")
	if err != nil {
		return fmt.Errorf("retrieving application_font from db: %w", err)
	}

	fm := fontManager.Get()
	textShader, ok := shaderManager.Get().Get("text")
	if !ok {
		return fmt.Errorf("unable to fetch text shader")
	}

	// Create title texture
	titleTextureID, titleWidth, titleHeight, err := fm.CreateStringTexture(applicationFont, p.title, textShader)
	if err != nil {
		return fmt.Errorf("failed to create title texture: %w", err)
	}
	if titleTextureID == 0 {
		return fmt.Errorf("CreateStringTexture returned a nil texture ID")
	}

	logger.InfoF("Created title texture %d", titleTextureID)
	p.titleTextureID = titleTextureID
	p.titleTextureWidth = titleWidth
	p.titleTextureHeight = titleHeight

	// Load existing license key from database
	licenseKey, err := db.GetConfigValue("license_key")
	if err != nil {
		return fmt.Errorf("retrieving license_key from db: %w", err)
	}

	if licenseKey != "" {
		p.inputText = licenseKey

		// Create current license key display texture
		currentLicenseText := fmt.Sprintf("Current License Key: %s", licenseKey)
		currentLicenseTextureID, currentLicenseWidth, currentLicenseHeight, err := fm.CreateStringTexture(applicationFont, currentLicenseText, textShader)
		if err != nil {
			return fmt.Errorf("failed to create current license display texture: %w", err)
		}
		if currentLicenseTextureID == 0 {
			return fmt.Errorf("CreateStringTexture returned a nil texture ID for current license display")
		}

		logger.InfoF("Created current license display texture %d", currentLicenseTextureID)
		p.currentLicenseDisplayTextureID = currentLicenseTextureID
		p.currentLicenseDisplayTextureWidth = currentLicenseWidth
		p.currentLicenseDisplayTextureHeight = currentLicenseHeight
	} else {
		p.inputText = ""
	}

	sdl.StartTextInput()

	return nil
}

// formatLicenseKey formats the input string with dashes every 4 characters
// and limits to 16 alphanumeric characters (not counting dashes)
func formatLicenseKey(input string) string {
	// Remove all dashes and non-alphanumeric characters
	cleaned := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, input)

	// Limit to 16 characters
	if len(cleaned) > 16 {
		cleaned = cleaned[:16]
	}

	// Insert dashes every 4 characters
	var result strings.Builder
	for i, char := range cleaned {
		if i > 0 && i%4 == 0 {
			result.WriteRune('-')
		}
		result.WriteRune(char)
	}

	return result.String()
}

// HandleEvent processes SDL events and handles them appropriately for the LicenseKeyPage.
func (p *LicenseKeyPage) HandleEvent(event *sdl.Event) error {
	actualEvent := *event
	if !sdl.IsTextInputActive() {
		sdl.StartTextInput()
	}

	switch e := actualEvent.(type) {
	case *sdl.TextInputEvent:
		// Add the new character and reformat
		p.inputText += e.GetText()
		p.inputText = formatLicenseKey(p.inputText)
		logger.InfoF("Input text updated: %s", p.inputText)
	case *sdl.KeyboardEvent:
		if e.Type == sdl.KEYDOWN {
			switch e.Keysym.Sym {
			case sdl.K_RETURN:
				sdl.StopTextInput()

				// Save license key to database
				if err := db.SetConfigValue("license_key", p.inputText); err != nil {
					return fmt.Errorf("failed to save license key: %w", err)
				}

				logger.InfoF("LicenseKey Submitted: %s", p.inputText)
				p.Base.App.PopPage() // Pop this page from the pageStack
			case sdl.K_BACKSPACE:
				if len(p.inputText) > 0 {
					// Remove last character and reformat
					// Strip all dashes first to get raw characters
					cleaned := strings.ReplaceAll(p.inputText, "-", "")
					if len(cleaned) > 0 {
						cleaned = cleaned[:len(cleaned)-1]
					}
					p.inputText = formatLicenseKey(cleaned)
					logger.InfoF("Input text updated: %s", p.inputText)
				}
			}
		}
	}
	return nil
}

// Update updates animations and changing textures and other changes using `dt` which represents a time delta in float32.
func (p *LicenseKeyPage) Update(dt float32) error {
	if len(p.inputText) == 0 {
		return nil
	}

	// Get application font from database
	applicationFont, err := db.GetConfigValue("application_font")
	if err != nil {
		return fmt.Errorf("retrieving application_font from db: %w", err)
	}

	fm := fontManager.Get()
	sm := shaderManager.Get()
	shader, ok := sm.Get("text")
	if !ok {
		return fmt.Errorf("unable to fetch text shader")
	}

	// Create the new inputText texture
	inputTextTextureID, inputTextTextureWidth, inputTextTextureHeight, err := fm.CreateStringTexture(
		applicationFont,
		p.inputText,
		shader,
	)
	if err != nil {
		return fmt.Errorf("unable to create new input text texture: %w", err)
	}

	// Delete old texture if it exists
	if p.inputTextTextureID != 0 {
		gl.DeleteTextures(1, &p.inputTextTextureID)
	}

	p.inputTextTextureID = inputTextTextureID
	p.inputTextTextureWidth = inputTextTextureWidth
	p.inputTextTextureHeight = inputTextTextureHeight

	return nil
}

// Render performs page drawing logic.
func (p *LicenseKeyPage) Render() error {
	// Get base font size from database
	baseFontSizeStr, err := db.GetConfigValue("init_font_size")
	if err != nil {
		return fmt.Errorf("retrieving init_font_size from db: %w", err)
	}
	baseFontSize, err := strconv.Atoi(baseFontSizeStr)
	if err != nil {
		return fmt.Errorf("parsing init_font_size: %w", err)
	}

	sm := shaderManager.Get()

	// 1. Calculate Scale
	titleFontSize := int32(64)
	titleScale := float32(titleFontSize) / float32(baseFontSize)
	currentLicenseFontSize := int32(28)
	currentLicenseScale := float32(currentLicenseFontSize) / float32(baseFontSize)
	padding := int32(4)
	verticalMargin := int32(20)

	// 2. Define dimensions
	targetFontSize := int32(36)
	scale := float32(targetFontSize) / float32(baseFontSize)
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

	var currentLicenseDisplayH, currentLicenseDisplayW float32
	var currentLicenseDisplayDimensions mgl32.Vec2
	if p.currentLicenseDisplayTextureID != 0 {
		currentLicenseDisplayH = float32(p.currentLicenseDisplayTextureHeight) * currentLicenseScale
		currentLicenseDisplayW = float32(p.currentLicenseDisplayTextureWidth) * currentLicenseScale
		currentLicenseDisplayDimensions = mgl32.Vec2{currentLicenseDisplayW, currentLicenseDisplayH}
	}

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

	var currentLicenseDisplayPosition mgl32.Vec2
	if p.currentLicenseDisplayTextureID != 0 {
		currentLicenseDisplayY := borderBoxY - int32(currentLicenseDisplayH) - verticalMargin
		currentLicenseDisplayX := p.Base.ScreenCenterX - int32(currentLicenseDisplayW/2)
		currentLicenseDisplayPosition = mgl32.Vec2{float32(currentLicenseDisplayX), float32(currentLicenseDisplayY)}
	}

	// 4. Get Shaders
	textShader, _ := sm.Get("text")
	solidShader, _ := sm.Get("solid_color")

	// 5. Draw everything
	p.Base.RenderSolidColorQuad(solidShader, borderBoxPosition, borderBoxDimensions, colors.Gray)
	p.Base.RenderSolidColorQuad(solidShader, boxPosition, boxDimensions, colors.DarkGray)
	p.Base.RenderTexture(textShader, p.titleTextureID, titlePosition, titleDimensions, colors.White)

	// Draw current license display if it exists
	if p.currentLicenseDisplayTextureID != 0 {
		p.Base.RenderTexture(textShader, p.currentLicenseDisplayTextureID, currentLicenseDisplayPosition, currentLicenseDisplayDimensions, colors.White)
	}

	if len(p.inputText) > 0 {
		p.Base.RenderTexture(textShader, p.inputTextTextureID, inputTextPosition, inputTextDimensions, colors.White)
	}
	return nil
}

// Destroy cleans up page-specific resources.
func (p *LicenseKeyPage) Destroy() error {
	logger.InfoF("Destroying LicenseKeyPage...")

	gl.DeleteTextures(1, &p.titleTextureID)

	// Delete current license display texture if it exists
	if p.currentLicenseDisplayTextureID != 0 {
		gl.DeleteTextures(1, &p.currentLicenseDisplayTextureID)
		p.currentLicenseDisplayTextureID = 0
	}

	// Delete input text texture if it exists
	if p.inputTextTextureID != 0 {
		gl.DeleteTextures(1, &p.inputTextTextureID)
		p.inputTextTextureID = 0
	}

	sdl.StopTextInput()
	return p.Base.Destroy()
}

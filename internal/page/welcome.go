package page

import (
	"fmt"
	"os"
	"path/filepath"
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

var GitVersion = "dev"

type Welcome struct {
	Base
	title              StringItem
	version            StringItem
	prompt             StringItem
	note               StringItem
	licenseInfoHeading StringItem
	emailDisplay       StringItem
	licenseKeyDisplay  StringItem
	displayStatus      StringItem
	hasLicenseInfo     bool
}

// countConnectedDisplays returns the number of connected displays
func countConnectedDisplays() int {
	matches, err := filepath.Glob("/sys/class/drm/card*/status")
	if err != nil {
		logger.ErrorF("Error globbing display devices: %v", err)
		return 0
	}

	connected := 0
	for _, path := range matches {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		status := strings.TrimSpace(string(content))
		if status == "connected" {
			connected++
		}
	}

	return connected
}

// Init performs basic page initialization and creation of static assets.
func (p *Welcome) Init(app ApplicationInterface) error {
	logger.InfoF("Initializing Welcome page")

	title := "Radiant Wave"
	version := GitVersion
	prompt := "Press F2 to Start, or F3 for Settings"
	note := "You can always press F1 to return here"

	if err := p.Base.Init(app); err != nil {
		return fmt.Errorf("failed base page initialization: %w", err)
	}

	// Get font
	fm := fontManager.Get()
	font, ok := fm.GetFont("Roboto-Regular")
	if !ok {
		return fmt.Errorf("unable to fetch font")
	}

	// Create title texture
	var err error
	p.title, err = NewStringItem(title, font, colors.White)
	if err != nil {
		return fmt.Errorf("unable to create title string texture: %w", err)
	}
	logger.InfoF("Created title texture")

	// Create version texture
	p.version, err = NewStringItem(version, font, colors.White)
	if err != nil {
		return fmt.Errorf("unable to create version string texture: %w", err)
	}

	// Create prompt texture
	p.prompt, err = NewStringItem(prompt, font, colors.White)
	if err != nil {
		return fmt.Errorf("unable to create prompt string texture: %w", err)
	}

	// Create note texture
	p.note, err = NewStringItem(note, font, colors.White)
	if err != nil {
		return fmt.Errorf("unable to create note string texture: %w", err)
	}

	// Count connected displays and create status texture
	displayCount := countConnectedDisplays()
	displayStatusText := fmt.Sprintf("Connected Displays: %d", displayCount)
	p.displayStatus, err = NewStringItem(displayStatusText, font, colors.White)
	if err != nil {
		return fmt.Errorf("unable to create display status texture: %w", err)
	}

	// Load license information from database
	emailAddress, err := db.GetConfigValue("email_address")
	if err != nil {
		return fmt.Errorf("retrieving email_address from db: %w", err)
	}

	licenseKey, err := db.GetConfigValue("license_key")
	if err != nil {
		return fmt.Errorf("retrieving license_key from db: %w", err)
	}

	// Create license info textures if data exists
	if emailAddress != "" || licenseKey != "" {
		p.hasLicenseInfo = true

		// Create heading
		p.licenseInfoHeading, err = NewStringItem("License Info", font, colors.White)
		if err != nil {
			return fmt.Errorf("unable to create license info heading texture: %w", err)
		}

		// Create email display if exists
		if emailAddress != "" {
			emailText := fmt.Sprintf("Email: %s", emailAddress)
			p.emailDisplay, err = NewStringItem(emailText, font, colors.White)
			if err != nil {
				return fmt.Errorf("unable to create email display texture: %w", err)
			}
		}

		// Create license key display if exists
		if licenseKey != "" {
			licenseKeyText := fmt.Sprintf("License Key: %s", licenseKey)
			p.licenseKeyDisplay, err = NewStringItem(licenseKeyText, font, colors.White)
			if err != nil {
				return fmt.Errorf("unable to create license key display texture: %w", err)
			}
		}
	}

	// Get base font size from database
	baseFontSizeStr, err := db.GetConfigValue("init_font_size")
	if err != nil {
		return fmt.Errorf("retrieving init_font_size from db: %w", err)
	}
	baseFontSize, err := strconv.Atoi(baseFontSizeStr)
	if err != nil {
		return fmt.Errorf("parsing init_font_size: %w", err)
	}

	// Scale the note text
	fontSize := int32(32)
	scale := float32(fontSize) / float32(baseFontSize)
	p.note.Scale(scale * 0.75)

	// Scale display status text
	displayStatusScale := float32(24) / float32(baseFontSize)
	p.displayStatus.Scale(displayStatusScale)

	// Scale license info texts
	if p.hasLicenseInfo {
		licenseInfoScale := float32(28) / float32(baseFontSize)
		p.licenseInfoHeading.Scale(licenseInfoScale)
		if p.emailDisplay.ID != 0 {
			p.emailDisplay.Scale(licenseInfoScale * 0.85)
		}
		if p.licenseKeyDisplay.ID != 0 {
			p.licenseKeyDisplay.Scale(licenseInfoScale * 0.85)
		}
	}

	return nil
}

// HandleEvent processes SDL events and handles them appropriately for the Welcome page.
func (p *Welcome) HandleEvent(event *sdl.Event) error {
	return nil
}

// Update updates animations and changing textures and other changes using `dt` which represents a time delta in float32.
func (p *Welcome) Update(dt float32) error {
	return nil
}

func (p *Welcome) Render() error {
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
	verticalMargin := int32(20)

	// 1) Calculate scales
	titleFontSize := int32(64)
	bodyFontSize := int32(32)

	titleScale := float32(titleFontSize) / float32(baseFontSize)
	bodyScale := float32(bodyFontSize) / float32(baseFontSize)

	// 2) Calculate dimensions (scaled)
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

	// 3) Calculate total height of the vertical stack
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

	// 6) Position display status in bottom left corner
	p.displayStatus.X = 15
	p.displayStatus.Y = 15

	// 7) Position license info at bottom center
	if p.hasLicenseInfo {
		licenseInfoMargin := int32(10)
		bottomMargin := int32(20)

		// License info items are already scaled in Init
		licenseHeadingDim := p.licenseInfoHeading.RawDimensions()
		licenseHeadingW, licenseHeadingH := licenseHeadingDim[0], licenseHeadingDim[1]

		var emailDisplayH, licenseKeyDisplayH float32
		if p.emailDisplay.ID != 0 {
			emailDisplayDim := p.emailDisplay.RawDimensions()
			emailDisplayH = emailDisplayDim[1]
		}
		if p.licenseKeyDisplay.ID != 0 {
			licenseKeyDisplayDim := p.licenseKeyDisplay.RawDimensions()
			licenseKeyDisplayH = licenseKeyDisplayDim[1]
		}

		// Calculate total height of license info block
		licenseInfoTotalH := int32(licenseHeadingH)
		if p.emailDisplay.ID != 0 {
			licenseInfoTotalH += int32(emailDisplayH) + licenseInfoMargin
		}
		if p.licenseKeyDisplay.ID != 0 {
			licenseInfoTotalH += int32(licenseKeyDisplayH) + licenseInfoMargin
		}

		// Start from bottom and work up
		currentY := bottomMargin

		// License key (bottom-most if it exists)
		if p.licenseKeyDisplay.ID != 0 {
			licenseKeyDisplayDim := p.licenseKeyDisplay.RawDimensions()
			licenseKeyDisplayW := licenseKeyDisplayDim[0]
			p.licenseKeyDisplay.X = p.Base.ScreenCenterX - int32(licenseKeyDisplayW/2)
			p.licenseKeyDisplay.Y = currentY
			currentY += int32(licenseKeyDisplayH) + licenseInfoMargin
		}

		// Email
		if p.emailDisplay.ID != 0 {
			emailDisplayDim := p.emailDisplay.RawDimensions()
			emailDisplayW := emailDisplayDim[0]
			p.emailDisplay.X = p.Base.ScreenCenterX - int32(emailDisplayW/2)
			p.emailDisplay.Y = currentY
			currentY += int32(emailDisplayH) + licenseInfoMargin
		}

		// Heading (top-most)
		p.licenseInfoHeading.X = p.Base.ScreenCenterX - int32(licenseHeadingW/2)
		p.licenseInfoHeading.Y = currentY
	}

	// 8) Get shader and render
	textShader, ok := sm.Get("text")
	if !ok {
		return fmt.Errorf("failed to fetch text shader")
	}
	color := colors.White

	p.Base.RenderTexture(textShader, p.title.ID, titlePos, titleDim, color)
	p.Base.RenderTexture(textShader, p.version.ID, versionPos, versionDim, color)
	p.Base.RenderTexture(textShader, p.prompt.ID, p.prompt.Position(), promptDim, color)
	p.Base.RenderTexture(textShader, p.note.ID, p.note.Position(), p.note.RawDimensions(), color)

	// Render display status
	p.Base.RenderTexture(textShader, p.displayStatus.ID, p.displayStatus.Position(), p.displayStatus.RawDimensions(), color)

	// Render license info if it exists
	if p.hasLicenseInfo {
		p.Base.RenderTexture(textShader, p.licenseInfoHeading.ID, p.licenseInfoHeading.Position(), p.licenseInfoHeading.RawDimensions(), color)
		if p.emailDisplay.ID != 0 {
			p.Base.RenderTexture(textShader, p.emailDisplay.ID, p.emailDisplay.Position(), p.emailDisplay.RawDimensions(), color)
		}
		if p.licenseKeyDisplay.ID != 0 {
			p.Base.RenderTexture(textShader, p.licenseKeyDisplay.ID, p.licenseKeyDisplay.Position(), p.licenseKeyDisplay.RawDimensions(), color)
		}
	}

	return nil
}

// Destroy cleans up page-specific resources.
func (p *Welcome) Destroy() error {
	logger.InfoF("Destroying Welcome page...")
	gl.DeleteTextures(1, &p.title.ID)
	gl.DeleteTextures(1, &p.version.ID)
	gl.DeleteTextures(1, &p.prompt.ID)
	gl.DeleteTextures(1, &p.note.ID)
	gl.DeleteTextures(1, &p.displayStatus.ID)

	if p.hasLicenseInfo {
		gl.DeleteTextures(1, &p.licenseInfoHeading.ID)
		if p.emailDisplay.ID != 0 {
			gl.DeleteTextures(1, &p.emailDisplay.ID)
		}
		if p.licenseKeyDisplay.ID != 0 {
			gl.DeleteTextures(1, &p.licenseKeyDisplay.ID)
		}
	}

	return p.Base.Destroy()
}

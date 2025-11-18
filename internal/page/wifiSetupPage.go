package page

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/godbus/dbus/v5"
	"github.com/veandco/go-sdl2/sdl"
	"radiantwavetech.com/radiantwave/internal/colors"
	"radiantwavetech.com/radiantwave/internal/db"
	"radiantwavetech.com/radiantwave/internal/fontManager"
	"radiantwavetech.com/radiantwave/internal/logger"
	"radiantwavetech.com/radiantwave/internal/network"
	"radiantwavetech.com/radiantwave/internal/shaderManager"
)

// accessPointUIItem holds both the network data and its rendered texture.
type accessPointUIItem struct {
	AP   network.AccessPoint
	Item StringItem
}

// WiFiSetupPage manages the discovery and connection to Wi-Fi networks.
type WiFiSetupPage struct {
	Base
	networkManager  *network.Manager
	applicationFont *fontManager.FontEntry

	// Page-specific UI elements
	titleItem            StringItem
	scanningStatusItem   StringItem
	connectionStatusItem StringItem
	passwordPromptItem   StringItem
	passwordInputItem    StringItem
	passwordErrorItem    StringItem

	// Network list and selection
	accessPoints  []accessPointUIItem
	selectedIndex int
	scrollOffset  int

	// State management
	isScanning            bool
	isEnteringPassword    bool
	isConnecting          bool
	connectionError       string
	passwordInput         string
	selectedAP            *network.AccessPoint
	wifiDevice            *network.Device
	connectionStatus      *network.ConnectionStatus
	activeSSID            string // Tracks the CONFIRMED active network
	pendingSSID           string // Tracks the network we are TRYING to connect to
	connectionAttemptTime time.Time
	connectionTimeout     time.Duration

	// Async management
	resultsChan chan []network.AccessPoint
	statusChan  chan *network.ConnectionStatus
	errorChan   chan error
	stopScan    chan struct{}
}

// Init sets up the page, initializes the network manager, and starts a scan.
func (p *WiFiSetupPage) Init(app ApplicationInterface) error {
	logger.InfoF("Initializing WiFiSetupPage")

	if err := p.Base.Init(app); err != nil {
		return fmt.Errorf("failed base page initialization: %w", err)
	}

	// Get application font from database
	applicationFontName, err := db.GetConfigValue("application_font")
	if err != nil {
		return fmt.Errorf("retrieving application_font from db: %w", err)
	}

	fm := fontManager.Get()
	font, ok := fm.GetFont(applicationFontName)
	if !ok {
		return fmt.Errorf("could not find application font: %s", applicationFontName)
	}
	p.applicationFont = font

	// Initialize network manager
	nm, err := network.New()
	if err != nil {
		return fmt.Errorf("failed to initialize network manager: %w", err)
	}
	p.networkManager = nm

	// Find Wi-Fi device
	if err := p.networkManager.FindDevices(); err != nil {
		logger.WarningF("Could not scan for network devices: %v", err)
	}
	for _, dev := range p.networkManager.Devices {
		if dev.Type == network.TypeWifi {
			d := dev
			p.wifiDevice = &d
			break
		}
	}

	// Create UI elements
	white := sdl.Color{R: 255, G: 255, B: 255, A: 255}
	p.titleItem, err = NewStringItem("Wi-Fi Setup", p.applicationFont, white)
	if err != nil {
		return fmt.Errorf("failed to create title item: %w", err)
	}
	p.scanningStatusItem, err = NewStringItem("Scanning for networks...", p.applicationFont, white)
	if err != nil {
		return fmt.Errorf("failed to create scanning status item: %w", err)
	}

	// Initialize channels and timeout
	p.resultsChan = make(chan []network.AccessPoint)
	p.statusChan = make(chan *network.ConnectionStatus, 1)
	p.errorChan = make(chan error)
	p.stopScan = make(chan struct{})
	p.connectionTimeout = 15 * time.Second

	// Start scanning and status monitoring
	p.startScan()
	go p.checkConnectionStatusLoop()

	return nil
}

// checkConnectionStatusLoop periodically checks the network status.
func (p *WiFiSetupPage) checkConnectionStatusLoop() {
	ticker := time.NewTicker(2 * time.Second) // Check more frequently for better UX
	defer ticker.Stop()

	status, _ := p.networkManager.CheckInternetConnection()
	p.statusChan <- status

	for {
		select {
		case <-ticker.C:
			status, _ := p.networkManager.CheckInternetConnection()
			p.statusChan <- status
		case <-p.stopScan:
			return
		}
	}
}

// startScan triggers a new asynchronous network scan.
func (p *WiFiSetupPage) startScan() {
	p.isScanning = true
	p.selectedIndex = 0
	p.scrollOffset = 0
	p.accessPoints = nil

	if p.wifiDevice != nil {
		go p.networkManager.StartScan(p.resultsChan, p.errorChan, p.stopScan)
	} else {
		p.isScanning = false
	}
}

// HandleEvent processes user input.
func (p *WiFiSetupPage) HandleEvent(event *sdl.Event) error {
	switch e := (*event).(type) {
	case *sdl.TextInputEvent:
		if p.isEnteringPassword {
			p.passwordInput += e.GetText()
			p.updatePasswordTexture()
		}
	case *sdl.KeyboardEvent:
		if e.Type != sdl.KEYDOWN {
			return nil
		}
		if p.isEnteringPassword {
			p.handlePasswordInput(e.Keysym.Sym)
		} else {
			p.handleNetworkSelection(e.Keysym.Sym)
		}
	}
	return nil
}

// handlePasswordInput manages keyboard events during password entry.
func (p *WiFiSetupPage) handlePasswordInput(sym sdl.Keycode) {
	switch sym {
	case sdl.K_RETURN:
		logger.InfoF("Attempting to connect to %s", p.selectedAP.SSID)
		sdl.StopTextInput()
		p.pendingSSID = p.selectedAP.SSID
		p.isConnecting = true
		p.connectionError = ""
		p.connectionAttemptTime = time.Now()
		err := p.networkManager.Connect(*p.wifiDevice, *p.selectedAP, p.passwordInput)
		if err != nil {
			logger.ErrorF("Failed to send connect command: %v", err)
			p.isConnecting = false
			p.pendingSSID = ""
			p.connectionError = "Failed to initiate connection"
		}
		p.isEnteringPassword = false
	case sdl.K_BACKSPACE:
		if len(p.passwordInput) > 0 {
			p.passwordInput = p.passwordInput[:len(p.passwordInput)-1]
			p.updatePasswordTexture()
		}
	case sdl.K_ESCAPE:
		p.isEnteringPassword = false
		p.pendingSSID = ""
		p.connectionError = ""
		sdl.StopTextInput()
	}
}

// handleNetworkSelection manages keyboard events for the network list.
func (p *WiFiSetupPage) handleNetworkSelection(sym sdl.Keycode) {
	switch sym {
	case sdl.K_UP:
		if p.selectedIndex > 0 {
			p.selectedIndex--
		}
	case sdl.K_DOWN:
		if p.selectedIndex < len(p.accessPoints)-1 {
			p.selectedIndex++
		}
	case sdl.K_RETURN:
		if len(p.accessPoints) > 0 && p.selectedIndex < len(p.accessPoints) {
			p.selectedAP = &p.accessPoints[p.selectedIndex].AP
			p.pendingSSID = p.selectedAP.SSID
			p.connectionError = ""
			if p.selectedAP.IsProtected {
				p.isEnteringPassword = true
				p.passwordInput = ""
				p.updatePasswordTexture()
				p.updatePasswordPromptTexture()
				sdl.StartTextInput()
			} else {
				logger.InfoF("Connecting to open network: %s", p.selectedAP.SSID)
				p.isConnecting = true
				p.connectionAttemptTime = time.Now()
				err := p.networkManager.Connect(*p.wifiDevice, *p.selectedAP, "")
				if err != nil {
					logger.ErrorF("Failed to send connect command: %v", err)
					p.isConnecting = false
					p.pendingSSID = ""
					p.connectionError = "Failed to initiate connection"
				}
			}
		}
	case sdl.K_r:
		if !p.isScanning {
			p.startScan()
		}
	}
}

// Update checks for async results and updates UI state.
func (p *WiFiSetupPage) Update(dt float32) error {
	select {
	case results := <-p.resultsChan:
		p.isScanning = false
		p.buildNetworkUIList(results)
	case status := <-p.statusChan:
		p.connectionStatus = status
		p.handleConnectionStatusUpdate(status)
		p.updateConnectionStatusItem()
	case err := <-p.errorChan:
		p.isScanning = false
		logger.ErrorF("Network scan error: %v", err)
	default:
	}

	// Check for connection timeout
	if p.isConnecting && time.Since(p.connectionAttemptTime) > p.connectionTimeout {
		logger.WarningF("Connection to %s timed out", p.pendingSSID)
		p.isConnecting = false
		p.connectionError = "Connection timed out - Incorrect password?"
		if p.selectedAP != nil && p.selectedAP.IsProtected {
			p.isEnteringPassword = true
			p.passwordInput = ""
			p.updatePasswordTexture()
			p.updatePasswordPromptTexture()
			p.updatePasswordErrorTexture()
			sdl.StartTextInput()
		}
	}

	return nil
}

// handleConnectionStatusUpdate checks if connection succeeded or failed
func (p *WiFiSetupPage) handleConnectionStatusUpdate(status *network.ConnectionStatus) {
	if !p.isConnecting {
		return
	}

	// Check if we successfully connected to our pending SSID
	switch status.StatusCode {
	case network.StatusWifiConnectedInternetUp:
		activeSSID := p.getActiveSSID()
		if activeSSID == p.pendingSSID {
			logger.InfoF("Successfully connected to %s", p.pendingSSID)
			p.isConnecting = false
			p.pendingSSID = ""
			p.connectionError = ""
			p.refreshNetworkColors()
		}
	case network.StatusWifiAvailableNotConnected:
		// Connection failed - we're trying to connect but now show as not connected
		if time.Since(p.connectionAttemptTime) > 5*time.Second {
			logger.WarningF("Connection to %s failed", p.pendingSSID)
			p.isConnecting = false
			if p.selectedAP != nil && p.selectedAP.IsProtected {
				p.connectionError = "Connection failed - Incorrect password?"
				p.isEnteringPassword = true
				p.passwordInput = ""
				p.updatePasswordTexture()
				p.updatePasswordPromptTexture()
				p.updatePasswordErrorTexture()
				sdl.StartTextInput()
			} else {
				p.connectionError = "Connection failed"
				p.pendingSSID = ""
			}
		}
	}
}

// getActiveSSID queries NetworkManager for the currently active WiFi SSID
func (p *WiFiSetupPage) getActiveSSID() string {
	if p.networkManager == nil || p.wifiDevice == nil {
		return ""
	}

	nmObj := p.networkManager.Conn().Object("org.freedesktop.NetworkManager", "/org/freedesktop/NetworkManager")
	prop, err := nmObj.GetProperty("org.freedesktop.NetworkManager.ActiveConnections")
	if err != nil {
		return ""
	}

	if paths, ok := prop.Value().([]dbus.ObjectPath); ok {
		for _, path := range paths {
			acObj := p.networkManager.Conn().Object("org.freedesktop.NetworkManager", path)
			devProp, err := acObj.GetProperty("org.freedesktop.NetworkManager.Connection.Active.Devices")
			if err == nil {
				if devPaths, ok := devProp.Value().([]dbus.ObjectPath); ok {
					for _, devPath := range devPaths {
						if string(devPath) == string(p.wifiDevice.Path) {
							idProp, err := acObj.GetProperty("org.freedesktop.NetworkManager.Connection.Active.Id")
							if err == nil {
								if id, ok := idProp.Value().(string); ok {
									return id
								}
							}
						}
					}
				}
			}
		}
	}
	return ""
}

// refreshNetworkColors updates the colors of network items based on active status
func (p *WiFiSetupPage) refreshNetworkColors() {
	activeSSID := p.getActiveSSID()
	p.activeSSID = activeSSID

	for i := range p.accessPoints {
		if p.accessPoints[i].AP.SSID == activeSSID {
			p.accessPoints[i].Item.Color = colors.LawnGreen
		} else {
			p.accessPoints[i].Item.Color = colors.White
		}
	}
}

// updateConnectionStatusItem creates the colored status message item.
func (p *WiFiSetupPage) updateConnectionStatusItem() {
	if p.connectionStatusItem.ID != 0 {
		gl.DeleteTextures(1, &p.connectionStatusItem.ID)
	}

	// Show connecting message if in progress
	if p.isConnecting {
		message := fmt.Sprintf("Connecting to %s...", p.pendingSSID)
		item, err := NewStringItem(message, p.applicationFont, colors.Yellow)
		if err == nil {
			p.connectionStatusItem = item
		}
		return
	}

	// Show connection error if present
	if p.connectionError != "" {
		item, err := NewStringItem(p.connectionError, p.applicationFont, colors.Red)
		if err == nil {
			p.connectionStatusItem = item
		}
		return
	}

	if p.connectionStatus == nil {
		return
	}

	switch p.connectionStatus.StatusCode {
	case network.StatusWifiConnectedInternetUp:
		message := fmt.Sprintf("Connected to %s - Internet accessible | F1 to exit", p.activeSSID)
		item, err := NewStringItem(message, p.applicationFont, colors.LawnGreen)
		if err == nil {
			p.connectionStatusItem = item
		}
		p.refreshNetworkColors()

	case network.StatusWifiAvailableNotConnected:
		message := "WiFi available - Not connected | Select network or press R to rescan"
		item, err := NewStringItem(message, p.applicationFont, colors.Yellow)
		if err == nil {
			p.connectionStatusItem = item
		}

	case network.StatusEthernetConnectedInternetUp:
		message := "Ethernet connected - Internet accessible | F1 to exit"
		item, err := NewStringItem(message, p.applicationFont, colors.LawnGreen)
		if err == nil {
			p.connectionStatusItem = item
		}

	case network.StatusEthernetNotConnected:
		message := "No network connection | Select WiFi network or press R to rescan"
		item, err := NewStringItem(message, p.applicationFont, colors.Red)
		if err == nil {
			p.connectionStatusItem = item
		}

	case network.StatusNoDevicesFound:
		message := "No network devices found"
		item, err := NewStringItem(message, p.applicationFont, colors.Red)
		if err == nil {
			p.connectionStatusItem = item
		}

	case network.StatusError:
		message := "Network status check failed"
		item, err := NewStringItem(message, p.applicationFont, colors.Red)
		if err == nil {
			p.connectionStatusItem = item
		}

	default:
		item, err := NewStringItem(p.connectionStatus.Message, p.applicationFont, colors.White)
		if err == nil {
			p.connectionStatusItem = item
		}
	}
}

// buildNetworkUIList creates the renderable textures for the list of discovered networks.
func (p *WiFiSetupPage) buildNetworkUIList(aps []network.AccessPoint) {
	// Get the currently active Wi-Fi SSID
	activeSSID := p.getActiveSSID()
	p.activeSSID = activeSSID

	// Set IsActive flag based on the actual active SSID
	for i := range aps {
		aps[i].IsActive = (aps[i].SSID == p.activeSSID)
	}

	p.accessPoints = make([]accessPointUIItem, len(aps))
	white := sdl.Color{R: 255, G: 255, B: 255, A: 255}

	for i, ap := range aps {
		ssid := ap.SSID
		if ap.IsProtected {
			ssid += " [L]"
		}
		color := white
		if ap.IsActive {
			color = colors.LawnGreen
		}
		item, err := NewStringItem(ssid, p.applicationFont, color)
		if err != nil {
			logger.WarningF("Could not create texture for SSID %s: %v", ap.SSID, err)
			continue
		}
		p.accessPoints[i] = accessPointUIItem{AP: ap, Item: item}
	}
}

// updatePasswordTexture regenerates the texture for the password input field.
func (p *WiFiSetupPage) updatePasswordTexture() {
	if p.passwordInputItem.ID != 0 {
		gl.DeleteTextures(1, &p.passwordInputItem.ID)
	}
	maskedPassword := strings.Repeat("*", len(p.passwordInput))
	if maskedPassword == "" {
		maskedPassword = " "
	}
	item, err := NewStringItem(maskedPassword, p.applicationFont, sdl.Color{R: 255, G: 255, B: 255, A: 255})
	if err == nil {
		p.passwordInputItem = item
	}
}

// updatePasswordPromptTexture creates the texture for the password prompt label.
func (p *WiFiSetupPage) updatePasswordPromptTexture() {
	if p.passwordPromptItem.ID != 0 {
		gl.DeleteTextures(1, &p.passwordPromptItem.ID)
	}
	prompt := fmt.Sprintf("Password for %s:", p.selectedAP.SSID)
	item, err := NewStringItem(prompt, p.applicationFont, sdl.Color{R: 255, G: 255, B: 255, A: 255})
	if err == nil {
		p.passwordPromptItem = item
	}
}

// updatePasswordErrorTexture creates the texture for password error message.
func (p *WiFiSetupPage) updatePasswordErrorTexture() {
	if p.passwordErrorItem.ID != 0 {
		gl.DeleteTextures(1, &p.passwordErrorItem.ID)
	}
	if p.connectionError != "" {
		item, err := NewStringItem(p.connectionError, p.applicationFont, colors.DarkRed)
		if err == nil {
			p.passwordErrorItem = item
		}
	}
}

// Render draws the entire page.
func (p *WiFiSetupPage) Render() error {
	// Get base font size from database
	baseFontSizeStr, err := db.GetConfigValue("init_font_size")
	if err != nil {
		return fmt.Errorf("retrieving init_font_size from db: %w", err)
	}
	baseFontSize, err := strconv.ParseFloat(baseFontSizeStr, 32)
	if err != nil {
		return fmt.Errorf("parsing init_font_size: %w", err)
	}

	// Colors & Config
	bgColor := sdl.Color{R: 0, G: 0, B: 0, A: 255}
	selectionColor := sdl.Color{R: 68, G: 71, B: 90, A: 255}
	white := sdl.Color{R: 255, G: 255, B: 255, A: 255}
	verticalMargin := float32(20)

	solidShader, _ := shaderManager.Get().Get("solid_color")
	textShader, _ := shaderManager.Get().Get("text")

	// Background
	p.Base.RenderSolidColorQuad(solidShader, mgl32.Vec2{0, 0}, mgl32.Vec2{float32(p.ScreenWidth), float32(p.ScreenHeight)}, bgColor)

	// Title
	titleScale := float32(48) / float32(baseFontSize)
	titleW := float32(p.titleItem.W) * titleScale
	titleH := float32(p.titleItem.H) * titleScale
	titleX := float32(p.ScreenCenterX) - (titleW / 2)
	titleY := float32(p.ScreenHeight) - titleH - 40
	p.Base.RenderTexture(textShader, p.titleItem.ID, mgl32.Vec2{titleX, titleY}, mgl32.Vec2{titleW, titleH}, white)

	// Status message
	statusY := titleY - titleH - verticalMargin
	if p.isScanning {
		statusScale := float32(24) / float32(baseFontSize)
		statusW := float32(p.scanningStatusItem.W) * statusScale
		statusH := float32(p.scanningStatusItem.H) * statusScale
		statusX := float32(p.ScreenCenterX) - (statusW / 2)
		p.Base.RenderTexture(textShader, p.scanningStatusItem.ID, mgl32.Vec2{statusX, statusY}, mgl32.Vec2{statusW, statusH}, white)
	} else if p.connectionStatusItem.ID != 0 {
		statusScale := float32(24) / float32(baseFontSize)
		statusW := float32(p.connectionStatusItem.W) * statusScale
		statusH := float32(p.connectionStatusItem.H) * statusScale
		statusX := float32(p.ScreenCenterX) - (statusW / 2)
		p.Base.RenderTexture(textShader, p.connectionStatusItem.ID, mgl32.Vec2{statusX, statusY}, mgl32.Vec2{statusW, statusH}, p.connectionStatusItem.Color)
	}

	// Network list
	listY := statusY - verticalMargin
	if len(p.accessPoints) > 0 {
		itemHeight := int32(40)
		listPadding := int32(10)
		for i, apItem := range p.accessPoints {
			itemYpos := listY - float32(int32(i+1)*itemHeight)
			itemXpos := float32(p.ScreenCenterX - 350)
			itemW := float32(700)
			if i == p.selectedIndex {
				p.Base.RenderSolidColorQuad(solidShader, mgl32.Vec2{itemXpos, itemYpos}, mgl32.Vec2{itemW, float32(itemHeight)}, selectionColor)
			}

			textScale := float32(24) / float32(baseFontSize)
			textW := float32(apItem.Item.W) * textScale
			textH := float32(apItem.Item.H) * textScale
			textY := itemYpos + (float32(itemHeight) / 2) - (textH / 2)
			textX := itemXpos + float32(listPadding)
			p.Base.RenderTexture(textShader, apItem.Item.ID, mgl32.Vec2{textX, textY}, mgl32.Vec2{textW, textH}, apItem.Item.Color)
		}
	}

	// Password dialog overlay
	if p.isEnteringPassword {
		if err := p.renderPasswordDialog(solidShader, textShader, float32(baseFontSize)); err != nil {
			return err
		}
	}

	return nil
}

// renderPasswordDialog draws the password entry modal.
func (p *WiFiSetupPage) renderPasswordDialog(solidShader, textShader *shaderManager.Shader, baseFontSize float32) error {
	overlayColor := sdl.Color{R: 0, G: 0, B: 0, A: 150}
	dialogBgColor := sdl.Color{R: 40, G: 42, B: 54, A: 255}
	boxBorderColor := sdl.Color{R: 120, G: 120, B: 120, A: 255}
	boxBgColor := sdl.Color{R: 30, G: 30, B: 30, A: 255}
	white := sdl.Color{R: 255, G: 255, B: 255, A: 255}

	dialogW, dialogH := int32(600), int32(250)
	if p.connectionError != "" {
		dialogH = int32(300) // Make dialog taller if there's an error
	}
	dialogX, dialogY := p.ScreenCenterX-(dialogW/2), p.ScreenCenterY-(dialogH/2)

	// Overlay and dialog background
	p.Base.RenderSolidColorQuad(solidShader, mgl32.Vec2{0, 0}, mgl32.Vec2{float32(p.ScreenWidth), float32(p.ScreenHeight)}, overlayColor)
	p.Base.RenderSolidColorQuad(solidShader, mgl32.Vec2{float32(dialogX), float32(dialogY)}, mgl32.Vec2{float32(dialogW), float32(dialogH)}, dialogBgColor)

	currentY := float32(dialogY+dialogH) - 30

	// Password error (if exists) - shown at the top in red
	if p.passwordErrorItem.ID != 0 {
		errorScale := float32(20) / baseFontSize
		errorW := float32(p.passwordErrorItem.W) * errorScale
		errorH := float32(p.passwordErrorItem.H) * errorScale
		errorX := float32(p.ScreenCenterX) - (errorW / 2)
		currentY -= errorH
		p.Base.RenderTexture(textShader, p.passwordErrorItem.ID, mgl32.Vec2{errorX, currentY}, mgl32.Vec2{errorW, errorH}, colors.DarkRed)
		currentY -= 20
	}

	// Password prompt
	if p.passwordPromptItem.ID != 0 {
		promptScale := float32(24) / baseFontSize
		promptW := float32(p.passwordPromptItem.W) * promptScale
		promptH := float32(p.passwordPromptItem.H) * promptScale
		promptX := float32(p.ScreenCenterX) - (promptW / 2)
		currentY -= promptH
		p.Base.RenderTexture(textShader, p.passwordPromptItem.ID, mgl32.Vec2{promptX, currentY}, mgl32.Vec2{promptW, promptH}, white)
		currentY -= 30
	}

	// Input box
	padding, border := int32(4), int32(3)
	boxW := int32(500)
	targetFontSize := int32(36)
	boxH := targetFontSize + (padding * 2)
	borderBoxW, borderBoxH := boxW+(border*2), boxH+(border*2)
	borderBoxX := p.ScreenCenterX - (borderBoxW / 2)
	borderBoxY := int32(currentY) - boxH - (padding * 2)
	boxX, boxY := p.ScreenCenterX-(boxW/2), int32(currentY)-boxH-padding

	p.Base.RenderSolidColorQuad(solidShader, mgl32.Vec2{float32(borderBoxX), float32(borderBoxY)}, mgl32.Vec2{float32(borderBoxW), float32(borderBoxH)}, boxBorderColor)
	p.Base.RenderSolidColorQuad(solidShader, mgl32.Vec2{float32(boxX), float32(boxY)}, mgl32.Vec2{float32(boxW), float32(boxH)}, boxBgColor)

	// Password input text
	if p.passwordInputItem.ID != 0 {
		inputScale := float32(targetFontSize) / baseFontSize
		inputW := float32(p.passwordInputItem.W) * inputScale
		inputH := float32(p.passwordInputItem.H) * inputScale
		inputX := float32(boxX + padding)
		inputY := float32(boxY + padding)
		p.Base.RenderTexture(textShader, p.passwordInputItem.ID, mgl32.Vec2{inputX, inputY}, mgl32.Vec2{inputW, inputH}, white)
	}

	return nil
}

// Destroy cleans up all resources used by the page.
func (p *WiFiSetupPage) Destroy() error {
	logger.InfoF("Destroying WiFiSetupPage...")

	if p.stopScan != nil {
		close(p.stopScan)
		p.stopScan = nil
	}

	p.networkManager.Close()

	gl.DeleteTextures(1, &p.titleItem.ID)
	gl.DeleteTextures(1, &p.scanningStatusItem.ID)
	if p.connectionStatusItem.ID != 0 {
		gl.DeleteTextures(1, &p.connectionStatusItem.ID)
	}
	if p.passwordPromptItem.ID != 0 {
		gl.DeleteTextures(1, &p.passwordPromptItem.ID)
	}
	if p.passwordInputItem.ID != 0 {
		gl.DeleteTextures(1, &p.passwordInputItem.ID)
	}
	if p.passwordErrorItem.ID != 0 {
		gl.DeleteTextures(1, &p.passwordErrorItem.ID)
	}

	for _, apItem := range p.accessPoints {
		if apItem.Item.ID != 0 {
			gl.DeleteTextures(1, &apItem.Item.ID)
		}
	}

	return p.Base.Destroy()
}

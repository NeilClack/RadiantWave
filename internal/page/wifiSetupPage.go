package page

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/godbus/dbus/v5"
	"github.com/veandco/go-sdl2/sdl"
	"radiantwavetech.com/radiantwave/internal/colors"
	"radiantwavetech.com/radiantwave/internal/config"
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
	config          *config.Config
	networkManager  *network.Manager
	applicationFont *fontManager.FontEntry

	// Page-specific UI elements
	titleItem            StringItem
	scanningStatusItem   StringItem
	connectionStatusItem StringItem
	passwordPromptItem   StringItem
	passwordInputItem    StringItem

	// Network list and selection
	accessPoints  []accessPointUIItem
	selectedIndex int
	scrollOffset  int

	// State management
	isScanning         bool
	isEnteringPassword bool
	passwordInput      string
	selectedAP         *network.AccessPoint
	wifiDevice         *network.Device
	connectionStatus   *network.ConnectionStatus
	activeSSID         string // Tracks the CONFIRMED active network
	pendingSSID        string // Tracks the network we are TRYING to connect to

	// Async management
	resultsChan chan []network.AccessPoint
	statusChan  chan *network.ConnectionStatus
	errorChan   chan error
	stopScan    chan struct{}
}

// Init sets up the page, initializes the network manager, and starts a scan.
func (p *WiFiSetupPage) Init(app ApplicationInterface) error {
	logger.LogInfo("Initializing WiFiSetupPage")
	p.config = config.Get()

	if err := p.Base.Init(app); err != nil {
		return fmt.Errorf("failed base page initialization: %w", err)
	}

	fm := fontManager.Get()
	font, ok := fm.GetFont(p.config.ApplicationFont)
	if !ok {
		return fmt.Errorf("could not find application font: %s", p.config.ApplicationFont)
	}
	p.applicationFont = font

	nm, err := network.New()
	if err != nil {
		return fmt.Errorf("failed to initialize network manager: %w", err)
	}
	p.networkManager = nm

	if err := p.networkManager.FindDevices(); err != nil {
		logger.LogWarningF("Could not scan for network devices: %v", err)
	}
	for _, dev := range p.networkManager.Devices {
		if dev.Type == network.TypeWifi {
			d := dev
			p.wifiDevice = &d
			break
		}
	}

	white := sdl.Color{R: 255, G: 255, B: 255, A: 255}
	p.titleItem, err = NewStringItem("Wi-Fi Setup", p.applicationFont, white)
	if err != nil {
		return fmt.Errorf("failed to create title item: %w", err)
	}
	p.scanningStatusItem, err = NewStringItem("Scanning for networks...", p.applicationFont, white)
	if err != nil {
		return fmt.Errorf("failed to create scanning status item: %w", err)
	}

	p.resultsChan = make(chan []network.AccessPoint)
	p.statusChan = make(chan *network.ConnectionStatus, 1)
	p.errorChan = make(chan error)
	p.stopScan = make(chan struct{})

	p.startScan()
	go p.checkConnectionStatusLoop()

	return nil
}

// checkConnectionStatusLoop periodically checks the network status.
func (p *WiFiSetupPage) checkConnectionStatusLoop() {
	ticker := time.NewTicker(5 * time.Second) // Check more frequently
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
		logger.LogInfoF("Attempting to connect to %s", p.selectedAP.SSID)
		sdl.StopTextInput()
		p.pendingSSID = p.selectedAP.SSID // Set pending state
		err := p.networkManager.Connect(*p.wifiDevice, *p.selectedAP, p.passwordInput)
		if err != nil {
			logger.LogErrorF("Failed to send connect command: %v", err)
			p.pendingSSID = "" // Clear pending state on immediate failure
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
			p.pendingSSID = p.selectedAP.SSID // Set pending state
			if p.selectedAP.IsProtected {
				p.isEnteringPassword = true
				p.passwordInput = ""
				p.updatePasswordTexture()
				p.updatePasswordPromptTexture()
				sdl.StartTextInput()
			} else {
				logger.LogInfoF("Connecting to open network: %s", p.selectedAP.SSID)
				err := p.networkManager.Connect(*p.wifiDevice, *p.selectedAP, "")
				if err != nil {
					logger.LogErrorF("Failed to send connect command: %v", err)
					p.pendingSSID = "" // Clear pending state on immediate failure
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
		p.updateConnectionStatusItem()
	case err := <-p.errorChan:
		p.isScanning = false
		logger.LogErrorF("Network scan error: %v", err)
	default:
	}
	return nil
}

// updateConnectionStatusItem creates the colored status message item.
func (p *WiFiSetupPage) updateConnectionStatusItem() {
	if p.connectionStatusItem.ID != 0 {
		gl.DeleteTextures(1, &p.connectionStatusItem.ID)
	}
	if p.connectionStatus == nil {
		return
	}

	switch p.connectionStatus.StatusCode {
	case network.StatusWifiConnectedInternetUp, network.StatusEthernetConnectedInternetUp:
		for i, item := range p.accessPoints {
			if item.AP.IsActive {
				p.accessPoints[i].Item.Color = colors.LawnGreen
			}
		}
	case network.StatusWifiAvailableNotConnected, network.StatusEthernetNotConnected, network.StatusNoDevicesFound, network.StatusError:
		// **HERE: We are explicitly disconnected. Clear the active and pending states.**
		p.activeSSID = ""
		p.pendingSSID = ""
	}

	if p.connectionStatus.StatusCode == network.StatusWifiConnectedInternetUp {
		message := "Wifi Connected, Internet Accessible. Press F1 to exit WiFi Setup"
		item, err := NewStringItem(message, p.applicationFont, colors.LawnGreen)
		if err == nil {
			p.connectionStatusItem = item
		}
	} else {
		item, err := NewStringItem(p.connectionStatus.Message, p.applicationFont, colors.White)
		if err == nil {
			p.connectionStatusItem = item
		}

	}
}

// buildNetworkUIList creates the renderable textures for the list of discovered networks.
func (p *WiFiSetupPage) buildNetworkUIList(aps []network.AccessPoint) {
	// Query NetworkManager for the currently active Wi-Fi SSID
	activeSSID := ""
	if p.networkManager != nil && p.wifiDevice != nil {
		nmObj := p.networkManager.Conn().Object("org.freedesktop.NetworkManager", "/org/freedesktop/NetworkManager")
		// Get the "ActiveConnections" property (array of object paths)
		prop, err := nmObj.GetProperty("org.freedesktop.NetworkManager.ActiveConnections")
		if err == nil {
			if paths, ok := prop.Value().([]dbus.ObjectPath); ok {
				for _, path := range paths {
					acObj := p.networkManager.Conn().Object("org.freedesktop.NetworkManager", path)
					// Get the Devices property for this active connection
					devProp, err := acObj.GetProperty("org.freedesktop.NetworkManager.Connection.Active.Devices")
					if err == nil {
						if devPaths, ok := devProp.Value().([]dbus.ObjectPath); ok {
							for _, devPath := range devPaths {
								if string(devPath) == string(p.wifiDevice.Path) {
									// This active connection uses our Wi-Fi device
									// Get the connection ID (SSID)
									idProp, err := acObj.GetProperty("org.freedesktop.NetworkManager.Connection.Active.Id")
									if err == nil {
										if id, ok := idProp.Value().(string); ok {
											activeSSID = id
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
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
			ssid += " *"
		}
		color := white
		if ap.IsActive {
			color = colors.LawnGreen
		}
		item, err := NewStringItem(ssid, p.applicationFont, color)
		if err != nil {
			logger.LogWarningF("Could not create texture for SSID %s: %v", ap.SSID, err)
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

// Render draws the entire page.
func (p *WiFiSetupPage) Render() error {
	// Colors & Config
	bgColor := sdl.Color{R: 40, G: 42, B: 54, A: 255}
	selectionColor := sdl.Color{R: 68, G: 71, B: 90, A: 255}
	white := sdl.Color{R: 255, G: 255, B: 255, A: 255}
	baseFontSize := float32(p.config.BaseFontSize)
	verticalMargin := float32(20)

	solidShader, _ := shaderManager.Get().Get("solid_color")
	textShader, _ := shaderManager.Get().Get("text")

	p.Base.RenderSolidColorQuad(solidShader, mgl32.Vec2{0, 0}, mgl32.Vec2{float32(p.ScreenWidth), float32(p.ScreenHeight)}, bgColor)

	titleScale := float32(48) / baseFontSize
	titleW := float32(p.titleItem.W) * titleScale
	titleH := float32(p.titleItem.H) * titleScale
	titleX := float32(p.ScreenCenterX) - (titleW / 2)
	titleY := float32(p.ScreenHeight) - titleH - 40
	p.Base.RenderTexture(textShader, p.titleItem.ID, mgl32.Vec2{titleX, titleY}, mgl32.Vec2{titleW, titleH}, white)

	statusY := titleY - titleH - verticalMargin
	if p.isScanning {
		statusScale := float32(24) / baseFontSize
		statusW := float32(p.scanningStatusItem.W) * statusScale
		statusH := float32(p.scanningStatusItem.H) * statusScale
		statusX := float32(p.ScreenCenterX) - (statusW / 2)
		p.Base.RenderTexture(textShader, p.scanningStatusItem.ID, mgl32.Vec2{statusX, statusY}, mgl32.Vec2{statusW, statusH}, white)
	} else if p.connectionStatusItem.ID != 0 {
		statusScale := float32(24) / baseFontSize
		statusW := float32(p.connectionStatusItem.W) * statusScale
		statusH := float32(p.connectionStatusItem.H) * statusScale
		statusX := float32(p.ScreenCenterX) - (statusW / 2)
		p.Base.RenderTexture(textShader, p.connectionStatusItem.ID, mgl32.Vec2{statusX, statusY}, mgl32.Vec2{statusW, statusH}, p.connectionStatusItem.Color)
	}

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

			textScale := float32(24) / baseFontSize
			textW := float32(apItem.Item.W) * textScale
			textH := float32(apItem.Item.H) * textScale
			textY := itemYpos + (float32(itemHeight) / 2) - (textH / 2)
			textX := itemXpos + float32(listPadding)
			p.Base.RenderTexture(textShader, apItem.Item.ID, mgl32.Vec2{textX, textY}, mgl32.Vec2{textW, textH}, apItem.Item.Color)
		}
	}

	if p.isEnteringPassword {
		p.renderPasswordDialog(solidShader, textShader)
	}

	return nil
}

// renderPasswordDialog draws the password entry modal.
func (p *WiFiSetupPage) renderPasswordDialog(solidShader, textShader *shaderManager.Shader) {
	overlayColor := sdl.Color{R: 0, G: 0, B: 0, A: 150}
	dialogBgColor := sdl.Color{R: 40, G: 42, B: 54, A: 255}
	boxBorderColor := sdl.Color{R: 120, G: 120, B: 120, A: 255}
	boxBgColor := sdl.Color{R: 30, G: 30, B: 30, A: 255}
	white := sdl.Color{R: 255, G: 255, B: 255, A: 255}
	baseFontSize := float32(p.config.BaseFontSize)

	dialogW, dialogH := int32(600), int32(200)
	dialogX, dialogY := p.ScreenCenterX-(dialogW/2), p.ScreenCenterY-(dialogH/2)

	p.Base.RenderSolidColorQuad(solidShader, mgl32.Vec2{0, 0}, mgl32.Vec2{float32(p.ScreenWidth), float32(p.ScreenHeight)}, overlayColor)
	p.Base.RenderSolidColorQuad(solidShader, mgl32.Vec2{float32(dialogX), float32(dialogY)}, mgl32.Vec2{float32(dialogW), float32(dialogH)}, dialogBgColor)

	if p.passwordPromptItem.ID != 0 {
		promptScale := float32(24) / baseFontSize
		promptW := float32(p.passwordPromptItem.W) * promptScale
		promptH := float32(p.passwordPromptItem.H) * promptScale
		promptX := float32(p.ScreenCenterX) - (promptW / 2)
		promptY := float32(dialogY+dialogH) - promptH - 20
		p.Base.RenderTexture(textShader, p.passwordPromptItem.ID, mgl32.Vec2{promptX, promptY}, mgl32.Vec2{promptW, promptH}, white)
	}

	padding, border := int32(4), int32(3)
	boxW := int32(500)
	targetFontSize := int32(36)
	boxH := targetFontSize + (padding * 2)
	borderBoxW, borderBoxH := boxW+(border*2), boxH+(border*2)
	borderBoxX, borderBoxY := p.ScreenCenterX-(borderBoxW/2), p.ScreenCenterY-(borderBoxH/2)
	boxX, boxY := p.ScreenCenterX-(boxW/2), p.ScreenCenterY-(boxH/2)

	p.Base.RenderSolidColorQuad(solidShader, mgl32.Vec2{float32(borderBoxX), float32(borderBoxY)}, mgl32.Vec2{float32(borderBoxW), float32(borderBoxH)}, boxBorderColor)
	p.Base.RenderSolidColorQuad(solidShader, mgl32.Vec2{float32(boxX), float32(boxY)}, mgl32.Vec2{float32(boxW), float32(boxH)}, boxBgColor)

	if p.passwordInputItem.ID != 0 {
		inputScale := float32(targetFontSize) / baseFontSize
		inputW := float32(p.passwordInputItem.W) * inputScale
		inputH := float32(p.passwordInputItem.H) * inputScale
		inputX := float32(boxX + padding)
		inputY := float32(boxY + padding)
		p.Base.RenderTexture(textShader, p.passwordInputItem.ID, mgl32.Vec2{inputX, inputY}, mgl32.Vec2{inputW, inputH}, white)
	}
}

// Destroy cleans up all resources used by the page.
func (p *WiFiSetupPage) Destroy() error {
	logger.LogInfo("Destroying WiFiSetupPage...")

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

	for _, apItem := range p.accessPoints {
		if apItem.Item.ID != 0 {
			gl.DeleteTextures(1, &apItem.Item.ID)
		}
	}

	return p.Base.Destroy()
}

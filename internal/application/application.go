// package application is the entry point to RadiantWave. It handles all managers, pages, and the event loop.
package application

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
	"radiantwavetech.com/radiantwave/internal/db"
	"radiantwavetech.com/radiantwave/internal/fontManager"
	"radiantwavetech.com/radiantwave/internal/graphics"
	"radiantwavetech.com/radiantwave/internal/keybinds"
	"radiantwavetech.com/radiantwave/internal/logger"
	"radiantwavetech.com/radiantwave/internal/mixer"
	"radiantwavetech.com/radiantwave/internal/network"
	"radiantwavetech.com/radiantwave/internal/page"
	"radiantwavetech.com/radiantwave/internal/shaderManager"
)

// Application configuration constants
const (
	targetFPS          = 60.0
	volumeStepSize     = 16
	networkMaxWait     = 25 * time.Second
	networkStableTime  = 2 * time.Second
	openGLMajorVersion = 3
	openGLMinorVersion = 3
)

// Application holds the core state and dependencies for the application.
type Application struct {
	Window           *sdl.Window
	ValidationReport *ValidationReport
	pageStack        []page.Page
	pendingAction    func()
	running          bool
}

// ValidationCheck defines the type for specific validation steps.
type ValidationCheck int

// Validation check types
const (
	EmailAddressCheck ValidationCheck = iota
	LicenseKeyCheck
	NetworkConnectionCheck
	SubscriptionCheck
)

// CheckResult holds the outcome of a single validation check.
type CheckResult struct {
	OK      bool
	Message string
}

// ValidationReport contains the results of all validation checks.
type ValidationReport struct {
	Checks   map[ValidationCheck]CheckResult
	AllValid bool
}

// String returns the string representation of a ValidationCheck
func (vc ValidationCheck) String() string {
	names := map[ValidationCheck]string{
		EmailAddressCheck:      "EmailAddressCheck",
		LicenseKeyCheck:        "LicenseKeyCheck",
		NetworkConnectionCheck: "NetworkConnectionCheck",
		SubscriptionCheck:      "SubscriptionCheck",
	}
	if name, ok := names[vc]; ok {
		return name
	}
	return "UnknownCheck"
}

// ----------------------------------------------------------------------------
// Main Entry Point
// ----------------------------------------------------------------------------

// Run is the core application entry point.
func Run() error {
	appDataDir, err := initializeApplicationDirectory()
	if err != nil {
		return err
	}

	logger.InitLogger(appDataDir)

	if err := initializeDatabase(appDataDir); err != nil {
		return err
	}

	app := &Application{}

	if err := app.initializeSDLAndOpenGL(); err != nil {
		return err
	}
	defer app.cleanup()

	if err := app.initializeManagers(); err != nil {
		return err
	}

	if err := app.validate(); err != nil {
		return fmt.Errorf("application validation failed: %w", err)
	}

	if err := app.initializeAudio(); err != nil {
		return err
	}

	if err := app.runEventLoop(); err != nil {
		logger.ErrorF("Event loop exited with error: %v", err)
		return err
	}

	return nil
}

// ----------------------------------------------------------------------------
// Initialization Functions
// ----------------------------------------------------------------------------

// initializeApplicationDirectory creates and returns the application data directory path
func initializeApplicationDirectory() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine user home directory: %w", err)
	}
	return filepath.Join(homeDir, ".radiantwave"), nil
}

// initializeDatabase sets up the database connection
func initializeDatabase(appDataDir string) error {
	dbPath := filepath.Join(appDataDir, "data.db")
	if err := db.InitDatabase(dbPath); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	return nil
}

// initializeSDLAndOpenGL sets up SDL, TTF, and OpenGL
func (app *Application) initializeSDLAndOpenGL() error {
	logger.Info("Initializing SDL systems")

	if err := initializeSDL(); err != nil {
		return fmt.Errorf("SDL initialization failed: %w", err)
	}

	window, err := app.createWindow()
	if err != nil {
		return err
	}
	app.Window = window

	if err := app.initializeOpenGL(); err != nil {
		return err
	}

	app.initKeybinds()
	return nil
}

// initializeSDL initializes SDL subsystems
func initializeSDL() error {
	if err := sdl.Init(sdl.INIT_VIDEO | sdl.INIT_AUDIO); err != nil {
		logger.ErrorF("Failed to initialize SDL subsystems: %v", err)
		return err
	}
	logger.Info("Successfully initialized VIDEO and AUDIO subsystems")
	logger.InfoF("SDL audio backend: %s", sdl.GetCurrentAudioDriver())

	if err := ttf.Init(); err != nil {
		logger.ErrorF("Failed to initialize TTF: %v", err)
		return fmt.Errorf("TTF initialization failed: %w", err)
	}
	logger.Info("Successfully initialized TTF")

	sdl.GLSetAttribute(sdl.GL_CONTEXT_MAJOR_VERSION, openGLMajorVersion)
	sdl.GLSetAttribute(sdl.GL_CONTEXT_MINOR_VERSION, openGLMinorVersion)
	sdl.GLSetAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_CORE)
	sdl.ShowCursor(sdl.DISABLE)
	sdl.SetRelativeMouseMode(true)

	return nil
}

// createWindow creates the SDL window with proper display mode detection
func (app *Application) createWindow() (*sdl.Window, error) {
	displayMode, err := sdl.GetDisplayMode(0, 0)
	if err != nil {
		logger.ErrorF("Failed to query display mode: %v", err)
		return nil, fmt.Errorf("display mode query failed: %w", err)
	}

	logger.InfoF("Display mode: %dx%d", displayMode.W, displayMode.H)

	if displayMode.W == 0 || displayMode.H == 0 {
		return nil, fmt.Errorf("invalid display dimensions: %dx%d", displayMode.W, displayMode.H)
	}

	width, height := graphics.SetDisplayOrientation(displayMode.W, displayMode.H)

	window, err := sdl.CreateWindow(
		"Radiant Wave",
		sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED,
		width,
		height,
		sdl.WINDOW_OPENGL|sdl.WINDOW_FULLSCREEN_DESKTOP|sdl.WINDOW_SHOWN,
	)
	if err != nil {
		return nil, fmt.Errorf("window creation failed: %w", err)
	}

	return window, nil
}

// initializeOpenGL sets up OpenGL context and logs version info
func (app *Application) initializeOpenGL() error {
	glContext, err := app.Window.GLCreateContext()
	if err != nil {
		return fmt.Errorf("OpenGL context creation failed: %w", err)
	}
	_ = glContext // Mark as intentionally unused

	if err := sdl.GLSetSwapInterval(1); err != nil {
		return fmt.Errorf("failed to enable vsync: %w", err)
	}

	if err := gl.InitWithProcAddrFunc(sdl.GLGetProcAddress); err != nil {
		return fmt.Errorf("go-gl initialization failed: %w", err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	renderer := gl.GoStr(gl.GetString(gl.RENDERER))
	logger.InfoF("OpenGL version: %s", version)
	logger.InfoF("OpenGL renderer: %s", renderer)
	logger.Info("SDL & OpenGL successfully initialized")

	return nil
}

// initializeManagers initializes shader and font managers
func (app *Application) initializeManagers() error {
	if err := shaderManager.InitShaderManager(); err != nil {
		return fmt.Errorf("ShaderManager initialization failed: %w", err)
	}

	if err := fontManager.InitFontManager(); err != nil {
		return fmt.Errorf("FontManager initialization failed: %w", err)
	}

	return nil
}

// initializeAudio initializes the music mixer with configured audio device
func (app *Application) initializeAudio() error {
	audioDevice, err := db.GetConfigValue("audio_device_name")
	if err != nil {
		return fmt.Errorf("failed to get audio device name: %w", err)
	}
	mixer.Init(audioDevice)
	return nil
}

// cleanup handles proper resource cleanup
func (app *Application) cleanup() {
	for _, p := range app.pageStack {
		p.Destroy()
	}
	app.pageStack = nil

	if app.Window != nil {
		app.Window.Destroy()
	}

	ttf.Quit()
	sdl.Quit()
}

// ----------------------------------------------------------------------------
// Event Loop
// ----------------------------------------------------------------------------

// runEventLoop contains the main loop for the application.
func (app *Application) runEventLoop() error {
	if len(app.pageStack) == 0 {
		return fmt.Errorf("no initial page loaded, pageStack empty")
	}
	logger.Info("Successfully initialized first page in pageStack")

	const frameDelay = time.Second / targetFPS
	app.running = true
	lastTime := time.Now()

	for app.running {
		frameStart := time.Now()

		app.handleEvents()
		if !app.running {
			break
		}

		deltaTime := float32(time.Since(lastTime).Seconds())
		lastTime = time.Now()

		app.updateCurrentPage(deltaTime)
		app.renderFrame()
		app.applyPendingAction()

		// More accurate frame timing
		frameTime := time.Since(frameStart)
		if frameTime < frameDelay {
			time.Sleep(frameDelay - frameTime)
		}
	}

	return nil
}

// handleEvents processes all pending SDL events
func (app *Application) handleEvents() {
	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
		switch e := event.(type) {
		case *sdl.QuitEvent:
			app.running = false
			return

		case *sdl.KeyboardEvent:
			if e.Type == sdl.KEYDOWN && e.Repeat == 0 {
				// Check for Ctrl+C or Cmd+C
				isCtrlPressed := (e.Keysym.Mod&sdl.KMOD_CTRL) != 0 || (e.Keysym.Mod&sdl.KMOD_GUI) != 0
				if e.Keysym.Sym == sdl.K_c && isCtrlPressed {
					logger.Info("User attempted to exit program")
					continue
				}
				keybinds.PerformAction(e)
			}
		}

		// Forward event to current page
		if currentPage := app.currentPage(); currentPage != nil {
			if err := currentPage.HandleEvent(&event); err != nil {
				logger.WarningF("Error handling event in page: %v", err)
			}
		}
	}
}

// updateCurrentPage updates the current page with delta time
func (app *Application) updateCurrentPage(deltaTime float32) {
	if currentPage := app.currentPage(); currentPage != nil {
		currentPage.Update(deltaTime)
	}
}

// renderFrame clears and renders the current frame
func (app *Application) renderFrame() {
	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	if currentPage := app.currentPage(); currentPage != nil {
		currentPage.Render()
	}

	app.Window.GLSwap()
}

// ----------------------------------------------------------------------------
// Keybind Registration
// ----------------------------------------------------------------------------

// initKeybinds registers all application-level keyboard shortcuts
func (app *Application) initKeybinds() {
	keybinds.Register(sdl.K_F3, sdl.KMOD_NONE, func() {
		app.PushPage(&page.Settings{})
	})

	keybinds.Register(sdl.K_F2, sdl.KMOD_NONE, func() {
		app.PushPage(&page.ScrollerPage{})
	})

	keybinds.Register(sdl.K_F1, sdl.KMOD_NONE, func() {
		app.UnwindToPage(&page.Welcome{})
	})

	keybinds.Register(sdl.K_q, sdl.KMOD_RCTRL, func() {
		app.Stop()
	})

	keybinds.Register(sdl.K_UP, sdl.KMOD_SHIFT, func() {
		logger.InfoF("Increasing volume: %d + %d", mixer.GetVolume128(), volumeStepSize)
		mixer.SetVolume128(volumeStepSize)
	})

	keybinds.Register(sdl.K_DOWN, sdl.KMOD_SHIFT, func() {
		logger.InfoF("Decreasing volume: %d - %d", mixer.GetVolume128(), volumeStepSize)
		mixer.SetVolume128(-volumeStepSize)
	})
}

// Stop sets the running flag to false, triggering application shutdown
func (app *Application) Stop() {
	app.running = false
}

// ----------------------------------------------------------------------------
// Page Management
// ----------------------------------------------------------------------------

// currentPage returns the page at the top of the stack, or nil if empty
func (app *Application) currentPage() page.Page {
	if len(app.pageStack) == 0 {
		return nil
	}
	return app.pageStack[len(app.pageStack)-1]
}

// applyPendingAction executes the scheduled page transition
func (app *Application) applyPendingAction() {
	if app.pendingAction != nil {
		app.pendingAction()
		app.pendingAction = nil
	}
}

// schedulePush schedules a new page to be added to the top of the stack
func (app *Application) schedulePush(p page.Page) {
	app.pendingAction = func() {
		if err := p.Init(app); err != nil {
			logger.ErrorF("Failed to initialize page, aborting push: %v", err)
			return
		}

		if topPage := app.currentPage(); topPage != nil {
			topPage.Destroy()
		}

		app.pageStack = append(app.pageStack, p)
		logger.InfoF("Pushed new page onto stack: %T", p)
	}
}

// scheduleSwitch schedules the current page to be replaced by a new one
func (app *Application) scheduleSwitch(p page.Page) {
	app.pendingAction = func() {
		if topPage := app.currentPage(); topPage != nil {
			topPage.Destroy()
			app.pageStack = app.pageStack[:len(app.pageStack)-1]
		}

		if err := p.Init(app); err != nil {
			logger.ErrorF("Failed to initialize page, aborting switch: %v", err)
			return
		}

		app.pageStack = append(app.pageStack, p)
		logger.InfoF("Switched to new page: %T", p)
	}
}

// scheduleUnwind schedules the removal of all pages above a target page
func (app *Application) scheduleUnwind(targetPage page.Page) {
	app.pendingAction = func() {
		// Search for the target page in the stack
		foundIndex := -1
		for i := len(app.pageStack) - 1; i >= 0; i-- {
			if app.pageStack[i] == targetPage {
				foundIndex = i
				break
			}
		}

		if foundIndex != -1 {
			// Destroy pages above the target
			for i := len(app.pageStack) - 1; i > foundIndex; i-- {
				app.pageStack[i].Destroy()
			}
			app.pageStack = app.pageStack[:foundIndex+1]
			logger.InfoF("Unwound to existing page: %T", targetPage)
		} else {
			// Page not found, replace current with new instance
			if topPage := app.currentPage(); topPage != nil {
				topPage.Destroy()
				app.pageStack = app.pageStack[:len(app.pageStack)-1]
			}

			if err := targetPage.Init(app); err != nil {
				logger.ErrorF("Failed to initialize target page: %v", err)
				return
			}

			app.pageStack = append(app.pageStack, targetPage)
			logger.InfoF("Unwound to new page instance: %T", targetPage)
		}
	}
}

// ----------------------------------------------------------------------------
// Public Page Navigation Interface
// ----------------------------------------------------------------------------

// PushPage adds a new page to the pageStack.
// Creates a page 'history' rather than a simple switch.
func (app *Application) PushPage(p page.Page) {
	app.schedulePush(p)
}

// SwitchPage replaces the current page with a new one.
// This is the standard method for page transitions.
func (app *Application) SwitchPage(p page.Page) {
	app.scheduleSwitch(p)
}

// UnwindToPage reduces the pageStack back to a specific page.
// Like a "back button" but you choose the destination.
func (app *Application) UnwindToPage(p page.Page) {
	app.scheduleUnwind(p)
}

// PopPage removes the current page from the stack.
// Typically called from within a page to close itself.
func (app *Application) PopPage() {
	app.pendingAction = func() {
		if topPage := app.currentPage(); topPage != nil {
			topPage.Destroy()
			app.pageStack = app.pageStack[:len(app.pageStack)-1]
			logger.Info("Popped page from stack")
		}
	}
}

// GetDrawableSize returns the current OpenGL drawable size (typically screen dimensions)
func (app *Application) GetDrawableSize() (w, h int32) {
	return app.Window.GLGetDrawableSize()
}

// ----------------------------------------------------------------------------
// Validation
// ----------------------------------------------------------------------------

// validate performs a series of checks to ensure the application is ready to run
func (app *Application) validate() error {
	logger.Info("Validating program state")

	report := &ValidationReport{
		Checks:   make(map[ValidationCheck]CheckResult),
		AllValid: true,
	}

	// Initialize welcome page as base
	welcomePage := &page.Welcome{}
	if err := welcomePage.Init(app); err != nil {
		return fmt.Errorf("failed to initialize welcome page: %w", err)
	}
	app.pageStack = append(app.pageStack, welcomePage)

	// Validate network connectivity
	isOnline := app.validateNetwork(report)

	// If offline, push WiFi setup page
	if !isOnline {
		report.AllValid = false
		wifiPage := &page.WiFiSetupPage{}
		if err := wifiPage.Init(app); err != nil {
			return fmt.Errorf("failed to initialize WiFi setup page: %w", err)
		}
		app.pageStack = append(app.pageStack, wifiPage)
	}

	// Validate license key
	if err := app.validateLicenseKey(report); err != nil {
		return err
	}

	// Validate email address
	if err := app.validateEmailAddress(report); err != nil {
		return err
	}

	// Check subscription status
	app.validateSubscription(report)

	// Log validation results
	for checkType, result := range report.Checks {
		logger.InfoF("Validation: %s - %s", checkType.String(), result.Message)
	}

	app.ValidationReport = report
	return nil
}

// validateNetwork checks network connectivity and returns online status
func (app *Application) validateNetwork(report *ValidationReport) bool {
	netManager, err := network.New()
	if err != nil {
		logger.ErrorF("Failed to initialize network manager: %v", err)
		report.Checks[NetworkConnectionCheck] = CheckResult{false, "Network manager initialization failed"}
		return false
	}
	defer func() {
		logger.Info("Closing network manager")
		netManager.Close()
	}()

	status, err := netManager.WaitForInternet(networkMaxWait, networkStableTime)
	if err != nil {
		logger.InfoF("Network wait error: %v", err)
	}

	isOnline := status.StatusCode == network.StatusEthernetConnectedInternetUp ||
		status.StatusCode == network.StatusWifiConnectedInternetUp

	report.Checks[NetworkConnectionCheck] = CheckResult{isOnline, status.Message}
	return isOnline
}

// validateLicenseKey checks for a valid license key in the config
func (app *Application) validateLicenseKey(report *ValidationReport) error {
	licenseKey, err := db.GetConfigValue("license_key")
	if err != nil {
		return fmt.Errorf("failed to get license key: %w", err)
	}

	if licenseKey == "" {
		report.Checks[LicenseKeyCheck] = CheckResult{false, "License key is missing"}
		report.AllValid = false

		licenseKeyPage := &page.LicenseKeyPage{}
		if err := licenseKeyPage.Init(app); err != nil {
			return fmt.Errorf("failed to initialize license key page: %w", err)
		}
		app.pageStack = append(app.pageStack, licenseKeyPage)
	} else {
		report.Checks[LicenseKeyCheck] = CheckResult{true, "License key found"}
	}

	return nil
}

// validateEmailAddress checks for a valid email address in the config
func (app *Application) validateEmailAddress(report *ValidationReport) error {
	emailAddress, err := db.GetConfigValue("email_address")
	if err != nil {
		return fmt.Errorf("failed to get email address: %w", err)
	}

	if emailAddress == "" {
		report.Checks[EmailAddressCheck] = CheckResult{false, "Email address is missing"}
		report.AllValid = false

		emailPage := &page.EmailAddressPage{}
		if err := emailPage.Init(app); err != nil {
			return fmt.Errorf("failed to initialize email address page: %w", err)
		}
		app.pageStack = append(app.pageStack, emailPage)
	} else {
		report.Checks[EmailAddressCheck] = CheckResult{true, "Email address found"}
	}

	return nil
}

// validateSubscription checks subscription status (currently simulated)
func (app *Application) validateSubscription(report *ValidationReport) {
	if report.AllValid {
		report.Checks[SubscriptionCheck] = CheckResult{true, "Subscription status is valid (Simulated)"}
	} else {
		report.Checks[SubscriptionCheck] = CheckResult{false, "Skipped due to previous validation failures"}
	}
}

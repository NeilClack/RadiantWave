// package application is the entry point to RadiantWave. It handles all managers, pages, and the event loop.
package application

import (
	"fmt"
	"time"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
	"radiantwavetech.com/radiant_wave/internal/config"
	"radiantwavetech.com/radiant_wave/internal/fontManager"
	"radiantwavetech.com/radiant_wave/internal/graphics"
	"radiantwavetech.com/radiant_wave/internal/keybinds"
	"radiantwavetech.com/radiant_wave/internal/logger"
	"radiantwavetech.com/radiant_wave/internal/mixer"
	"radiantwavetech.com/radiant_wave/internal/network"
	"radiantwavetech.com/radiant_wave/internal/page"
	"radiantwavetech.com/radiant_wave/internal/shaderManager"
)

// Application holds the core state and dependencies for the application.
// It no longer holds a direct reference to Config, using the singleton instead.
type Application struct {
	Window           *sdl.Window
	ValidationReport *ValidationReport
	pageStack        []page.Page
	pendingAction    func()
	running          bool
}

// ValidationCheck defines the type for specific validation steps.
type ValidationCheck int

// Defines the set of checks the Validate method will perform.
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
	AllValid bool // A helper field, true only if all essential checks passed.
}

// Run is the core application entry point.
func Run() error {
	// Create the Application object without the Config field
	app := &Application{}

	logger.InitLogger()

	logger := logger.Get()

	logger.Info("Application starting")

	// Initializing SDL systems
	logger.Info("Initializing SDL systems")
	if err := initializeSDL(logger); err != nil {
		return fmt.Errorf("failed to initialize SDL: %v", err)
	}
	defer ttf.Quit()
	defer sdl.Quit()

	displayMode, err := sdl.GetDisplayMode(0, 0)
	if err != nil {
		logger.Errorf("failed to query diapay mode for display 0: %v", err)
		return fmt.Errorf("failed to query diapay mode for display 0: %v", err)
	}
	logger.Infof("Display mode: %dx%d", displayMode.W, displayMode.H)
	if displayMode.W == 0 || displayMode.H == 0 {
		return fmt.Errorf("invalid display mode dimensions: %dx%d", displayMode.W, displayMode.H)
	}
	w, h := graphics.SetDisplayOrientation(displayMode.W, displayMode.H)
	window, err := sdl.CreateWindow("Radiant Wave", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		w, h, sdl.WINDOW_OPENGL|sdl.WINDOW_FULLSCREEN_DESKTOP|sdl.WINDOW_SHOWN)
	if err != nil {
		return fmt.Errorf("failed to create window: %w", err)
	}
	app.Window = window
	defer window.Destroy()

	glContext, err := window.GLCreateContext()
	if err != nil {
		return fmt.Errorf("failed to create OpenGL Context: %v", err)
	}
	defer sdl.GLDeleteContext(glContext)

	if err := sdl.GLSetSwapInterval(1); err != nil {
		return fmt.Errorf("unable to enable vsync: %v", err)
	}

	if err := gl.InitWithProcAddrFunc(sdl.GLGetProcAddress); err != nil {
		return fmt.Errorf("failed to initialize go-gl function pointers: %v", err)
	}
	version := gl.GoStr(gl.GetString(gl.VERSION))
	renderer := gl.GoStr(gl.GetString(gl.RENDERER))
	logger.Infof("OpenGL version: %s", version)
	logger.Infof("OpenGL renderer: %s", renderer)
	logger.Info("SDL & OpenGL successfully initialized.")
	app.initKeybinds()

	// Initialize the ShaderManager
	if err := shaderManager.InitShaderManager(); err != nil {
		return fmt.Errorf("failed to initialize ShaderManager in Application.Run(): %v", err)
	}

	// Initialize the FontManager
	if err := fontManager.InitFontManager(); err != nil {
		return fmt.Errorf("failed to initialize FontManager in Application.Run(): %v", err)
	}

	// Validate the application configuration
	if err := app.Validate(); err != nil {
		return fmt.Errorf("failed to complete application validation: %v", err)
	}

	// Initialize the Music Mixer
	mixer.Init(config.Get().AudioDeviceName)

	// Start the main application loop
	if err := app.runEventLoop(); err != nil {
		logger.Errorf("The main event loop exited with an error: %v", err)
		return err
	}

	app.Close()
	return nil
}

// runEventLoop contains the main loop for the application.
func (app *Application) runEventLoop() error {
	var now uint64
	FPS := 60.0
	frameDelay := 1.0 / FPS
	app.running = true

	lastTime := time.Now()
	pageStackLength := len(app.pageStack)
	if pageStackLength == 0 {
		logger.LogErrorF("no initial page loaded, pageStack empty!")
		return fmt.Errorf("no initial page loaded, pageStack empty")
	} else {
		logger.LogInfo("Successfully initialized first page in pageStack")
	}

	for app.running {
		now = sdl.GetPerformanceCounter()

		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch e := event.(type) {
			case *sdl.QuitEvent:
				app.running = false
				continue
			case *sdl.KeyboardEvent:
				if e.Type == sdl.KEYDOWN && e.Repeat == 0 {
					isCtrlPressed := (e.Keysym.Mod&sdl.KMOD_CTRL) != 0 || (e.Keysym.Mod&sdl.KMOD_GUI) != 0
					if e.Keysym.Sym == sdl.K_c && isCtrlPressed {
						logger.LogInfo("User attemped to exit program!")
						continue
					}
					keybinds.PerformAction(e)
				}
			}

			// Always use the current top of the stack for event handling
			if len(app.pageStack) > 0 {
				if err := app.pageStack[len(app.pageStack)-1].HandleEvent(&event); err != nil {
					logger.LogWarningF("Error handling event in page: %v", err)
				}
			}
		}

		if !app.running {
			break
		}

		currentTime := time.Now()
		deltaTime := float32(currentTime.Sub(lastTime).Seconds())
		lastTime = currentTime

		// Always use the current top of the stack for updating
		if len(app.pageStack) > 0 {
			app.pageStack[len(app.pageStack)-1].Update(deltaTime)
		}

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		gl.ClearColor(0.0, 0.0, 0.0, 1.0) // Black background
		if len(app.pageStack) > 0 {
			app.pageStack[len(app.pageStack)-1].Render()
		}

		app.Window.GLSwap()
		app.applyPendingAction()
		frameTime := sdl.GetPerformanceCounter() - now/sdl.GetPerformanceFrequency()
		if float64(frameTime) < frameDelay {
			sdl.Delay(uint32(frameDelay/float64(frameTime)) * 1000)
		}
	}
	return nil
}

func (app *Application) initKeybinds() {
	keybinds.Register(
		sdl.K_F3,
		sdl.KMOD_NONE,
		func() {
			app.PushPage(&page.Settings{})
		},
	)

	keybinds.Register(
		sdl.K_F2,
		sdl.KMOD_NONE,
		func() {
			app.PushPage(&page.ScrollerPage{})
		},
	)

	keybinds.Register(
		sdl.K_F1,
		sdl.KMOD_NONE,
		func() {
			app.UnwindToPage(&page.Welcome{})
		},
	)

	keybinds.Register(
		sdl.K_q,
		sdl.KMOD_RCTRL,
		func() {
			app.Stop()
		},
	)

	// Volume Up
	keybinds.Register(
		sdl.K_UP,
		sdl.KMOD_SHIFT,
		func() {
			logger.LogInfoF("Increasing volume %d + 16", mixer.GetVolume128())
			mixer.SetVolume128(16)
		},
	)

	// Volume Down
	keybinds.Register(
		sdl.K_DOWN,
		sdl.KMOD_SHIFT,
		func() {
			logger.LogInfoF("Decreasing volume %d - 16", mixer.GetVolume128())
			mixer.SetVolume128(-16)
		},
	)
}

func (app *Application) Stop() {
	app.running = false
}

// ----------------------------------------------------------------------------
// Page Management Logic
// ----------------------------------------------------------------------------

// currentPage is a private helper method that gets the page at the top of the
// stack without needing a dedicated field. It returns nil if the stack is empty.
func (app *Application) currentPage() page.Page {
	if len(app.pageStack) == 0 {
		return nil
	}
	return app.pageStack[len(app.pageStack)-1]
}

// applyPendingAction executes the scheduled page transition.
func (app *Application) applyPendingAction() {
	if app.pendingAction != nil {
		app.pendingAction()
		app.pendingAction = nil
		// Removed: Do NOT call Init on the new top page here.
		// Initialization is handled by schedulePush/Switch/Unwind.
		// The runEventLoop will naturally pick up the new current page.
	}
}

// schedulePush schedules a new page to be added to the top of the stack.
func (app *Application) schedulePush(p page.Page) {
	// Update the current pendingAction with a function that performs a "push" to the pageStack
	app.pendingAction = func() {
		// Initialize the new page and pass it the current app for configuration
		if err := p.Init(app); err != nil {
			// Log an error if Init fails
			logger.LogErrorF("Failed to initialize new page, aborting push: %v", err)
			return
		}
		// Get the current page at the end of the pageStack and check that it is not nil (pageStack is not empty)
		if topPage := app.currentPage(); topPage != nil {
			// If there is a page, destroy it
			topPage.Destroy()
		}
		// Append the new page to the pageStack
		app.pageStack = append(app.pageStack, p)
		logger.LogInfoF("Pushed new page onto stack: %T", p)
	}
}

// scheduleSwitch schedules the current page to be replaced by a new one.
func (app *Application) scheduleSwitch(p page.Page) {
	app.pendingAction = func() {
		logger := logger.Get()
		// Get the top page using the helper method and destroy it.
		if topPage := app.currentPage(); topPage != nil {
			topPage.Destroy()
			app.pageStack = app.pageStack[:len(app.pageStack)-1]
		}
		// Push the new page.
		if err := p.Init(app); err != nil {
			logger.Errorf("Failed to initialize new page, aborting switch: %v", err)
			return
		}
		app.pageStack = append(app.pageStack, p)
		logger.Infof("Switched to new page: %T", p)
	}
}

// scheduleUnwind schedules the removal of all pages above a target page.
func (app *Application) scheduleUnwind(targetPage page.Page) {
	app.pendingAction = func() {
		foundIndex := -1
		for i := len(app.pageStack) - 1; i >= 0; i-- {
			if app.pageStack[i] == targetPage {
				foundIndex = i
				break
			}
		}

		if foundIndex != -1 {
			for i := len(app.pageStack) - 1; i > foundIndex; i-- {
				app.pageStack[i].Destroy()
			}
			app.pageStack = app.pageStack[:foundIndex+1]
		} else {
			if topPage := app.currentPage(); topPage != nil {
				topPage.Destroy()
				app.pageStack = app.pageStack[:len(app.pageStack)-1]
			}
			if err := targetPage.Init(app); err != nil {
				return
			}
			app.pageStack = append(app.pageStack, targetPage)
		}
	}
}

// --------------------------------------------------------
// ApplicationInterface implementation
// --------------------------------------------------------

// PushPage adds a new page to the pageStack
// Simple page switches should use SwitchPage instead of PushPage.
// PushPage is not intended to switch pages, but to create a page 'history'
func (app *Application) PushPage(p page.Page) { app.schedulePush(p) }

// SwitchPage switches the current pageStack with a single, new page
// This is the method that should normally be called when switching pages.
func (app *Application) SwitchPage(p page.Page) { app.scheduleSwitch(p) }

// UnwindToPage reduces the pageStack, no matter the length, back to a specific page.
// Think of this function as the "back button" in a web browser, except that you get to
// pick which page you return to along the stack.
func (app *Application) UnwindToPage(p page.Page) { app.scheduleUnwind(p) }

// PopPage pops the last page off of the stack
// This is intended to be called from pages in a stack, since the page being viewed
// will be the last page on the stack, therefore, itself.
func (app *Application) PopPage() {
	app.pendingAction = func() {
		if topPage := app.currentPage(); topPage != nil {
			topPage.Destroy()
			app.pageStack = app.pageStack[:len(app.pageStack)-1]
			logger.Get().Info("Popped page from stack.")
		}
	}
}

// GetDrawableSize queries the current window for the current OpenGL Context Drawable Size
// Typically results in the dimensions of the screen.
func (app *Application) GetDrawableSize() (w int32, h int32) {
	return app.Window.GLGetDrawableSize()
}

// ----------------------------------------------------------------------------
// Other functions
// ----------------------------------------------------------------------------

// Close cleans up application resources for a clean exit
func (app *Application) Close() {
	for _, page := range app.pageStack {
		page.Destroy()
	}
	app.pageStack = nil
}

// Validate performs a series of checks to ensure the application is ready to run.
func (app *Application) Validate() error {
	logger.LogInfoF("Validating program state")
	report := &ValidationReport{
		Checks:   make(map[ValidationCheck]CheckResult),
		AllValid: true,
	}

	appConfig := config.Get() // Get the singleton instance

	welcomePage := &page.Welcome{}
	welcomePage.Init(app)
	app.pageStack = append(app.pageStack, welcomePage)

	// Check if a network device is found, and if we are connected to a network with internet access
	// Initialize manager once
	netManager, err := network.New()
	if err != nil {
		return fmt.Errorf("failed to initialize network manager: %w", err)
	}
	defer func() {
		logger.LogInfoF("Closing network manager.")
		netManager.Close()
	}()

	// Wait for NM to connect (or timeout). Tune these as you like.
	const (
		maxWait   = 25 * time.Second // total time budget at boot
		stableFor = 2 * time.Second  // require "up" to be stable briefly
	)

	status, err := netManager.WaitForInternet(maxWait, stableFor)
	if err != nil {
		// Not fatal by itself; we still have a status to report below.
		logger.LogInfoF("Network wait error: %v", err)
	}

	isOnline := (status.StatusCode == network.StatusEthernetConnectedInternetUp ||
		status.StatusCode == network.StatusWifiConnectedInternetUp)

	report.Checks[NetworkConnectionCheck] = CheckResult{isOnline, status.Message}
	if !isOnline {
		report.AllValid = false
		wifiSetupPage := &page.WiFiSetupPage{}
		wifiSetupPage.Init(app)
		app.pageStack = append(app.pageStack, wifiSetupPage)
	}

	if appConfig.LicenseKey == "" {
		report.Checks[LicenseKeyCheck] = CheckResult{false, "License key is missing from config."}
		report.AllValid = false
		licenseKeyPage := &page.LicenseKeyPage{}
		licenseKeyPage.Init(app)
		app.pageStack = append(app.pageStack, licenseKeyPage)
	} else {
		report.Checks[LicenseKeyCheck] = CheckResult{true, "License key found."}
	}

	if appConfig.EmailAddress == "" {
		report.Checks[EmailAddressCheck] = CheckResult{false, "Email address is missing from config."}
		report.AllValid = false
		emailAddressPage := &page.EmailAddressPage{}
		emailAddressPage.Init(app)
		app.pageStack = append(app.pageStack, emailAddressPage)

	} else {
		report.Checks[EmailAddressCheck] = CheckResult{true, "Email address found."}
	}

	if report.AllValid {
		report.Checks[SubscriptionCheck] = CheckResult{true, "Subscription status is valid (Simulated)."}
	} else {
		report.Checks[SubscriptionCheck] = CheckResult{false, "Skipped due to previous validation failures."}
	}
	for checkType, result := range report.Checks {
		logger.LogInfoF("Validation Report: %s : %s", checkType.String(), result.Message)
	}
	app.ValidationReport = report
	return nil
}

func (vc ValidationCheck) String() string {
	switch vc {
	case EmailAddressCheck:
		return "EmailAddressCheck"
	case LicenseKeyCheck:
		return "LicenseKeyCheck"
	case NetworkConnectionCheck:
		return "NetworkConnectionCheck"
	case SubscriptionCheck:
		return "SubscriptionCheck"
	default:
		return "UnknownCheck"
	}
}

func initializeSDL(logger *logger.Logger) error {
	// os.Setenv("SDL_VIDEODRIVER", "kmsdrm") // This is done in the kiosk-session as ENV variables
	if err := sdl.Init(sdl.INIT_VIDEO | sdl.INIT_AUDIO); err != nil {
		logger.Errorf("failed to initialize critical SDL subsystems: %v", err)
		return err
	}
	logger.Info("Successfully initialized VIDEO and AUDIO subsystems")
	logger.Infof("SDL audio backend: %s", sdl.GetCurrentAudioDriver())
	if err := ttf.Init(); err != nil {
		logger.Errorf("failed to initialize critical component TTF : %v", err)
		return fmt.Errorf("failed to initialize critical component TTF : %v", err)
	}
	logger.Info("Successfully initialized TTF")

	sdl.GLSetAttribute(sdl.GL_CONTEXT_MAJOR_VERSION, 3)
	sdl.GLSetAttribute(sdl.GL_CONTEXT_MINOR_VERSION, 3)
	sdl.GLSetAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_CORE)
	sdl.ShowCursor(sdl.DISABLE)
	sdl.SetRelativeMouseMode(true)

	return nil
}

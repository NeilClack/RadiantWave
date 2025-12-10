# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

RadiantWave is a Go-based fullscreen SDL2/OpenGL application designed to run on Linux kiosk systems. It displays affirmations with audio playback and animated visual patterns. The application uses a page-based navigation system and includes network connectivity validation, license key management, and an auto-update mechanism.

## Build & Deployment

RadiantWave uses Debian packages (.deb) for distribution and updates. There are two package variants:
- **`radiantwave`** - Release channel (stable versions from git tags)
- **`radiantwave-dev`** - Development channel (versions from commit hashes)

These packages conflict with each other (only one can be installed at a time).

### Building Packages

```bash
# Build dev package (version = 0.0.{commit}, e.g., 0.0.ab01651)
./build.sh dev

# Build release package (version = git tag without 'v', e.g., 2.0.0)
./build.sh release
```

The build script:
1. Compiles the binary with CGO enabled (required for SDL2)
2. Embeds version info via ldflags into `internal/page.GitVersion`
3. Templates the updater script with the package name (`radiantwave` or `radiantwave-dev`)
4. Sets file permissions for installed files
5. Creates staging directory with proper Debian package structure
6. Generates DEBIAN/control file with package metadata and dependencies
7. Builds .deb package using `dpkg-deb`
8. Generates SHA256 checksum

**Output files:**
- `radiantwave_2.0.0_amd64.deb` (release)
- `radiantwave-dev_0.0.ab01651_amd64.deb` (dev)
- Corresponding `.sha256` files
- All files set to 644 permissions (readable by _apt user)

### Installation

**Manual installation:**
```bash
sudo dpkg -i radiantwave_2.0.0_amd64.deb
sudo apt install -f  # Fix any dependency issues
```

**Via apt repository (recommended):**
```bash
# Add repository (one-time setup)
echo "deb [trusted=yes] https://repository.radiantwavetech.com/release ./" | \
  sudo tee /etc/apt/sources.list.d/radiantwave.list

# Install
sudo apt update
sudo apt install radiantwave
```

For dev channel, use `https://repository.radiantwavetech.com/dev` and install `radiantwave-dev`.

### Auto-Updates

The updater script (`/usr/local/bin/radiantwave-updater.py`) uses apt to check for and install updates:
1. Runs `pkexec apt update` to refresh package lists
2. Checks if upgrade is available using `apt list --upgradable`
3. Runs `pkexec apt install --only-upgrade -y <package>` to upgrade

Polkit rules allow the `kiosk` user to run these apt commands without a password.

### Testing
There are currently no automated tests in this codebase.

## Architecture

### Application Entry Point
- `radiantwave.go` is the main entry point, calls `internal/application.Run()`
- `internal/application/application.go` contains the core application lifecycle

### Core Application Flow
1. **Initialization**
   - Create application data directory (`/usr/local/share/radiantwave`)
   - Initialize SQLite database (`data.db`) with GORM
   - Initialize SDL2 subsystems (VIDEO, AUDIO)
   - Initialize SDL_ttf for font rendering
   - Create fullscreen OpenGL 3.3 Core Profile window
   - Initialize shader and font managers
   - Hide cursor and enable relative mouse mode

2. **Validation Phase**
   - Check network connectivity (Ethernet or WiFi)
   - Validate license key (pushes LicenseKeyPage if missing)
   - Validate email address (pushes EmailAddressPage if missing)
   - Check subscription status
   - If validation fails, appropriate setup pages are pushed onto the page stack

3. **Event Loop** (60 FPS target)
   - Poll SDL events
   - Handle keyboard events via keybinds system
   - Update current page with delta time
   - Render current page
   - Apply pending page transitions

### Page System
Pages implement the `page.Page` interface:
- `Init(app ApplicationInterface) error` - Setup resources
- `HandleEvent(event *sdl.Event) error` - Handle input
- `Update(dt float32) error` - Update state
- `Render() error` - Draw to screen
- `Destroy() error` - Cleanup resources

Pages can embed `page.Base` for common functionality (OpenGL quad rendering, projection matrix, screen dimensions).

**Page Navigation:**
- `PushPage(p)` - Add page to stack (creates history)
- `SwitchPage(p)` - Replace current page
- `UnwindToPage(p)` - Pop back to a specific page (or create if not in stack)

**Available Pages:**
- `Welcome` - Initial screen showing version
- `ScrollerPage` - Main affirmation player (F2)
- `Settings` - Settings menu (F3)
- `WiFiSetupPage` - Network configuration
- `LicenseKeyPage` - License key entry
- `EmailAddressPage` - Email address entry
- `AudioDevices` - Audio device selection
- `AffirmationOptions` - Affirmation selection

### Key Global Keybinds
Defined in `application.initKeybinds()`:
- `F1` - Navigate to Welcome page
- `F2` - Navigate to Player (ScrollerPage)
- `F3` - Navigate to Settings
- `SHIFT+UP` - Increase volume
- `SHIFT+DOWN` - Decrease volume
- `RIGHT_CTRL+Q` - Quit application

### Package Structure

**`internal/application`** - Core application lifecycle, event loop, page management, validation

**`internal/page`** - Page interface and all page implementations

**`internal/db`** - Database layer using GORM with SQLite
- Models: `Config`, `Affirmations`, `LogEntry`
- Manages configuration key-value pairs
- Seeds default affirmations from `/usr/local/share/radiantwave/affirmations/`

**`internal/audio`** - SDL2 audio subsystem integration (currently minimal wrapper)

**`internal/mixer`** - Audio playback manager for music and affirmations

**`internal/graphics`** - OpenGL utilities and display orientation handling

**`internal/shaderManager`** - Loads and manages OpenGL shaders from `/usr/local/share/radiantwave/shaders/`

**`internal/fontManager`** - Loads and manages TrueType fonts from `/usr/local/share/radiantwave/fonts/`

**`internal/keybinds`** - Global keyboard shortcut registration and dispatch

**`internal/network`** - Network connectivity manager (uses D-Bus for NetworkManager on Linux)

**`internal/logger`** - Logging system that writes to database and stdout

**`internal/colors`** - Color utilities

**`internal/pattern`** - Visual pattern generation (line patterns, color patterns)

### Deployment Structure

The `system/` directory mirrors the Linux filesystem and contains all deployed files:

```
system/
├── usr/local/bin/
│   ├── radiantwave                    # Main binary
│   ├── radiantwave-updater.py         # Auto-updater script (templated with package name)
│   └── scripts/
│       └── post-install.sh            # Post-installation tasks (sets group ownership)
├── usr/local/share/radiantwave/
│   ├── affirmations/                  # Affirmation text files
│   │   ├── home/                      # Non-commercial affirmations
│   │   └── commercial/                # Commercial affirmations
│   ├── audio/                         # Audio files (7.83.wav, etc.)
│   ├── fonts/                         # TrueType fonts
│   ├── shaders/                       # GLSL shader files
│   └── VERSION                        # Current version file
└── etc/polkit-1/rules.d/
    └── 99-radiantwave-updater.rules   # PolicyKit rules for apt update/upgrade
```

The `debian/` directory contains package metadata templates:
```
debian/
├── control.template   # Package metadata (templated with package name, version, conflicts)
├── postinst           # Post-installation script (runs post-install.sh)
└── prerm              # Pre-removal script
```

### Important Implementation Details

**OpenGL Context:** Uses OpenGL 3.3 Core Profile with VSync enabled. The page base class sets up a reusable quad VAO/VBO for texture and solid color rendering.

**Database Location:** `/usr/local/share/radiantwave/data.db` - Contains configuration, affirmations, and log entries

**Fullscreen Mode:** Uses `SDL_WINDOW_FULLSCREEN_DESKTOP` to match native display resolution

**Display Orientation:** Handled by `graphics.SetDisplayOrientation()` which swaps width/height for portrait displays

**Version Injection:** Git version is injected at build time via ldflags: `-X 'radiantwavetech.com/radiantwave/internal/page.GitVersion=${VERSION}'`
- Release builds: Use git tag without 'v' prefix (e.g., `2.0.0` from tag `v2.0.0`)
- Dev builds: Use `0.0.{commit}` format (e.g., `0.0.ab01651`) - the `0.0.` prefix ensures Debian compliance (versions must start with digit)

**Auto-updater:** Python script that uses apt to check for and install updates:
- Runs `pkexec apt update` to refresh package lists
- Checks `apt list --upgradable` to see if update is available
- Runs `pkexec apt install --only-upgrade -y <package>` to install update
- Polkit rules allow kiosk user to run these specific apt commands without password

**Package Management:** Uses Debian packaging system with two package variants (`radiantwave` and `radiantwave-dev`) that conflict with each other

**Network Manager:** Uses D-Bus to communicate with NetworkManager for WiFi configuration and connectivity status

## Development Workflow

### Adding a New Page
1. Create `internal/page/<pagename>.go`
2. Implement the `page.Page` interface
3. Embed `page.Base` for common functionality
4. Register keybind in `application.initKeybinds()` if needed
5. Add navigation logic from existing pages

### Modifying Database Schema
1. Update models in `internal/db/db.go`
2. GORM AutoMigrate will handle schema changes on next run
3. Add seed data logic if needed in `seedDefaults()` or `seedDefaultAffirmations()`

### Adding New Assets
1. Place files in appropriate `system/usr/local/share/radiantwave/` subdirectory
2. Build script will include them in tarball with correct permissions
3. Access via config values (`assets_dir`) or direct paths

### Working with Shaders
1. Create `.vert` and `.frag` files in `system/usr/local/share/radiantwave/shaders/`
2. Load via `shaderManager.GetShader("<name>")`
3. Shader must have uniforms: `u_mvpMatrix`, `u_color` or `u_textColor`, `u_texture` (if using textures)

## Important Constraints

- Must build with `CGO_ENABLED=1` for SDL2 bindings
- Target platform is Linux (uses D-Bus for NetworkManager)
- Application expects to run as kiosk user with polkit rules for system operations
- All paths are absolute under `/usr/local/` for system-wide installation
- Version string comes from git tags (release) or commit hash (dev)

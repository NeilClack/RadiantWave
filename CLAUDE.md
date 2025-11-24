# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
# Build the application (interactive - prompts for confirmation)
./build.sh [SYSTEM_TYPE] [RELEASE_TYPE]

# Examples:
./build.sh                    # Builds home/dev (default)
./build.sh home release       # Builds home/release
./build.sh commercial beta    # Builds commercial/beta

# SYSTEM_TYPE: home | commercial
# RELEASE_TYPE: dev | beta | release (auto-detected from git branch if omitted)
```

The build script:
- Compiles with CGO enabled and embeds version info via ldflags
- Packages binary + assets into a tarball under `pkgroot/`
- Creates `radiantwave-{type}-{version}.tar.xz`

## Architecture Overview

**RadiantWave** is a Go desktop application using SDL2 and OpenGL 3.3 to display affirmations with synchronized visual patterns.

### Core Components

- **Entry point**: `radiantwave.go` → `internal/application/application.go`
- **Page system**: `internal/page/` - UI screens implementing the `Page` interface (Init, HandleEvent, Update, Render, Destroy)
- **Database**: `internal/db/` - GORM/SQLite storing config, affirmations, and logs at `~/.radiantwave/data.db`
- **Graphics**: `internal/graphics/`, `internal/fontManager/`, `internal/shaderManager/` - OpenGL rendering pipeline
- **Pattern generation**: `internal/pattern/` - Fibonacci-based frequency patterns for spatial/temporal modulation
- **Network**: `internal/network/` - D-Bus integration with NetworkManager for WiFi

### Key Pages

- `welcome.go` - Home screen with license info
- `scrollerPage.go` - Main affirmation display with Radiant Wave patterns
- `settingsPage.go` - Configuration menu
- `wifiSetupPage.go` - WiFi network selection

### Application Flow

1. Initialize SDL/OpenGL and database
2. Validate network, license, subscription
3. Run 60 FPS event loop with page stack navigation
4. Keybinds: F1=Welcome, F2=Scroller, F3=Settings, Ctrl+Q=Exit

### Data Paths

- User data: `~/.radiantwave/`
- Assets: `~/.local/share/radiantwave/`
- Affirmations source: `assets/affirmations/`

## Dependencies

- Go 1.24+ with CGO
- SDL2, SDL2_ttf, SDL2_mixer
- OpenGL 3.3

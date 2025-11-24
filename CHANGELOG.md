# Radiant Wave Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Added first-time setup flow for Email and License Key pages with mandatory completion
- Added email validation with regex pattern matching and red error message display
- Added license key validation requiring 16 alphanumeric characters with red error message display
- Added `--release` flag to build.sh to distinguish local dev builds from release builds
- Added CLAUDE.md documentation with build commands and architecture overview
- Added comprehensive logging for audio device operations (initialization, switching, failures)
- Added device validation before saving audio device selection

### Changed
- Changed database and assets path from `/usr/local/` to `~/.local/share/radiantwave/` for XDG compliance
- Changed build.sh local builds to extract to `$HOME` without requiring sudo
- Changed build.sh release builds to extract to `/` with `localuser` paths
- Changed usb/setup.sh to fix ownership with chown after extraction
- Changed audio mixer to use system default SDL audio driver instead of hardcoded ALSA
- Changed audio device selection to validate device before saving to database

### Deprecated
### Removed
- Removed hardcoded ALSA preference from mixer to support PulseAudio, PipeWire, and other audio systems

### Fixed
- Fixed bug in Email and License key pages causing text in Settings to become solid white rectangles
- Fixed error message positioning in Email and License Key pages to not overlap with current value display
- Fixed permissions and ownership issues in build script
- Fixed upload commands in build.sh (uncommented)
- Fixed audio device selection page using wrong package (audio instead of mixer)
- Fixed silent fallback to default audio device - now logs warnings and errors appropriately
- Fixed missing error recovery in audio device switching - now attempts to restore previous device on failure

### Security
---

## [v0.1.2]

### Added
### Changed
- Changed database path from `~/.radiantwave/` to `~/.local/share/radiantwave/` for XDG compliance
- Changed build.sh local builds to extract to `$HOME` without requiring sudo
- Changed build.sh release builds to extract to `/` with `localuser` paths
- Changed usb/setup.sh to fix ownership with chown after extraction

### Deprecated
### Removed
### Fixed  
- Fixed bug in Email and License key pages causing text in Settings to become solid white rectangles.  
- Fixed error message positioning in Email and License Key pages to not overlap with current value display

### Security

---  

## [v0.1.2]

### Added  
- Added first-time setup flow for Email and License Key pages with mandatory completion
- Added email validation with regex pattern matching and error display
- Added license key validation requiring 16 alphanumeric characters with error display
- Added `--release` flag to build.sh to distinguish local dev builds from release builds
- Added CLAUDE.md documentation with build commands and architecture overview


---  

## [v0.1.1]

### Added  
- Settings menu options for editing email address and license key
- Current email address and license key display on their respective configuration pages
- License information display on welcome page (email address and license key)
- Automatic dash formatting for license key input (XXXX-XXXX-XXXX-XXXX format)
- Database logging support for application logs

### Changed  
- Improved WiFi configuration page user experience
- Updated WiFi page background color
- Enhanced affirmation selection user interface

### Fixed 
- Database configuration values not saving due to UNIQUE constraint conflict in `SetConfigValue` function

### Deprecated  
### Removed  
### Security
--- 

## [v0.1.0]  

### Added  
- Added sqlite database with GORM integration
- Added new logger package (`internal/logger`) with singleton pattern and consistent API
- Added database schema for affirmations with `Selected`, `Available`, and `Favorite` fields
- Added `GetAffirmations()` function to retrieve affirmations from database
- Added memory management improvements with proper texture cleanup in Update methods
- Added helper methods in ScrollerPage for line creation (`createLine`, `addLineAtBottom`, `addLineAtTop`)
- Added application configuration constants for FPS, volume step size, and network timeouts
- Added centralized `cleanup()` function for proper resource management
- Added dedicated initialization functions: `initializeApplicationDirectory()`, `initializeDatabase()`, `initializeSDL()`, `createWindow()`, `initializeOpenGL()`, `initializeManagers()`, `initializeAudio()`
- Added separate event handling function `handleEvents()` for cleaner event loop
- Added `updateCurrentPage()` and `renderFrame()` helper functions
- Added validation helper functions: `validateNetwork()`, `validateLicenseKey()`, `validateEmailAddress()`, `validateSubscription()`

### Changed  
- **BREAKING**: Migrated from config package to db package for all configuration storage
- **PERFORMANCE**: Fixed critical frame timing bug - reduced CPU usage from ~100% to ~12%
- **PERFORMANCE**: Replaced incorrect SDL performance counter calculation with Go's `time` package for accurate 60 FPS limiting
- **PERFORMANCE**: Optimized event loop with reduced redundant checks and better frame timing
- Refactored all page files to use db package instead of config:
  - `AffirmationOptions`: Now loads affirmations from database with selection state
  - `AudioDevices`: Uses database for device settings and font configuration
  - `EmailAddressPage`: Stores email in database instead of config file
  - `LicenseKeyPage`: Stores license key in database instead of config file
  - `ScrollerPage`: Loads affirmations from database with major performance optimizations
  - `Settings`: Uses database for font size configuration
  - `Welcome`: Uses database for font size configuration
  - `WiFiSetupPage`: Uses database for font and network configuration
- Refactored `shaderManager` to load shader paths from database
- Refactored `application.go` for improved maintainability:
  - Broke down 180-line `Run()` function into 15+ focused, single-responsibility functions
  - Extracted magic numbers into named constants (`targetFPS`, `volumeStepSize`, `networkMaxWait`, etc.)
  - Reorganized code into logical sections with clear separators
  - Simplified page management logic with consistent helper method usage
- Improved `ValidationCheck.String()` to use map-based lookup instead of switch statement
- Standardized logger calls across entire codebase:
  - All instances now use `logger.InfoF()`, `logger.ErrorF()`, `logger.WarningF()`, `logger.DebugF()`, `logger.FatalF()`
  - Removed inconsistent logging patterns (`logger.Get()`, `logger.LogInfo()`, `logger.LogInfoF()`)
- Renamed database functions for clarity: `GetValue` → `GetConfigValue`, `GetValues` → `GetConfigValues`, `SetValue` → `SetConfigValue`
- Improved error handling with proper error wrapping using `fmt.Errorf` with `%w` verb throughout application
- ScrollerPage optimizations:
  - Eliminated file I/O by loading affirmations directly from database
  - Cached font size calculations (converted once at init instead of repeated parsing)
  - Pre-calculated space width during initialization
  - Reduced code duplication with extracted helper methods

### Removed  
- Removed config package dependencies from all page files
- Removed filesystem-based affirmation loading in favor of database queries
- Removed all deprecated logging functions and inconsistent logger patterns
- Removed `p.config` and `p.logger` fields from page structs (now use singletons directly)
- Removed incorrect SDL performance counter arithmetic that caused frame timing issues

### Fixed  
- Fixed critical frame timing bug causing 100% CPU usage on single core
- Fixed incorrect frame delay calculation using SDL performance counters
- Fixed memory leaks in `EmailAddressPage` and `LicenseKeyPage` (now properly delete old textures in Update)
- Fixed incorrect Destroy message in `Welcome` page (was saying "LicenseKeyPage")
- Fixed missing texture cleanup in `Settings` page options
- Fixed missing note texture cleanup in `Welcome` page Destroy method
- Fixed logger level mismatches (warnings now use `WarningF`, errors use `ErrorF`)
- Fixed slow program startup
- Fixed resource cleanup order in application shutdown  
- Fixed audio filepath in `Scroller`

---  

## [v0.0.3]  

### Added  
- Added user manual markdown page  
- Added logs to volume controls  

### Changed   
- Changed default volume from 0% to 100%  
- Applied high-pass filter to audio for better speaker compatibility  

### Fixed  
- Fixed audio devices not switching when selected  
- Fixed volume reseting to zero
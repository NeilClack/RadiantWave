# Radiant Wave Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
### Changed
### Deprecated
### Removed
### Fixed  
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

### Changed  
- **BREAKING**: Migrated from config package to db package for all configuration storage
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
- Standardized logger calls across entire codebase:
  - All instances now use `logger.InfoF()`, `logger.ErrorF()`, `logger.WarningF()`, `logger.DebugF()`, `logger.FatalF()`
  - Removed inconsistent logging patterns (`logger.Get()`, `logger.LogInfo()`, `logger.LogInfoF()`)
- Renamed database functions for clarity: `GetValue` → `GetConfigValue`, `GetValues` → `GetConfigValues`, `SetValue` → `SetConfigValue`
- Improved error handling with proper error wrapping using `fmt.Errorf` with `%w` verb
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

### Fixed  
- Fixed memory leaks in `EmailAddressPage` and `LicenseKeyPage` (now properly delete old textures in Update)
- Fixed incorrect Destroy message in `Welcome` page (was saying "LicenseKeyPage")
- Fixed missing texture cleanup in `Settings` page options
- Fixed missing note texture cleanup in `Welcome` page Destroy method
- Fixed logger level mismatches (warnings now use `WarningF`, errors use `ErrorF`)
- Fixed slow program startup

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
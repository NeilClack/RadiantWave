// Package config handles the loading and management of application configuration.
// It uses a singleton pattern to ensure that there is only one configuration
// object active throughout the application's lifecycle.
package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

var SystemType string

// Config holds the application's configuration.
// Fields that are not persisted to disk are marked with `json:"-"`.
type Config struct {
	// Runtime-only fields (not saved in settings.json)
	HomeDir   string `json:"-"`
	AssetsDir string `json:"-"` // System-wide assets, e.g., /opt/RadiantWave
	LogDir    string `json:"-"`

	// Persisted fields (saved in settings.json)
	LicenseKey            string   `json:"license_key"`
	SystemCapabilities    string   `json:"system_capabilities"`
	SystemType            string   `json:"system_type"` // Is this a "commercial" system, or a "home" system?
	EmailAddress          string   `json:"email_address"`
	SelectedFilePaths     []string `json:"selected_file_paths"`
	SelectedAudioFilePath string   `json:"selected_audio_paths"`
	AudioDeviceName       string   `json:"audio_device_name"`
	ApplicationFont       string   `json:"application_font"`   // Used to store the name of the font to use in Application text
	BaseFontSize          int      `json:"font_size"`          // Used to create the font map with a large size in order to scale down to StandardFontSize for crips text
	StandardFontSize      int32    `json:"standard_font_size"` // Used for standard text and represents the visual font size seen by the end-user
	LinePattern           string   `json:"line_pattern"`
	DisplayOrientation    int      `json:"display_orientation"` // 1 = standard horizontal 2 = vertical
	LastVolume            int      `json:"last_volume"`         // 0..128, last volume set by the user
}

var (
	instance *Config
	once     sync.Once
)

// Get returns the singleton instance of the application configuration.
// On its first call, it initializes the configuration by loading it from
// ~/.radiantwave/settings.json. If the file does not exist, it's created
// with default values.
func Get() *Config {
	once.Do(func() {
		instance = &Config{}
		if err := instance.Load(); err != nil {
			// If configuration fails to load, the application cannot run correctly.
			log.Fatalf("FATAL: could not load configuration: %v", err)
		}
	})
	return instance
}

// Load reads the configuration from ~/.radiantwave/settings.json.
// If the file doesn't exist, it applies default settings and saves the new file.
// It is called automatically on first access via Get().
func (c *Config) Load() error {
	log.Println("Loading application configuration...")

	// 1) Resolve paths
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("config: could not determine user home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".radiantwave")
	configFilePath := filepath.Join(configDir, "settings.json")

	// Runtime-only paths
	c.LogDir = filepath.Join(homeDir, ".radiantwave")

	// 2) Ensure config dir exists
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("config: could not create config directory %s: %w", configDir, err)
	}

	// 3) Try to read existing settings (may not exist on first run)
	data, readErr := os.ReadFile(configFilePath)
	fileMissing := false
	if readErr != nil {
		if !os.IsNotExist(readErr) {
			return fmt.Errorf("config: failed to read settings file %s: %w", configFilePath, readErr)
		}
		fileMissing = true
		log.Println("settings.json not found, will create with defaults.")
	}

	// 4) If we have JSON, probe for key existence and unmarshal into c exactly once
	raw := map[string]json.RawMessage{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("config: could not parse json (raw probe): %w", err)
		}
		if err := json.Unmarshal(data, c); err != nil {
			return fmt.Errorf("config: could not parse json from %s: %w", configFilePath, err)
		}
	}

	// 5) Runtime-only fields; keep your existing package logic
	c.HomeDir = homeDir
	c.AssetsDir = "/usr/local/share/radiantwave"

	if SystemType == "" {
		c.SystemType = "home"
	} else {
		c.SystemType = SystemType
	}

	// 6) Apply defaults strictly based on "did the key exist?"
	// Treat first-run (no file) as "all keys missing".
	defaultsApplied := false
	missing := func(key string) bool { return fileMissing || raw[key] == nil }

	// selected_file_paths
	if missing("selected_file_paths") {
		var def string
		switch c.SystemType {
		case "home":
			def = filepath.Join(c.AssetsDir, "affirmations", "home", "Relaxation.txt")
		case "commercial":
			def = filepath.Join(c.AssetsDir, "affirmations", "commercial", "Standard.txt")
		default:
			def = filepath.Join(c.AssetsDir, "affirmations", "home", "Relaxation.txt")
		}
		c.SelectedFilePaths = []string{def}
		defaultsApplied = true
	}

	// selected_audio_paths  (note: tag is plural per your struct)
	if missing("selected_audio_paths") {
		c.SelectedAudioFilePath = filepath.Join(c.AssetsDir, "audio", "7.83.wav")
		defaultsApplied = true
	}

	// application_font
	if missing("application_font") {
		c.ApplicationFont = "Roboto-Regular"
	}

	// font_size (BaseFontSize)
	if missing("font_size") {
		c.BaseFontSize = 128
	}

	// standard_font_size
	if missing("standard_font_size") {
		c.StandardFontSize = 32
	}

	// line_pattern
	if missing("line_pattern") {
		c.LinePattern = "fibonacci"
	}

	// display_orientation
	if missing("display_orientation") {
		c.DisplayOrientation = 1
	}

	// last_volume  (range validation happens elsewhere; here we only set when absent)
	if missing("last_volume") {
		c.LastVolume = 96 // 75% of 128
	}

	// 7) Persist when first run or when first-run defaults were applied
	if fileMissing || defaultsApplied {
		log.Println("Applying default settings and saving configuration.")
		if saveErr := c.Save(); saveErr != nil {
			return fmt.Errorf("config: failed to save initial/default settings: %w", saveErr)
		}
	}

	log.Printf("Configuration loaded successfully from %s", configDir)
	return nil
}

// Save writes the current configuration state to ~/.radiantwave/settings.json.
func (c *Config) Save() error {
	// HomeDir should be populated by Load(), but we check for safety.
	if c.HomeDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("config: could not determine user home directory for saving: %w", err)
		}
		c.HomeDir = homeDir
	}

	configFilePath := filepath.Join(c.HomeDir, ".radiantwave", "settings.json")

	// Ensure the directory exists, as Save() could theoretically be called before Load().
	if err := os.MkdirAll(filepath.Dir(configFilePath), 0755); err != nil {
		return fmt.Errorf("config: could not create directory for saving: %w", err)
	}

	// Marshal the config struct into pretty-printed JSON.
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("config: could not marshal config to json: %w", err)
	}

	// Write the JSON data to the file.
	log.Printf("Saving configuration to %s", configFilePath)
	return os.WriteFile(configFilePath, data, 0644)
}

// Update saves the current in-memory configuration to disk and then immediately
// reloads it from the file. This is useful for ensuring the application state is
// synchronized with the settings.json file after making programmatic changes.
func (c *Config) Update() error {
	log.Println("Updating configuration...")
	if err := c.Save(); err != nil {
		return fmt.Errorf("config: failed to save during update: %w", err)
	}
	// Reload the configuration from the file to ensure consistency.
	return c.Load()
}

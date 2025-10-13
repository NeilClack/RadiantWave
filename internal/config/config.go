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

	// 1) Paths
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("config: could not determine user home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".radiantwave")
	configFilePath := filepath.Join(configDir, "settings.json")

	c.LogDir = filepath.Join(homeDir, ".radiantwave")

	// 2) Ensure config dir
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("config: could not create config directory %s: %w", configDir, err)
	}

	// 3) Read file (may not exist)
	data, readErr := os.ReadFile(configFilePath)
	fileMissing := false
	if readErr != nil {
		if !os.IsNotExist(readErr) {
			return fmt.Errorf("config: failed to read settings file %s: %w", configFilePath, readErr)
		}
		fileMissing = true
		log.Println("settings.json not found, will create with defaults.")
	}

	// 4) If we have JSON, unmarshal and also probe for field existence
	hasLastVolume := false
	if len(data) > 0 {
		// Probe keys without forcing defaults
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("config: could not parse json (raw probe): %w", err)
		}
		// IMPORTANT: this key must match your struct tag (see note below).
		_, hasLastVolume = raw["last_volume"]

		// Now unmarshal into the struct
		if err := json.Unmarshal(data, c); err != nil {
			return fmt.Errorf("config: could not parse json from %s: %w", configFilePath, err)
		}
	}

	// 5) Populate runtime fields
	c.HomeDir = homeDir
	c.AssetsDir = "/usr/local/share/radiantwave"

	if SystemType == "" {
		c.SystemType = "home"
	} else {
		c.SystemType = SystemType
	}

	// 6) Apply first-run defaults
	defaultsApplied := false
	if len(c.SelectedFilePaths) == 0 {
		var defaultFile string
		switch c.SystemType {
		case "home":
			defaultFile = filepath.Join(c.AssetsDir, "affirmations", "home", "Relaxation.txt")
		case "commercial":
			defaultFile = filepath.Join(c.AssetsDir, "affirmations", "commercial", "Standard.txt")
		default:
			defaultFile = filepath.Join(c.AssetsDir, "affirmations", "home", "Relaxation.txt")
		}
		c.SelectedFilePaths = []string{defaultFile}
		defaultsApplied = true
	}
	if c.SelectedAudioFilePath == "" {
		c.SelectedAudioFilePath = filepath.Join(c.AssetsDir, "audio", "7.83.wav")
		defaultsApplied = true
	}

	// 7) Fonts and visuals
	if c.ApplicationFont == "" {
		c.ApplicationFont = "Roboto-Regular"
	}
	if c.BaseFontSize == 0 {
		c.BaseFontSize = 128
	}
	if c.StandardFontSize == 0 {
		c.StandardFontSize = 32
	}
	if c.LinePattern == "" {
		c.LinePattern = "fibonacci"
	}
	if c.DisplayOrientation == 0 || (c.DisplayOrientation != 1 && c.DisplayOrientation != 2) {
		c.DisplayOrientation = 1
	}

	// 8) LastVolume default ONLY if key was missing (or bad value)
	// Treat first-run (no file) the same as "missing key".
	if !hasLastVolume || c.LastVolume < 0 || c.LastVolume > 128 {
		c.LastVolume = 96                                // 75% of 128
		defaultsApplied = defaultsApplied || fileMissing // saving makes sense on fresh runs
	}

	// 9) Save if first run or we applied defaults
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

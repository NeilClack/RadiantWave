package db

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

type Config struct {
	gorm.Model
	Key   string `gorm:"uniqueIndex"`
	Value string
}

type Affirmations struct {
	gorm.Model
	Title      string
	Content    string
	Commercial bool
	Selected   bool
	Favorite   bool
	Available  bool
}

type LogEntry struct {
	gorm.Model
	Timestamp string
	Level     string
	Message   string
}

var defaultConfigValues = map[string]string{
	"home_dir":            "",
	"assets_dir":          "", // Set dynamically in seedDefaults() to $HOME/.local/share/radiantwave
	"log_dir":             "", // Set dynamically in seedDefaults() to $HOME/.local/share/radiantwave/logs.log
	"email_address":       "",
	"license_key":         "",
	"system_type":         "home",
	"system_capabilities": "",
	"selected_audio":      "audio/7.83.wav",
	"audio_device_name":   "",
	"application_font":    "Roboto-Regular",
	"init_font_size":      "128",
	"standard_font_size":  "32",
	"line_pattern":        "fibonacci",
	"last_volume":         "128",
}

// InitDatabase initializes the database connection
// and performs auto-migration for all models
func InitDatabase(dbpath string) error {
	// Create the directory for the database if it doesn't exist
	dbDir := filepath.Dir(dbpath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Printf("Error creating database directory %s: %v", dbDir, err)
		return err
	}

	// Check if database file exists
	_, err := os.Stat(dbpath)
	dbExists := !os.IsNotExist(err)

	if !dbExists {
		log.Printf("Database file does not exist at %s, creating new database", dbpath)
		// Create empty database file
		file, err := os.Create(dbpath)
		if err != nil {
			log.Printf("Error creating database file: %v", err)
			return err
		}
		file.Close()
	}

	DB, err = gorm.Open(sqlite.Open(dbpath), &gorm.Config{})
	if err != nil {
		return err
	}
	log.Println("Database connected successfully")
	err = DB.AutoMigrate(&Config{}, &Affirmations{}, &LogEntry{})
	if err != nil {
		return err
	}
	log.Println("Database migrated successfully")
	err = seedDefaults()
	if err != nil {
		log.Println("Error seeding default configuration values:", err)
		return err
	}
	err = seedDefaultAffirmations()
	if err != nil {
		log.Println("Error seeding default affirmations:", err)
		return err
	}
	log.Println("Database initialized successfully")
	return nil
}

// seedDefaults seeds the database with default configuration values
// default values are set via the defaultConfigValues map
func seedDefaults() error {
	var count int64
	err := DB.Model(&Config{}).Count(&count).Error
	if err != nil {
		log.Println("Error counting configuration entries:", err)
		return err
	}
	if count == 0 {
		shareDir := filepath.Join("/usr", "local", "share", "radiantwave")

		for key, value := range defaultConfigValues {
			// Replace placeholder paths with user-local paths
			if key == "assets_dir" {
				value = shareDir
			} else if key == "log_dir" {
				value = filepath.Join(shareDir, "logs.log")
			}
			config := Config{Key: key, Value: value}
			if err := DB.Create(&config).Error; err != nil {
				return err
			}
		}
		log.Println("Seeded default configuration values")
	}
	return nil
}

// seedDefaultAffirmations reads affirmation text files from the assets directory
// and populates the database on first run
func seedDefaultAffirmations() error {
	var count int64
	err := DB.Model(&Affirmations{}).Count(&count).Error
	if err != nil {
		log.Println("Error counting affirmations:", err)
		return err
	}

	// Only seed if table is empty
	if count > 0 {
		return nil
	}

	// Get the assets directory from config
	assetsDir, err := GetConfigValue("assets_dir")
	if err != nil {
		log.Println("Error getting assets_dir:", err)
		return err
	}

	affirmationsBaseDir := assetsDir + "/affirmations"

	// Process both home and commercial directories
	subdirs := []string{"home", "commercial"}
	totalSeeded := 0

	for _, subdir := range subdirs {
		affirmationsDir := affirmationsBaseDir + "/" + subdir
		isCommercial := (subdir == "commercial")

		// Read all .txt files from the subdirectory
		files, err := os.ReadDir(affirmationsDir)
		if err != nil {
			log.Printf("Error reading affirmations directory %s: %v", affirmationsDir, err)
			continue // Continue with other directories even if one fails
		}

		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".txt") {
				continue
			}

			filePath := affirmationsDir + "/" + file.Name()
			content, err := os.ReadFile(filePath)
			if err != nil {
				log.Printf("Error reading file %s: %v", filePath, err)
				continue
			}

			// Extract title from filename (remove .txt extension)
			title := strings.TrimSuffix(file.Name(), ".txt")

			// Select "Relaxation" as the default affirmation
			isSelected := (title == "Relaxation")

			affirmation := Affirmations{
				Title:      title,
				Content:    string(content),
				Commercial: isCommercial,
				Selected:   isSelected,
				Favorite:   false,
				Available:  true,
			}

			if err := DB.Create(&affirmation).Error; err != nil {
				log.Printf("Error creating affirmation from %s: %v", file.Name(), err)
				return err
			}

			if isSelected {
				log.Printf("Set default affirmation: %s", title)
			}

			totalSeeded++
		}
	}

	log.Printf("Seeded %d default affirmations from %s", totalSeeded, affirmationsBaseDir)
	return nil
}

// GetConfigValue retrieves a configuration value by key
// Returns the value OR any error encountered
func GetConfigValue(key string) (string, error) {
	var config Config
	result := DB.First(&config, "key = ?", key)
	if result.Error != nil {
		return "", result.Error
	}
	return config.Value, nil
}

// GetConfigValues retrieves all configuration key-value pairs
// Returns a map of key-value pairs OR any error encountered
// This is mainly for compatibility with legacy code, but can
// be useful for bulkl config referrals
func GetConfigValues() (map[string]string, error) {
	var configs []Config
	result := DB.Find(&configs)
	if result.Error != nil {
		return nil, result.Error
	}
	configMap := make(map[string]string)
	for _, config := range configs {
		configMap[config.Key] = config.Value
	}
	return configMap, nil
}

// SetConfigValue sets a configuration value by key
// Returns any error encountered || nil
func SetConfigValue(key, value string) error {
	// First try to find existing config
	var config Config
	result := DB.Where("key = ?", key).First(&config)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Record doesn't exist, create new one
			config = Config{Key: key, Value: value}
			if err := DB.Create(&config).Error; err != nil {
				return fmt.Errorf("failed to create config value for key %s: %w", key, err)
			}
		} else {
			// Some other error occurred
			return fmt.Errorf("failed to query config for key %s: %w", key, result.Error)
		}
	} else {
		// Record exists, update it
		config.Value = value
		if err := DB.Save(&config).Error; err != nil {
			return fmt.Errorf("failed to update config value for key %s: %w", key, err)
		}
	}

	return nil
}

// GetAffirmations retrieves all affirmation entries from the database
// Returns a slice of Affirmations OR any error encountered
func GetAffirmations() ([]Affirmations, error) {
	var affirmations []Affirmations
	result := DB.Find(&affirmations)
	if result.Error != nil {
		return nil, result.Error
	}
	return affirmations, nil
}

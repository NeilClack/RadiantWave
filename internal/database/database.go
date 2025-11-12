package database

import (
	"log"

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
	Title     string
	Content   string
	Selected  bool
	Favorite  bool
	Available bool
}

var defaultConfigValues = map[string]string{
	"home_dir":            "",
	"assets_dir":          "/usr/local/share/radiantwave",
	"log_dir":             "/usr/local/share/radiantwave/logs.log", // TODO: Convert the log to use the DB instead of a file
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
	var err error
	DB, err = gorm.Open(sqlite.Open(dbpath), &gorm.Config{})
	if err != nil {
		return err
	}
	log.Println("Database connected successfully")
	err = DB.AutoMigrate(&Config{}, &Affirmations{})
	if err != nil {
		return err
	}
	log.Println("Database migrated successfully")
	err = seedDefaults()
	if err != nil {
		log.Println("Error seeding default configuration values:", err)
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
		for key, value := range defaultConfigValues {
			config := Config{Key: key, Value: value}
			if err := DB.Create(&config).Error; err != nil {
				return err
			}
		}
		log.Println("Seeded default configuration values")
	}
	return nil
}

// GetValue retrieves a configuration value by key
// Returns the value OR any error encountered
func GetValue(key string) (string, error) {
	var config Config
	result := DB.First(&config, "key = ?", key)
	if result.Error != nil {
		return "", result.Error
	}
	return config.Value, nil
}

// SetValue sets a configuration value by key
// Returns any error encountered || nil
func SetValue(key string, value string) error {
	config := Config{Key: key, Value: value}
	return DB.Save(&config).Error
}

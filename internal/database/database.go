package database

import (
	"github.com/radiantwave/radiantwave/internal/logger"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Settings struct {
	gorm.Model
	HomeDir              string
	AssetsDir            string
	LogDir               string
	EmailAddress         string
	LicenseKey           string
	SystemCapabilities   string
	SystemType           string
	SelectedAdioFilePath string
	AudioDeviceName      string
	ApplicationFont      string
	BaseFontSize         int
	StandardFontSize     int
	LinePattern          string
	LastVolume           int
}

type Affirmations struct {
	gorm.Model
	Title     string
	Content   string
	Selected  bool
	Favorite  bool
	Available bool
}

func init() {
	db, err := gorm.Open(sqlite.Open("/var/lib/radiantwave/data.db"), &gorm.Config{})
	if err != nil {
		logger.Fatalf("failed to connect database: %v", err)
	}
}

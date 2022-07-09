package config

import (
	"os"
	"path/filepath"
)

var Config struct {
	Enabled   bool
	LogPath   string
	CachePath string
}

func init() {
	Config.LogPath = filepath.Join(os.TempDir(), "debug.log")
}

package config

import (
	"dlog/utils"
	"os"
	"path/filepath"
)

var Config struct {
	Enabled     bool
	LogPath     string
	CachePath   string
	HistoryPath string
}

func init() {
	Config.LogPath = filepath.Join(os.TempDir(), "debug.log")

	dlogdir := os.Getenv("DLOG_DIR")
	if dlogdir == "" {
		dlogdir = filepath.Join(utils.GetHomeDir(), ".dlog")
	}
	Config.HistoryPath = filepath.Join(dlogdir, "history")
}

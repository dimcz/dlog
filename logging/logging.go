package logging

import (
	"dlog/config"
	"log"
	"os"
)

func Debug(l ...any) {
	if !config.Config.Enabled {
		return
	}

	f, err := os.OpenFile(config.Config.LogPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, os.FileMode(0600))
	if err != nil {
		log.Println(err)

		return
	}

	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	log.SetOutput(f)
	log.Println(l...)
}

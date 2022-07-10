package logging

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

var DebugDisabled = true
var DebugConfigPath = filepath.Join(os.TempDir(), "debug.log")

func init() {
	if len(os.Getenv("DEBUG")) > 0 {
		DebugDisabled = false
	}
}

func LogOnErr(err error) {
	if err != nil {
		Debug(err)
	}
}

func Debug(l ...interface{}) {
	if DebugDisabled {
		return
	}

	f, err := os.OpenFile(DebugConfigPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, os.FileMode(0600))
	if err != nil {
		log.Println(err)

		return
	}

	defer LogOnErr(f.Close())

	log.SetOutput(f)
	log.Println(l...)
}

func Timeit(l ...interface{}) func() {
	start := time.Now()
	Debug("->", l)

	return func() {
		Debug("<- ", l, time.Since(start))
	}
}

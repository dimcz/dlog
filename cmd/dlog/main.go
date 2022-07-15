package main

import (
	"fmt"
	"os"

	"github.com/dimcz/dlog"
	"github.com/dimcz/dlog/config"
	"github.com/dimcz/dlog/logging"
	"github.com/dimcz/dlog/utils"
)

const VERSION = "1.0.0"

func main() {
	if config.GetValue().Version {
		fmt.Println("Dlog Version: ", VERSION)
		os.Exit(0)
	}

	logging.Debug("<-- DLOG -->", VERSION)

	d, err := dlog.NewWithDocker()
	utils.ExitOnErr(err)

	defer d.Shutdown()

	d.Display()
}

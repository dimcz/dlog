package main

import (
	"flag"
	"fmt"
	"os"

	"dlog"
	"dlog/config"
	"dlog/utils"
)

const VERSION = "1.0.0"

func main() {
	showVersion := false

	flag.BoolVar(&config.Config.Enabled, "debug", false,
		"Enables debug messages, written to /tmp/dlog.log")
	flag.BoolVar(&showVersion, "version", false, "Print version information")
	flag.Parse()

	if showVersion {
		fmt.Println("Dlog Version: ", VERSION)
		os.Exit(0)
	}

	if !dlog.CheckDaemon() {
		fmt.Println("Can't connect to docker daemon")
		os.Exit(1)
	}

	d, err := dlog.NewWithDocker()
	utils.ExitOnErr(err)

	defer d.Shutdown()

	d.Display()
}

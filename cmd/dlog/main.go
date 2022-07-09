package main

import (
	"dlog"
	"dlog/utils"
	"flag"
	"fmt"
	"os"
)

const VERSION = "1.0.0"

func main() {
	showVersion := false
	waitForShortStdin := 10000

	flag.BoolVar(&showVersion, "version", false, "Print version information")
	flag.IntVar(&waitForShortStdin, "short-stdin-timeout", 10000,
		"Maximum duration(ms) to wait for delayed short stdin(won't delay long stdin)")
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

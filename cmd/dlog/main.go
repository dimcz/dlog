package main

import (
	"flag"
	"fmt"
	"os"

	"dlog"
	"dlog/utils"
)

const VERSION = "1.0.0"

func main() {
	showVersion := flag.Bool("version", false, "Print version information")
	flag.Parse()

	if *showVersion {
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

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dimcz/dlog"
	"github.com/dimcz/dlog/logging"
	"github.com/dimcz/dlog/utils"
)

var gitCommit string

const VERSION = "1.0.0"

func main() {
	showVersion := flag.Bool("version", false, "Print version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Dlog Version: %s-%s\n", VERSION, gitCommit)
		os.Exit(0)
	}

	logging.Debug("<-- DLOG -->", VERSION)

	d, err := dlog.NewWithDocker()
	utils.ExitOnErr(err)

	defer d.Shutdown()

	d.Display()
}

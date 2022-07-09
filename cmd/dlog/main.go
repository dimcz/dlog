package main

import (
	"dlog"
	"dlog/utils"
	"flag"
	"fmt"
)

const VERSION = "1.0.0"

func main() {
	showVersion := false
	flag.BoolVar(&showVersion, "version", false, "Print version information")
	flag.Parse()

	if showVersion {
		fmt.Println("Dlog Version: ", VERSION)
	}

	path := flag.Arg(0)
	d, err := dlog.NewFromFile(path)
	utils.ExitOnErr(err)

	defer d.Shutdown()

	d.Display()
}

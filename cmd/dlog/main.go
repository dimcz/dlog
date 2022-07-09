package main

import (
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
}

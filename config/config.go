package config

import "flag"

type Config struct {
	Version   bool
	Tail      int
	NoLoad    bool
	TimeShift int64
}

var values Config

func init() {
	flag.BoolVar(&(values.Version), "version", false, "Print version information")
	flag.IntVar(&(values.Tail), "tail", 50_000, "Number of lines to show from the end of the logs")
	flag.BoolVar(&(values.NoLoad), "noload", false, "Disable loading previous logs")
	flag.Int64Var(&(values.TimeShift), "shift", 24*60*60, "time chunk to download logs")
	flag.Parse()
}

func GetValue() Config {
	return values
}

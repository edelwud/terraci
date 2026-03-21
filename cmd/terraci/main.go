package main

import (
	"os"

	"github.com/edelwud/terraci/cmd/terraci/cmd"
)

// Version information (set via ldflags)
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	if err := cmd.NewRootCmd(version, commit, date).Execute(); err != nil {
		os.Exit(1)
	}
}

package main

import (
	"fmt"
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
	cmd.SetVersion(version, commit, date)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

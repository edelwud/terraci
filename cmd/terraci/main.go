package main

import (
	"os"

	"github.com/edelwud/terraci/cmd/terraci/cmd"

	// Built-in plugins (blank imports trigger init() registration)
	_ "github.com/edelwud/terraci/plugins/cost"
	_ "github.com/edelwud/terraci/plugins/git"
	_ "github.com/edelwud/terraci/plugins/github"
	_ "github.com/edelwud/terraci/plugins/gitlab"
	_ "github.com/edelwud/terraci/plugins/policy"
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

package main

import (
	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/cmd/terraci/cmd"

	// Built-in plugins (blank imports trigger init() registration)
	_ "github.com/edelwud/terraci/plugins/cost"
	_ "github.com/edelwud/terraci/plugins/diskblob"
	_ "github.com/edelwud/terraci/plugins/git"
	_ "github.com/edelwud/terraci/plugins/github"
	_ "github.com/edelwud/terraci/plugins/gitlab"
	_ "github.com/edelwud/terraci/plugins/inmemcache"
	_ "github.com/edelwud/terraci/plugins/localexec"
	_ "github.com/edelwud/terraci/plugins/policy"
	_ "github.com/edelwud/terraci/plugins/summary"
	_ "github.com/edelwud/terraci/plugins/tfupdate"
)

// Version information (set via ldflags)
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	if err := cmd.NewRootCmd(version, commit, date).Execute(); err != nil {
		log.WithError(err).Fatal("command failed")
	}
}

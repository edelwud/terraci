// xterraci builds custom TerraCi binaries with selected plugins,
// similar to xcaddy for Caddy.
package main

import (
	"github.com/edelwud/terraci/cmd/xterraci/cmd"
	"github.com/edelwud/terraci/pkg/log"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	log.Init()

	if err := cmd.NewRootCmd(version, commit, date).Execute(); err != nil {
		log.WithError(err).Fatal("command failed")
	}
}

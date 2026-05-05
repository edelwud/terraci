// xterraci builds custom TerraCi binaries with selected plugins,
// similar to xcaddy for Caddy.
package main

import (
	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/cmd/xterraci/cmd"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	cmd.InitLogger()

	if err := cmd.NewRootCmd(version, commit, date).Execute(); err != nil {
		log.WithError(err).Fatal("command failed")
	}
}

package main

// BuiltinPlugins lists the import paths of all built-in plugins.
// These are included by default unless excluded with --without.
var BuiltinPlugins = map[string]string{
	"gitlab":  "github.com/edelwud/terraci/plugins/gitlab",
	"github":  "github.com/edelwud/terraci/plugins/github",
	"cost": "github.com/edelwud/terraci/plugins/cost",
	"policy":  "github.com/edelwud/terraci/plugins/policy",
	"git":     "github.com/edelwud/terraci/plugins/git",
}

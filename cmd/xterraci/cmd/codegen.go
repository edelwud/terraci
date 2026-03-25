package cmd

import (
	"fmt"
	"sort"
	"strings"
)

// GenerateMainGo produces the main.go source code with the specified plugin imports.
func GenerateMainGo(builtinImports, externalImports []string) string {
	var b strings.Builder

	b.WriteString(`package main

import (
	"github.com/edelwud/terraci/cmd/terraci/cmd"
	"github.com/edelwud/terraci/pkg/log"
`)

	// Built-in plugins
	if len(builtinImports) > 0 {
		b.WriteString("\n\t// Built-in plugins\n")
		sort.Strings(builtinImports)
		for _, imp := range builtinImports {
			fmt.Fprintf(&b, "\t_ %q\n", imp)
		}
	}

	// External plugins
	if len(externalImports) > 0 {
		b.WriteString("\n\t// External plugins\n")
		sort.Strings(externalImports)
		for _, imp := range externalImports {
			fmt.Fprintf(&b, "\t_ %q\n", imp)
		}
	}

	b.WriteString(`)

var (
	version = "custom"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	if err := cmd.NewRootCmd(version, commit, date).Execute(); err != nil {
		log.WithError(err).Fatal("command failed")
	}
}
`)

	return b.String()
}

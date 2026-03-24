package main

import (
	"fmt"
	"sort"
	"strings"
)

// GenerateMainGo produces the main.go source code with the specified plugin imports.
func GenerateMainGo(builtinImports []string, externalImports []string) string {
	var b strings.Builder

	b.WriteString(`package main

import (
	"os"

	"github.com/edelwud/terraci/cmd/terraci/cmd"
`)

	// Built-in plugins
	if len(builtinImports) > 0 {
		b.WriteString("\n\t// Built-in plugins\n")
		sort.Strings(builtinImports)
		for _, imp := range builtinImports {
			b.WriteString(fmt.Sprintf("\t_ %q\n", imp))
		}
	}

	// External plugins
	if len(externalImports) > 0 {
		b.WriteString("\n\t// External plugins\n")
		sort.Strings(externalImports)
		for _, imp := range externalImports {
			b.WriteString(fmt.Sprintf("\t_ %q\n", imp))
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
		os.Exit(1)
	}
}
`)

	return b.String()
}

package source

import (
	"context"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

type Loader struct{}

func NewLoader() *Loader {
	return &Loader{}
}

func (l *Loader) Load(ctx context.Context, modulePath string) (*Snapshot, error) {
	return newLoadSession(modulePath).Run(ctx)
}

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func parseHCLFile(path string) (*hcl.File, hcl.Diagnostics, error) {
	content, err := readFile(path)
	if err != nil {
		return nil, nil, err
	}
	file, diags := hclsyntax.ParseConfig(content, path, hcl.Pos{Line: 1, Column: 1})
	return file, diags, nil
}

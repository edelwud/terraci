package source

import (
	"context"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
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

func parseHCLFile(hclParser *hclparse.Parser, path string) (*hcl.File, hcl.Diagnostics, error) {
	content, err := readFile(path)
	if err != nil {
		return nil, nil, err
	}
	file, diags := hclParser.ParseHCL(content, path)
	return file, diags, nil
}

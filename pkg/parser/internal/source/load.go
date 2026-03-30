package source

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
)

type Loader struct{}

func NewLoader() *Loader {
	return &Loader{}
}

func (l *Loader) Load(ctx context.Context, modulePath string) (*Index, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	hclParser := hclparse.NewParser()
	index := NewIndex(modulePath, hclParser)

	tfFiles, err := filepath.Glob(filepath.Join(modulePath, "*.tf"))
	if err != nil {
		return nil, fmt.Errorf("glob .tf files: %w", err)
	}

	for _, tfFile := range tfFiles {
		file, err := index.ParseHCLFile(tfFile)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", tfFile, err)
		}
		if file != nil {
			index.AddFile(tfFile, file)
		}
	}

	return index, nil
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

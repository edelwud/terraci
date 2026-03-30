package parser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
)

type moduleLoader struct{}

func newModuleLoader() *moduleLoader {
	return &moduleLoader{}
}

func (l *moduleLoader) Load(ctx context.Context, modulePath string) (*moduleIndex, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	hclParser := hclparse.NewParser()
	index := newModuleIndex(modulePath, hclParser)

	tfFiles, err := filepath.Glob(filepath.Join(modulePath, "*.tf"))
	if err != nil {
		return nil, fmt.Errorf("glob .tf files: %w", err)
	}

	for _, tfFile := range tfFiles {
		file, err := index.parseHCLFile(tfFile)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", tfFile, err)
		}
		if file != nil {
			index.addFile(tfFile, file)
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

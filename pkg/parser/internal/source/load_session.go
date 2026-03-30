package source

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/hashicorp/hcl/v2/hclparse"
)

type loadSession struct {
	modulePath string
	index      *Index
}

func newLoadSession(modulePath string) *loadSession {
	return &loadSession{
		modulePath: modulePath,
		index:      NewIndex(modulePath, hclparse.NewParser()),
	}
}

func (s *loadSession) Run(ctx context.Context) (*Index, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	tfFiles, err := s.discoverTFFiles()
	if err != nil {
		return nil, err
	}

	if err := s.parseFiles(tfFiles); err != nil {
		return nil, err
	}

	return s.index, nil
}

func (s *loadSession) discoverTFFiles() ([]string, error) {
	tfFiles, err := filepath.Glob(filepath.Join(s.modulePath, "*.tf"))
	if err != nil {
		return nil, fmt.Errorf("glob .tf files: %w", err)
	}

	return tfFiles, nil
}

func (s *loadSession) parseFiles(tfFiles []string) error {
	for _, tfFile := range tfFiles {
		file, err := s.index.ParseHCLFile(tfFile)
		if err != nil {
			return fmt.Errorf("read %s: %w", tfFile, err)
		}
		if file != nil {
			s.index.AddFile(tfFile, file)
		}
	}

	return nil
}

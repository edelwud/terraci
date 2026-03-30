package source

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/hashicorp/hcl/v2/hclparse"
)

type loadSession struct {
	modulePath string
	builder    *indexBuilder
}

func newLoadSession(modulePath string) *loadSession {
	return &loadSession{
		modulePath: modulePath,
		builder:    newIndexBuilder(modulePath, hclparse.NewParser()),
	}
}

func (s *loadSession) Run(ctx context.Context) (*Snapshot, error) {
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

	return s.builder.Snapshot(), nil
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
		file, err := s.builder.ParseHCLFile(tfFile)
		if err != nil {
			return fmt.Errorf("read %s: %w", tfFile, err)
		}
		if file != nil {
			s.builder.AddFile(tfFile, file)
		}
	}

	return nil
}

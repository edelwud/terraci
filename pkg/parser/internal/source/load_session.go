package source

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2/hclparse"
)

type loadSession struct {
	modulePath string
	builder    *indexBuilder
}

func newLoadSession(modulePath string) *loadSession {
	return &loadSession{
		modulePath: modulePath,
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

	s.builder = newIndexBuilder(s.modulePath, hclparse.NewParser(), len(tfFiles))

	if err := s.parseFiles(tfFiles); err != nil {
		return nil, err
	}

	return s.builder.Snapshot(), nil
}

func (s *loadSession) discoverTFFiles() ([]string, error) {
	entries, err := os.ReadDir(s.modulePath)
	if err != nil {
		return nil, fmt.Errorf("read module dir: %w", err)
	}

	tfFiles := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tf") {
			continue
		}
		tfFiles = append(tfFiles, filepath.Join(s.modulePath, entry.Name()))
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

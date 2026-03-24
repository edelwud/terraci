package policy

import (
	"context"
	"fmt"
	"os"
)

// PathSource represents a local filesystem path source
type PathSource struct {
	Path string
}

// Pull for path sources is a no-op since files are already local
// It just validates that the path exists
func (s *PathSource) Pull(_ context.Context, _ string) error {
	info, err := os.Stat(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("policy path does not exist: %s", s.Path)
		}
		return fmt.Errorf("failed to access policy path: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("policy path is not a directory: %s", s.Path)
	}

	return nil
}

// String returns a human-readable description
func (s *PathSource) String() string {
	return fmt.Sprintf("path:%s", s.Path)
}

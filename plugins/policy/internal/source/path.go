package source

import (
	"context"
	"fmt"
	"os"
)

type PathSource struct {
	Path string
}

func (s *PathSource) Materialize(_ context.Context, _ string) (string, error) {
	info, err := os.Stat(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("policy path does not exist: %s", s.Path)
		}
		return "", fmt.Errorf("access policy path: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("policy path is not a directory: %s", s.Path)
	}
	return s.Path, nil
}

func (s *PathSource) String() string {
	return "path:" + s.Path
}

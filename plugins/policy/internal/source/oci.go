package source

import (
	"context"
	"fmt"
	"os"
	"strings"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
)

type OCISource struct {
	URL string
}

func (s *OCISource) Materialize(ctx context.Context, dest string) (string, error) {
	if err := os.RemoveAll(dest); err != nil {
		return "", fmt.Errorf("clean destination: %w", err)
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return "", fmt.Errorf("create destination: %w", err)
	}

	ref, err := s.ref()
	if err != nil {
		return "", err
	}
	repo, err := remote.NewRepository(ref)
	if err != nil {
		return "", fmt.Errorf("create repository: %w", err)
	}

	fs, err := file.New(dest)
	if err != nil {
		return "", fmt.Errorf("create file store: %w", err)
	}
	defer fs.Close()

	if _, err := oras.Copy(ctx, repo, ref, fs, ref, oras.DefaultCopyOptions); err != nil {
		return "", fmt.Errorf("pull OCI bundle: %w", err)
	}

	return dest, nil
}

func (s *OCISource) ref() (string, error) {
	ref := strings.TrimPrefix(s.URL, "oci://")
	if ref == "" {
		return "", fmt.Errorf("invalid OCI URL: %s", s.URL)
	}
	return ref, nil
}

func (s *OCISource) String() string {
	return "oci:" + s.URL
}

package policy

import (
	"context"
	"fmt"
	"os"
	"strings"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
)

// OCISource represents an OCI registry source
type OCISource struct {
	URL string // oci://registry.example.com/policies:v1.0
}

// Pull downloads the OCI bundle to the destination directory
func (s *OCISource) Pull(ctx context.Context, dest string) error {
	// Remove existing directory if it exists
	if err := os.RemoveAll(dest); err != nil {
		return fmt.Errorf("failed to clean destination: %w", err)
	}

	// Create destination directory
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}

	// Parse OCI URL
	ref, err := s.parseURL()
	if err != nil {
		return err
	}

	// Create repository
	repo, err := remote.NewRepository(ref)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	// Create file store for destination
	fs, err := file.New(dest)
	if err != nil {
		return fmt.Errorf("failed to create file store: %w", err)
	}
	defer fs.Close()

	// Copy from remote to local
	_, err = oras.Copy(ctx, repo, ref, fs, ref, oras.DefaultCopyOptions)
	if err != nil {
		return fmt.Errorf("failed to pull OCI bundle: %w", err)
	}

	return nil
}

// parseURL parses the OCI URL and returns the reference
func (s *OCISource) parseURL() (string, error) {
	url := s.URL

	// Remove oci:// prefix
	url = strings.TrimPrefix(url, "oci://")

	if url == "" {
		return "", fmt.Errorf("invalid OCI URL: %s", s.URL)
	}

	return url, nil
}

// String returns a human-readable description
func (s *OCISource) String() string {
	return fmt.Sprintf("oci:%s", s.URL)
}

package lockfile

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"golang.org/x/mod/sumdb/dirhash"
)

func normalizePlatforms(platforms []string) []string {
	if len(platforms) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(platforms))
	normalized := make([]string, 0, len(platforms))
	for _, platform := range platforms {
		if platform == "" {
			continue
		}
		if _, ok := seen[platform]; ok {
			continue
		}
		seen[platform] = struct{}{}
		normalized = append(normalized, platform)
	}

	sort.Strings(normalized)
	return normalized
}

func filterPlatforms(available, wanted []string) []string {
	set := make(map[string]struct{}, len(wanted))
	for _, p := range wanted {
		set[p] = struct{}{}
	}

	filtered := make([]string, 0, len(wanted))
	for _, p := range available {
		if _, ok := set[p]; ok {
			filtered = append(filtered, p)
		}
	}

	return filtered
}

func normalizeHashes(hashes []string) []string {
	if len(hashes) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(hashes))
	normalized := make([]string, 0, len(hashes))
	for _, hash := range hashes {
		if hash == "" {
			continue
		}
		if _, ok := seen[hash]; ok {
			continue
		}
		seen[hash] = struct{}{}
		normalized = append(normalized, hash)
	}

	sort.Strings(normalized)
	return normalized
}

// ReadLockedProviders reads all provider entries from an existing lock file.
func ReadLockedProviders(lockPath string) []LockedProviderEntry {
	doc, err := ParseDocument(lockPath)
	if err != nil {
		return nil
	}

	entries := make([]LockedProviderEntry, 0, len(doc.Providers))
	for _, provider := range doc.Providers {
		entries = append(entries, LockedProviderEntry{
			Source:      provider.Source,
			Version:     provider.Version,
			Constraints: provider.Constraints,
		})
	}

	return entries
}

func verifyPackageChecksum(path, want string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return fmt.Errorf("hash %s: %w", path, err)
	}

	got := hex.EncodeToString(sum.Sum(nil))
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("checksum mismatch: got %s want %s", got, want)
	}

	return nil
}

func hashZip(path string) (string, error) {
	hash, err := dirhash.HashZip(path, dirhash.Hash1)
	if err != nil {
		return "", fmt.Errorf("compute h1 for %s: %w", path, err)
	}

	return hash, nil
}

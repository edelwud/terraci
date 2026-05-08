package workspacepath

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
)

// Join joins workspace-relative path components with POSIX separators while
// preserving parent segments for later validation.
func Join(parts ...string) string {
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		normalized := strings.ReplaceAll(part, "\\", "/")
		if len(cleaned) == 0 {
			normalized = strings.TrimRight(normalized, "/")
		} else {
			normalized = strings.Trim(normalized, "/")
		}
		if normalized == "" {
			continue
		}
		cleaned = append(cleaned, normalized)
	}
	if len(cleaned) == 0 {
		return ""
	}
	return strings.Join(cleaned, "/")
}

// Validate rejects empty, absolute, drive-prefixed, and parent-traversing
// paths. Values are always interpreted as workspace-relative POSIX paths.
func Validate(value string) error {
	normalized := strings.ReplaceAll(value, "\\", "/")
	if normalized == "" {
		return errors.New("path is empty")
	}
	return validateNonEmpty(value, normalized)
}

// ValidateOptional applies Validate only when value is not empty.
func ValidateOptional(value string) error {
	normalized := strings.ReplaceAll(value, "\\", "/")
	if normalized == "" {
		return nil
	}
	return validateNonEmpty(value, normalized)
}

func validateNonEmpty(original, normalized string) error {
	if path.IsAbs(normalized) || hasWindowsDriveSegment(normalized) {
		return fmt.Errorf("path %q must be workspace-relative", original)
	}
	if slices.Contains(strings.Split(normalized, "/"), "..") {
		return fmt.Errorf("path %q must not contain parent directory segments", original)
	}
	return nil
}

func hasWindowsDriveSegment(value string) bool {
	for segment := range strings.SplitSeq(value, "/") {
		if len(segment) < 2 || segment[1] != ':' {
			continue
		}
		if (segment[0] >= 'A' && segment[0] <= 'Z') || (segment[0] >= 'a' && segment[0] <= 'z') {
			return true
		}
	}
	return false
}

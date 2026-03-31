// Package ciprovider contains shared types used by CI provider plugins (gitlab, github).
package ciprovider

import (
	"strings"

	"github.com/edelwud/terraci/pkg/ci"
)

// Image defines a Docker image configuration for CI jobs.
type Image struct {
	// Name is the image name (e.g., "hashicorp/terraform:1.6")
	Name string `yaml:"name,omitempty" json:"name,omitempty" jsonschema:"description=Docker image name"`
	// Entrypoint overrides the default entrypoint
	Entrypoint []string `yaml:"entrypoint,omitempty" json:"entrypoint,omitempty" jsonschema:"description=Override default entrypoint"`
}

// UnmarshalYAML supports both string shorthand ("image:1.0") and full object ({name: "image:1.0"}).
func (img *Image) UnmarshalYAML(unmarshal func(any) error) error {
	// Try string shorthand first (just image name)
	var shorthand string
	if err := unmarshal(&shorthand); err == nil {
		img.Name = shorthand
		return nil
	}

	// Try full object syntax
	type imageAlias Image
	var alias imageAlias
	if err := unmarshal(&alias); err != nil {
		return err
	}
	*img = Image(alias)
	return nil
}

// String returns the image name.
func (img *Image) String() string {
	return img.Name
}

// HasEntrypoint returns true if entrypoint is configured.
func (img *Image) HasEntrypoint() bool {
	return len(img.Entrypoint) > 0
}

// MRCommentConfig contains settings for MR/PR comments (shared by gitlab, github).
type MRCommentConfig struct {
	// Enabled enables MR comments (default: true when in MR pipeline)
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty" jsonschema:"description=Enable MR comments,default=true"`
	// OnChangesOnly only comment when there are changes (default: false)
	OnChangesOnly bool `yaml:"on_changes_only,omitempty" json:"on_changes_only,omitempty" jsonschema:"description=Only comment when there are changes"`
	// IncludeDetails includes full plan output in collapsible sections
	IncludeDetails bool `yaml:"include_details,omitempty" json:"include_details,omitempty" jsonschema:"description=Include full plan output in expandable sections,default=true"`
}

// CommentEnabled returns the effective enabled state for MR/PR comments.
// Nil config, nil Enabled, and missing config all default to true.
func CommentEnabled(cfg *MRCommentConfig) bool {
	if cfg == nil || cfg.Enabled == nil {
		return true
	}
	return *cfg.Enabled
}

// HasCommentMarker reports whether the provided body contains the terraci marker.
func HasCommentMarker(body string) bool {
	return strings.Contains(body, ci.CommentMarker)
}

package ci

import "strings"

// Image defines a Docker image configuration for CI jobs.
type Image struct {
	// Name is the image name (e.g., "hashicorp/terraform:1.6")
	Name string `yaml:"name,omitempty" json:"name,omitempty" jsonschema:"description=Docker image name"`
	// Entrypoint overrides the default entrypoint.
	Entrypoint []string `yaml:"entrypoint,omitempty" json:"entrypoint,omitempty" jsonschema:"description=Override default entrypoint"`
}

// UnmarshalYAML supports both string shorthand ("image:1.0") and full object ({name: "image:1.0"}).
func (img *Image) UnmarshalYAML(unmarshal func(any) error) error {
	var shorthand string
	if err := unmarshal(&shorthand); err == nil {
		img.Name = shorthand
		return nil
	}

	type imageAlias Image
	var alias imageAlias
	if err := unmarshal(&alias); err != nil {
		return err
	}

	*img = Image(alias)
	return nil
}

// MarshalYAML emits the short string form ("image:1.0") when only Name is set,
// preserving round-trip symmetry with UnmarshalYAML. Configs that started as
// shorthand stay shorthand after `terraci init` writes them back; only configs
// with an Entrypoint override expand to the full mapping.
func (img Image) MarshalYAML() (any, error) {
	if len(img.Entrypoint) == 0 {
		return img.Name, nil
	}
	type imageAlias Image
	return imageAlias(img), nil
}

// String returns the image name.
func (img *Image) String() string {
	return img.Name
}

// HasEntrypoint returns true if entrypoint is configured.
func (img *Image) HasEntrypoint() bool {
	return len(img.Entrypoint) > 0
}

// HasCommentMarker reports whether the provided body contains the TerraCI review-comment marker.
func HasCommentMarker(body string) bool {
	return strings.Contains(body, CommentMarker)
}

package updateengine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

const (
	blockTerraform         = "terraform"
	blockRequiredProviders = "required_providers"
)

// WriteModuleVersion updates the version attribute of a module block in a .tf file.
func WriteModuleVersion(filePath, moduleName, newVersion string) error {
	filePath = filepath.Clean(filePath)
	src, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read %s: %w", filePath, err)
	}

	file, diags := hclwrite.ParseConfig(src, filePath, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return fmt.Errorf("parse %s: %s", filePath, diags.Error())
	}

	for _, block := range file.Body().Blocks() {
		if block.Type() == "module" && len(block.Labels()) > 0 && block.Labels()[0] == moduleName {
			block.Body().SetAttributeValue("version", cty.StringVal(newVersion))
			return os.WriteFile(filePath, file.Bytes(), 0o600) //nolint:gosec // filePath is cleaned above
		}
	}

	return fmt.Errorf("module %q not found in %s", moduleName, filePath)
}

// WriteProviderVersion updates the version constraint in a required_providers block.
// Since hclwrite doesn't support nested object manipulation directly, we use
// token-level string replacement within the provider attribute.
func WriteProviderVersion(filePath, providerName, newConstraint string) error {
	filePath = filepath.Clean(filePath)
	src, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read %s: %w", filePath, err)
	}

	file, diags := hclwrite.ParseConfig(src, filePath, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return fmt.Errorf("parse %s: %s", filePath, diags.Error())
	}

	for _, block := range file.Body().Blocks() {
		if block.Type() != blockTerraform {
			continue
		}
		for _, sub := range block.Body().Blocks() {
			if sub.Type() != blockRequiredProviders {
				continue
			}

			attr := sub.Body().GetAttribute(providerName)
			if attr == nil {
				continue
			}

			// Get the raw tokens of the attribute value and replace the version string.
			tokens := attr.Expr().BuildTokens(nil)
			replaced := replaceVersionInTokens(tokens, newConstraint)
			sub.Body().SetAttributeRaw(providerName, replaced)

			return os.WriteFile(filePath, file.Bytes(), 0o600) //nolint:gosec // filePath is cleaned above
		}
	}

	return fmt.Errorf("provider %q not found in required_providers in %s", providerName, filePath)
}

// replaceVersionInTokens finds the version value token in a required_providers object
// and replaces it with the new constraint.
func replaceVersionInTokens(tokens hclwrite.Tokens, newConstraint string) hclwrite.Tokens {
	// Look for pattern: "version" = "constraint"
	// We need to find the string token after "version" key.
	foundVersionKey := false
	for i, tok := range tokens {
		if tok.Type == 9 { // hclsyntax.TokenQuotedLit
			val := strings.Trim(string(tok.Bytes), `"`)
			if val == "version" {
				foundVersionKey = true
				continue
			}
			if foundVersionKey {
				// This is the version value token — replace it.
				tokens[i].Bytes = []byte(`"` + newConstraint + `"`)
				return tokens
			}
		}
		// Reset if we hit a key that isn't "version".
		if tok.Type == 9 && !foundVersionKey {
			foundVersionKey = false
		}
	}
	return tokens
}

// containsModuleBlock checks if a .tf file contains a module block with the given name.
func containsModuleBlock(filePath, moduleName string) bool {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}
	// Quick string check before parsing.
	if !strings.Contains(string(src), "module") {
		return false
	}

	file, diags := hclwrite.ParseConfig(src, filePath, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return false
	}

	for _, block := range file.Body().Blocks() {
		if block.Type() == "module" && len(block.Labels()) > 0 && block.Labels()[0] == moduleName {
			return true
		}
	}
	return false
}

// containsProviderBlock checks if a .tf file contains a required_providers block
// with the given provider name.
func containsProviderBlock(filePath, providerName string) bool {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}
	if !strings.Contains(string(src), blockRequiredProviders) {
		return false
	}

	file, diags := hclwrite.ParseConfig(src, filePath, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return false
	}

	for _, block := range file.Body().Blocks() {
		if block.Type() != blockTerraform {
			continue
		}
		for _, sub := range block.Body().Blocks() {
			if sub.Type() == blockRequiredProviders && sub.Body().GetAttribute(providerName) != nil {
				return true
			}
		}
	}
	return false
}

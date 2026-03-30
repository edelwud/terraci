package tfwrite

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
	foundVersionKey := false
	for i, tok := range tokens {
		if tok.Type == 9 { // hclsyntax.TokenQuotedLit
			val := strings.Trim(string(tok.Bytes), `"`)
			if val == "version" {
				foundVersionKey = true
				continue
			}
			if foundVersionKey {
				tokens[i].Bytes = []byte(`"` + newConstraint + `"`)
				return tokens
			}
		}
		if tok.Type == 9 && !foundVersionKey {
			foundVersionKey = false
		}
	}
	return tokens
}

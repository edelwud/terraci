package tfwrite

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

const (
	blockTerraform         = "terraform"
	blockRequiredProviders = "required_providers"
	blockProvider          = "provider"
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
			return os.WriteFile(filePath, file.Bytes(), 0o600) //nolint:gosec // filePath is sanitized via filepath.Clean above
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

			return os.WriteFile(filePath, file.Bytes(), 0o600) //nolint:gosec // filePath is sanitized via filepath.Clean above
		}
	}

	return fmt.Errorf("provider %q not found in required_providers in %s", providerName, filePath)
}

// replaceVersionInTokens finds the version value token in a required_providers object
// and replaces it with the new constraint.
func replaceVersionInTokens(tokens hclwrite.Tokens, newConstraint string) hclwrite.Tokens {
	foundVersionKey := false
	for i, tok := range tokens {
		if tok.Type == hclsyntax.TokenIdent && string(tok.Bytes) == "version" {
			foundVersionKey = true
			continue
		}
		if foundVersionKey && tok.Type == hclsyntax.TokenQuotedLit {
			tokens[i].Bytes = []byte(newConstraint)
			return tokens
		}
	}
	return tokens
}

// WriteProviderLock creates or updates a provider entry in .terraform.lock.hcl.
func WriteProviderLock(filePath, providerSource, version, constraints string, hashes []string) error {
	filePath = filepath.Clean(filePath)

	file, err := parseOrCreateFile(filePath)
	if err != nil {
		return err
	}

	block := findProviderLockBlock(file.Body(), providerSource)
	if block == nil {
		block = file.Body().AppendNewBlock(blockProvider, []string{providerSource})
	}

	block.Body().SetAttributeValue("version", cty.StringVal(version))
	if constraints != "" {
		block.Body().SetAttributeValue("constraints", cty.StringVal(constraints))
	}
	if len(hashes) > 0 {
		block.Body().SetAttributeRaw("hashes", buildHashesTokens(hashes))
	}

	return os.WriteFile(filePath, file.Bytes(), 0o600)
}

func parseOrCreateFile(filePath string) (*hclwrite.File, error) {
	src, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return hclwrite.NewEmptyFile(), nil
		}
		return nil, fmt.Errorf("read %s: %w", filePath, err)
	}

	file, diags := hclwrite.ParseConfig(src, filePath, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("parse %s: %s", filePath, diags.Error())
	}

	return file, nil
}

// UpdateProviderLockConstraints updates only the constraints attribute of an
// existing provider entry in .terraform.lock.hcl. Returns false if the provider
// block was not found.
func UpdateProviderLockConstraints(filePath, providerSource, constraints string) (bool, error) {
	filePath = filepath.Clean(filePath)
	src, err := os.ReadFile(filePath)
	if err != nil {
		return false, fmt.Errorf("read %s: %w", filePath, err)
	}

	file, diags := hclwrite.ParseConfig(src, filePath, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return false, fmt.Errorf("parse %s: %s", filePath, diags.Error())
	}

	block := findProviderLockBlock(file.Body(), providerSource)
	if block == nil {
		return false, nil
	}

	block.Body().SetAttributeValue("constraints", cty.StringVal(constraints))

	return true, os.WriteFile(filePath, file.Bytes(), 0o600) //nolint:gosec // filePath is sanitized via filepath.Clean above
}

func findProviderLockBlock(body *hclwrite.Body, providerSource string) *hclwrite.Block {
	for _, block := range body.Blocks() {
		if block.Type() != blockProvider {
			continue
		}
		labels := block.Labels()
		if len(labels) == 1 && labels[0] == providerSource {
			return block
		}
	}

	return nil
}

// buildHashesTokens constructs HCL tokens for a multi-line list of hashes,
// matching the format Terraform uses in .terraform.lock.hcl.
func buildHashesTokens(hashes []string) hclwrite.Tokens {
	tokens := make(hclwrite.Tokens, 0, 2+6*len(hashes)+2)
	tokens = append(tokens,
		&hclwrite.Token{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")},
		&hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
	)

	for _, hash := range hashes {
		tokens = append(tokens,
			&hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("    ")},
			&hclwrite.Token{Type: hclsyntax.TokenOQuote, Bytes: []byte("\"")},
			&hclwrite.Token{Type: hclsyntax.TokenQuotedLit, Bytes: []byte(hash)},
			&hclwrite.Token{Type: hclsyntax.TokenCQuote, Bytes: []byte("\"")},
			&hclwrite.Token{Type: hclsyntax.TokenComma, Bytes: []byte(",")},
			&hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
		)
	}

	tokens = append(tokens,
		&hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("  ")},
		&hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("]")},
	)

	return tokens
}

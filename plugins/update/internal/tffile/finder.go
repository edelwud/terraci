package tffile

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

const (
	blockTerraform         = "terraform"
	blockRequiredProviders = "required_providers"
)

// FindModuleBlockFile searches for the .tf file containing a specific module block.
func FindModuleBlockFile(modulePath, callName string) string {
	files, err := filepath.Glob(filepath.Join(modulePath, "*.tf"))
	if err != nil {
		return ""
	}
	for _, f := range files {
		if containsModuleBlock(f, callName) {
			return f
		}
	}
	return ""
}

// FindProviderBlockFile searches for the .tf file containing the required_providers block.
func FindProviderBlockFile(modulePath, providerName string) string {
	files, err := filepath.Glob(filepath.Join(modulePath, "*.tf"))
	if err != nil {
		return ""
	}
	for _, f := range files {
		if containsProviderBlock(f, providerName) {
			return f
		}
	}
	return ""
}

func containsModuleBlock(filePath, moduleName string) bool {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}
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

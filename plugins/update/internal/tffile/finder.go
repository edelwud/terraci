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

// Index caches Terraform block lookups for a single module path.
type Index struct {
	moduleFiles   map[string]string
	providerFiles map[string]string
}

type fileFacts struct {
	moduleNames   []string
	providerNames []string
}

// FindModuleBlockFile searches for the .tf file containing a specific module block.
func FindModuleBlockFile(modulePath, callName string) string {
	index, err := BuildIndex(modulePath)
	if err != nil || index == nil {
		return ""
	}
	return index.FindModuleBlockFile(callName)
}

// FindProviderBlockFile searches for the .tf file containing the required_providers block.
func FindProviderBlockFile(modulePath, providerName string) string {
	index, err := BuildIndex(modulePath)
	if err != nil || index == nil {
		return ""
	}
	return index.FindProviderBlockFile(providerName)
}

// BuildIndex builds a file lookup index for all .tf files in a module directory.
func BuildIndex(modulePath string) (*Index, error) {
	index := &Index{
		moduleFiles:   make(map[string]string),
		providerFiles: make(map[string]string),
	}

	files, err := filepath.Glob(filepath.Join(modulePath, "*.tf"))
	if err != nil {
		return index, err
	}

	for _, f := range files {
		facts := inspectFile(f)
		for _, moduleName := range facts.moduleNames {
			if _, exists := index.moduleFiles[moduleName]; !exists {
				index.moduleFiles[moduleName] = f
			}
		}
		for _, providerName := range facts.providerNames {
			if _, exists := index.providerFiles[providerName]; !exists {
				index.providerFiles[providerName] = f
			}
		}
	}

	return index, nil
}

// FindModuleBlockFile returns the first indexed .tf file containing the given module call.
func (i *Index) FindModuleBlockFile(callName string) string {
	if i == nil {
		return ""
	}
	return i.moduleFiles[callName]
}

// FindProviderBlockFile returns the first indexed .tf file containing the given provider declaration.
func (i *Index) FindProviderBlockFile(providerName string) string {
	if i == nil {
		return ""
	}
	return i.providerFiles[providerName]
}

func inspectFile(filePath string) fileFacts {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return fileFacts{}
	}

	hasModuleKeyword := strings.Contains(string(src), "module")
	hasProviderKeyword := strings.Contains(string(src), blockRequiredProviders)
	if !hasModuleKeyword && !hasProviderKeyword {
		return fileFacts{}
	}

	file, diags := hclwrite.ParseConfig(src, filePath, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return fileFacts{}
	}

	facts := fileFacts{}
	for _, block := range file.Body().Blocks() {
		if hasModuleKeyword && block.Type() == "module" && len(block.Labels()) > 0 {
			facts.moduleNames = append(facts.moduleNames, block.Labels()[0])
		}
		if !hasProviderKeyword || block.Type() != blockTerraform {
			continue
		}
		for _, sub := range block.Body().Blocks() {
			if sub.Type() != blockRequiredProviders {
				continue
			}
			for providerName := range sub.Body().Attributes() {
				facts.providerNames = append(facts.providerNames, providerName)
			}
		}
	}

	return facts
}

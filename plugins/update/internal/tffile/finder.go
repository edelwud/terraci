package tffile

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
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
		index.addFacts(f, facts)
	}

	return index, nil
}

// BuildIndexFromParsedFiles builds an index from already parsed Terraform files.
func BuildIndexFromParsedFiles(files map[string]*hcl.File) *Index {
	index := &Index{
		moduleFiles:   make(map[string]string),
		providerFiles: make(map[string]string),
	}

	for path, file := range files {
		index.addFacts(path, inspectParsedFile(file))
	}

	return index
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

func (i *Index) addFacts(path string, facts fileFacts) {
	for _, moduleName := range facts.moduleNames {
		if _, exists := i.moduleFiles[moduleName]; !exists {
			i.moduleFiles[moduleName] = path
		}
	}
	for _, providerName := range facts.providerNames {
		if _, exists := i.providerFiles[providerName]; !exists {
			i.providerFiles[providerName] = path
		}
	}
}

func inspectFile(filePath string) fileFacts {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return fileFacts{}
	}

	hasModuleKeyword := bytes.Contains(src, []byte("module"))
	hasProviderKeyword := bytes.Contains(src, []byte(blockRequiredProviders))
	if !hasModuleKeyword && !hasProviderKeyword {
		return fileFacts{}
	}

	file, diags := hclsyntax.ParseConfig(src, filePath, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return fileFacts{}
	}

	return inspectParsedFileWithHints(file, hasModuleKeyword, hasProviderKeyword)
}

func inspectParsedFile(file *hcl.File) fileFacts {
	return inspectParsedFileWithHints(file, true, true)
}

func inspectParsedFileWithHints(file *hcl.File, hasModuleKeyword, hasProviderKeyword bool) fileFacts {
	if file == nil {
		return fileFacts{}
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return fileFacts{}
	}

	facts := fileFacts{}
	for _, block := range body.Blocks {
		if hasModuleKeyword && block.Type == "module" && len(block.Labels) > 0 {
			facts.moduleNames = append(facts.moduleNames, block.Labels[0])
		}
		if !hasProviderKeyword || block.Type != blockTerraform {
			continue
		}
		for _, sub := range block.Body.Blocks {
			if sub.Type != blockRequiredProviders {
				continue
			}
			for providerName := range sub.Body.Attributes {
				facts.providerNames = append(facts.providerNames, providerName)
			}
		}
	}

	return facts
}

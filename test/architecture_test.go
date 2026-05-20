package test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const moduleImportPath = "github.com/edelwud/terraci"

func TestArchitecture_ImportBoundaries(t *testing.T) {
	root := repoRoot(t)
	var violations []string

	for _, rel := range goFiles(t, root, "pkg", "plugins") {
		imports := fileImports(t, filepath.Join(root, rel))
		for _, imp := range imports {
			if isProductionFile(rel) && strings.HasPrefix(rel, "pkg/") {
				if strings.HasPrefix(imp, moduleImportPath+"/plugins/") {
					violations = append(violations, rel+" imports plugin implementation package "+imp)
				}
				if !strings.HasPrefix(rel, "pkg/plugin/") && importsPluginSDK(imp) {
					violations = append(violations, rel+" imports plugin SDK package "+imp)
				}
			}

			if strings.HasPrefix(rel, "plugins/") {
				if reason, ok := siblingPluginImportViolation(rel, imp); ok {
					violations = append(violations, reason)
				}
			}
		}
	}

	if len(violations) > 0 {
		t.Fatalf("architecture import boundary violations:\n%s", strings.Join(violations, "\n"))
	}
}

func repoRoot(tb testing.TB) string {
	tb.Helper()

	dir, err := os.Getwd()
	if err != nil {
		tb.Fatalf("Getwd() error = %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			tb.Fatal("repository root with go.mod not found")
		}
		dir = parent
	}
}

func goFiles(tb testing.TB, root string, roots ...string) []string {
	tb.Helper()

	var files []string
	for _, scanRoot := range roots {
		err := filepath.WalkDir(filepath.Join(root, scanRoot), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				switch d.Name() {
				case ".git", "build", "dist", "testdata", "vendor":
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) != ".go" {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, filepath.ToSlash(rel))
			return nil
		})
		if err != nil {
			tb.Fatalf("walk %s: %v", scanRoot, err)
		}
	}
	return files
}

func fileImports(tb testing.TB, path string) []string {
	tb.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		tb.Fatalf("parse imports for %s: %v", path, err)
	}

	imports := make([]string, 0, len(file.Imports))
	for _, spec := range file.Imports {
		importPath := importPath(tb, spec)
		if importPath != "" {
			imports = append(imports, importPath)
		}
	}
	return imports
}

func importPath(tb testing.TB, spec *ast.ImportSpec) string {
	tb.Helper()
	path, err := strconv.Unquote(spec.Path.Value)
	if err != nil {
		tb.Fatalf("unquote import path %s: %v", spec.Path.Value, err)
	}
	return path
}

func isProductionFile(rel string) bool {
	return !strings.HasSuffix(rel, "_test.go")
}

func importsPluginSDK(imp string) bool {
	return imp == moduleImportPath+"/pkg/plugin" || strings.HasPrefix(imp, moduleImportPath+"/pkg/plugin/")
}

func siblingPluginImportViolation(rel, imp string) (string, bool) {
	if !strings.HasPrefix(imp, moduleImportPath+"/plugins/") {
		return "", false
	}

	parts := strings.Split(rel, "/")
	if len(parts) < 2 {
		return "", false
	}
	pluginName := parts[1]
	if pluginName == "internal" {
		return "", false
	}

	importRest := strings.TrimPrefix(imp, moduleImportPath+"/plugins/")
	importedPlugin := strings.Split(importRest, "/")[0]
	if importedPlugin == pluginName || importedPlugin == "internal" {
		return "", false
	}

	scope := "production"
	if strings.HasSuffix(rel, "_test.go") {
		scope = "test"
	}
	return rel + " has " + scope + " import of sibling plugin package " + imp, true
}

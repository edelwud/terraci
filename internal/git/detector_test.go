package git

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/edelwud/terraci/internal/discovery"
)

func TestIsTerraformRelated(t *testing.T) {
	tests := []struct {
		file string
		want bool
	}{
		{"main.tf", true},
		{"variables.tfvars", true},
		{".terraform.lock.hcl", true},
		{"config.tf.json", true},
		{"modules/vpc/main.tf", true},
		{"service/prod/us-east-1/vpc/outputs.tf", true},
		{"readme.md", false},
		{"script.sh", false},
		{"terraform.tfstate", false},
		{"Makefile", false},
		{"go.mod", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			got := isTerraformRelated(tt.file)
			if got != tt.want {
				t.Errorf("isTerraformRelated(%q) = %v, want %v", tt.file, got, tt.want)
			}
		})
	}
}

func makeModule(relPath string) *discovery.Module {
	segments := []string{"service", "environment", "region", "module"}
	parts := filepath.SplitList(relPath)
	if len(parts) == 0 {
		parts = splitPath(relPath)
	}
	values := make([]string, len(segments))
	for i := range segments {
		if i < len(parts) {
			values[i] = parts[i]
		}
	}
	return discovery.NewModule(segments, values, "/root/"+relPath, relPath)
}

func splitPath(p string) []string {
	var parts []string
	for p != "" && p != "." {
		dir, base := filepath.Split(filepath.Clean(p))
		parts = append([]string{base}, parts...)
		p = filepath.Clean(dir)
	}
	return parts
}

func TestFilesToModules(t *testing.T) {
	mod1 := makeModule("myapp/prod/us-east-1/vpc")
	mod2 := makeModule("myapp/prod/us-east-1/eks")
	index := discovery.NewModuleIndex([]*discovery.Module{mod1, mod2})

	client := NewClient(t.TempDir())
	detector := NewChangedModulesDetector(client, index, "")

	tests := []struct {
		name    string
		files   []string
		wantIDs []string
	}{
		{
			name:    "terraform files under known module",
			files:   []string{"myapp/prod/us-east-1/vpc/main.tf"},
			wantIDs: []string{"myapp/prod/us-east-1/vpc"},
		},
		{
			name:    "multiple files in same module deduped",
			files:   []string{"myapp/prod/us-east-1/vpc/main.tf", "myapp/prod/us-east-1/vpc/variables.tf"},
			wantIDs: []string{"myapp/prod/us-east-1/vpc"},
		},
		{
			name:    "files in different modules",
			files:   []string{"myapp/prod/us-east-1/vpc/main.tf", "myapp/prod/us-east-1/eks/main.tf"},
			wantIDs: []string{"myapp/prod/us-east-1/eks", "myapp/prod/us-east-1/vpc"},
		},
		{
			name:    "non-terraform files skipped",
			files:   []string{"myapp/prod/us-east-1/vpc/README.md"},
			wantIDs: nil,
		},
		{
			name:    "unknown paths skipped",
			files:   []string{"unknown/path/main.tf"},
			wantIDs: nil,
		},
		{
			name:    "empty file list",
			files:   nil,
			wantIDs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modules := detector.filesToModules(tt.files)
			gotIDs := make([]string, len(modules))
			for i, m := range modules {
				gotIDs[i] = m.ID()
			}
			sort.Strings(gotIDs)
			sort.Strings(tt.wantIDs)
			if len(gotIDs) != len(tt.wantIDs) {
				t.Fatalf("got %d modules %v, want %d modules %v", len(gotIDs), gotIDs, len(tt.wantIDs), tt.wantIDs)
			}
			for i := range gotIDs {
				if gotIDs[i] != tt.wantIDs[i] {
					t.Errorf("module[%d] = %q, want %q", i, gotIDs[i], tt.wantIDs[i])
				}
			}
		})
	}
}

func TestFilesToLibraryPaths(t *testing.T) {
	index := discovery.NewModuleIndex(nil)
	client := NewClient(t.TempDir())
	detector := NewChangedModulesDetector(client, index, "/root")

	tests := []struct {
		name         string
		files        []string
		libraryPaths []string
		want         []string
	}{
		{
			name:         "file under library path",
			files:        []string{"_modules/kafka/main.tf"},
			libraryPaths: []string{"_modules"},
			want:         []string{"/root/_modules/kafka"},
		},
		{
			name:         "file in nested library dir",
			files:        []string{"_modules/kafka/src/main.tf"},
			libraryPaths: []string{"_modules"},
			want:         []string{"/root/_modules/kafka"},
		},
		{
			name:         "file not under library path",
			files:        []string{"myapp/prod/us-east-1/vpc/main.tf"},
			libraryPaths: []string{"_modules"},
			want:         nil,
		},
		{
			name:         "non-terraform file under library",
			files:        []string{"_modules/kafka/README.md"},
			libraryPaths: []string{"_modules"},
			want:         nil,
		},
		{
			name:         "multiple files same library deduped",
			files:        []string{"_modules/kafka/main.tf", "_modules/kafka/variables.tf"},
			libraryPaths: []string{"_modules"},
			want:         []string{"/root/_modules/kafka"},
		},
		{
			name:         "multiple library paths",
			files:        []string{"_modules/kafka/main.tf", "shared/modules/vpc/main.tf"},
			libraryPaths: []string{"_modules", "shared/modules"},
			want:         []string{"/root/_modules/kafka", "/root/shared/modules/vpc"},
		},
		{
			name:         "file directly in library root",
			files:        []string{"_modules/main.tf"},
			libraryPaths: []string{"_modules"},
			want:         []string{"/root/_modules/main.tf"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detector.filesToLibraryPaths(tt.files, tt.libraryPaths)
			sort.Strings(got)
			sort.Strings(tt.want)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("path[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestFindOwningModule(t *testing.T) {
	mod := makeModule("myapp/prod/us-east-1/vpc")
	index := discovery.NewModuleIndex([]*discovery.Module{mod})
	client := NewClient(t.TempDir())
	detector := NewChangedModulesDetector(client, index, "")

	tests := []struct {
		name string
		dir  string
		want string
	}{
		{
			name: "exact module path",
			dir:  "myapp/prod/us-east-1/vpc",
			want: "myapp/prod/us-east-1/vpc",
		},
		{
			name: "subdirectory of module",
			dir:  "myapp/prod/us-east-1/vpc/subdir",
			want: "myapp/prod/us-east-1/vpc",
		},
		{
			name: "unknown path",
			dir:  "unknown/path/to/module",
			want: "",
		},
		{
			name: "parent of module",
			dir:  "myapp/prod/us-east-1",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detector.findOwningModule(tt.dir)
			if tt.want == "" {
				if got != nil {
					t.Errorf("findOwningModule(%q) = %v, want nil", tt.dir, got.ID())
				}
				return
			}
			if got == nil {
				t.Fatalf("findOwningModule(%q) = nil, want %q", tt.dir, tt.want)
			}
			if got.ID() != tt.want {
				t.Errorf("findOwningModule(%q) = %q, want %q", tt.dir, got.ID(), tt.want)
			}
		})
	}
}

func TestNewChangedModulesDetector(t *testing.T) {
	client := NewClient("/some/dir")
	index := discovery.NewModuleIndex(nil)
	detector := NewChangedModulesDetector(client, index, "/root")

	if detector.gitClient != client {
		t.Error("gitClient not set")
	}
	if detector.index != index {
		t.Error("index not set")
	}
	if detector.rootDir != "/root" {
		t.Errorf("rootDir = %q, want /root", detector.rootDir)
	}
}

func TestDetectChangedModules_Integration(t *testing.T) {
	dir, repo := initTestRepo(t)

	// Create module structure and add terraform files
	modPath := "myapp/prod/us-east-1/vpc"
	mod := discovery.NewModule(
		[]string{"service", "environment", "region", "module"},
		[]string{"myapp", "prod", "us-east-1", "vpc"},
		filepath.Join(dir, modPath),
		modPath,
	)
	index := discovery.NewModuleIndex([]*discovery.Module{mod})

	// Add a terraform file in a second commit
	addCommit(t, dir, repo, filepath.Join(modPath, "main.tf"), "resource {}", "add vpc module")

	client := NewClient(dir)
	detector := NewChangedModulesDetector(client, index, "")

	modules, err := detector.DetectChangedModules("HEAD~1")
	if err != nil {
		t.Fatalf("DetectChangedModules error: %v", err)
	}
	if len(modules) != 1 {
		t.Fatalf("got %d modules, want 1", len(modules))
	}
	if modules[0].ID() != modPath {
		t.Errorf("module ID = %q, want %q", modules[0].ID(), modPath)
	}
}

func TestDetectChangedModulesVerbose_Integration(t *testing.T) {
	dir, repo := initTestRepo(t)

	modPath := "myapp/prod/us-east-1/vpc"
	mod := discovery.NewModule(
		[]string{"service", "environment", "region", "module"},
		[]string{"myapp", "prod", "us-east-1", "vpc"},
		filepath.Join(dir, modPath),
		modPath,
	)
	index := discovery.NewModuleIndex([]*discovery.Module{mod})

	addCommit(t, dir, repo, filepath.Join(modPath, "main.tf"), "resource {}", "add vpc module")

	client := NewClient(dir)
	detector := NewChangedModulesDetector(client, index, "")

	modules, files, err := detector.DetectChangedModulesVerbose("HEAD~1")
	if err != nil {
		t.Fatalf("DetectChangedModulesVerbose error: %v", err)
	}
	if len(modules) != 1 {
		t.Fatalf("got %d modules, want 1", len(modules))
	}
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	if files[0] != filepath.Join(modPath, "main.tf") {
		t.Errorf("files[0] = %q, want %q", files[0], filepath.Join(modPath, "main.tf"))
	}
}

func TestDetectUncommittedModules_Integration(t *testing.T) {
	dir, _ := initTestRepo(t)

	modPath := "myapp/prod/us-east-1/vpc"
	mod := discovery.NewModule(
		[]string{"service", "environment", "region", "module"},
		[]string{"myapp", "prod", "us-east-1", "vpc"},
		filepath.Join(dir, modPath),
		modPath,
	)
	index := discovery.NewModuleIndex([]*discovery.Module{mod})

	// Create an uncommitted terraform file
	fullDir := filepath.Join(dir, modPath)
	if err := os.MkdirAll(fullDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fullDir, "main.tf"), []byte("resource {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	client := NewClient(dir)
	detector := NewChangedModulesDetector(client, index, "")

	modules, err := detector.DetectUncommittedModules()
	if err != nil {
		t.Fatalf("DetectUncommittedModules error: %v", err)
	}
	if len(modules) != 1 {
		t.Fatalf("got %d modules, want 1", len(modules))
	}
	if modules[0].ID() != modPath {
		t.Errorf("module ID = %q, want %q", modules[0].ID(), modPath)
	}
}

func TestGetChangedModuleIDs_Integration(t *testing.T) {
	dir, repo := initTestRepo(t)

	modPath := "myapp/prod/us-east-1/vpc"
	mod := discovery.NewModule(
		[]string{"service", "environment", "region", "module"},
		[]string{"myapp", "prod", "us-east-1", "vpc"},
		filepath.Join(dir, modPath),
		modPath,
	)
	index := discovery.NewModuleIndex([]*discovery.Module{mod})

	addCommit(t, dir, repo, filepath.Join(modPath, "main.tf"), "resource {}", "add vpc module")

	client := NewClient(dir)
	detector := NewChangedModulesDetector(client, index, "")

	ids, err := detector.GetChangedModuleIDs("HEAD~1")
	if err != nil {
		t.Fatalf("GetChangedModuleIDs error: %v", err)
	}
	if len(ids) != 1 {
		t.Fatalf("got %d IDs, want 1", len(ids))
	}
	if ids[0] != modPath {
		t.Errorf("id = %q, want %q", ids[0], modPath)
	}
}

func TestDetectChangedLibraryModules_Integration(t *testing.T) {
	dir, repo := initTestRepo(t)

	index := discovery.NewModuleIndex(nil)

	addCommit(t, dir, repo, "_modules/kafka/main.tf", "module {}", "add library module")

	client := NewClient(dir)
	detector := NewChangedModulesDetector(client, index, dir)

	paths, err := detector.DetectChangedLibraryModules("HEAD~1", []string{"_modules"})
	if err != nil {
		t.Fatalf("DetectChangedLibraryModules error: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("got %d paths, want 1: %v", len(paths), paths)
	}
	want := filepath.Join(dir, "_modules", "kafka")
	if paths[0] != want {
		t.Errorf("path = %q, want %q", paths[0], want)
	}
}

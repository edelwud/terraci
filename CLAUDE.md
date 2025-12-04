# TerraCi - AI Assistant Guide

CLI tool for analyzing Terraform projects, building dependency graphs, and generating GitLab CI pipelines.

## Project Structure

```
terraci/
├── cmd/terraci/
│   ├── main.go                 # Entry point
│   └── cmd/                    # Cobra commands
│       ├── root.go             # Root command, global flags
│       ├── generate.go         # Pipeline generation
│       ├── validate.go         # Project validation
│       ├── graph.go            # Graph visualization
│       ├── init.go             # Config initialization
│       └── version.go          # Version info
├── internal/
│   ├── discovery/              # Module discovery
│   │   └── module.go           # Scanner, Module, ModuleIndex
│   ├── parser/                 # HCL parsing
│   │   ├── hcl.go              # Parser, ParsedModule, RemoteStateRef
│   │   └── dependency.go       # DependencyExtractor
│   ├── graph/                  # Dependency graph
│   │   └── dependency.go       # DependencyGraph, TopologicalSort
│   ├── pipeline/gitlab/        # GitLab CI generation
│   │   └── generator.go        # Generator, Pipeline, Job
│   ├── filter/                 # Module filtering
│   │   └── glob.go             # GlobFilter, CompositeFilter
│   └── git/                    # Git integration
│       └── diff.go             # Client, ChangedModulesDetector
├── pkg/config/                 # Public configuration package
│   └── config.go               # Config, Load(), Validate()
├── Makefile
├── go.mod
└── .terraci.example.yaml
```

## Key Types

### discovery.Module
```go
type Module struct {
    Service      string    // cdp
    Environment  string    // stage, prod
    Region       string    // eu-central-1
    Module       string    // vpc, eks
    Submodule    string    // optional: rabbitmq (for ec2/rabbitmq)
    Path         string    // absolute path
    RelativePath string    // relative path
    Parent       *Module   // parent module reference
    Children     []*Module // child submodules
}

func (m *Module) ID() string      // service/env/region/module[/submodule]
func (m *Module) Name() string    // module or module/submodule
func (m *Module) IsSubmodule() bool
```

### discovery.Scanner
```go
type Scanner struct {
    RootDir  string
    MinDepth int  // default: 4
    MaxDepth int  // default: 5
}

func (s *Scanner) Scan() ([]*Module, error)
```

Directory pattern: `service/environment/region/module[/submodule]`

### parser.RemoteStateRef
```go
type RemoteStateRef struct {
    Name         string            // data block name
    Backend      string            // s3, gcs, etc.
    Config       map[string]string // backend config
    ForEach      bool              // has for_each
    WorkspaceDir string            // resolved path
}
```

### graph.DependencyGraph
```go
func (g *DependencyGraph) AddNode(module *discovery.Module)
func (g *DependencyGraph) AddEdge(from, to *discovery.Module)
func (g *DependencyGraph) TopologicalSort() ([]*discovery.Module, error)
func (g *DependencyGraph) ExecutionLevels() [][]*discovery.Module
func (g *DependencyGraph) DetectCycles() [][]string
func (g *DependencyGraph) ToDOT() string
```

## CLI Commands

```bash
# Pipeline generation
terraci generate -o .gitlab-ci.yml
terraci generate --changed-only --base-ref main
terraci generate --exclude "*/test/*" --environment prod

# Validation
terraci validate

# Dependency graph
terraci graph --format dot -o deps.dot
terraci graph --format levels
terraci graph --module cdp/stage/eu-central-1/vpc --dependents

# Initialization
terraci init
```

## Global Flags

- `-c, --config` — config file path (defaults to `.terraci.yaml`)
- `-d, --dir` — working directory
- `-v, --verbose` — verbose output

## Configuration (.terraci.yaml)

```yaml
structure:
  pattern: "{service}/{environment}/{region}/{module}"
  min_depth: 4
  max_depth: 5
  allow_submodules: true

exclude:
  - "*/test/*"
  - "*/sandbox/*"

gitlab:
  terraform_image: "hashicorp/terraform:1.6"
  parallelism: 5
  plan_enabled: true
  auto_approve: false
  tags: [terraform, docker]
```

## Build and Test

```bash
make build      # Build binary
make test       # Run tests
make lint       # Lint code
make install    # Install to $GOPATH/bin
```

## Data Flow

1. `Scanner.Scan()` — discover modules in directories
2. `ModuleIndex` — index for fast lookups
3. `Parser.ParseModule()` — parse HCL, extract locals and remote_state
4. `DependencyExtractor` — determine dependencies between modules
5. `DependencyGraph` — build DAG, topological sort
6. `Generator.Generate()` — generate GitLab CI YAML

## Algorithms

- **Topological Sort**: Kahn's algorithm for module ordering
- **Cycle Detection**: DFS for finding circular dependencies
- **Execution Levels**: grouping modules for parallel execution
- **Path Resolution**: variable interpolation in state file paths

## Dependencies

- `github.com/spf13/cobra` — CLI framework
- `github.com/hashicorp/hcl/v2` — HCL parsing
- `github.com/zclconf/go-cty` — CTY types for HCL
- `gopkg.in/yaml.v3` — YAML

## Known Behaviors

- Modules can exist at depth 4 (base) and depth 5 (submodules) simultaneously
- `for_each` in remote_state expands to multiple dependencies
- Filters support `**` for arbitrary path depth

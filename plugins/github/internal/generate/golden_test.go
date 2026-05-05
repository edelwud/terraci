package generate

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
)

// updateGolden allows refreshing the YAML fixtures with `go test -update`.
var updateGolden = flag.Bool("update", false, "regenerate golden YAML fixtures")

// goldenCase locks a deterministic generator scenario against silent YAML
// regressions. Run `go test -run TestGoldenYAML -update ./plugins/github/...`
// after intentional shape changes to refresh fixtures.
type goldenCase struct {
	name      string
	scenario  func(t *testing.T) *generatorScenario
	goldenRel string
}

func TestGoldenYAML(t *testing.T) {
	cases := []goldenCase{
		{
			name: "single_module",
			scenario: func(t *testing.T) *generatorScenario {
				return newGeneratorScenario(t).
					withModules(discovery.TestModule("platform", "stage", "eu-central-1", "vpc"))
			},
			goldenRel: "testdata/golden/single_module.yaml",
		},
		{
			name: "two_modules_with_dependency",
			scenario: func(t *testing.T) *generatorScenario {
				vpc := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
				eks := discovery.TestModule("platform", "stage", "eu-central-1", "eks")
				return newGeneratorScenario(t).
					withModules(vpc, eks).
					withDependencies(map[string][]string{
						eks.ID(): {vpc.ID()},
					})
			},
			goldenRel: "testdata/golden/two_modules_with_dependency.yaml",
		},
		{
			name: "plan_only",
			scenario: func(t *testing.T) *generatorScenario {
				return newGeneratorScenario(t).
					withConfig(func(cfg *configpkg.Config) { cfg.PlanOnly = true }).
					withModules(discovery.TestModule("platform", "stage", "eu-central-1", "vpc"))
			},
			goldenRel: "testdata/golden/plan_only.yaml",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			scenario := tc.scenario(t)
			workflow := scenario.generate()
			yamlBytes, err := workflow.ToYAML()
			if err != nil {
				t.Fatalf("ToYAML() error = %v", err)
			}

			if *updateGolden {
				if mkErr := os.MkdirAll(filepath.Dir(tc.goldenRel), 0o755); mkErr != nil {
					t.Fatalf("MkdirAll: %v", mkErr)
				}
				if wErr := os.WriteFile(tc.goldenRel, yamlBytes, 0o644); wErr != nil {
					t.Fatalf("write golden: %v", wErr)
				}
				t.Logf("wrote %s (%d bytes)", tc.goldenRel, len(yamlBytes))
				return
			}

			want, readErr := os.ReadFile(tc.goldenRel)
			if readErr != nil {
				t.Fatalf("read golden %s: %v (run `go test -update` to regenerate)", tc.goldenRel, readErr)
			}
			if !bytes.Equal(yamlBytes, want) {
				t.Errorf("golden YAML mismatch for %s.\n--- got ---\n%s\n--- want ---\n%s",
					tc.name, string(yamlBytes), string(want))
			}
		})
	}
}

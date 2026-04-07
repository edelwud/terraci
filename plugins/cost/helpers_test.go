package cost

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	"github.com/edelwud/terraci/pkg/plugin/registry"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/engine"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/diskblob"
)

// newRuntimeWithEstimator wraps an estimator in a costRuntime for tests that
// need to bypass the full runtime construction path.
func newRuntimeWithEstimator(estimator *engine.Estimator) *costRuntime {
	return &costRuntime{estimator: estimator}
}

// testPlanEC2 is a minimal plan.json with a single aws_instance.web (t3.micro, create).
const testPlanEC2 = `{
	"format_version": "1.2",
	"terraform_version": "1.6.0",
	"resource_changes": [{
		"address": "aws_instance.web",
		"module_address": "",
		"type": "aws_instance",
		"name": "web",
		"change": {
			"actions": ["create"],
			"before": null,
			"after": {"instance_type": "t3.micro", "ami": "ami-12345"},
			"after_unknown": {}
		}
	}]
}`

// newTestPlugin creates a fresh Plugin with the same configuration as the one
// registered in init(), but without touching the global plugin registry.
func newTestPlugin(t *testing.T) *Plugin {
	t.Helper()
	registry.ResetPlugins()
	p := &Plugin{
		BasePlugin: plugin.BasePlugin[*model.CostConfig]{
			PluginName: "cost",
			PluginDesc: "Cloud cost estimation from Terraform plans",
			EnableMode: plugin.EnabledExplicitly,
			DefaultCfg: func() *model.CostConfig {
				return &model.CostConfig{}
			},
			IsEnabledFn: func(cfg *model.CostConfig) bool {
				return cfg != nil && cfg.HasEnabledProviders()
			},
		},
	}
	t.Cleanup(p.Reset)
	return p
}

// enablePlugin configures the plugin with the given config, marking it as configured.
func enablePlugin(t *testing.T, p *Plugin, cfg *model.CostConfig) {
	t.Helper()
	p.SetTypedConfig(cfg)
}

// fakePricingServer returns an httptest server that serves minimal AWS pricing JSON
// for EC2 (t3.micro) and RDS (db.t3.micro). Server is automatically closed via t.Cleanup.
func fakePricingServer(t *testing.T) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Serve EC2 pricing for any request — sufficient for t3.micro lookups
		fmt.Fprint(w, `{
			"formatVersion": "v1.0",
			"offerCode": "AmazonEC2",
			"version": "test",
			"products": {
				"SKU1": {
					"sku": "SKU1",
					"productFamily": "Compute Instance",
					"attributes": {
						"instanceType": "t3.micro",
						"location": "US East (N. Virginia)",
						"tenancy": "Shared",
						"operatingSystem": "Linux",
						"preInstalledSw": "NA",
						"capacitystatus": "Used"
					}
				},
				"SKU2": {
					"sku": "SKU2",
					"productFamily": "Storage",
					"attributes": {
						"volumeApiName": "gp2",
						"location": "US East (N. Virginia)"
					}
				}
			},
			"terms": {
				"OnDemand": {
					"SKU1": {
						"SKU1.T1": {
							"offerTermCode": "JRTCKXETXF",
							"sku": "SKU1",
							"priceDimensions": {
								"SKU1.T1.D1": {
									"unit": "Hrs",
									"pricePerUnit": {"USD": "0.0104"}
								}
							}
						}
					},
					"SKU2": {
						"SKU2.T1": {
							"offerTermCode": "JRTCKXETXF",
							"sku": "SKU2",
							"priceDimensions": {
								"SKU2.T1.D1": {
									"unit": "GB-Mo",
									"pricePerUnit": {"USD": "0.10"}
								}
							}
						}
					}
				}
			}
		}`)
	}))
	t.Cleanup(ts.Close)
	return ts
}

// newTestEstimator creates an Estimator backed by a fake pricing server.
func newTestEstimator(t *testing.T) *engine.Estimator {
	t.Helper()
	ts := fakePricingServer(t)
	cacheDir := filepath.Join(t.TempDir(), "cache")
	fetcher := &awskit.Fetcher{
		Client:  ts.Client(),
		BaseURL: ts.URL,
	}
	cfg := &model.CostConfig{
		Providers: model.CostProvidersConfig{"aws": {Enabled: true}},
	}
	cache := blobcache.New(diskblob.NewStore(cacheDir), model.DefaultBlobCacheNamespace, cfg.CacheTTLDuration())
	e, err := engine.NewEstimatorFromConfig(cfg, cache)
	if err != nil {
		t.Fatalf("newTestEstimator: %v", err)
	}
	e.SetFetcherForProvider(awskit.ProviderID, fetcher)
	return e
}

// writePlanJSON creates the module directory and writes plan.json into it.
// If planJSON is empty, uses testPlanEC2 as default.
func writePlanJSON(t *testing.T, moduleDir, planJSON string) {
	t.Helper()
	if planJSON == "" {
		planJSON = testPlanEC2
	}
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "plan.json"), []byte(planJSON), 0o600); err != nil {
		t.Fatal(err)
	}
}

// newTestAppContext creates a minimal AppContext suitable for plugin testing.
// workDir should contain the module directories with plan.json files.
func newTestAppContext(t *testing.T, workDir string) *plugin.AppContext {
	return plugintest.NewAppContext(t, workDir)
}

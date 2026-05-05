# TerraCi Examples

Example configurations demonstrating TerraCi features.

## Examples

| Example | Description |
|---------|-------------|
| [basic/](basic/) | Minimal setup — 3 modules with dependencies |
| [github-actions/](github-actions/) | GitHub Actions configuration with OIDC |
| [cross-env-deps/](cross-env-deps/) | Cross-environment dependencies and `abspath` pattern |
| [library-modules/](library-modules/) | Shared library modules with change detection |
| [cost-estimation/](cost-estimation/) | AWS cost estimation in MR/PR comments |
| [policy-checks/](policy-checks/) | OPA policy enforcement with Rego rules |
| [external-plugin/](external-plugin/) | Minimal CLI-only custom plugin built with xterraci |
| [plugin-skeleton/](plugin-skeleton/) | Reference plugin showing producer + consumer report patterns |

## Quick Start

```bash
# Copy an example
cp -r examples/basic my-infra
cd my-infra

# Initialize
terraci init

# Validate
terraci validate

# Generate pipeline
terraci generate -o .gitlab-ci.yml         # GitLab CI
terraci generate -o .github/workflows/terraform.yml  # GitHub Actions

# Preview dependencies
terraci graph --format levels
```

## Configuration Reference

Add the schema comment to your `.terraci.yaml` for IDE autocomplete:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/edelwud/terraci/main/terraci.schema.json
```

See [full documentation](https://edelwud.github.io/terraci) for details.

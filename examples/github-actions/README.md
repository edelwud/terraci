# GitHub Actions Example

TerraCi with GitHub Actions instead of GitLab CI.

## Configuration

Uses `terraci init --ci --provider github`; outside GitHub Actions, set `TERRACI_PROVIDER=github` when generating.

## Usage

```bash
terraci init --ci --provider github
terraci generate -o .github/workflows/terraform.yml
```

## CI Integration

GitHub Actions doesn't support dynamic workflow generation at runtime.
Use a **pre-commit hook** to regenerate the workflow on each commit:

```bash
# .husky/pre-commit
terraci generate -o .github/workflows/terraform.yml
git add .github/workflows/terraform.yml
```

# GitHub Actions Example

TerraCi with GitHub Actions instead of GitLab CI.

## Configuration

Uses `provider: github` to generate GitHub Actions workflow files.

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

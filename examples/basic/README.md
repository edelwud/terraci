# Basic Example

Minimal TerraCi setup with 3 modules: VPC -> EKS -> App.

## Structure

```
platform/stage/eu-central-1/
├── vpc/       # No dependencies
├── eks/       # Depends on VPC
└── app/       # Depends on EKS
```

## Usage

```bash
terraci init --ci
terraci validate
terraci generate -o .gitlab-ci.yml        # GitLab
terraci generate -o .github/workflows/terraform.yml  # GitHub
terraci graph --format levels
```

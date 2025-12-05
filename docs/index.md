---
layout: home

hero:
  name: "TerraCi"
  text: "Terraform Pipeline Generator"
  tagline: Automatically generate GitLab CI pipelines with proper dependency ordering for your Terraform/OpenTofu monorepos
  image:
    src: /logo.svg
    alt: TerraCi
  actions:
    - theme: brand
      text: Get Started
      link: /guide/getting-started
    - theme: alt
      text: View on GitHub
      link: https://github.com/edelwud/terraci

features:
  - icon: 🔍
    title: Smart Discovery
    details: Automatically discovers Terraform modules based on your directory structure. Supports nested submodules at depth 4 and 5.
  - icon: 🔗
    title: Dependency Resolution
    details: Parses terraform_remote_state blocks to build an accurate dependency graph. Handles for_each and dynamic references.
  - icon: ⚡
    title: Parallel Execution
    details: Groups independent modules into execution levels for maximum parallelism while respecting dependencies.
  - icon: 🎯
    title: Changed-Only Pipelines
    details: Git integration detects changed files and generates pipelines only for affected modules and their dependents.
  - icon: 🔄
    title: OpenTofu Support
    details: First-class support for both Terraform and OpenTofu. Just change a single config option.
  - icon: 📊
    title: Graph Visualization
    details: Export dependency graphs to DOT format for visualization with GraphViz.
---

## Quick Example

```bash
# Initialize configuration
terraci init

# Generate pipeline for all modules
terraci generate -o .gitlab-ci.yml

# Generate pipeline only for changed modules
terraci generate --changed-only --base-ref main -o .gitlab-ci.yml
```

## How It Works

TerraCi analyzes your Terraform project structure:

```
infrastructure/
├── service/
│   └── environment/
│       └── region/
│           ├── vpc/          # Module at depth 4
│           ├── eks/          # Depends on vpc
│           └── ec2/
│               └── rabbitmq/ # Submodule at depth 5
```

It parses `terraform_remote_state` data sources to understand dependencies:

```hcl
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    bucket = "terraform-state"
    key    = "cdp/stage/eu-central-1/vpc/terraform.tfstate"
  }
}
```

And generates a GitLab CI pipeline with proper job ordering:

```yaml
stages:
  - deploy-plan-0
  - deploy-apply-0
  - deploy-plan-1
  - deploy-apply-1

plan-cdp-stage-eu-central-1-vpc:
  stage: deploy-plan-0
  script:
    - ${TERRAFORM_BINARY} plan -out=plan.tfplan

plan-cdp-stage-eu-central-1-eks:
  stage: deploy-plan-1
  needs:
    - apply-cdp-stage-eu-central-1-vpc
```


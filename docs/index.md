---
layout: home

hero:
  name: TerraCi
  text: Terraform Pipeline Generator
  tagline: Dependency-aware CI pipelines for Terraform/OpenTofu monorepos — GitLab CI & GitHub Actions, with cost estimation and policy checks
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
  - icon:
      src: /icons/graph.svg
    title: Dependency Graph
    details: Parses terraform_remote_state to build a DAG. Topological sort ensures correct execution order across modules.
    link: /guide/dependencies
    linkText: How it works
  - icon:
      src: /icons/zap.svg
    title: Parallel Execution
    details: Groups modules into execution levels. Independent modules run in parallel — only dependent modules wait.
    link: /guide/pipeline-generation
    linkText: See pipeline structure
  - icon:
      src: /icons/git.svg
    title: Changed-Only Mode
    details: Detects modified files via git diff. Generates pipelines only for affected modules and their dependents.
    link: /guide/git-integration
    linkText: Git integration
  - icon:
      src: /icons/shield.svg
    title: Policy Checks
    details: Enforce compliance with OPA policies on every plan. Block or warn on violations — results appear in MR comments.
    link: /config/policy
    linkText: Configure policies
  - icon:
      src: /icons/dollar.svg
    title: Cost Estimation
    details: Estimate monthly AWS costs from terraform plans. See before/after diffs per module in MR comments.
    link: /config/cost
    linkText: Configure costs
  - icon:
      src: /icons/tofu.svg
    title: OpenTofu Ready
    details: First-class support for both Terraform and OpenTofu. Switch with a single config option.
    link: /guide/opentofu
    linkText: Configure
---

## Quick Start

```bash
# Install
brew install edelwud/tap/terraci

# Initialize & generate (GitLab)
terraci init
terraci generate -o .gitlab-ci.yml

# Initialize & generate (GitHub Actions)
terraci init --provider github
terraci generate -o .github/workflows/terraform.yml

# Only changed modules
terraci generate --changed-only --base-ref main
```

## How It Works

```mermaid
flowchart LR
  subgraph repo["Your Repo"]
    r1["vpc/"]
    r2["eks/"]
    r3["rds/"]
  end
  subgraph terraci["TerraCi"]
    t1["Discover"] --> t2["Parse"] --> t3["Sort"] --> t4["Generate"]
  end
  subgraph ci["CI Pipeline"]
    g1["plan vpc"] --> g2["apply vpc"]
    g2 --> g3["plan eks + rds"]
    g3 --> g4["apply eks + rds"]
  end
  repo --> terraci --> ci
```

[Full configuration reference →](/config/)

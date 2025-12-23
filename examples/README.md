# TerraCi Examples

Example GitLab CI configuration that uses TerraCi to generate and run Terraform pipelines.

## Files

- `.gitlab-ci.yml` - Parent pipeline that generates child pipeline with TerraCi
- `.terraci.yaml` - TerraCi configuration

## How It Works

```
┌─────────────────────────────────────────────────────────────┐
│                    Parent Pipeline                          │
├─────────────────────────────────────────────────────────────┤
│  generate-pipeline:                                         │
│    - terraci generate -o terraform.gitlab-ci.yml            │
│    - artifact: terraform.gitlab-ci.yml                      │
│                         │                                   │
│                         ▼                                   │
│  deploy:                                                    │
│    - trigger child pipeline (terraform.gitlab-ci.yml)       │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│              Child Pipeline (generated)                     │
├─────────────────────────────────────────────────────────────┤
│  plan-vpc → apply-vpc → plan-eks → apply-eks → ...          │
└─────────────────────────────────────────────────────────────┘
```

## Usage

1. Copy `.gitlab-ci.yml` and `.terraci.yaml` to your Terraform project
2. Adjust `.terraci.yaml` for your project structure
3. Commit and push - GitLab will run the pipeline

## Pipeline Stages

1. **generate** - TerraCi generates the Terraform pipeline
2. **deploy** - Triggers the generated child pipeline with all Terraform jobs

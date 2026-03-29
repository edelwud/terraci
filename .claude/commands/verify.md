Run full PR-level verification locally.

Execute `task ci:pr` which runs: deps → fmt-check → lint → unit tests → integration tests → goreleaser check.

Analyze any failures and fix them. This is the same pipeline that CI runs, so everything must pass before creating a PR.

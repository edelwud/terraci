Run security analysis on the project.

Execute `task dev:security` to run govulncheck.

For each vulnerability found:
1. Identify which dependency has the vulnerability
2. Check if an update is available with `go list -m -u <module>`
3. Suggest or apply the fix (usually `go get <module>@latest && go mod tidy`)

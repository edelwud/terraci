package policyengine

type CheckRequest struct {
	ModulePath string
}

type PullRequest struct {
	CacheDir string
}

type PullResult struct {
	PolicyDirs []string
	CacheDir   string
}

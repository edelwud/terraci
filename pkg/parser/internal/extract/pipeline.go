package extract

type extractorStep func(*Context)

func RunDefault(ctx *Context) {
	for _, step := range defaultPipeline() {
		step(ctx)
	}
}

func defaultPipeline() []extractorStep {
	return []extractorStep{
		extractLocals,
		extractTfvars,
		extractBackendConfig,
		extractRequiredProviders,
		extractLockFile,
		extractRemoteStates,
		extractModuleCalls,
	}
}

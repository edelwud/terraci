package extract

type session struct {
	ctx *Context
}

func newSession(ctx *Context) *session {
	return &session{ctx: ctx}
}

func (s *session) Run() {
	for _, step := range s.pipeline() {
		step(s.ctx)
	}
}

func (s *session) pipeline() []extractorStep {
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

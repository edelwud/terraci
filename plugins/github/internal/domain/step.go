package domain

type StepOptions struct {
	Name            string
	Uses            string
	With            map[string]string
	Run             string
	Env             map[string]string
	If              string
	ContinueOnError bool
}

type Step struct {
	name            string
	uses            string
	with            map[string]string
	run             string
	env             map[string]string
	ifExpr          string
	continueOnError bool
}

func NewStep(opts StepOptions) Step {
	return Step{
		name:            opts.Name,
		uses:            opts.Uses,
		with:            cloneStringMap(opts.With),
		run:             opts.Run,
		env:             cloneStringMap(opts.Env),
		ifExpr:          opts.If,
		continueOnError: opts.ContinueOnError,
	}
}

func (s Step) Name() string { return s.name }

func (s Step) Uses() string { return s.uses }

func (s Step) With() map[string]string { return cloneStringMap(s.with) }

func (s Step) Run() string { return s.run }

func (s Step) Env() map[string]string { return cloneStringMap(s.env) }

func (s Step) If() string { return s.ifExpr }

func (s Step) ContinueOnError() bool { return s.continueOnError }

func (s Step) clone() Step {
	return NewStep(StepOptions{
		Name:            s.name,
		Uses:            s.uses,
		With:            s.with,
		Run:             s.run,
		Env:             s.env,
		If:              s.ifExpr,
		ContinueOnError: s.continueOnError,
	})
}

package filter

// Flags holds parsed filter values from CLI flags.
// Source-agnostic — works with any CLI framework (cobra, etc.).
// Plugins and core commands both use this to collect filter inputs.
type Flags struct {
	Excludes    []string
	Includes    []string
	SegmentArgs []string
}

// Merge combines config defaults with flag overrides into filter Options.
func (f *Flags) Merge(cfgExcludes, cfgIncludes []string) Options {
	if f == nil {
		f = &Flags{}
	}
	return Options{
		Excludes: append(append([]string{}, cfgExcludes...), f.Excludes...),
		Includes: append(append([]string{}, cfgIncludes...), f.Includes...),
		Segments: ParseSegmentFilters(f.SegmentArgs),
	}
}

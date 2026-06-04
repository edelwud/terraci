package ci

import (
	"encoding/json"
	"time"
)

// Report is a CI enrichment artifact written by a tool and consumed by report
// aggregation flows. Producers persist reports as
// {serviceDir}/{producer}-report.json.
type Report struct {
	producer   string
	title      string
	status     ReportStatus
	summary    string
	provenance *ReportProvenance
	sections   []ReportSection
}

type reportJSON struct {
	Producer   string            `json:"producer"`
	Title      string            `json:"title"`
	Status     ReportStatus      `json:"status"`
	Summary    string            `json:"summary"`
	Provenance *ReportProvenance `json:"provenance,omitempty"`
	Sections   []ReportSection   `json:"sections,omitempty"`
}

// Clone returns a defensive copy of the report.
func (r *Report) Clone() *Report {
	if r == nil {
		return nil
	}
	cloned := *r
	cloned.provenance = r.provenance.Clone()
	if len(r.sections) > 0 {
		cloned.sections = make([]ReportSection, len(r.sections))
		for i := range r.sections {
			cloned.sections[i] = r.sections[i].Clone()
		}
	} else {
		cloned.sections = nil
	}
	return &cloned
}

// Producer returns the report producer key.
func (r *Report) Producer() string {
	if r == nil {
		return ""
	}
	return r.producer
}

// Title returns the report title.
func (r *Report) Title() string {
	if r == nil {
		return ""
	}
	return r.title
}

// Status returns the report status.
func (r *Report) Status() ReportStatus {
	if r == nil {
		return ""
	}
	return r.status
}

// Summary returns the short report summary.
func (r *Report) Summary() string {
	if r == nil {
		return ""
	}
	return r.summary
}

// Provenance returns a defensive copy of report provenance.
func (r *Report) Provenance() *ReportProvenance {
	if r == nil {
		return nil
	}
	return r.provenance.Clone()
}

// Sections returns defensive copies of render-ready report sections.
func (r *Report) Sections() []ReportSection {
	if r == nil || len(r.sections) == 0 {
		return nil
	}
	sections := make([]ReportSection, len(r.sections))
	for i := range r.sections {
		sections[i] = r.sections[i].Clone()
	}
	return sections
}

// MarshalJSON preserves the public report artifact shape while keeping the
// in-process report immutable to external packages.
func (r *Report) MarshalJSON() ([]byte, error) {
	if r == nil {
		return []byte("null"), nil
	}
	return json.Marshal(reportJSON{
		Producer:   r.producer,
		Title:      r.title,
		Status:     r.status,
		Summary:    r.summary,
		Provenance: r.provenance.Clone(),
		Sections:   r.Sections(),
	})
}

// UnmarshalJSON decodes the public report artifact shape.
func (r *Report) UnmarshalJSON(data []byte) error {
	var raw reportJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.producer = raw.Producer
	r.title = raw.Title
	r.status = raw.Status
	r.summary = raw.Summary
	r.provenance = raw.Provenance.Clone()
	if len(raw.Sections) > 0 {
		r.sections = make([]ReportSection, len(raw.Sections))
		for i := range raw.Sections {
			r.sections[i] = raw.Sections[i].Clone()
		}
	} else {
		r.sections = nil
	}
	return nil
}

// ReportProvenance captures the source run identity for a persisted report.
//
// Producers should populate provenance for every persisted report so local
// consumers (e.g. localexec/render) can decide whether the artifact still
// matches the current workspace. Producers should derive it from
// ArtifactContext instead of assembling it by hand.
type ReportProvenance struct {
	generatedAt            time.Time
	commitSHA              string
	pipelineID             string
	planResultsFingerprint string
}

type reportProvenanceJSON struct {
	GeneratedAt            time.Time `json:"generated_at"`
	CommitSHA              string    `json:"commit_sha,omitempty"`
	PipelineID             string    `json:"pipeline_id,omitempty"`
	PlanResultsFingerprint string    `json:"plan_results_fingerprint,omitempty"`
}

// Clone returns a defensive copy of p.
func (p *ReportProvenance) Clone() *ReportProvenance {
	if p == nil {
		return nil
	}
	cloned := *p
	return &cloned
}

// GeneratedAt returns the report generation timestamp.
func (p *ReportProvenance) GeneratedAt() time.Time {
	if p == nil {
		return time.Time{}
	}
	return p.generatedAt
}

// CommitSHA returns the best-known source commit SHA.
func (p *ReportProvenance) CommitSHA() string {
	if p == nil {
		return ""
	}
	return p.commitSHA
}

// PipelineID returns the best-known CI pipeline identifier.
func (p *ReportProvenance) PipelineID() string {
	if p == nil {
		return ""
	}
	return p.pipelineID
}

// PlanResultsFingerprint returns the plan result collection fingerprint used
// to decide whether a report is fresh for local consumers.
func (p *ReportProvenance) PlanResultsFingerprint() string {
	if p == nil {
		return ""
	}
	return p.planResultsFingerprint
}

// MarshalJSON preserves the public provenance artifact shape.
func (p *ReportProvenance) MarshalJSON() ([]byte, error) {
	if p == nil {
		return []byte("null"), nil
	}
	return json.Marshal(reportProvenanceJSON{
		GeneratedAt:            p.generatedAt,
		CommitSHA:              p.commitSHA,
		PipelineID:             p.pipelineID,
		PlanResultsFingerprint: p.planResultsFingerprint,
	})
}

// UnmarshalJSON decodes persisted provenance.
func (p *ReportProvenance) UnmarshalJSON(data []byte) error {
	var raw reportProvenanceJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	p.generatedAt = raw.GeneratedAt
	p.commitSHA = raw.CommitSHA
	p.pipelineID = raw.PipelineID
	p.planResultsFingerprint = raw.PlanResultsFingerprint
	return nil
}

// ArtifactContext describes the current artifact-producing run. Producers pass
// it to NewRenderedReport so provenance is derived consistently for every
// persisted report.
type ArtifactContext struct {
	serviceDir             string
	workDir                string
	commitSHA              string
	pipelineID             string
	planResultsFingerprint string
	generatedAt            time.Time
}

// ArtifactContextOptions describes how to construct an ArtifactContext.
type ArtifactContextOptions struct {
	ServiceDir             string
	WorkDir                string
	CommitSHA              string
	PipelineID             string
	PlanResultsFingerprint string
	GeneratedAt            time.Time
}

// NewArtifactContext normalizes run metadata used by report producers.
func NewArtifactContext(opts ArtifactContextOptions) ArtifactContext {
	generatedAt := opts.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	return ArtifactContext{
		serviceDir:             opts.ServiceDir,
		workDir:                opts.WorkDir,
		commitSHA:              opts.CommitSHA,
		pipelineID:             opts.PipelineID,
		planResultsFingerprint: opts.PlanResultsFingerprint,
		generatedAt:            generatedAt.UTC(),
	}
}

// ServiceDir returns the service directory used for report artifacts.
func (c ArtifactContext) ServiceDir() string { return c.serviceDir }

// WorkDir returns the workspace directory for the artifact-producing run.
func (c ArtifactContext) WorkDir() string { return c.workDir }

// CommitSHA returns the best-known source commit SHA.
func (c ArtifactContext) CommitSHA() string { return c.commitSHA }

// PipelineID returns the best-known CI pipeline identifier.
func (c ArtifactContext) PipelineID() string { return c.pipelineID }

// PlanResultsFingerprint returns the plan result collection fingerprint.
func (c ArtifactContext) PlanResultsFingerprint() string { return c.planResultsFingerprint }

// GeneratedAt returns the normalized UTC artifact timestamp.
func (c ArtifactContext) GeneratedAt() time.Time { return c.generatedAt }

// ArtifactRun describes one producer's report/result artifact write.
type ArtifactRun struct {
	producer    string
	artifact    ArtifactContext
	planResults *PlanResultCollection
}

// ArtifactRunOptions describes how to construct an ArtifactRun.
type ArtifactRunOptions struct {
	Producer    string
	Artifact    ArtifactContext
	PlanResults *PlanResultCollection
}

// NewArtifactRun normalizes producer run metadata. When PlanResults is present
// and the artifact context has no explicit fingerprint, the plan collection
// fingerprint is used. Empty fingerprints remain valid for non-plan producers.
func NewArtifactRun(opts ArtifactRunOptions) (ArtifactRun, error) {
	if err := validateArtifactProducer(opts.Producer); err != nil {
		return ArtifactRun{}, err
	}

	artifact := opts.Artifact
	if opts.PlanResults != nil && artifact.planResultsFingerprint == "" {
		artifact.planResultsFingerprint = opts.PlanResults.Fingerprint()
	}
	artifact = normalizeArtifactContext(artifact)

	return ArtifactRun{
		producer:    opts.Producer,
		artifact:    artifact,
		planResults: opts.PlanResults.Clone(),
	}, nil
}

// Producer returns the artifact-producing plugin key.
func (r ArtifactRun) Producer() string { return r.producer }

// Artifact returns the normalized artifact context.
func (r ArtifactRun) Artifact() ArtifactContext { return r.artifact }

// PlanResults returns a defensive copy of the plan result collection.
func (r ArtifactRun) PlanResults() *PlanResultCollection {
	return r.planResults.Clone()
}

// Provenance converts the artifact context into persisted report provenance.
func (c ArtifactContext) Provenance() *ReportProvenance {
	generatedAt := c.generatedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	return &ReportProvenance{
		generatedAt:            generatedAt.UTC(),
		commitSHA:              c.commitSHA,
		pipelineID:             c.pipelineID,
		planResultsFingerprint: c.planResultsFingerprint,
	}
}

func normalizeArtifactContext(artifact ArtifactContext) ArtifactContext {
	return NewArtifactContext(ArtifactContextOptions{
		ServiceDir:             artifact.serviceDir,
		WorkDir:                artifact.workDir,
		CommitSHA:              artifact.commitSHA,
		PipelineID:             artifact.pipelineID,
		PlanResultsFingerprint: artifact.planResultsFingerprint,
		GeneratedAt:            artifact.generatedAt,
	})
}

// ReportSectionKind identifies a report section payload shape. Summary-facing
// producer reports should use ReportSectionKindRendered so consumers can render
// plugin output without knowing the producer's domain model.
type ReportSectionKind string

// ReportSection is a neutral value object describing one slice of a CI report.
// Persisted producer reports must publish ReportSectionKindRendered payloads.
// Use NewRenderedSection or NewRenderedReport instead of constructing report
// JSON or payloads by hand.
//
//nolint:recvcheck // MarshalJSON must work on values; UnmarshalJSON must mutate the receiver.
type ReportSection struct {
	kind           ReportSectionKind
	title          string
	status         ReportStatus
	sectionSummary string
	payload        json.RawMessage
}

type reportSectionJSON struct {
	Kind           ReportSectionKind `json:"kind"`
	Title          string            `json:"title,omitempty"`
	Status         ReportStatus      `json:"status,omitempty"`
	SectionSummary string            `json:"section_summary,omitempty"`
	Payload        json.RawMessage   `json:"payload,omitempty"`
}

// Kind returns the section payload shape.
func (s ReportSection) Kind() ReportSectionKind {
	return s.kind
}

// Title returns the section title.
func (s ReportSection) Title() string {
	return s.title
}

// Status returns the section status.
func (s ReportSection) Status() ReportStatus {
	return s.status
}

// Summary returns the short section summary.
func (s ReportSection) Summary() string {
	return s.sectionSummary
}

// Clone returns a defensive copy of the report section.
func (s ReportSection) Clone() ReportSection {
	cloned := s
	if len(s.payload) > 0 {
		cloned.payload = append([]byte(nil), s.payload...)
	}
	return cloned
}

// MarshalJSON preserves the public persisted report section shape while
// keeping the in-process value object immutable from external packages.
func (s ReportSection) MarshalJSON() ([]byte, error) {
	return json.Marshal(reportSectionJSON{
		Kind:           s.kind,
		Title:          s.title,
		Status:         s.status,
		SectionSummary: s.sectionSummary,
		Payload:        s.payload,
	})
}

// UnmarshalJSON preserves the public persisted report section shape while
// keeping direct payload construction out of the Go API.
func (s *ReportSection) UnmarshalJSON(data []byte) error {
	var raw reportSectionJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	s.kind = raw.Kind
	s.title = raw.Title
	s.status = raw.Status
	s.sectionSummary = raw.SectionSummary
	if len(raw.Payload) > 0 {
		s.payload = append([]byte(nil), raw.Payload...)
	} else {
		s.payload = nil
	}
	return nil
}

// ReportStatus indicates the outcome of a producer's check.
type ReportStatus string

const (
	ReportStatusPass ReportStatus = "pass"
	ReportStatusWarn ReportStatus = "warn"
	ReportStatusFail ReportStatus = "fail"
)

// Valid reports whether the status is one of the supported CI report outcomes.
func (s ReportStatus) Valid() bool {
	switch s {
	case ReportStatusPass, ReportStatusWarn, ReportStatusFail:
		return true
	default:
		return false
	}
}

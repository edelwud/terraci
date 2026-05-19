package ci

import (
	"encoding/json"
	"time"
)

// Report is a CI enrichment artifact written by a tool and consumed by report
// aggregation flows. Producers persist reports as
// {serviceDir}/{producer}-report.json.
type Report struct {
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
	if r.Provenance != nil {
		provenance := *r.Provenance
		cloned.Provenance = &provenance
	}
	if len(r.Sections) > 0 {
		cloned.Sections = make([]ReportSection, len(r.Sections))
		for i := range r.Sections {
			cloned.Sections[i] = r.Sections[i].Clone()
		}
	} else {
		cloned.Sections = nil
	}
	return &cloned
}

// ReportProvenance captures the source run identity for a persisted report.
//
// Producers should populate provenance for every persisted report so local
// consumers (e.g. localexec/render) can decide whether the artifact still
// matches the current workspace. Producers should derive it from
// ArtifactContext instead of assembling it by hand.
type ReportProvenance struct {
	GeneratedAt            time.Time `json:"generated_at"`
	CommitSHA              string    `json:"commit_sha,omitempty"`
	PipelineID             string    `json:"pipeline_id,omitempty"`
	PlanResultsFingerprint string    `json:"plan_results_fingerprint,omitempty"`
}

// ArtifactContext describes the current artifact-producing run. Producers pass
// it to NewRenderedReport so provenance is derived consistently for every
// persisted report.
type ArtifactContext struct {
	ServiceDir             string
	WorkDir                string
	CommitSHA              string
	PipelineID             string
	PlanResultsFingerprint string
	GeneratedAt            time.Time
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
		ServiceDir:             opts.ServiceDir,
		WorkDir:                opts.WorkDir,
		CommitSHA:              opts.CommitSHA,
		PipelineID:             opts.PipelineID,
		PlanResultsFingerprint: opts.PlanResultsFingerprint,
		GeneratedAt:            generatedAt.UTC(),
	}
}

// Provenance converts the artifact context into persisted report provenance.
func (c ArtifactContext) Provenance() *ReportProvenance {
	generatedAt := c.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	return &ReportProvenance{
		GeneratedAt:            generatedAt.UTC(),
		CommitSHA:              c.CommitSHA,
		PipelineID:             c.PipelineID,
		PlanResultsFingerprint: c.PlanResultsFingerprint,
	}
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

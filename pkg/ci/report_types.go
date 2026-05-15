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

// ReportProvenance captures the source run identity for a persisted report.
//
// Producers should populate provenance for every persisted report so local
// consumers (e.g. localexec/render) can decide whether the artifact still
// matches the current workspace. Use NewProvenance to fill GeneratedAt
// consistently and let producer-specific fields stay omitempty.
type ReportProvenance struct {
	GeneratedAt            time.Time `json:"generated_at"`
	CommitSHA              string    `json:"commit_sha,omitempty"`
	PipelineID             string    `json:"pipeline_id,omitempty"`
	PlanResultsFingerprint string    `json:"plan_results_fingerprint,omitempty"`
}

// NewProvenance returns a ReportProvenance with GeneratedAt = time.Now().UTC().
// Pass empty strings for fields the producer does not have — they remain
// omitempty in the resulting JSON.
func NewProvenance(commitSHA, pipelineID, planResultsFingerprint string) *ReportProvenance {
	return &ReportProvenance{
		GeneratedAt:            time.Now().UTC(),
		CommitSHA:              commitSHA,
		PipelineID:             pipelineID,
		PlanResultsFingerprint: planResultsFingerprint,
	}
}

// ReportSectionKind identifies a report section payload shape. Summary-facing
// producer reports should use ReportSectionKindRendered so consumers can render
// plugin output without knowing the producer's domain model.
type ReportSectionKind string

// ReportSection is a neutral envelope describing one slice of a CI report. The
// Payload is opaque JSON owned by the producer — consumers decode it according
// to Kind. Use EncodeRenderSection for summary-facing reports.
type ReportSection struct {
	Kind           ReportSectionKind `json:"kind"`
	Title          string            `json:"title,omitempty"`
	Status         ReportStatus      `json:"status,omitempty"`
	SectionSummary string            `json:"section_summary,omitempty"`
	Payload        json.RawMessage   `json:"payload,omitempty"`
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

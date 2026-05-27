package ci

// ReportSectionKindRendered is the canonical render-ready report payload. It is
// intentionally generic: producer plugins own their domain model and publish
// only typed view values needed by summary/local renderers.
const ReportSectionKindRendered ReportSectionKind = "rendered"

// RenderPayloadSchemaVersion is the current rendered payload wire schema.
const RenderPayloadSchemaVersion = 2

const (
	missingRenderPayloadVersionError = "rendered report payload is missing schema_version; rerun the producer command to regenerate reports"
	legacyRenderPayloadError         = "legacy rendered report payload uses string render values; rerun the producer command to regenerate reports"
)

type renderSectionJSON struct {
	SchemaVersion int           `json:"schema_version"`
	Blocks        []RenderBlock `json:"blocks,omitempty"`
}

type renderSectionVersionProbe struct {
	SchemaVersion *int `json:"schema_version"`
}

type renderBlockJSON struct {
	Kind    RenderBlockKind `json:"kind"`
	Title   string          `json:"title,omitempty"`
	Text    *RenderValue    `json:"text,omitempty"`
	Items   []RenderValue   `json:"items,omitempty"`
	Table   *RenderTable    `json:"table,omitempty"`
	Details *RenderDetails  `json:"details,omitempty"`
}

type renderValueJSON struct {
	Kind   RenderValueKind `json:"kind"`
	Text   string          `json:"text,omitempty"`
	Status ReportStatus    `json:"status,omitempty"`
	Tone   RenderTone      `json:"tone,omitempty"`
	Amount *float64        `json:"amount,omitempty"`
	Unit   RenderMoneyUnit `json:"unit,omitempty"`
	Parts  []RenderValue   `json:"parts,omitempty"`
}

type renderColumnJSON struct {
	Title RenderValue `json:"title"`
}

type renderRowJSON struct {
	Cells []RenderValue `json:"cells"`
}

type renderTableJSON struct {
	Columns []RenderColumn `json:"columns"`
	Rows    []RenderRow    `json:"rows,omitempty"`
}

type renderDetailsJSON struct {
	Summary  string `json:"summary"`
	Body     string `json:"body,omitempty"`
	Language string `json:"language,omitempty"`
}

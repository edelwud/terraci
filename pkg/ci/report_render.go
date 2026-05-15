package ci

// ReportSectionKindRendered is the canonical render-ready report payload. It is
// intentionally generic: producer plugins own their domain model and publish
// only the view model needed by summary/local renderers.
const ReportSectionKindRendered ReportSectionKind = "rendered"

// RenderSection is a plugin-produced, renderer-facing section body.
type RenderSection struct {
	Blocks []RenderBlock `json:"blocks,omitempty"`
}

// RenderBlockKind identifies one render-ready block type.
type RenderBlockKind string

const (
	RenderBlockKindText    RenderBlockKind = "text"
	RenderBlockKindTable   RenderBlockKind = "table"
	RenderBlockKindList    RenderBlockKind = "list"
	RenderBlockKindDetails RenderBlockKind = "details"
)

// RenderBlock is a generic block used by markdown and CLI renderers.
type RenderBlock struct {
	Kind    RenderBlockKind `json:"kind"`
	Title   string          `json:"title,omitempty"`
	Text    string          `json:"text,omitempty"`
	Items   []string        `json:"items,omitempty"`
	Table   *RenderTable    `json:"table,omitempty"`
	Details *RenderDetails  `json:"details,omitempty"`
}

// RenderTable is an ordered table payload.
type RenderTable struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows,omitempty"`
}

// RenderDetails is a collapsible detail block.
type RenderDetails struct {
	Summary  string `json:"summary"`
	Body     string `json:"body,omitempty"`
	Language string `json:"language,omitempty"`
}

// EncodeRenderSection builds a render-ready ReportSection.
func EncodeRenderSection(title, sectionSummary string, status ReportStatus, blocks ...RenderBlock) (ReportSection, error) {
	return encodeSection(ReportSectionKindRendered, title, sectionSummary, status, RenderSection{Blocks: blocks})
}

// RenderTextBlock returns a paragraph-like render block.
func RenderTextBlock(text string) RenderBlock {
	return RenderBlock{Kind: RenderBlockKindText, Text: text}
}

// RenderListBlock returns an ordered list of already formatted items.
func RenderListBlock(title string, items []string) RenderBlock {
	return RenderBlock{Kind: RenderBlockKindList, Title: title, Items: items}
}

// RenderTableBlock returns an ordered table render block.
func RenderTableBlock(title string, columns []string, rows [][]string) RenderBlock {
	return RenderBlock{
		Kind:  RenderBlockKindTable,
		Title: title,
		Table: &RenderTable{Columns: columns, Rows: rows},
	}
}

// RenderDetailsBlock returns a collapsible detail render block.
func RenderDetailsBlock(summary, body, language string) RenderBlock {
	return RenderBlock{
		Kind: RenderBlockKindDetails,
		Details: &RenderDetails{
			Summary:  summary,
			Body:     body,
			Language: language,
		},
	}
}

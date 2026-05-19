package ci

import (
	"errors"
	"fmt"
)

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

// RenderedReportOptions describes a complete producer report assembled from
// render-ready sections.
type RenderedReportOptions struct {
	Producer string
	Title    string
	Status   ReportStatus
	Summary  string
	Artifact ArtifactContext
	Sections []RenderedSectionOptions
}

// RenderedSectionOptions describes one render-ready report section.
type RenderedSectionOptions struct {
	Title   string
	Summary string
	Status  ReportStatus
	Blocks  []RenderBlock
}

// NewRenderedReport builds and validates a producer report from render-ready
// sections. Section statuses default to the report status when omitted.
func NewRenderedReport(opts RenderedReportOptions) (*Report, error) {
	sections := make([]ReportSection, 0, len(opts.Sections))
	for i, sectionOpts := range opts.Sections {
		if sectionOpts.Status == "" {
			sectionOpts.Status = opts.Status
		}
		section, err := NewRenderedSection(sectionOpts)
		if err != nil {
			return nil, fmt.Errorf("build rendered section %d: %w", i, err)
		}
		sections = append(sections, section)
	}

	report := &Report{
		Producer:   opts.Producer,
		Title:      opts.Title,
		Status:     opts.Status,
		Summary:    opts.Summary,
		Provenance: opts.Artifact.Provenance(),
		Sections:   sections,
	}
	if err := report.Validate(); err != nil {
		return nil, err
	}
	return report, nil
}

// NewRenderedSection builds and validates a render-ready report section.
func NewRenderedSection(opts RenderedSectionOptions) (ReportSection, error) {
	if !opts.Status.Valid() {
		return ReportSection{}, fmt.Errorf("rendered section status %q is invalid", opts.Status)
	}
	rendered := RenderSection{Blocks: opts.Blocks}
	if err := rendered.Validate(); err != nil {
		return ReportSection{}, err
	}
	return encodeSection(ReportSectionKindRendered, opts.Title, opts.Summary, opts.Status, rendered)
}

// DecodeRenderSection decodes and validates a render-ready report section.
func DecodeRenderSection(section ReportSection) (RenderSection, error) {
	if section.kind != ReportSectionKindRendered {
		return RenderSection{}, fmt.Errorf("report section %q is not %q", section.kind, ReportSectionKindRendered)
	}
	rendered, err := decodeSection[RenderSection](section)
	if err != nil {
		return RenderSection{}, err
	}
	if err := rendered.Validate(); err != nil {
		return RenderSection{}, fmt.Errorf("validate rendered section: %w", err)
	}
	return rendered, nil
}

// Validate verifies that all render blocks are well-formed.
func (s RenderSection) Validate() error {
	for i := range s.Blocks {
		if err := s.Blocks[i].Validate(); err != nil {
			return fmt.Errorf("render block %d: %w", i, err)
		}
	}
	return nil
}

// Validate verifies one render block.
func (b RenderBlock) Validate() error {
	switch b.Kind {
	case RenderBlockKindText:
		if b.Text == "" {
			return errors.New("text block requires text")
		}
	case RenderBlockKindList:
		if len(b.Items) == 0 {
			return errors.New("list block requires at least one item")
		}
	case RenderBlockKindTable:
		if b.Table == nil {
			return errors.New("table block requires table payload")
		}
		if err := b.Table.Validate(); err != nil {
			return err
		}
	case RenderBlockKindDetails:
		if b.Details == nil {
			return errors.New("details block requires details payload")
		}
		if err := b.Details.Validate(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported render block kind %q", b.Kind)
	}
	return nil
}

// Validate verifies table shape.
func (t RenderTable) Validate() error {
	if len(t.Columns) == 0 {
		return errors.New("table block requires at least one column")
	}
	for i, row := range t.Rows {
		if len(row) > len(t.Columns) {
			return fmt.Errorf("table row %d has %d cells for %d columns", i, len(row), len(t.Columns))
		}
	}
	return nil
}

// Validate verifies details shape.
func (d RenderDetails) Validate() error {
	if d.Summary == "" {
		return errors.New("details block requires summary")
	}
	return nil
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

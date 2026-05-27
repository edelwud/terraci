package ci

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
)

// ReportSectionKindRendered is the canonical render-ready report payload. It is
// intentionally generic: producer plugins own their domain model and publish
// only typed view values needed by summary/local renderers.
const ReportSectionKindRendered ReportSectionKind = "rendered"

const legacyRenderPayloadError = "legacy rendered report payload uses string render values; rerun the producer command to regenerate reports"

// RenderSection is a plugin-produced, renderer-facing section body.
//
//nolint:recvcheck // MarshalJSON works on values; UnmarshalJSON must mutate the receiver.
type RenderSection struct {
	blocks []RenderBlock
}

type renderSectionJSON struct {
	Blocks []RenderBlock `json:"blocks,omitempty"`
}

// Blocks returns a defensive copy of the section blocks.
func (s RenderSection) Blocks() []RenderBlock {
	return cloneRenderBlocks(s.blocks)
}

// MarshalJSON preserves the rendered section wire shape while keeping the Go
// value object immutable to external packages.
func (s RenderSection) MarshalJSON() ([]byte, error) {
	return json.Marshal(renderSectionJSON{Blocks: s.blocks})
}

// UnmarshalJSON decodes the rendered section wire shape.
func (s *RenderSection) UnmarshalJSON(data []byte) error {
	var raw renderSectionJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	s.blocks = cloneRenderBlocks(raw.Blocks)
	return nil
}

// Validate verifies that all render blocks are well-formed.
func (s RenderSection) Validate() error {
	for i := range s.blocks {
		if err := s.blocks[i].Validate(); err != nil {
			return fmt.Errorf("render block %d: %w", i, err)
		}
	}
	return nil
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
//
//nolint:recvcheck // MarshalJSON works on values; UnmarshalJSON must mutate the receiver.
type RenderBlock struct {
	kind    RenderBlockKind
	title   string
	text    RenderValue
	items   []RenderValue
	table   *RenderTable
	details *RenderDetails
}

type renderBlockJSON struct {
	Kind    RenderBlockKind `json:"kind"`
	Title   string          `json:"title,omitempty"`
	Text    *RenderValue    `json:"text,omitempty"`
	Items   []RenderValue   `json:"items,omitempty"`
	Table   *RenderTable    `json:"table,omitempty"`
	Details *RenderDetails  `json:"details,omitempty"`
}

// Kind returns the block type.
func (b RenderBlock) Kind() RenderBlockKind {
	return b.kind
}

// Title returns the optional block title.
func (b RenderBlock) Title() string {
	return b.title
}

// Text returns the text block content.
func (b RenderBlock) Text() RenderValue {
	return b.text.Clone()
}

// Items returns a defensive copy of list block items.
func (b RenderBlock) Items() []RenderValue {
	return cloneRenderValues(b.items)
}

// Table returns a defensive copy of the table block payload.
func (b RenderBlock) Table() *RenderTable {
	if b.table == nil {
		return nil
	}
	cloned := b.table.Clone()
	return &cloned
}

// Details returns a defensive copy of the details block payload.
func (b RenderBlock) Details() *RenderDetails {
	if b.details == nil {
		return nil
	}
	cloned := b.details.Clone()
	return &cloned
}

// Clone returns a defensive copy of the render block.
func (b RenderBlock) Clone() RenderBlock {
	cloned := b
	cloned.text = b.text.Clone()
	cloned.items = cloneRenderValues(b.items)
	if b.table != nil {
		table := b.table.Clone()
		cloned.table = &table
	}
	if b.details != nil {
		details := b.details.Clone()
		cloned.details = &details
	}
	return cloned
}

// MarshalJSON preserves the render block wire shape.
func (b RenderBlock) MarshalJSON() ([]byte, error) {
	raw := renderBlockJSON{
		Kind:  b.kind,
		Title: b.title,
	}
	switch b.kind {
	case RenderBlockKindText:
		text := b.text.Clone()
		raw.Text = &text
	case RenderBlockKindList:
		raw.Items = cloneRenderValues(b.items)
	case RenderBlockKindTable:
		if b.table != nil {
			table := b.table.Clone()
			raw.Table = &table
		}
	case RenderBlockKindDetails:
		if b.details != nil {
			details := b.details.Clone()
			raw.Details = &details
		}
	}
	return json.Marshal(raw)
}

// UnmarshalJSON decodes the render block wire shape.
func (b *RenderBlock) UnmarshalJSON(data []byte) error {
	var raw renderBlockJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	b.kind = raw.Kind
	b.title = raw.Title
	if raw.Text != nil {
		b.text = raw.Text.Clone()
	} else {
		b.text = RenderValue{}
	}
	b.items = cloneRenderValues(raw.Items)
	if raw.Table != nil {
		table := raw.Table.Clone()
		b.table = &table
	} else {
		b.table = nil
	}
	if raw.Details != nil {
		details := raw.Details.Clone()
		b.details = &details
	} else {
		b.details = nil
	}
	return nil
}

// Validate verifies one render block.
func (b RenderBlock) Validate() error {
	switch b.kind {
	case RenderBlockKindText:
		if err := b.text.Validate(); err != nil {
			return fmt.Errorf("text block requires valid text value: %w", err)
		}
	case RenderBlockKindList:
		if len(b.items) == 0 {
			return errors.New("list block requires at least one item")
		}
		for i := range b.items {
			if err := b.items[i].Validate(); err != nil {
				return fmt.Errorf("list item %d: %w", i, err)
			}
		}
	case RenderBlockKindTable:
		if b.table == nil {
			return errors.New("table block requires table payload")
		}
		if err := b.table.Validate(); err != nil {
			return err
		}
	case RenderBlockKindDetails:
		if b.details == nil {
			return errors.New("details block requires details payload")
		}
		if err := b.details.Validate(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported render block kind %q", b.kind)
	}
	return nil
}

// RenderValueKind identifies one semantic value carried by rendered reports.
type RenderValueKind string

const (
	RenderValueKindText            RenderValueKind = "text"
	RenderValueKindCode            RenderValueKind = "code"
	RenderValueKindStatus          RenderValueKind = "status"
	RenderValueKindLabel           RenderValueKind = "label"
	RenderValueKindMoney           RenderValueKind = "money"
	RenderValueKindMoneyDelta      RenderValueKind = "money_delta"
	RenderValueKindModulePath      RenderValueKind = "module_path"
	RenderValueKindResourceAddress RenderValueKind = "resource_address"
	RenderValueKindInline          RenderValueKind = "inline"
)

// RenderTone describes presentation intent for labels.
type RenderTone string

const (
	RenderToneNeutral RenderTone = "neutral"
	RenderToneInfo    RenderTone = "info"
	RenderToneSuccess RenderTone = "success"
	RenderToneWarning RenderTone = "warning"
	RenderToneFailure RenderTone = "failure"
)

// Valid reports whether the tone is supported by renderers.
func (t RenderTone) Valid() bool {
	switch t {
	case RenderToneNeutral, RenderToneInfo, RenderToneSuccess, RenderToneWarning, RenderToneFailure:
		return true
	default:
		return false
	}
}

// RenderMoneyUnit identifies a display unit suffix for money values.
type RenderMoneyUnit string

const (
	RenderMoneyUnitNone  RenderMoneyUnit = ""
	RenderMoneyUnitMonth RenderMoneyUnit = "mo"
)

// Valid reports whether the money unit is supported by renderers.
func (u RenderMoneyUnit) Valid() bool {
	switch u {
	case RenderMoneyUnitNone, RenderMoneyUnitMonth:
		return true
	default:
		return false
	}
}

// RenderMoneyOptions configures money rendering.
type RenderMoneyOptions struct {
	Unit RenderMoneyUnit
}

// RenderValue is a typed semantic value rendered by the shared renderers.
//
//nolint:recvcheck // MarshalJSON works on values; UnmarshalJSON must mutate the receiver.
type RenderValue struct {
	kind   RenderValueKind
	text   string
	status ReportStatus
	tone   RenderTone
	amount float64
	unit   RenderMoneyUnit
	parts  []RenderValue
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

// RenderText builds a plain text render value.
func RenderText(text string) RenderValue {
	return RenderValue{kind: RenderValueKindText, text: text}
}

// RenderCode builds a monospace code render value.
func RenderCode(text string) RenderValue {
	return RenderValue{kind: RenderValueKindCode, text: text}
}

// RenderStatus builds a report status render value.
func RenderStatus(status ReportStatus) RenderValue {
	return RenderValue{kind: RenderValueKindStatus, status: status}
}

// RenderLabel builds a tone-aware label render value.
func RenderLabel(text string, tone RenderTone) RenderValue {
	return RenderValue{kind: RenderValueKindLabel, text: text, tone: tone}
}

// RenderMoney builds a money render value.
func RenderMoney(amount float64, opts RenderMoneyOptions) RenderValue {
	return RenderValue{kind: RenderValueKindMoney, amount: amount, unit: opts.Unit}
}

// RenderMoneyDelta builds a signed money delta render value.
func RenderMoneyDelta(amount float64, opts RenderMoneyOptions) RenderValue {
	return RenderValue{kind: RenderValueKindMoneyDelta, amount: amount, unit: opts.Unit}
}

// RenderModulePath builds a Terraform module path render value.
func RenderModulePath(path string) RenderValue {
	return RenderValue{kind: RenderValueKindModulePath, text: path}
}

// RenderResourceAddress builds a Terraform resource address render value.
func RenderResourceAddress(address string) RenderValue {
	return RenderValue{kind: RenderValueKindResourceAddress, text: address}
}

// RenderInline builds a composite inline value from typed fragments.
func RenderInline(parts ...RenderValue) RenderValue {
	if len(parts) == 1 {
		return parts[0].Clone()
	}
	return RenderValue{kind: RenderValueKindInline, parts: cloneRenderValues(parts)}
}

// Kind returns the semantic value type.
func (v RenderValue) Kind() RenderValueKind {
	return v.kind
}

// Text returns the value text for text-like values.
func (v RenderValue) Text() string {
	return v.text
}

// Status returns the report status for status values.
func (v RenderValue) Status() ReportStatus {
	return v.status
}

// Tone returns the label tone.
func (v RenderValue) Tone() RenderTone {
	return v.tone
}

// Amount returns the money amount.
func (v RenderValue) Amount() float64 {
	return v.amount
}

// Unit returns the money unit.
func (v RenderValue) Unit() RenderMoneyUnit {
	return v.unit
}

// Parts returns a defensive copy of inline fragments.
func (v RenderValue) Parts() []RenderValue {
	return cloneRenderValues(v.parts)
}

// Clone returns a defensive copy of the render value.
func (v RenderValue) Clone() RenderValue {
	cloned := v
	cloned.parts = cloneRenderValues(v.parts)
	return cloned
}

// MarshalJSON preserves the render value wire shape.
func (v RenderValue) MarshalJSON() ([]byte, error) {
	raw := renderValueJSON{Kind: v.kind}
	switch v.kind {
	case RenderValueKindText, RenderValueKindCode, RenderValueKindModulePath, RenderValueKindResourceAddress:
		raw.Text = v.text
	case RenderValueKindStatus:
		raw.Status = v.status
	case RenderValueKindLabel:
		raw.Text = v.text
		raw.Tone = v.tone
	case RenderValueKindMoney, RenderValueKindMoneyDelta:
		amount := v.amount
		raw.Amount = &amount
		raw.Unit = v.unit
	case RenderValueKindInline:
		raw.Parts = cloneRenderValues(v.parts)
	default:
		raw.Text = v.text
	}
	return json.Marshal(raw)
}

// UnmarshalJSON decodes the render value wire shape.
func (v *RenderValue) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, `"`) {
		return errors.New(legacyRenderPayloadError)
	}
	var raw renderValueJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	v.kind = raw.Kind
	v.text = raw.Text
	v.status = raw.Status
	v.tone = raw.Tone
	if raw.Amount != nil {
		v.amount = *raw.Amount
	} else {
		v.amount = 0
	}
	v.unit = raw.Unit
	v.parts = cloneRenderValues(raw.Parts)
	return nil
}

// Validate verifies one render value.
func (v RenderValue) Validate() error {
	switch v.kind {
	case RenderValueKindText, RenderValueKindCode, RenderValueKindModulePath, RenderValueKindResourceAddress:
		if v.text == "" {
			return fmt.Errorf("%s value requires text", v.kind)
		}
	case RenderValueKindStatus:
		if !v.status.Valid() {
			return fmt.Errorf("status value %q is invalid", v.status)
		}
	case RenderValueKindLabel:
		if v.text == "" {
			return errors.New("label value requires text")
		}
		if !v.tone.Valid() {
			return fmt.Errorf("label tone %q is invalid", v.tone)
		}
	case RenderValueKindMoney, RenderValueKindMoneyDelta:
		if math.IsNaN(v.amount) || math.IsInf(v.amount, 0) {
			return fmt.Errorf("%s value requires finite amount", v.kind)
		}
		if !v.unit.Valid() {
			return fmt.Errorf("money unit %q is invalid", v.unit)
		}
	case RenderValueKindInline:
		if len(v.parts) == 0 {
			return errors.New("inline value requires at least one part")
		}
		for i := range v.parts {
			if err := v.parts[i].Validate(); err != nil {
				return fmt.Errorf("inline part %d: %w", i, err)
			}
		}
	default:
		return fmt.Errorf("unsupported render value kind %q", v.kind)
	}
	return nil
}

// RenderColumn is one typed table column.
//
//nolint:recvcheck // MarshalJSON works on values; UnmarshalJSON must mutate the receiver.
type RenderColumn struct {
	title RenderValue
}

type renderColumnJSON struct {
	Title RenderValue `json:"title"`
}

// NewRenderColumn builds a text table column.
func NewRenderColumn(title string) RenderColumn {
	return RenderColumn{title: RenderText(title)}
}

// NewRenderValueColumn builds a table column from a typed value.
func NewRenderValueColumn(title RenderValue) RenderColumn {
	return RenderColumn{title: title.Clone()}
}

// Title returns the typed column title.
func (c RenderColumn) Title() RenderValue {
	return c.title.Clone()
}

// Clone returns a defensive copy of the column.
func (c RenderColumn) Clone() RenderColumn {
	return RenderColumn{title: c.title.Clone()}
}

// MarshalJSON preserves the table column wire shape.
func (c RenderColumn) MarshalJSON() ([]byte, error) {
	return json.Marshal(renderColumnJSON{Title: c.title})
}

// UnmarshalJSON decodes the table column wire shape.
func (c *RenderColumn) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, `"`) {
		return errors.New(legacyRenderPayloadError)
	}
	var raw renderColumnJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	c.title = raw.Title.Clone()
	return nil
}

// RenderRow is one typed table row.
//
//nolint:recvcheck // MarshalJSON works on values; UnmarshalJSON must mutate the receiver.
type RenderRow struct {
	cells []RenderValue
}

type renderRowJSON struct {
	Cells []RenderValue `json:"cells"`
}

// NewRenderRow builds a table row from typed values.
func NewRenderRow(cells ...RenderValue) RenderRow {
	return RenderRow{cells: cloneRenderValues(cells)}
}

// Cells returns a defensive copy of the row cells.
func (r RenderRow) Cells() []RenderValue {
	return cloneRenderValues(r.cells)
}

// Clone returns a defensive copy of the row.
func (r RenderRow) Clone() RenderRow {
	return RenderRow{cells: cloneRenderValues(r.cells)}
}

// MarshalJSON preserves the table row wire shape.
func (r RenderRow) MarshalJSON() ([]byte, error) {
	return json.Marshal(renderRowJSON{Cells: r.cells})
}

// UnmarshalJSON decodes the table row wire shape.
func (r *RenderRow) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "[") {
		return errors.New(legacyRenderPayloadError)
	}
	var raw renderRowJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.cells = cloneRenderValues(raw.Cells)
	return nil
}

// RenderTable is an ordered table payload.
//
//nolint:recvcheck // MarshalJSON works on values; UnmarshalJSON must mutate the receiver.
type RenderTable struct {
	columns []RenderColumn
	rows    []RenderRow
}

type renderTableJSON struct {
	Columns []RenderColumn `json:"columns"`
	Rows    []RenderRow    `json:"rows,omitempty"`
}

// Columns returns a defensive copy of table columns.
func (t RenderTable) Columns() []RenderColumn {
	return cloneRenderColumns(t.columns)
}

// Rows returns a defensive copy of table rows.
func (t RenderTable) Rows() []RenderRow {
	return cloneRenderRows(t.rows)
}

// Clone returns a defensive copy of the table.
func (t RenderTable) Clone() RenderTable {
	return RenderTable{
		columns: cloneRenderColumns(t.columns),
		rows:    cloneRenderRows(t.rows),
	}
}

// MarshalJSON preserves the table wire shape.
func (t RenderTable) MarshalJSON() ([]byte, error) {
	return json.Marshal(renderTableJSON{Columns: t.columns, Rows: t.rows})
}

// UnmarshalJSON decodes the table wire shape.
func (t *RenderTable) UnmarshalJSON(data []byte) error {
	var raw renderTableJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	t.columns = cloneRenderColumns(raw.Columns)
	t.rows = cloneRenderRows(raw.Rows)
	return nil
}

// Validate verifies table shape.
func (t RenderTable) Validate() error {
	if len(t.columns) == 0 {
		return errors.New("table block requires at least one column")
	}
	for i := range t.columns {
		if err := t.columns[i].title.Validate(); err != nil {
			return fmt.Errorf("table column %d: %w", i, err)
		}
	}
	for i, row := range t.rows {
		if len(row.cells) > len(t.columns) {
			return fmt.Errorf("table row %d has %d cells for %d columns", i, len(row.cells), len(t.columns))
		}
		for j := range row.cells {
			if err := row.cells[j].Validate(); err != nil {
				return fmt.Errorf("table row %d cell %d: %w", i, j, err)
			}
		}
	}
	return nil
}

// RenderDetails is a collapsible detail block.
//
//nolint:recvcheck // MarshalJSON works on values; UnmarshalJSON must mutate the receiver.
type RenderDetails struct {
	summary  string
	body     string
	language string
}

type renderDetailsJSON struct {
	Summary  string `json:"summary"`
	Body     string `json:"body,omitempty"`
	Language string `json:"language,omitempty"`
}

// Summary returns the details summary.
func (d RenderDetails) Summary() string {
	return d.summary
}

// Body returns the details body.
func (d RenderDetails) Body() string {
	return d.body
}

// Language returns the optional details language.
func (d RenderDetails) Language() string {
	return d.language
}

// Clone returns a defensive copy of the details value.
func (d RenderDetails) Clone() RenderDetails {
	return d
}

// MarshalJSON preserves the details wire shape.
func (d RenderDetails) MarshalJSON() ([]byte, error) {
	return json.Marshal(renderDetailsJSON{Summary: d.summary, Body: d.body, Language: d.language})
}

// UnmarshalJSON decodes the details wire shape.
func (d *RenderDetails) UnmarshalJSON(data []byte) error {
	var raw renderDetailsJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	d.summary = raw.Summary
	d.body = raw.Body
	d.language = raw.Language
	return nil
}

// Validate verifies details shape.
func (d RenderDetails) Validate() error {
	if d.summary == "" {
		return errors.New("details block requires summary")
	}
	return nil
}

// NewTextBlock returns a paragraph-like render block.
func NewTextBlock(values ...RenderValue) RenderBlock {
	return RenderBlock{kind: RenderBlockKindText, text: RenderInline(values...)}
}

// NewListBlock returns an ordered list of typed items.
func NewListBlock(title string, items []RenderValue) RenderBlock {
	return RenderBlock{kind: RenderBlockKindList, title: title, items: cloneRenderValues(items)}
}

// NewTableBlock returns an ordered typed table render block.
func NewTableBlock(title string, columns []RenderColumn, rows []RenderRow) RenderBlock {
	table := RenderTable{
		columns: cloneRenderColumns(columns),
		rows:    cloneRenderRows(rows),
	}
	return RenderBlock{kind: RenderBlockKindTable, title: title, table: &table}
}

// NewDetailsBlock returns a collapsible detail block.
func NewDetailsBlock(summary, body, language string) RenderBlock {
	return RenderBlock{
		kind: RenderBlockKindDetails,
		details: &RenderDetails{
			summary:  summary,
			body:     body,
			language: language,
		},
	}
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
	rendered := RenderSection{blocks: cloneRenderBlocks(opts.Blocks)}
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
		return RenderSection{}, fmt.Errorf("decode rendered section: %w", err)
	}
	if err := rendered.Validate(); err != nil {
		return RenderSection{}, fmt.Errorf("validate rendered section: %w", err)
	}
	return rendered, nil
}

func cloneRenderBlocks(blocks []RenderBlock) []RenderBlock {
	if len(blocks) == 0 {
		return nil
	}
	cloned := make([]RenderBlock, len(blocks))
	for i := range blocks {
		cloned[i] = blocks[i].Clone()
	}
	return cloned
}

func cloneRenderValues(values []RenderValue) []RenderValue {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]RenderValue, len(values))
	for i := range values {
		cloned[i] = values[i].Clone()
	}
	return cloned
}

func cloneRenderColumns(columns []RenderColumn) []RenderColumn {
	if len(columns) == 0 {
		return nil
	}
	cloned := make([]RenderColumn, len(columns))
	for i := range columns {
		cloned[i] = columns[i].Clone()
	}
	return cloned
}

func cloneRenderRows(rows []RenderRow) []RenderRow {
	if len(rows) == 0 {
		return nil
	}
	cloned := make([]RenderRow, len(rows))
	for i := range rows {
		cloned[i] = rows[i].Clone()
	}
	return cloned
}

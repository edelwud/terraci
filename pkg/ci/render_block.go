package ci

import (
	"encoding/json"
	"errors"
	"fmt"
)

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

// RenderDetails is a collapsible detail block.
//
//nolint:recvcheck // MarshalJSON works on values; UnmarshalJSON must mutate the receiver.
type RenderDetails struct {
	summary  string
	body     string
	language string
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

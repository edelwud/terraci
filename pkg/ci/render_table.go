package ci

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// RenderColumn is one typed table column.
//
//nolint:recvcheck // MarshalJSON works on values; UnmarshalJSON must mutate the receiver.
type RenderColumn struct {
	title RenderValue
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

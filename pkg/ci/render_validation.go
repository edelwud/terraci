package ci

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

package ci

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewRenderedSection_RejectsMalformedBlocks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		block   RenderBlock
		wantErr string
	}{
		{
			name:    "empty text",
			block:   RenderBlock{Kind: RenderBlockKindText},
			wantErr: "text block requires text",
		},
		{
			name:    "empty list",
			block:   RenderBlock{Kind: RenderBlockKindList},
			wantErr: "list block requires at least one item",
		},
		{
			name:    "missing table",
			block:   RenderBlock{Kind: RenderBlockKindTable},
			wantErr: "table block requires table payload",
		},
		{
			name:    "wide table row",
			block:   RenderTableBlock("", []string{"Module"}, [][]string{{"app", "extra"}}),
			wantErr: "table row 0 has 2 cells for 1 columns",
		},
		{
			name:    "missing details",
			block:   RenderBlock{Kind: RenderBlockKindDetails},
			wantErr: "details block requires details payload",
		},
		{
			name:    "unsupported block kind",
			block:   RenderBlock{Kind: "custom"},
			wantErr: `unsupported render block kind "custom"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewRenderedSection(RenderedSectionOptions{
				Title:  "Broken",
				Status: ReportStatusWarn,
				Blocks: []RenderBlock{tt.block},
			})
			if err == nil {
				t.Fatal("NewRenderedSection() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("NewRenderedSection() error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestNewRenderedSection_RejectsInvalidStatus(t *testing.T) {
	t.Parallel()

	_, err := NewRenderedSection(RenderedSectionOptions{
		Title:  "Broken",
		Blocks: []RenderBlock{RenderTextBlock("body")},
	})
	if err == nil {
		t.Fatal("NewRenderedSection() error = nil, want invalid status error")
	}
	if !strings.Contains(err.Error(), `status "" is invalid`) {
		t.Fatalf("NewRenderedSection() error = %q, want invalid status message", err.Error())
	}
}

func TestNewRenderedReport_DefaultsSectionStatus(t *testing.T) {
	t.Parallel()

	report, err := NewRenderedReport(RenderedReportOptions{
		Producer: "producer",
		Title:    "Producer Report",
		Status:   ReportStatusWarn,
		Summary:  "summary",
		Sections: []RenderedSectionOptions{{
			Title:  "Findings",
			Blocks: []RenderBlock{RenderTextBlock("warning")},
		}},
	})
	if err != nil {
		t.Fatalf("NewRenderedReport() error = %v", err)
	}
	if report.Provenance == nil {
		t.Fatal("Provenance = nil, want default provenance")
	}
	if got := report.Sections[0].Status; got != ReportStatusWarn {
		t.Fatalf("section status = %q, want %q", got, ReportStatusWarn)
	}
	if _, err := DecodeRenderSection(report.Sections[0]); err != nil {
		t.Fatalf("DecodeRenderSection(defaulted section): %v", err)
	}
}

func TestDecodeRenderSection_RejectsWrongKind(t *testing.T) {
	t.Parallel()

	section := ReportSection{Kind: "findings", Payload: []byte(`{}`)}
	_, err := DecodeRenderSection(section)
	if err == nil {
		t.Fatal("DecodeRenderSection() error = nil, want wrong-kind error")
	}
	if !strings.Contains(err.Error(), `is not "rendered"`) {
		t.Fatalf("DecodeRenderSection() error = %q, want wrong-kind message", err.Error())
	}
}

func TestLoadReport_RejectsInvalidRenderedPayload(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, ReportFilename("broken"))
	content := `{
  "producer": "broken",
  "title": "Broken",
  "status": "warn",
  "sections": [
    {
      "kind": "rendered",
      "title": "Broken Section",
      "status": "warn",
      "payload": {
        "blocks": [
          {
            "kind": "table",
            "table": {
              "columns": ["Module"],
              "rows": [["app", "extra"]]
            }
          }
        ]
      }
    }
  ]
}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := LoadReport(path)
	if err == nil {
		t.Fatal("LoadReport() error = nil, want invalid rendered payload error")
	}
	if !strings.Contains(err.Error(), "table row 0 has 2 cells for 1 columns") {
		t.Fatalf("LoadReport() error = %q, want invalid table shape", err.Error())
	}
}

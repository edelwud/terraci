package ci

import (
	"encoding/json"
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
	if got := report.Sections[0].Status(); got != ReportStatusWarn {
		t.Fatalf("section status = %q, want %q", got, ReportStatusWarn)
	}
	if _, err := DecodeRenderSection(report.Sections[0]); err != nil {
		t.Fatalf("DecodeRenderSection(defaulted section): %v", err)
	}
}

func TestDecodeRenderSection_RejectsWrongKind(t *testing.T) {
	t.Parallel()

	var section ReportSection
	if err := json.Unmarshal([]byte(`{"kind":"findings","payload":{}}`), &section); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	_, err := DecodeRenderSection(section)
	if err == nil {
		t.Fatal("DecodeRenderSection() error = nil, want wrong-kind error")
	}
	if !strings.Contains(err.Error(), `is not "rendered"`) {
		t.Fatalf("DecodeRenderSection() error = %q, want wrong-kind message", err.Error())
	}
}

func TestReportSection_JSONRoundTripAndGetters(t *testing.T) {
	t.Parallel()

	section, err := NewRenderedSection(RenderedSectionOptions{
		Title:   "Findings",
		Summary: "1 finding",
		Status:  ReportStatusWarn,
		Blocks:  []RenderBlock{RenderTextBlock("body")},
	})
	if err != nil {
		t.Fatalf("NewRenderedSection() error = %v", err)
	}

	data, err := json.Marshal(section)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	for _, want := range []string{`"kind":"rendered"`, `"title":"Findings"`, `"status":"warn"`, `"section_summary":"1 finding"`, `"payload":`} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("marshaled section missing %s: %s", want, data)
		}
	}

	var decoded ReportSection
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if decoded.Kind() != ReportSectionKindRendered || decoded.Title() != "Findings" || decoded.Status() != ReportStatusWarn || decoded.Summary() != "1 finding" {
		t.Fatalf("decoded getters = kind:%q title:%q status:%q summary:%q", decoded.Kind(), decoded.Title(), decoded.Status(), decoded.Summary())
	}
	if _, err := DecodeRenderSection(decoded); err != nil {
		t.Fatalf("DecodeRenderSection() error = %v", err)
	}
}

func TestReportClone_DefensivelyCopiesSectionsAndProvenance(t *testing.T) {
	t.Parallel()

	report, err := NewRenderedReport(RenderedReportOptions{
		Producer:   "producer",
		Title:      "Producer Report",
		Status:     ReportStatusWarn,
		Provenance: NewProvenance("commit", "pipeline", "fingerprint"),
		Sections: []RenderedSectionOptions{{
			Title:  "Findings",
			Blocks: []RenderBlock{RenderTextBlock("original")},
		}},
	})
	if err != nil {
		t.Fatalf("NewRenderedReport() error = %v", err)
	}

	clone := report.Clone()
	report.Provenance.PlanResultsFingerprint = "mutated"
	report.Sections[0], err = NewRenderedSection(RenderedSectionOptions{
		Title:  "Findings",
		Status: ReportStatusWarn,
		Blocks: []RenderBlock{RenderTextBlock("mutated")},
	})
	if err != nil {
		t.Fatalf("NewRenderedSection() error = %v", err)
	}

	if clone.Provenance.PlanResultsFingerprint != "fingerprint" {
		t.Fatalf("clone provenance fingerprint = %q, want fingerprint", clone.Provenance.PlanResultsFingerprint)
	}
	rendered, err := DecodeRenderSection(clone.Sections[0])
	if err != nil {
		t.Fatalf("DecodeRenderSection(clone) error = %v", err)
	}
	if rendered.Blocks[0].Text != "original" {
		t.Fatalf("clone section text = %q, want original", rendered.Blocks[0].Text)
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

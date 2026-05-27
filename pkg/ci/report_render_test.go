package ci

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestArtifactContext_ProvenanceDefaultsGeneratedAt(t *testing.T) {
	t.Parallel()

	artifact := NewArtifactContext(ArtifactContextOptions{
		ServiceDir:             "/service",
		WorkDir:                "/work",
		CommitSHA:              "commit",
		PipelineID:             "pipeline",
		PlanResultsFingerprint: "fingerprint",
	})
	provenance := artifact.Provenance()

	if provenance.GeneratedAt.IsZero() {
		t.Fatal("GeneratedAt is zero, want default timestamp")
	}
	if provenance.GeneratedAt.Location() != time.UTC {
		t.Fatalf("GeneratedAt location = %v, want UTC", provenance.GeneratedAt.Location())
	}
	if provenance.CommitSHA != "commit" || provenance.PipelineID != "pipeline" || provenance.PlanResultsFingerprint != "fingerprint" {
		t.Fatalf("Provenance = %#v, want artifact metadata", provenance)
	}
}

func TestNewRenderedSection_RejectsMalformedBlocks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		block   RenderBlock
		wantErr string
	}{
		{
			name:    "empty text",
			block:   RenderBlock{kind: RenderBlockKindText},
			wantErr: "text block requires valid text value",
		},
		{
			name:    "empty list",
			block:   RenderBlock{kind: RenderBlockKindList},
			wantErr: "list block requires at least one item",
		},
		{
			name:    "missing table",
			block:   RenderBlock{kind: RenderBlockKindTable},
			wantErr: "table block requires table payload",
		},
		{
			name:    "wide table row",
			block:   NewTableBlock("", []RenderColumn{NewRenderColumn("Module")}, []RenderRow{NewRenderRow(RenderText("app"), RenderText("extra"))}),
			wantErr: "table row 0 has 2 cells for 1 columns",
		},
		{
			name:    "missing details",
			block:   RenderBlock{kind: RenderBlockKindDetails},
			wantErr: "details block requires details payload",
		},
		{
			name:    "unsupported block kind",
			block:   RenderBlock{kind: "custom"},
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
		Blocks: []RenderBlock{NewTextBlock(RenderText("body"))},
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
			Blocks: []RenderBlock{NewTextBlock(RenderText("warning"))},
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
		Blocks:  []RenderBlock{NewTextBlock(RenderText("body"))},
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

func TestRenderValue_JSONRoundTripAndValidation(t *testing.T) {
	t.Parallel()

	value := RenderInline(
		RenderStatus(ReportStatusWarn),
		RenderText(" total "),
		RenderMoneyDelta(12.5, RenderMoneyOptions{Unit: RenderMoneyUnitMonth}),
	)
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded RenderValue
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if err := decoded.Validate(); err != nil {
		t.Fatalf("decoded Validate() error = %v", err)
	}
	parts := decoded.Parts()
	if len(parts) != 3 || parts[0].Status() != ReportStatusWarn || parts[2].Amount() != 12.5 || parts[2].Unit() != RenderMoneyUnitMonth {
		t.Fatalf("decoded parts = %#v, want status/text/monthly delta", parts)
	}
}

func TestRenderValue_RejectsInvalidMoney(t *testing.T) {
	t.Parallel()

	value := RenderMoney(math.NaN(), RenderMoneyOptions{Unit: RenderMoneyUnitMonth})
	if err := value.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want finite amount error")
	}

	invalidUnit := RenderMoney(1, RenderMoneyOptions{Unit: RenderMoneyUnit("year")})
	if err := invalidUnit.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want invalid unit error")
	}
}

func TestRenderValue_RejectsLegacyStringJSON(t *testing.T) {
	t.Parallel()

	var value RenderValue
	err := json.Unmarshal([]byte(`"legacy"`), &value)
	if err == nil {
		t.Fatal("json.Unmarshal() error = nil, want legacy payload error")
	}
	if !strings.Contains(err.Error(), legacyRenderPayloadError) {
		t.Fatalf("json.Unmarshal() error = %q, want legacy payload message", err.Error())
	}
}

func TestReportClone_DefensivelyCopiesSectionsAndProvenance(t *testing.T) {
	t.Parallel()

	report, err := NewRenderedReport(RenderedReportOptions{
		Producer: "producer",
		Title:    "Producer Report",
		Status:   ReportStatusWarn,
		Artifact: NewArtifactContext(ArtifactContextOptions{
			CommitSHA:              "commit",
			PipelineID:             "pipeline",
			PlanResultsFingerprint: "fingerprint",
		}),
		Sections: []RenderedSectionOptions{{
			Title:  "Findings",
			Blocks: []RenderBlock{NewTextBlock(RenderText("original"))},
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
		Blocks: []RenderBlock{NewTextBlock(RenderText("mutated"))},
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
	blocks := rendered.Blocks()
	if got := blocks[0].Text().Text(); got != "original" {
		t.Fatalf("clone section text = %q, want original", got)
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
	if !strings.Contains(err.Error(), legacyRenderPayloadError) {
		t.Fatalf("LoadReport() error = %q, want legacy payload message", err.Error())
	}
}

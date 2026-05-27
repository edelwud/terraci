package ci

import (
	"encoding/json"
	"errors"
	"fmt"
)

// RenderSection is a plugin-produced, renderer-facing section body.
//
//nolint:recvcheck // MarshalJSON works on values; UnmarshalJSON must mutate the receiver.
type RenderSection struct {
	schemaVersion int
	blocks        []RenderBlock
}

// SchemaVersion returns the rendered payload schema version.
func (s RenderSection) SchemaVersion() int {
	if s.schemaVersion == 0 {
		return RenderPayloadSchemaVersion
	}
	return s.schemaVersion
}

// Blocks returns a defensive copy of the section blocks.
func (s RenderSection) Blocks() []RenderBlock {
	return cloneRenderBlocks(s.blocks)
}

// MarshalJSON preserves the rendered section wire shape while keeping the Go
// value object immutable to external packages.
func (s RenderSection) MarshalJSON() ([]byte, error) {
	return json.Marshal(renderSectionJSON{
		SchemaVersion: s.SchemaVersion(),
		Blocks:        s.blocks,
	})
}

// UnmarshalJSON decodes the rendered section wire shape.
func (s *RenderSection) UnmarshalJSON(data []byte) error {
	var probe renderSectionVersionProbe
	if err := json.Unmarshal(data, &probe); err != nil {
		return err
	}
	if probe.SchemaVersion == nil {
		return errors.New(missingRenderPayloadVersionError)
	}
	if *probe.SchemaVersion != RenderPayloadSchemaVersion {
		return fmt.Errorf("unsupported rendered report payload schema_version %d; rerun the producer command to regenerate reports", *probe.SchemaVersion)
	}

	var raw renderSectionJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	s.schemaVersion = raw.SchemaVersion
	s.blocks = cloneRenderBlocks(raw.Blocks)
	return nil
}

// Validate verifies that all render blocks are well-formed.
func (s RenderSection) Validate() error {
	if version := s.SchemaVersion(); version != RenderPayloadSchemaVersion {
		return fmt.Errorf("unsupported rendered report payload schema_version %d; rerun the producer command to regenerate reports", version)
	}
	for i := range s.blocks {
		if err := s.blocks[i].Validate(); err != nil {
			return fmt.Errorf("render block %d: %w", i, err)
		}
	}
	return nil
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
	rendered := RenderSection{
		schemaVersion: RenderPayloadSchemaVersion,
		blocks:        cloneRenderBlocks(opts.Blocks),
	}
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

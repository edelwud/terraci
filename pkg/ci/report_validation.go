package ci

import (
	"errors"
	"fmt"
	"strings"
)

// Validate verifies the persisted report artifact contract.
func (r *Report) Validate() error {
	if r == nil {
		return errors.New("ci report is nil")
	}
	if strings.TrimSpace(r.Producer) == "" {
		return errors.New("ci report producer is required")
	}
	if strings.ContainsAny(r.Producer, `/\`) {
		return fmt.Errorf("ci report producer %q is not a safe artifact name", r.Producer)
	}
	if strings.TrimSpace(r.Title) == "" {
		return errors.New("ci report title is required")
	}
	if !r.Status.Valid() {
		return fmt.Errorf("ci report status %q is invalid", r.Status)
	}
	for i := range r.Sections {
		if err := validateReportSection(r.Sections[i]); err != nil {
			return fmt.Errorf("ci report section %d: %w", i, err)
		}
	}
	return nil
}

func validateReportSection(s ReportSection) error {
	if strings.TrimSpace(string(s.Kind)) == "" {
		return errors.New("report section kind is required")
	}
	if len(s.Payload) == 0 {
		return fmt.Errorf("report section %q has empty payload", s.Kind)
	}
	return nil
}

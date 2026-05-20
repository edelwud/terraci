package skeleton

import (
	"fmt"
	"io"

	"github.com/edelwud/terraci/pkg/ci"
)

// WriteOutput renders command output. Keep this writer-based for tests.
func WriteOutput(w io.Writer, result *Result) error {
	if w == nil || result == nil {
		return nil
	}
	switch {
	case result.Producer != nil:
		fmt.Fprintln(w, result.Producer.Greeting)
		fmt.Fprintf(w, "wrote %s/%s\n", result.Producer.ServiceDir, ci.ReportFilename(pluginName))
	case result.Consumer != nil:
		if len(result.Consumer.Reports) == 0 {
			fmt.Fprintln(w, "no reports found in service directory")
			return nil
		}
		for _, report := range result.Consumer.Reports {
			fmt.Fprintf(w, "- %s [%s] %s\n", report.Producer, report.Status, report.Summary)
			for _, section := range report.Sections {
				if section.Error != "" {
					fmt.Fprintf(w, "    %s: decode error: %s\n", section.Title, section.Error)
					continue
				}
				fmt.Fprintf(w, "    %s: %d block(s)\n", section.Title, section.Blocks)
			}
		}
	}
	return nil
}

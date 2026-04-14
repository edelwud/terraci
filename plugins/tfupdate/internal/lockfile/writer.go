package lockfile

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type hclWriter struct{}

// NewWriter constructs the default lock-file writer.
func NewWriter() Writer {
	return hclWriter{}
}

func (hclWriter) WriteDocument(filePath string, doc *LockDocument) error {
	if doc == nil {
		doc = &LockDocument{}
	}

	providers := append([]LockProviderEntry(nil), doc.Providers...)
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].Source < providers[j].Source
	})

	var out strings.Builder
	for _, provider := range providers {
		out.WriteString(`provider `)
		out.WriteString(strconv.Quote(provider.Source))
		out.WriteString(" {\n")
		out.WriteString("  version     = ")
		out.WriteString(strconv.Quote(provider.Version))
		out.WriteString("\n")
		if provider.Constraints != "" {
			out.WriteString("  constraints = ")
			out.WriteString(strconv.Quote(provider.Constraints))
			out.WriteString("\n")
		}
		hashes := normalizeHashes(provider.Hashes)
		out.WriteString("  hashes      = [\n")
		for _, hash := range hashes {
			out.WriteString("    ")
			out.WriteString(strconv.Quote(hash))
			out.WriteString(",\n")
		}
		out.WriteString("  ]\n")
		out.WriteString("}\n\n")
	}

	if err := os.WriteFile(filePath, []byte(out.String()), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}
	return nil
}

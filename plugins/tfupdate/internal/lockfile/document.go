package lockfile

import (
	"fmt"
	"os"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

// ParseDocument reads a lock file into a typed document. Missing files return an empty document.
func ParseDocument(filePath string) (*LockDocument, error) {
	src, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &LockDocument{}, nil
		}
		return nil, fmt.Errorf("read %s: %w", filePath, err)
	}
	return ParseDocumentBytes(filePath, src)
}

// ParseDocumentBytes parses lock-file content from bytes.
func ParseDocumentBytes(filePath string, src []byte) (*LockDocument, error) {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL(src, filePath)
	if diags.HasErrors() {
		return nil, fmt.Errorf("parse %s: %s", filePath, diags.Error())
	}

	schema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{{Type: "provider", LabelNames: []string{"source"}}},
	}
	content, _, diags := file.Body.PartialContent(schema)
	if diags.HasErrors() {
		return nil, fmt.Errorf("decode %s: %s", filePath, diags.Error())
	}

	doc := &LockDocument{Providers: make([]LockProviderEntry, 0, len(content.Blocks))}
	for _, block := range content.Blocks {
		entry, err := parseProviderBlock(block)
		if err != nil {
			return nil, err
		}
		doc.Providers = append(doc.Providers, entry)
	}

	sort.Slice(doc.Providers, func(i, j int) bool {
		return doc.Providers[i].Source < doc.Providers[j].Source
	})
	return doc, nil
}

func parseProviderBlock(block *hcl.Block) (LockProviderEntry, error) {
	entry := LockProviderEntry{Source: block.Labels[0]}
	attrs, diags := block.Body.JustAttributes()
	if diags.HasErrors() {
		return LockProviderEntry{}, fmt.Errorf("decode provider block %q: %s", entry.Source, diags.Error())
	}

	if attr := attrs["version"]; attr != nil {
		value, diags := attr.Expr.Value(nil)
		if diags.HasErrors() {
			return LockProviderEntry{}, fmt.Errorf("decode version for %q: %s", entry.Source, diags.Error())
		}
		entry.Version = value.AsString()
	}
	if attr := attrs["constraints"]; attr != nil {
		value, diags := attr.Expr.Value(nil)
		if diags.HasErrors() {
			return LockProviderEntry{}, fmt.Errorf("decode constraints for %q: %s", entry.Source, diags.Error())
		}
		entry.Constraints = value.AsString()
	}
	if attr := attrs["hashes"]; attr != nil {
		value, diags := attr.Expr.Value(nil)
		if diags.HasErrors() {
			return LockProviderEntry{}, fmt.Errorf("decode hashes for %q: %s", entry.Source, diags.Error())
		}
		entry.Hashes = decodeStringCollection(value)
	}

	return entry, nil
}

func decodeStringCollection(value cty.Value) LockHashSet {
	if !value.IsKnown() || value.IsNull() {
		return nil
	}
	result := make([]string, 0, value.LengthInt())
	it := value.ElementIterator()
	for it.Next() {
		_, item := it.Element()
		if item.Type() == cty.String {
			result = append(result, item.AsString())
		}
	}
	return LockHashSet(normalizeHashes(result))
}

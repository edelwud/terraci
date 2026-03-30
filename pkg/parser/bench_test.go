package parser

import (
	"context"
	"testing"

	"github.com/edelwud/terraci/pkg/parser/internal/testutil"
)

var benchParsedModule *ParsedModule

func BenchmarkParseModule(b *testing.B) {
	for _, tc := range []struct {
		name      string
		fileCount int
	}{
		{name: "files=5", fileCount: 5},
		{name: "files=20", fileCount: 20},
		{name: "files=50", fileCount: 50},
	} {
		dir := b.TempDir()
		testutil.BuildParserBenchmarkModule(b, dir, tc.fileCount)
		parser := NewParser(nil)

		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				parsed, err := parser.ParseModule(context.Background(), dir)
				if err != nil {
					b.Fatalf("ParseModule() error = %v", err)
				}
				benchParsedModule = parsed
			}
		})
	}
}

package moduleparse

import (
	"context"
	"testing"

	"github.com/edelwud/terraci/pkg/parser/internal/testutil"
	"github.com/edelwud/terraci/pkg/parser/model"
)

var benchModuleParseResult *model.ParsedModule

func BenchmarkRun(b *testing.B) {
	for _, tc := range []struct {
		name      string
		fileCount int
	}{
		{name: "files=5", fileCount: 5},
		{name: "files=20", fileCount: 20},
		{name: "files=50", fileCount: 50},
	} {
		dir := b.TempDir()
		testutil.BuildModuleParseBenchmarkModule(b, dir, tc.fileCount)

		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				parsed, err := Run(context.Background(), dir, []string{"service", "environment", "region", "module"})
				if err != nil {
					b.Fatalf("Run() error = %v", err)
				}
				benchModuleParseResult = parsed
			}
		})
	}
}

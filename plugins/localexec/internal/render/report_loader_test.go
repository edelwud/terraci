package render

import "testing"

func TestSummaryReportLoaderNoops(t *testing.T) {
	t.Parallel()

	loader := NewSummaryReportLoader(t.TempDir(), t.TempDir(), nil)
	if err := loader.Reset(); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	report, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if report != nil {
		t.Fatalf("Load() report = %#v, want nil", report)
	}
}

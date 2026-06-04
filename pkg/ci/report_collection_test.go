package ci

import "testing"

func TestReportCollection_SortsDedupesAndFilters(t *testing.T) {
	older := testStoreReport("cost", "Old", ReportStatusPass)
	newer := testStoreReport("cost", "New", ReportStatusWarn)
	policy := testStoreReport("policy", "Policy", ReportStatusFail)

	collection := NewReportCollection(policy, older, nil, newer)
	if collection.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", collection.Len())
	}
	if got := collection.Producers(); len(got) != 2 || got[0] != "cost" || got[1] != "policy" {
		t.Fatalf("Producers() = %v, want [cost policy]", got)
	}
	report, ok := collection.Find("cost")
	if !ok {
		t.Fatal("Find(cost) ok = false, want true")
	}
	if report.Title() != "New" || report.Status() != ReportStatusWarn {
		t.Fatalf("Find(cost) = %#v, want last duplicate", report)
	}

	filtered := collection.WithoutProducers("policy")
	if filtered.Len() != 1 {
		t.Fatalf("filtered Len() = %d, want 1", filtered.Len())
	}
	if _, ok := filtered.Find("policy"); ok {
		t.Fatal("filtered Find(policy) ok = true, want false")
	}
}

func TestReportCollection_ReturnsDefensiveCopies(t *testing.T) {
	collection := NewReportCollection(testStoreReport("cost", "Cost", ReportStatusPass))
	reports := collection.Reports()
	reports[0].title = "mutated"

	again, ok := collection.Find("cost")
	if !ok {
		t.Fatal("Find(cost) ok = false, want true")
	}
	if again.Title() != "Cost" {
		t.Fatalf("stored report title = %q, want Cost", again.Title())
	}
}

package ci

import "sort"

// ReportCollection is a deterministic read-only set of reports keyed by producer.
type ReportCollection struct {
	reports []*Report
}

// NewReportCollection returns reports sorted by producer. When duplicate
// producers are present, the last report wins.
func NewReportCollection(reports ...*Report) ReportCollection {
	byProducer := make(map[string]*Report)
	for _, report := range reports {
		if report == nil {
			continue
		}
		byProducer[report.Producer()] = report.Clone()
	}
	producers := make([]string, 0, len(byProducer))
	for producer := range byProducer {
		producers = append(producers, producer)
	}
	sort.Strings(producers)

	out := ReportCollection{reports: make([]*Report, 0, len(producers))}
	for _, producer := range producers {
		out.reports = append(out.reports, byProducer[producer].Clone())
	}
	return out
}

// Reports returns defensive report copies in deterministic producer order.
func (c ReportCollection) Reports() []*Report {
	if len(c.reports) == 0 {
		return nil
	}
	out := make([]*Report, len(c.reports))
	for i, report := range c.reports {
		out[i] = report.Clone()
	}
	return out
}

// Find returns a defensive copy for producer.
func (c ReportCollection) Find(producer string) (*Report, bool) {
	for _, report := range c.reports {
		if report.Producer() == producer {
			return report.Clone(), true
		}
	}
	return nil, false
}

// Producers returns report producer names in deterministic order.
func (c ReportCollection) Producers() []string {
	if len(c.reports) == 0 {
		return nil
	}
	producers := make([]string, len(c.reports))
	for i, report := range c.reports {
		producers[i] = report.Producer()
	}
	return producers
}

// Len returns the number of reports.
func (c ReportCollection) Len() int {
	return len(c.reports)
}

// WithoutProducers returns a collection excluding the supplied producer names.
func (c ReportCollection) WithoutProducers(producers ...string) ReportCollection {
	if len(c.reports) == 0 {
		return ReportCollection{}
	}
	excluded := make(map[string]struct{}, len(producers))
	for _, producer := range producers {
		excluded[producer] = struct{}{}
	}
	reports := make([]*Report, 0, len(c.reports))
	for _, report := range c.reports {
		if _, skip := excluded[report.Producer()]; skip {
			continue
		}
		reports = append(reports, report)
	}
	return NewReportCollection(reports...)
}

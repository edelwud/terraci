package ci

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// ReportPublisher publishes renderer-ready producer reports.
type ReportPublisher interface {
	Publish(report *Report)
	SaveReport(ctx context.Context, report *Report) error
}

// ReportReader loads reports from the current process and/or service dir.
type ReportReader interface {
	Get(producer string) (*Report, bool)
	All() []*Report
	LoadReports(ctx context.Context) ([]*Report, error)
}

// ArtifactWriter persists producer result/report artifacts.
type ArtifactWriter interface {
	SaveResults(ctx context.Context, producer string, results any) error
	ReplaceResultsAndReport(ctx context.Context, producer string, results any, report *Report) error
}

// ReportStore is the canonical producer/consumer boundary for CI reports.
// Memory stores support in-process exchange; file stores additionally persist
// artifacts using the canonical {producer}-report.json / {producer}-results.json
// filenames.
type ReportStore interface {
	ReportPublisher
	ReportReader
	ArtifactWriter
}

// NewMemoryReportStore creates an in-process report store.
func NewMemoryReportStore() ReportStore {
	return newMemoryReportStore()
}

// NewFileReportStore creates a report store backed by serviceDir. It also keeps
// an in-memory overlay for reports published by in-process plugins.
func NewFileReportStore(serviceDir string) ReportStore {
	return &fileReportStore{
		serviceDir: serviceDir,
		memory:     newMemoryReportStore(),
	}
}

type memoryReportStore struct {
	mu      sync.RWMutex
	reports map[string]*Report
}

func newMemoryReportStore() *memoryReportStore {
	return &memoryReportStore{reports: make(map[string]*Report)}
}

func (s *memoryReportStore) Publish(report *Report) {
	if s == nil || report == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.reports[report.Producer] = report.Clone()
}

func (s *memoryReportStore) deleteReport(producer string) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.reports, producer)
}

func (s *memoryReportStore) Get(producer string) (*Report, bool) {
	if s == nil {
		return nil, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	report, ok := s.reports[producer]
	if !ok {
		return nil, false
	}
	return report.Clone(), true
}

func (s *memoryReportStore) All() []*Report {
	if s == nil {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	reports := make([]*Report, 0, len(s.reports))
	for _, report := range s.reports {
		reports = append(reports, report.Clone())
	}
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].Producer < reports[j].Producer
	})
	return reports
}

func (s *memoryReportStore) SaveReport(ctx context.Context, report *Report) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := report.Validate(); err != nil {
		return fmt.Errorf("validate report: %w", err)
	}
	s.Publish(report)
	return nil
}

func (s *memoryReportStore) SaveResults(ctx context.Context, producer string, _ any) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	return validateArtifactProducer(producer)
}

func (s *memoryReportStore) ReplaceResultsAndReport(ctx context.Context, producer string, results any, report *Report) error {
	var errs []error
	if err := s.SaveResults(ctx, producer, results); err != nil {
		errs = append(errs, fmt.Errorf("save results: %w", err))
	}
	if report == nil {
		s.deleteReport(producer)
		return errors.Join(errs...)
	}
	if err := validateReportProducer(producer, report); err != nil {
		errs = append(errs, fmt.Errorf("save report: %w", err))
	} else if err := s.SaveReport(ctx, report); err != nil {
		errs = append(errs, fmt.Errorf("save report: %w", err))
	}
	return errors.Join(errs...)
}

func (s *memoryReportStore) LoadReports(ctx context.Context) ([]*Report, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	return s.All(), nil
}

type fileReportStore struct {
	serviceDir string
	memory     *memoryReportStore
}

func (s *fileReportStore) Publish(report *Report) {
	s.memory.Publish(report)
}

func (s *fileReportStore) deleteReport(ctx context.Context, producer string) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := validateArtifactProducer(producer); err != nil {
		return err
	}
	s.memory.deleteReport(producer)
	if s.serviceDir == "" {
		return nil
	}
	err := os.Remove(filepath.Join(s.serviceDir, ReportFilename(producer)))
	if err == nil || errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("delete report: %w", err)
}

func (s *fileReportStore) Get(producer string) (*Report, bool) {
	return s.memory.Get(producer)
}

func (s *fileReportStore) All() []*Report {
	return s.memory.All()
}

func (s *fileReportStore) SaveReport(ctx context.Context, report *Report) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := report.Validate(); err != nil {
		return fmt.Errorf("validate report: %w", err)
	}
	if s.serviceDir == "" {
		s.Publish(report)
		return nil
	}
	if err := saveJSON(ctx, s.serviceDir, ReportFilename(report.Producer), report); err != nil {
		return err
	}
	s.Publish(report)
	return nil
}

func (s *fileReportStore) SaveResults(ctx context.Context, producer string, results any) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := validateArtifactProducer(producer); err != nil {
		return err
	}
	if s.serviceDir == "" {
		return nil
	}
	return saveJSON(ctx, s.serviceDir, ResultFilename(producer), results)
}

func (s *fileReportStore) ReplaceResultsAndReport(ctx context.Context, producer string, results any, report *Report) error {
	var errs []error
	if err := s.SaveResults(ctx, producer, results); err != nil {
		errs = append(errs, fmt.Errorf("save results: %w", err))
	}
	if report == nil {
		if err := s.deleteReport(ctx, producer); err != nil {
			errs = append(errs, err)
		}
		return errors.Join(errs...)
	}
	if err := validateReportProducer(producer, report); err != nil {
		errs = append(errs, fmt.Errorf("save report: %w", err))
	} else if err := s.SaveReport(ctx, report); err != nil {
		errs = append(errs, fmt.Errorf("save report: %w", err))
	}
	return errors.Join(errs...)
}

func (s *fileReportStore) LoadReports(ctx context.Context) ([]*Report, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}

	byProducer := make(map[string]*Report)
	if s.serviceDir != "" {
		files, err := reportFiles(s.serviceDir)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			report, err := loadReport(ctx, file)
			if err != nil {
				return nil, fmt.Errorf("load report %s: %w", filepath.Base(file), err)
			}
			byProducer[report.Producer] = report
		}
	}

	for _, report := range s.memory.All() {
		byProducer[report.Producer] = report
	}

	producers := make([]string, 0, len(byProducer))
	for producer := range byProducer {
		producers = append(producers, producer)
	}
	sort.Strings(producers)

	reports := make([]*Report, 0, len(producers))
	for _, producer := range producers {
		reports = append(reports, byProducer[producer].Clone())
	}
	return reports, nil
}

func reportFiles(serviceDir string) ([]string, error) {
	pattern := filepath.Join(serviceDir, "*-report.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob reports: %w", err)
	}
	sort.Strings(files)
	return files, nil
}

func validateArtifactProducer(producer string) error {
	if strings.TrimSpace(producer) == "" {
		return errors.New("artifact producer is required")
	}
	if strings.ContainsAny(producer, `/\`) {
		return fmt.Errorf("artifact producer %q is not a safe artifact name", producer)
	}
	return nil
}

func validateReportProducer(producer string, report *Report) error {
	if report == nil {
		return nil
	}
	if report.Producer != producer {
		return fmt.Errorf("report producer %q does not match artifact producer %q", report.Producer, producer)
	}
	return nil
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func saveJSON(ctx context.Context, serviceDir, filename string, v any) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	path := filepath.Join(serviceDir, filename)
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

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

// ReportLoader loads reports from the current process and/or service dir.
type ReportLoader interface {
	LoadReports(ctx context.Context) (ReportCollection, error)
}

// ArtifactPublisher publishes producer result/report artifacts.
type ArtifactPublisher interface {
	PublishArtifacts(ctx context.Context, publication ArtifactPublication) error
}

// ReportStore is the canonical producer/consumer boundary for CI reports.
// Memory stores support in-process exchange; file stores additionally persist
// artifacts using the canonical {producer}-report.json / {producer}-results.json
// filenames.
type ReportStore interface {
	ReportLoader
	ArtifactPublisher
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

func (s *memoryReportStore) publish(report *Report) {
	if s == nil || report == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.reports[report.Producer()] = report.Clone()
}

func (s *memoryReportStore) deleteReport(producer string) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.reports, producer)
}

func (s *memoryReportStore) all() []*Report {
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
		return reports[i].Producer() < reports[j].Producer()
	})
	return reports
}

func (s *memoryReportStore) saveReport(ctx context.Context, report *Report) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := report.Validate(); err != nil {
		return fmt.Errorf("validate report: %w", err)
	}
	s.publish(report)
	return nil
}

func (s *memoryReportStore) saveResults(ctx context.Context, producer string, _ any) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	return validateArtifactProducer(producer)
}

func (s *memoryReportStore) replaceResultsAndReport(ctx context.Context, producer string, results any, report *Report) error {
	var errs []error
	if err := s.saveResults(ctx, producer, results); err != nil {
		errs = append(errs, fmt.Errorf("save results: %w", err))
	}
	if report == nil {
		s.deleteReport(producer)
		return errors.Join(errs...)
	}
	if err := validateReportProducer(producer, report); err != nil {
		errs = append(errs, fmt.Errorf("save report: %w", err))
	} else if err := s.saveReport(ctx, report); err != nil {
		errs = append(errs, fmt.Errorf("save report: %w", err))
	}
	return errors.Join(errs...)
}

func (s *memoryReportStore) replaceReport(ctx context.Context, producer string, report *Report) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if report == nil {
		s.deleteReport(producer)
		return nil
	}
	if err := validateReportProducer(producer, report); err != nil {
		return err
	}
	return s.saveReport(ctx, report)
}

func (s *memoryReportStore) LoadReports(ctx context.Context) (ReportCollection, error) {
	if err := contextError(ctx); err != nil {
		return ReportCollection{}, err
	}
	return NewReportCollection(s.all()...), nil
}

func (s *memoryReportStore) PublishArtifacts(ctx context.Context, publication ArtifactPublication) error {
	return publishToStore(ctx, publication, s)
}

type fileReportStore struct {
	serviceDir string
	memory     *memoryReportStore
}

func (s *fileReportStore) publish(report *Report) {
	s.memory.publish(report)
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

func (s *fileReportStore) saveReport(ctx context.Context, report *Report) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := report.Validate(); err != nil {
		return fmt.Errorf("validate report: %w", err)
	}
	if s.serviceDir == "" {
		s.publish(report)
		return nil
	}
	if err := saveJSON(ctx, s.serviceDir, ReportFilename(report.Producer()), report); err != nil {
		return err
	}
	s.publish(report)
	return nil
}

func (s *fileReportStore) saveResults(ctx context.Context, producer string, results any) error {
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

func (s *fileReportStore) replaceResultsAndReport(ctx context.Context, producer string, results any, report *Report) error {
	var errs []error
	if err := s.saveResults(ctx, producer, results); err != nil {
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
	} else if err := s.saveReport(ctx, report); err != nil {
		errs = append(errs, fmt.Errorf("save report: %w", err))
	}
	return errors.Join(errs...)
}

func (s *fileReportStore) replaceReport(ctx context.Context, producer string, report *Report) error {
	if report == nil {
		return s.deleteReport(ctx, producer)
	}
	if err := validateReportProducer(producer, report); err != nil {
		return err
	}
	return s.saveReport(ctx, report)
}

func (s *fileReportStore) LoadReports(ctx context.Context) (ReportCollection, error) {
	if err := contextError(ctx); err != nil {
		return ReportCollection{}, err
	}

	byProducer := make(map[string]*Report)
	if s.serviceDir != "" {
		files, err := reportFiles(s.serviceDir)
		if err != nil {
			return ReportCollection{}, err
		}
		for _, file := range files {
			report, err := loadReport(ctx, file)
			if err != nil {
				return ReportCollection{}, fmt.Errorf("load report %s: %w", filepath.Base(file), err)
			}
			byProducer[report.Producer()] = report
		}
	}

	for _, report := range s.memory.all() {
		byProducer[report.Producer()] = report
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
	return NewReportCollection(reports...), nil
}

func (s *fileReportStore) PublishArtifacts(ctx context.Context, publication ArtifactPublication) error {
	return publishToStore(ctx, publication, s)
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
	if report.Producer() != producer {
		return fmt.Errorf("report producer %q does not match artifact producer %q", report.Producer(), producer)
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

type artifactStore interface {
	replaceResultsAndReport(ctx context.Context, producer string, results any, report *Report) error
	replaceReport(ctx context.Context, producer string, report *Report) error
}

func publishToStore(ctx context.Context, publication ArtifactPublication, store artifactStore) error {
	if err := contextPublicationError(ctx, publication); err != nil {
		return err
	}
	if store == nil {
		return nil
	}

	var errs []error
	report, err := buildPublicationReport(publication)
	if err != nil {
		errs = append(errs, err)
	}
	if results, ok := publication.results.valueToWrite(); ok {
		if err := store.replaceResultsAndReport(ctx, publication.producer, results, report); err != nil {
			errs = append(errs, fmt.Errorf("replace artifacts: %w", err))
		}
		return errors.Join(errs...)
	}
	if err := store.replaceReport(ctx, publication.producer, report); err != nil {
		errs = append(errs, fmt.Errorf("replace report: %w", err))
	}
	return errors.Join(errs...)
}

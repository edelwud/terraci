package source

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"golang.org/x/sync/errgroup"
)

type loadSession struct {
	modulePath string
	builder    *indexBuilder
}

const parallelParseThreshold = 8

type parsedFileResult struct {
	path  string
	file  *hcl.File
	diags hcl.Diagnostics
	err   error
}

func newLoadSession(modulePath string) *loadSession {
	return &loadSession{
		modulePath: modulePath,
	}
}

func (s *loadSession) Run(ctx context.Context) (*Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	tfFiles, err := s.discoverTFFiles()
	if err != nil {
		return nil, err
	}

	s.builder = newIndexBuilder(s.modulePath, len(tfFiles))

	if err := s.parseFiles(tfFiles); err != nil {
		return nil, err
	}

	s.parseLockFile()

	return s.builder.Snapshot(), nil
}

func (s *loadSession) discoverTFFiles() ([]string, error) {
	entries, err := os.ReadDir(s.modulePath)
	if err != nil {
		return nil, fmt.Errorf("read module dir: %w", err)
	}

	tfFiles := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tf") {
			continue
		}
		tfFiles = append(tfFiles, filepath.Join(s.modulePath, entry.Name()))
	}

	return tfFiles, nil
}

func (s *loadSession) parseFiles(tfFiles []string) error {
	if len(tfFiles) < parallelParseThreshold {
		return s.parseFilesSequential(tfFiles)
	}

	return s.parseFilesParallel(tfFiles)
}

func (s *loadSession) parseFilesSequential(tfFiles []string) error {
	for _, tfFile := range tfFiles {
		file, err := s.builder.ParseHCLFile(tfFile)
		if err != nil {
			return fmt.Errorf("read %s: %w", tfFile, err)
		}
		if file != nil {
			s.builder.AddFile(tfFile, file)
		}
	}

	return nil
}

func (s *loadSession) parseFilesParallel(tfFiles []string) error {
	results := make([]parsedFileResult, len(tfFiles))

	var group errgroup.Group
	group.SetLimit(runtime.GOMAXPROCS(0))

	for i, tfFile := range tfFiles {
		group.Go(func() error {
			file, diags, err := parseHCLFile(tfFile)
			results[i] = parsedFileResult{
				path:  tfFile,
				file:  file,
				diags: diags,
				err:   err,
			}
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return err
	}

	for _, result := range results {
		s.builder.AddDiagnostics(result.diags)
		if result.err != nil {
			return fmt.Errorf("read %s: %w", result.path, result.err)
		}
		if result.file != nil {
			s.builder.AddFile(result.path, result.file)
		}
	}

	return nil
}

func (s *loadSession) parseLockFile() {
	lockPath := filepath.Join(s.modulePath, ".terraform.lock.hcl")
	file, diags, err := parseHCLFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		s.builder.SetLockFile(nil, diags)
		return
	}

	s.builder.SetLockFile(file, diags)
}

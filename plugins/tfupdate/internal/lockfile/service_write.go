package lockfile

import "fmt"

func (s *Service) writeProviderEntry(lockPath string, entry LockProviderEntry) error {
	doc, err := ParseDocument(lockPath)
	if err != nil {
		return err
	}

	doc.UpsertProvider(entry)

	if err := s.writer.WriteDocument(lockPath, doc); err != nil {
		return fmt.Errorf("write provider lock file %s: %w", lockPath, err)
	}

	return nil
}

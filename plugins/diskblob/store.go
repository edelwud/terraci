package diskblob

import "github.com/edelwud/terraci/plugins/diskblob/internal/fsstore"

// Store is a filesystem-backed blob store.
type Store = fsstore.Store

// NewStore constructs a filesystem-backed blob store rooted at rootDir.
func NewStore(rootDir string) *Store {
	return fsstore.New(rootDir)
}

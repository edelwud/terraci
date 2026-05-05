package fsstore

import (
	"testing"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
	"github.com/edelwud/terraci/pkg/cache/blobcache/contracttest"
)

// TestStoreContract validates the diskblob backend against the shared
// blobcache.Store contract. Adding a new contract assertion in the suite
// retroactively covers diskblob — that's the whole point.
func TestStoreContract(t *testing.T) {
	contracttest.RunStoreContractTests(t, func(t *testing.T) blobcache.Store {
		return New(t.TempDir())
	})
}

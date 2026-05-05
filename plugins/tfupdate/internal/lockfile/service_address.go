package lockfile

import (
	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/plugins/tfupdate/internal/sourceaddr"
)

func (s *Service) resolveProviderAddress(source, lockPath string) (ProviderAddress, error) {
	address, err := ParseProviderAddress(source)
	if err != nil {
		return ProviderAddress{}, err
	}

	namespace, typeName, shortErr := sourceaddr.ParseProviderSource(source)
	if shortErr != nil {
		return address, nil //nolint:nilerr // short source parse failure is non-fatal; fall back to default address
	}

	existing, status := lookupLockedProviderAddress(lockPath, namespace, typeName)
	switch status {
	case lockAddressFound:
		return existing, nil
	case lockAddressAmbiguous:
		log.WithField("provider_source", source).
			WithField("lock_file", lockPath).
			Warn("update: multiple matching provider lock entries found; using default hostname resolution")
	case lockAddressNotFound:
	}

	return address, nil
}

type lockAddressStatus int

const (
	lockAddressNotFound lockAddressStatus = iota
	lockAddressFound
	lockAddressAmbiguous
)

func lookupLockedProviderAddress(lockPath, namespace, typeName string) (ProviderAddress, lockAddressStatus) {
	doc, err := ParseDocument(lockPath)
	if err != nil {
		return ProviderAddress{}, lockAddressNotFound
	}

	var matched ProviderAddress
	var found bool
	for _, provider := range doc.Providers {
		address, err := ParseProviderAddress(provider.Source)
		if err != nil {
			continue
		}
		if address.Namespace != namespace || address.Type != typeName {
			continue
		}

		if found {
			return ProviderAddress{}, lockAddressAmbiguous
		}
		matched = address
		found = true
	}

	if found {
		return matched, lockAddressFound
	}

	return ProviderAddress{}, lockAddressNotFound
}

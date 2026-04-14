package lockfile

import "github.com/edelwud/terraci/plugins/tfupdate/internal/sourceaddr"

// ProviderAddress is a normalized Terraform provider address.
type ProviderAddress struct {
	Hostname  string
	Namespace string
	Type      string
}

// ParseProviderAddress parses a short or fully-qualified provider source.
func ParseProviderAddress(source string) (ProviderAddress, error) {
	address, err := sourceaddr.ParseProviderAddress(source)
	if err != nil {
		return ProviderAddress{}, err
	}

	return ProviderAddress{
		Hostname:  address.Hostname,
		Namespace: address.Namespace,
		Type:      address.Type,
	}, nil
}

// LockSource returns the canonical source form used in .terraform.lock.hcl.
func (a ProviderAddress) LockSource() string {
	return a.Hostname + "/" + a.Namespace + "/" + a.Type
}

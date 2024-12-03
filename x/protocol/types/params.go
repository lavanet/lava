package types

import (
	"fmt"
	"strconv"
	"strings"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*Params)(nil)

const (
	TARGET_VERSION = "4.1.6"
	MIN_VERSION    = "3.1.0"
)

var (
	KeyVersion     = []byte("Version")
	DefaultVersion = Version{
		ProviderTarget: TARGET_VERSION,
		ProviderMin:    MIN_VERSION,
		ConsumerTarget: TARGET_VERSION,
		ConsumerMin:    MIN_VERSION,
	}
)

const (
	MAX_MINOR    = 10000
	MAX_REVISION = 10000
)

// This is problematic, since for version parameters we would like to validate that:
// (1) the min version is not greater than the version (limit a,b); and
// (2) the provider and consumer versions are in (for now) sync (limit a,b)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	version Version,
) Params {
	return Params{
		Version: version,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultVersion,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyVersion, &p.Version, validateVersion),
	}
}

// helper to convert Version to an integer (easily compared)
func versionToInteger(v string) (int, error) {
	s := strings.Split(v, ".")
	if len(s) != 3 {
		return 0, fmt.Errorf("invalid format")
	}

	maj, err := strconv.ParseUint(s[0], 10, 0)
	if err != nil {
		return 0, fmt.Errorf("%w (major)", err)
	}
	min, err := strconv.ParseUint(s[1], 10, 0)
	if err != nil {
		return 0, fmt.Errorf("%w (minor)", err)
	}
	rev, err := strconv.ParseUint(s[2], 10, 0)
	if err != nil {
		return 0, fmt.Errorf("%w (revision)", err)
	}

	if min > MAX_MINOR {
		return 0, fmt.Errorf("minor too big: %d > max %d", min, MAX_MINOR)
	}
	if rev > MAX_REVISION {
		return 0, fmt.Errorf("revision too big: %d > max %d", rev, MAX_REVISION)
	}

	n := maj*MAX_MINOR*MAX_REVISION + min*MAX_REVISION + rev

	return int(n), nil
}

// Validate validates the set of params
func (p Params) Validate() error {
	return validateVersion(p.Version)
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// validateVersion validates the Version param
func validateVersion(v interface{}) error {
	version, ok := v.(Version)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	return version.validateVersion()
}

package types

import (
	"fmt"
	"strconv"
	"strings"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyVersion     = []byte("Version")
	DefaultVersion = Version{
		ProviderTarget: "0.23.5",
		ProviderMin:    "0.21.0",
		ConsumerTarget: "0.23.5",
		ConsumerMin:    "0.21.0",
	}
)

const (
	MAX_MINOR    = 10000
	MAX_REVISION = 10000
)

// support for params (via cosmos-sdk's x/params) is limited: parameter change
// proposals (a) handle one parameter at a time, and (b) do not allow access to
// to a module's state during validation.
//
// This is problematic, since for version parameters we would like to validate that:
// (1) a new version is not smaller than the existing (limit b); and
// (2) the min version is not greater than the version (limit a,b); and
// (3) the provider and consumer versions are in (for now) sync (limit a,b)
//
// We address (a) by packing all versions into a single struct. We hack around (b)
// by using BeginBlock() callback to always update the latest params from the
// store at the beginning of each block, so the latest values are always available
// for the validation callbacks when a proposal arrives.

// a copy of the latest params (from keeper store) to workaround limit (b) above
var latestParams Params = DefaultParams()

// UpdateLatestParams updates the local (in memory) copy of the params from the store
func UpdateLatestParams(params Params) {
	latestParams = params
}

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
	if err := validateVersion(p.Version); err != nil {
		return err
	}

	return nil
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

	providerTarget, _ := versionToInteger(latestParams.Version.ProviderTarget)
	providerMin, _ := versionToInteger(latestParams.Version.ProviderMin)
	consumerTarget, _ := versionToInteger(latestParams.Version.ConsumerTarget)
	consumerMin, _ := versionToInteger(latestParams.Version.ConsumerMin)

	newProviderTarget, err := versionToInteger(version.ProviderTarget)
	if err != nil {
		return fmt.Errorf("provider target version: %w", err)
	}
	newProviderMin, err := versionToInteger(version.ProviderMin)
	if err != nil {
		return fmt.Errorf("provider min version: %w", err)
	}
	newConsumerTarget, err := versionToInteger(version.ConsumerTarget)
	if err != nil {
		return fmt.Errorf("consumer target version: %w", err)
	}
	newConsumerMin, err := versionToInteger(version.ConsumerMin)
	if err != nil {
		return fmt.Errorf("consumer min version: %w", err)
	}

	// versions may not decrease
	if newProviderTarget < providerTarget {
		return fmt.Errorf("provider target version smaller than latest: %d < %d",
			newProviderTarget, providerTarget)
	}
	if newProviderMin < providerMin {
		return fmt.Errorf("provider min version smaller than latest: %d < %d",
			newProviderMin, providerMin)
	}
	if newConsumerTarget < consumerTarget {
		return fmt.Errorf("consumer target version smaller than latest: %d < %d",
			newConsumerTarget, consumerTarget)
	}
	if newConsumerMin < consumerMin {
		return fmt.Errorf("consumer min version smaller than latest: %d < %d",
			newConsumerMin, consumerMin)
	}

	// min version may not exceed target version
	if newProviderMin > newProviderTarget {
		return fmt.Errorf("provider min version exceeds target version: %d > %d",
			newProviderMin, newProviderTarget)
	}
	if newConsumerMin > newConsumerTarget {
		return fmt.Errorf("consumer min version exceeds target version: %d > %d",
			newConsumerMin, newConsumerTarget)
	}

	// provider and consumer versions must match (for now)
	if newProviderTarget != newConsumerTarget {
		return fmt.Errorf("provider and consumer target versions mismatch: %d != %d",
			newProviderTarget, newConsumerTarget)
	}
	if newProviderMin != newConsumerMin {
		return fmt.Errorf("provider and consumer min versions mismatch: %d != %d",
			newProviderMin, newConsumerMin)
	}

	return nil
}

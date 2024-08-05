package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/v2/utils/decoder"
	"github.com/lavanet/lava/v2/utils/lavaslices"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	"github.com/mitchellh/mapstructure"
)

const WILDCARD_CHAIN_POLICY = "*" // wildcard allows you to define only part of the chains and allow all others

// init policy default values (for fields that their natural zero value is not good)
// the values were chosen in a way that they will not influence the strictest policy calculation
var policyDefaultValues = map[string]interface{}{
	"geolocation_profile":   int32(Geolocation_GL),
	"max_providers_to_pair": uint64(math.MaxUint64),
}

func (policy *Policy) ContainsChainID(chainID string) bool {
	if policy == nil {
		return false
	}
	if len(policy.ChainPolicies) == 0 {
		// empty chainPolicies -> support all chains
		return true
	}

	for _, chain := range policy.ChainPolicies {
		if chain.ChainId == chainID {
			return true
		}
	}
	return false
}

// gets the chainPolicy if exists, null safe
func (policy *Policy) ChainPolicy(chainID string) (chainPolicy ChainPolicy, allowed bool) {
	// empty policy | chainPolicies -> support all chains
	if policy == nil || len(policy.ChainPolicies) == 0 {
		return ChainPolicy{ChainId: chainID}, true
	}
	wildcard := false
	for _, chain := range policy.ChainPolicies {
		if chain.ChainId == chainID {
			return chain, true
		}
		if chain.ChainId == WILDCARD_CHAIN_POLICY {
			wildcard = true
		}
	}
	if wildcard {
		return ChainPolicy{ChainId: chainID}, true
	}
	return ChainPolicy{}, false
}

func (policy *Policy) GetSupportedAddons(specID string) (addons []string, err error) {
	chainPolicy, allowed := policy.ChainPolicy(specID)
	if !allowed {
		return nil, fmt.Errorf("specID %s not allowed by current policy", specID)
	}
	addons = []string{""} // always allow an empty addon
	for _, requirement := range chainPolicy.Requirements {
		addons = append(addons, requirement.Collection.AddOn)
	}
	return addons, nil
}

func (policy *Policy) GetSupportedExtensions(specID string) (extensions []epochstoragetypes.EndpointService, err error) {
	chainPolicy, allowed := policy.ChainPolicy(specID)
	if !allowed {
		return nil, fmt.Errorf("specID %s not allowed by current policy", specID)
	}
	extensions = []epochstoragetypes.EndpointService{}
	for _, requirement := range chainPolicy.Requirements {
		// always allow an empty extension
		emptyExtension := epochstoragetypes.EndpointService{
			ApiInterface: requirement.Collection.ApiInterface,
			Addon:        requirement.Collection.AddOn,
			Extension:    "",
		}
		extensions = append(extensions, emptyExtension)
		for _, extension := range requirement.Extensions {
			extensionServiceToAdd := epochstoragetypes.EndpointService{
				ApiInterface: requirement.Collection.ApiInterface,
				Addon:        requirement.Collection.AddOn,
				Extension:    extension,
			}
			extensions = append(extensions, extensionServiceToAdd)
		}
	}
	return extensions, nil
}

func (policy Policy) ValidateBasicPolicy(isPlanPolicy bool) error {
	// plan policy checks
	if isPlanPolicy {
		if policy.EpochCuLimit == 0 || policy.TotalCuLimit == 0 {
			return sdkerrors.Wrapf(ErrInvalidPolicyCuFields, `plan's compute units fields can't be zero 
				(EpochCuLimit = %v, TotalCuLimit = %v)`, policy.EpochCuLimit, policy.TotalCuLimit)
		}

		if policy.SelectedProvidersMode == SELECTED_PROVIDERS_MODE_DISABLED && len(policy.SelectedProviders) != 0 {
			return sdkerrors.Wrap(ErrPolicyInvalidSelectedProvidersConfig, `cannot configure mode = 3 (selected 
				providers feature is disabled) and non-empty list of selected providers`)
		}

		// non-plan policy checks
	} else if policy.SelectedProvidersMode == SELECTED_PROVIDERS_MODE_DISABLED {
		return sdkerrors.Wrap(ErrPolicyInvalidSelectedProvidersConfig, `cannot configure mode = 3 (selected 
				providers feature is disabled) for a policy that is not plan policy`)
	}

	// general policy checks
	if policy.EpochCuLimit > policy.TotalCuLimit {
		return sdkerrors.Wrapf(ErrInvalidPolicyCuFields, "EpochCuLimit can't be larger than TotalCuLimit (EpochCuLimit = %v, TotalCuLimit = %v)", policy.EpochCuLimit, policy.TotalCuLimit)
	}

	if policy.MaxProvidersToPair <= 1 {
		return sdkerrors.Wrapf(ErrInvalidPolicyMaxProvidersToPair, "invalid policy's MaxProvidersToPair fields (MaxProvidersToPair = %v)", policy.MaxProvidersToPair)
	}

	if policy.SelectedProvidersMode == SELECTED_PROVIDERS_MODE_ALLOWED && len(policy.SelectedProviders) != 0 {
		return sdkerrors.Wrap(ErrPolicyInvalidSelectedProvidersConfig, `cannot configure mode = 0 (no 
			providers restrictions) and non-empty list of selected providers`)
	}

	if policy.GeolocationProfile == int32(Geolocation_GLS) && !isPlanPolicy {
		return sdkerrors.Wrap(ErrPolicyGeolocation, `cannot configure geolocation = GLS (0)`)
	}

	if !IsValidGeoEnum(policy.GeolocationProfile) {
		return sdkerrors.Wrap(ErrPolicyGeolocation, `invalid geolocation enum`)
	}

	seen := map[string]bool{}
	for _, addr := range policy.SelectedProviders {
		_, err := sdk.AccAddressFromBech32(addr)
		if err != nil {
			return sdkerrors.Wrapf(legacyerrors.ErrInvalidAddress, "invalid selected provider address (%s)", err)
		}

		if seen[addr] {
			return sdkerrors.Wrapf(ErrPolicyInvalidSelectedProvidersConfig, "found duplicate provider address %s", addr)
		}
		seen[addr] = true
	}
	for _, chainPolicy := range policy.ChainPolicies {
		for _, requirement := range chainPolicy.GetRequirements() {
			if requirement.Collection.ApiInterface == "" {
				return sdkerrors.Wrapf(legacyerrors.ErrInvalidRequest, "invalid requirement definition requirement must define collection with an apiInterface (%+v)", chainPolicy)
			}
		}
	}

	return nil
}

func GetStrictestChainPolicyForSpec(chainID string, policies []*Policy) (chainPolicyRet ChainPolicy, allowed bool) {
	requirements := []ChainRequirement{}
	for _, policy := range policies {
		chainPolicy, allowdChain := policy.ChainPolicy(chainID)
		if !allowdChain {
			return ChainPolicy{}, false
		}
		// get the strictest collection specification, while empty is allowed
		chainPolicyRequirements := chainPolicy.Requirements
		// if no collection data is specified in the policy previous allowed is stricter and no update is necessary
		if len(chainPolicyRequirements) == 0 {
			continue
		}
		// this policy is limiting collection data so overwrite what is allowed
		if len(requirements) == 0 {
			requirements = chainPolicyRequirements
			continue
		}
		// previous policies and current policy change collection data, we need the union of both
		requirements = lavaslices.UnionByFunc(chainPolicyRequirements, requirements)
	}

	return ChainPolicy{ChainId: chainID, Requirements: requirements}, true
}

func VerifyTotalCuUsage(effectiveTotalCu uint64, cuUsage uint64) bool {
	return cuUsage < effectiveTotalCu
}

// allows unmarshaling parser func
func (s SELECTED_PROVIDERS_MODE) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(SELECTED_PROVIDERS_MODE_name[int32(s)])
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON unmashals a quoted json string to the enum value
func (s *SELECTED_PROVIDERS_MODE) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}
	// Note that if the string cannot be found then the zero value is used ('Created' in this case)
	*s = SELECTED_PROVIDERS_MODE(SELECTED_PROVIDERS_MODE_value[j])
	return nil
}

func ParsePolicyFromYamlString(input string) (*Policy, error) {
	return parsePolicyFromYaml(input, false)
}

func ParsePolicyFromYamlPath(path string) (*Policy, error) {
	return parsePolicyFromYaml(path, true)
}

func parsePolicyFromYaml(from string, isPath bool) (*Policy, error) {
	var policy Policy

	enumHooks := []mapstructure.DecodeHookFunc{
		PolicyEnumDecodeHookFunc,
	}
	var (
		unused []string
		unset  []string
		err    error
	)

	if isPath {
		err = decoder.DecodeFile(from, "Policy", &policy, enumHooks, &unset, &unused)
	} else {
		err = decoder.Decode(from, "Policy", &policy, enumHooks, &unset, &unused)
	}

	if err != nil {
		return &policy, err
	}

	if len(unused) != 0 {
		return &policy, fmt.Errorf("invalid policy: unknown field(s): %v", unused)
	}
	if len(unset) != 0 {
		err = policy.HandleUnsetPolicyFields(unset)
		if err != nil {
			return &policy, err
		}
	}

	return &policy, nil
}

// handleMissingPolicyFields sets default values to missing fields
func (p *Policy) HandleUnsetPolicyFields(unset []string) error {
	defaultValues := make(map[string]interface{})

	for _, field := range unset {
		// fields without explicit default values use their natural default value
		if defValue, ok := policyDefaultValues[field]; ok {
			defaultValues[field] = defValue
		}
	}

	return decoder.SetDefaultValues(defaultValues, p)
}

func DecodeSelectedProvidersMode(dataStr string) (interface{}, error) {
	mode, found := SELECTED_PROVIDERS_MODE_value[dataStr]
	if found {
		return SELECTED_PROVIDERS_MODE(mode), nil
	} else {
		return 0, fmt.Errorf("invalid selected providers mode: %s", dataStr)
	}
}

func (cr ChainRequirement) Differentiator() string {
	if cr.Collection.ApiInterface == "" {
		return ""
	}
	diff := cr.Collection.String() + strings.Join(cr.Extensions, ",")
	if cr.Mixed {
		diff = "mixed-" + diff
	}
	return diff
}

func PolicyEnumDecodeHookFunc(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if t == reflect.TypeOf(Policy{}) {
		policyMap, ok := data.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("unexpected data type for policy field")
		}

		// geolocation enum handling
		geo, ok := policyMap["geolocation_profile"]
		if ok {
			if geoStr, ok := geo.(string); ok {
				geoUint, err := ParseGeoEnum(geoStr)
				if err != nil {
					return nil, err
				}
				policyMap["geolocation_profile"] = geoUint
			}
		}

		// selected providers mode enum handling
		mode, ok := policyMap["selected_providers_mode"]
		if ok {
			if modeStr, ok := mode.(string); ok {
				modeEnum, err := DecodeSelectedProvidersMode(modeStr)
				if err != nil {
					return nil, err
				}
				policyMap["selected_providers_mode"] = modeEnum
			}
		}

		data = policyMap
	}
	return data, nil
}

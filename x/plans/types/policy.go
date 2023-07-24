package types

import (
	"bytes"
	"encoding/json"
	fmt "fmt"
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	commontypes "github.com/lavanet/lava/common/types"
)

// init policy default values (for fields that their natural zero value is not good)
// the values were chosen in a way that they will not influence the strictest policy calculation
var policyDefaultValues = map[string]interface{}{
	"GeolocationProfile": uint64(Geolocation_value["GL"]),
	"MaxProvidersToPair": uint64(math.MaxUint64),
}

func (policy *Policy) ContainsChainID(chainID string) bool {
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

func (policy Policy) ValidateBasicPolicy(isPlanPolicy bool) error {
	// plan policy checks
	if isPlanPolicy {
		if policy.EpochCuLimit == 0 || policy.TotalCuLimit == 0 {
			return sdkerrors.Wrapf(ErrInvalidPolicyCuFields, `plan's compute units fields can't be zero 
				(EpochCuLimit = %v, TotalCuLimit = %v)`, policy.EpochCuLimit, policy.TotalCuLimit)
		}

		if policy.SelectedProvidersMode == SELECTED_PROVIDERS_MODE_DISABLED && len(policy.SelectedProviders) != 0 {
			return sdkerrors.Wrap(ErrInvalidSelectedProvidersConfig, `cannot configure mode = 3 (selected 
				providers feature is disabled) and non-empty list of selected providers`)
		}

		// non-plan policy checks
	} else if policy.SelectedProvidersMode == SELECTED_PROVIDERS_MODE_DISABLED {
		return sdkerrors.Wrap(ErrInvalidSelectedProvidersConfig, `cannot configure mode = 3 (selected 
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
		return sdkerrors.Wrap(ErrInvalidSelectedProvidersConfig, `cannot configure mode = 0 (no 
			providers restrictions) and non-empty list of selected providers`)
	}

	seen := map[string]bool{}
	for _, addr := range policy.SelectedProviders {
		_, err := sdk.AccAddressFromBech32(addr)
		if err != nil {
			return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid selected provider address (%s)", err)
		}

		if seen[addr] {
			return sdkerrors.Wrapf(ErrInvalidSelectedProvidersConfig, "found duplicate provider address %s", addr)
		}
		seen[addr] = true
	}

	return nil
}

func CheckChainIdExistsInPolicies(chainID string, policies []*Policy) bool {
	for _, policy := range policies {
		if policy != nil {
			if policy.ContainsChainID(chainID) {
				return true
			}
		}
	}

	return false
}

func VerifyTotalCuUsage(policies []*Policy, cuUsage uint64) bool {
	for _, policy := range policies {
		if policy != nil {
			if cuUsage >= policy.GetTotalCuLimit() {
				return false
			}
		}
	}

	return true
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
	// Note that if the string cannot be found then it will be set to the zero value, 'Created' in this case.
	*s = SELECTED_PROVIDERS_MODE(SELECTED_PROVIDERS_MODE_value[j])
	return nil
}

func ParsePolicyFromYaml(filePath string) (*Policy, error) {
	var policy Policy
	enumHooks := []commontypes.EnumDecodeHookFuncType{
		commontypes.EnumDecodeHook(uint64(0), parsePolicyEnumValue), // for geolocation
		commontypes.EnumDecodeHook(SELECTED_PROVIDERS_MODE(0), parsePolicyEnumValue),
		// Add more enum hook functions for other enum types as needed
	}

	missingFields, err := commontypes.ReadYaml(filePath, "Policy", &policy, enumHooks)
	if err != nil {
		return &policy, err
	}

	handleMissingPolicyFields(missingFields, &policy)

	return &policy, nil
}

// handleMissingPolicyFields sets default values to missing fields
func handleMissingPolicyFields(missingFields []string, policy *Policy) {
	missingFieldsDefaultValues := make(map[string]interface{})

	for _, field := range missingFields {
		defValue, ok := policyDefaultValues[field]
		// not checking if not ok because fields without default values can use
		// their natural default value (it's not an error)
		if ok {
			missingFieldsDefaultValues[field] = defValue
		}
	}

	commontypes.SetDefaultValues(policy, missingFieldsDefaultValues)
}

// parseEnumValue is a helper function to parse the enum value based on the provided enumType.
func parsePolicyEnumValue(enumType interface{}, strVal string) (interface{}, error) {
	switch v := enumType.(type) {
	case uint64:
		geo, err := ParseGeoEnum(strVal)
		if err != nil {
			return 0, fmt.Errorf("invalid geolocation %s", strVal)
		}
		return geo, nil
	case SELECTED_PROVIDERS_MODE:
		return DecodeSelectedProvidersMode(strVal)
	// Add cases for other enum types as needed
	default:
		return nil, fmt.Errorf("unsupported enum type: %T", v)
	}
}

func DecodeSelectedProvidersMode(dataStr string) (interface{}, error) {
	mode, found := SELECTED_PROVIDERS_MODE_value[dataStr]
	if found {
		return SELECTED_PROVIDERS_MODE(mode), nil
	} else {
		return 0, fmt.Errorf("invalid selected providers mode: %s", dataStr)
	}
}

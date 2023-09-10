package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeMissingFields(t *testing.T) {
	var epochCuLimitDefault uint64
	var maxProivdersToPairDefault uint64

	epochVal, ok := policyDefaultValues["epoch_cu_limit"]
	if !ok {
		epochCuLimitDefault = 0
	} else {
		epochCuLimitDefault, ok = epochVal.(uint64)
		require.True(t, ok)
	}

	maxProvidersVal, ok := policyDefaultValues["max_providers_to_pair"]
	if !ok {
		maxProivdersToPairDefault = uint64(0)
	} else {
		maxProivdersToPairDefault, ok = maxProvidersVal.(uint64)
		require.True(t, ok)
	}

	expectedPolicy := Policy{
		ChainPolicies: []ChainPolicy{
			{ChainId: "ETH1", Apis: []string{"eth_blockNumber"}},
			{ChainId: "FTM250", Apis: []string{"ftm_blockNumber"}},
		},
		GeolocationProfile:    1,
		TotalCuLimit:          1000,
		EpochCuLimit:          epochCuLimitDefault,
		MaxProvidersToPair:    maxProivdersToPairDefault,
		SelectedProvidersMode: SELECTED_PROVIDERS_MODE_EXCLUSIVE,
		SelectedProviders: []string{
			"lava@1kgd936x3tlz2er9untunk7texfanmaud8yp9kf",
			"lava@18puklmhr7u2f9g524tttm24ttf4ud842wtfcna",
			"lava@1hvfeuhp5x94wwf972mfyls8gl0lgxeluklr202",
		},
	}
	input := `
Policy:
  chain_policies:
    - chain_id: ETH1
      apis:
        - eth_blockNumber
      #collections: ...                # MISSING
    - chain_id: FTM250
      apis:
        - ftm_blockNumber
      #collections: ...                # MISSING
  geolocation_profile: 1
  total_cu_limit: 1000
  #epoch_cu_limit: 100                 # MISSING
  #max_providers_to_pair: 2            # MISSING
  selected_providers_mode: 2
  selected_providers:
    - lava@1kgd936x3tlz2er9untunk7texfanmaud8yp9kf
    - lava@18puklmhr7u2f9g524tttm24ttf4ud842wtfcna
    - lava@1hvfeuhp5x94wwf972mfyls8gl0lgxeluklr202
`
	policy, err := ParsePolicyFromYamlString(input)
	require.NoError(t, err)
	require.True(t, policy.Equal(expectedPolicy))
}

func TestDecodeExtraFields(t *testing.T) {
	expectedPolicy := Policy{
		ChainPolicies: []ChainPolicy{
			{ChainId: "ETH1", Apis: []string{"eth_blockNumber"}},
			{ChainId: "FTM250", Apis: []string{"ftm_blockNumber"}},
			{ChainId: "*", Apis: []string{}},
		},
		GeolocationProfile:    1,
		TotalCuLimit:          1000,
		EpochCuLimit:          100,
		MaxProvidersToPair:    2,
		SelectedProvidersMode: SELECTED_PROVIDERS_MODE_EXCLUSIVE,
		SelectedProviders: []string{
			"lava@1kgd936x3tlz2er9untunk7texfanmaud8yp9kf",
			"lava@18puklmhr7u2f9g524tttm24ttf4ud842wtfcna",
			"lava@1hvfeuhp5x94wwf972mfyls8gl0lgxeluklr202",
		},
	}
	input := `
Policy:
  chain_policies:
    - chain_id: ETH1
      apis:
        - eth_blockNumber
    - chain_id: FTM250
      apis:
        - ftm_blockNumber
    - chain_id: "*" # allows all other chains without specifying
  geolocation_profile: 1
  total_cu_limit: 1000
  epoch_cu_limit: 100
  max_providers_to_pair: 2
  selected_providers_mode: EXCLUSIVE
  selected_providers:
    - lava@1kgd936x3tlz2er9untunk7texfanmaud8yp9kf
    - lava@18puklmhr7u2f9g524tttm24ttf4ud842wtfcna
    - lava@1hvfeuhp5x94wwf972mfyls8gl0lgxeluklr202
  myCustomField: 23                    # EXTRA
  dummyListField:                      # EXTRA
    - dummy: abc                       # EXTRA
`
	policy, err := ParsePolicyFromYamlString(input)
	require.Error(t, err)
	require.True(t, policy.Equal(expectedPolicy))
}

func TestDecodeStringGeoloc(t *testing.T) {
	expectedPolicy := Policy{
		ChainPolicies: []ChainPolicy{
			{ChainId: "ETH1", Apis: []string{"eth_blockNumber"}},
			{ChainId: "FTM250", Apis: []string{"ftm_blockNumber"}},
			{ChainId: "*", Apis: []string{}},
		},
		GeolocationProfile:    int32(Geolocation_USE),
		TotalCuLimit:          1000,
		EpochCuLimit:          100,
		MaxProvidersToPair:    2,
		SelectedProvidersMode: 2,
		SelectedProviders: []string{
			"lava@1kgd936x3tlz2er9untunk7texfanmaud8yp9kf",
			"lava@18puklmhr7u2f9g524tttm24ttf4ud842wtfcna",
			"lava@1hvfeuhp5x94wwf972mfyls8gl0lgxeluklr202",
		},
	}
	input := `
Policy:
  chain_policies:
    - chain_id: ETH1
      apis:
        - eth_blockNumber
    - chain_id: FTM250
      apis:
        - ftm_blockNumber
    - chain_id: "*" # allows all other chains without specifying
  geolocation_profile: USE
  total_cu_limit: 1000
  epoch_cu_limit: 100
  max_providers_to_pair: 2
  selected_providers_mode: 2
  selected_providers:
    - lava@1kgd936x3tlz2er9untunk7texfanmaud8yp9kf
    - lava@18puklmhr7u2f9g524tttm24ttf4ud842wtfcna
    - lava@1hvfeuhp5x94wwf972mfyls8gl0lgxeluklr202
`
	policy, err := ParsePolicyFromYamlString(input)
	require.NoError(t, err)
	require.True(t, policy.Equal(expectedPolicy))
}

func TestDecodeSelectedProvidersMode(t *testing.T) {
	expectedPolicy := Policy{
		ChainPolicies: []ChainPolicy{
			{ChainId: "ETH1", Apis: []string{"eth_blockNumber"}},
			{ChainId: "FTM250", Apis: []string{"ftm_blockNumber"}},
			{ChainId: "*", Apis: []string{}},
		},
		GeolocationProfile:    1,
		TotalCuLimit:          1000,
		EpochCuLimit:          100,
		MaxProvidersToPair:    2,
		SelectedProvidersMode: SELECTED_PROVIDERS_MODE_EXCLUSIVE,
		SelectedProviders: []string{
			"lava@1kgd936x3tlz2er9untunk7texfanmaud8yp9kf",
			"lava@18puklmhr7u2f9g524tttm24ttf4ud842wtfcna",
			"lava@1hvfeuhp5x94wwf972mfyls8gl0lgxeluklr202",
		},
	}
	input := `
Policy:
  chain_policies:
    - chain_id: ETH1
      apis:
        - eth_blockNumber
    - chain_id: FTM250
      apis:
        - ftm_blockNumber
    - chain_id: "*" # allows all other chains without specifying
  geolocation_profile: 1
  total_cu_limit: 1000
  epoch_cu_limit: 100
  max_providers_to_pair: 2
  selected_providers_mode: EXCLUSIVE
  selected_providers:
    - lava@1kgd936x3tlz2er9untunk7texfanmaud8yp9kf
    - lava@18puklmhr7u2f9g524tttm24ttf4ud842wtfcna
    - lava@1hvfeuhp5x94wwf972mfyls8gl0lgxeluklr202
`
	policy, err := ParsePolicyFromYamlString(input)
	require.NoError(t, err)
	require.True(t, policy.Equal(expectedPolicy))
}

func TestDecodeStrGeolocSelectedProvidersMode(t *testing.T) {
	expectedPolicy := Policy{
		ChainPolicies: []ChainPolicy{
			{ChainId: "ETH1", Apis: []string{"eth_blockNumber"}},
			{ChainId: "FTM250", Apis: []string{"ftm_blockNumber"}},
			{ChainId: "*", Apis: []string{}},
		},
		GeolocationProfile:    int32(Geolocation_USE),
		TotalCuLimit:          1000,
		EpochCuLimit:          100,
		MaxProvidersToPair:    2,
		SelectedProvidersMode: SELECTED_PROVIDERS_MODE_EXCLUSIVE,
		SelectedProviders: []string{
			"lava@1kgd936x3tlz2er9untunk7texfanmaud8yp9kf",
			"lava@18puklmhr7u2f9g524tttm24ttf4ud842wtfcna",
			"lava@1hvfeuhp5x94wwf972mfyls8gl0lgxeluklr202",
		},
	}
	input := `
Policy:
  chain_policies:
    - chain_id: ETH1
      apis:
        - eth_blockNumber
    - chain_id: FTM250
      apis:
        - ftm_blockNumber
    - chain_id: "*" # allows all other chains without specifying
  geolocation_profile: USE
  total_cu_limit: 1000
  epoch_cu_limit: 100
  max_providers_to_pair: 2
  selected_providers_mode: EXCLUSIVE
  selected_providers:
    - lava@1kgd936x3tlz2er9untunk7texfanmaud8yp9kf
    - lava@18puklmhr7u2f9g524tttm24ttf4ud842wtfcna
    - lava@1hvfeuhp5x94wwf972mfyls8gl0lgxeluklr202
`
	policy, err := ParsePolicyFromYamlString(input)
	require.NoError(t, err)
	require.True(t, policy.Equal(expectedPolicy))
}

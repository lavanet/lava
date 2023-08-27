package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeMissingFields(t *testing.T) {
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
	_, err := ParsePolicyFromYamlString(input)
	require.NoError(t, err)
}

func TestDecodeExtraFields(t *testing.T) {
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
	_, err := ParsePolicyFromYamlString(input)
	require.Error(t, err)
}

func TestDecodeStringGeoloc(t *testing.T) {
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
	_, err := ParsePolicyFromYamlString(input)
	require.NoError(t, err)
}

func TestDecodeSelectedProvidersMode(t *testing.T) {
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
	_, err := ParsePolicyFromYamlString(input)
	require.NoError(t, err)
}

func TestDecodeStrGeolocSelectedProvidersMode(t *testing.T) {
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
	_, err := ParsePolicyFromYamlString(input)
	require.NoError(t, err)
}

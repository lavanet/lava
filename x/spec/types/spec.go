package types

import (
	fmt "fmt"
	"strconv"

	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

const minCU = 1

func (spec Spec) ValidateSpec(maxCU uint64) (map[string]string, error) {
	details := map[string]string{"spec": spec.Name, "status": strconv.FormatBool(spec.Enabled), "chainID": spec.Index}
	functionTags := map[string]bool{}

	availableAPIInterface := map[string]struct{}{
		APIInterfaceJsonRPC:       {},
		APIInterfaceTendermintRPC: {},
		APIInterfaceRest:          {},
		APIInterfaceGrpc:          {},
	}
	availavleEncodings := map[string]struct{}{
		EncodingBase64: {},
		EncodingHex:    {},
	}

	if spec.ReliabilityThreshold == 0 {
		return details, fmt.Errorf("ReliabilityThreshold can't be zero")
	}

	if spec.BlocksInFinalizationProof == 0 {
		return details, fmt.Errorf("BlocksInFinalizationProof can't be zero")
	}

	if spec.AverageBlockTime <= 0 {
		return details, fmt.Errorf("AverageBlockTime can't be zero")
	}

	if spec.AllowedBlockLagForQosSync <= 0 {
		return details, fmt.Errorf("AllowedBlockLagForQosSync can't be zero")
	}

	if spec.MinStakeClient.Denom != epochstoragetypes.TokenDenom || spec.MinStakeClient.Amount.IsZero() {
		return details, fmt.Errorf("MinStakeClient can't be zero andmust have denom of ulava")
	}

	if spec.MinStakeProvider.Denom != epochstoragetypes.TokenDenom || spec.MinStakeProvider.Amount.IsZero() {
		return details, fmt.Errorf("MinStakeProvider can't be zero andmust have denom of ulava")
	}

	for _, api := range spec.Apis {
		if api.ComputeUnits < minCU || api.ComputeUnits > maxCU {
			details["api"] = api.Name
			return details, fmt.Errorf("compute units out or range")
		}

		if len(api.ApiInterfaces) == 0 {
			return details, fmt.Errorf("api interface list empty for %v", api.Name)
		}

		for _, apiInterface := range api.ApiInterfaces {
			if _, ok := availableAPIInterface[apiInterface.Interface]; !ok {
				return details, fmt.Errorf("unsupported api interface %v", apiInterface.Interface)
			}
		}

		if api.Parsing.FunctionTag != "" {
			// Validate tag name
			result := false
			for _, tag := range SupportedTags {
				if tag == api.Parsing.FunctionTag {
					result = true
					functionTags[api.Parsing.FunctionTag] = true
				}
			}

			if !result {
				details["api"] = api.Name
				return details, fmt.Errorf("unsupported function tag")
			}
			if api.Parsing.ResultParsing.Encoding != "" {
				if _, ok := availavleEncodings[api.Parsing.ResultParsing.Encoding]; !ok {
					return details, fmt.Errorf("unsupported api encoding %s in api %v ", api.Parsing.ResultParsing.Encoding, api)
				}
			}
		}
	}

	if spec.DataReliabilityEnabled && spec.Enabled {
		for _, tag := range []string{GET_BLOCKNUM, GET_BLOCK_BY_NUM} {
			if found := functionTags[tag]; !found {
				return details, fmt.Errorf("missing tagged functions for hash comparison: %s", tag)
			}
		}
	}

	return details, nil
}

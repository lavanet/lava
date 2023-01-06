package types

import (
	fmt "fmt"
	"strconv"
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

	for _, api := range spec.Apis {
		if api.ComputeUnits < minCU || api.ComputeUnits > maxCU {
			details["api"] = api.Name
			return details, fmt.Errorf("compute units out or range")
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
		}
	}

	if spec.DataReliabilityEnabled {
		for _, tag := range []string{GET_BLOCKNUM, GET_BLOCK_BY_NUM} {
			if found := functionTags[tag]; !found {
				return details, fmt.Errorf("missing tagged functions for hash comparison: %s", tag)
			}
		}
	}

	return details, nil
}

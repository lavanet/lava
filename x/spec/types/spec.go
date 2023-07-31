package types

import (
	fmt "fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

const minCU = 1

func (spec Spec) ValidateSpec(maxCU uint64) (map[string]string, error) {
	details := map[string]string{"spec": spec.Name, "status": strconv.FormatBool(spec.Enabled), "chainID": spec.Index}
	functionTags := map[FUNCTION_TAG]bool{}

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

	for _, char := range spec.Name {
		if !unicode.IsLower(char) && char != ' ' {
			return details, fmt.Errorf("spec name must contain lowercase characters only")
		}
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
		return details, fmt.Errorf("MinStakeClient can't be zero and must have denom of ulava")
	}

	if spec.MinStakeProvider.Denom != epochstoragetypes.TokenDenom || spec.MinStakeProvider.Amount.IsZero() {
		return details, fmt.Errorf("MinStakeProvider can't be zero and must have denom of ulava")
	}

	for _, apiCollection := range spec.ApiCollections {
		if len(apiCollection.Apis) == 0 {
			return details, fmt.Errorf("api apiCollection list empty for %v", apiCollection.CollectionData)
		}
		if _, ok := availableAPIInterface[apiCollection.CollectionData.ApiInterface]; !ok {
			// if the apiCollection is disabled and has an empty apiInterface it's allowed since this is a base collection for expand
			if apiCollection.CollectionData.ApiInterface != "" || apiCollection.Enabled {
				return details, fmt.Errorf("unsupported api interface %v", apiCollection.CollectionData.ApiInterface)
			}
		}
		// validate function tags
		for _, parsing := range apiCollection.ParseDirectives {
			// Validate function tag
			if parsing.FunctionTag == FUNCTION_TAG_DISABLED {
				details["apiCollection"] = fmt.Sprintf("%v", apiCollection.CollectionData)
				return details, fmt.Errorf("empty or unsupported function tag %s", parsing.FunctionTag)
			}
			functionTags[parsing.FunctionTag] = true

			if parsing.ResultParsing.Encoding != "" {
				if _, ok := availavleEncodings[parsing.ResultParsing.Encoding]; !ok {
					return details, fmt.Errorf("unsupported api encoding %s in apiCollection %v ", parsing.ResultParsing.Encoding, apiCollection.CollectionData)
				}
			}
		}
		currentApis := map[string]struct{}{}
		// validate apis
		for _, api := range apiCollection.Apis {
			_, ok := currentApis[api.Name]
			if ok {
				details["api"] = api.Name
				return details, fmt.Errorf("api defined twice %s", api.Name)
			}
			currentApis[api.Name] = struct{}{}
			if api.ComputeUnits < minCU || api.ComputeUnits > maxCU {
				details["api"] = api.Name
				return details, fmt.Errorf("compute units out or range %s", api.Name)
			}
		}
		currentHeaders := map[string]struct{}{}
		for _, header := range apiCollection.Headers {
			_, ok := currentHeaders[header.Name]
			if ok {
				details["header"] = header.Name
				return details, fmt.Errorf("header defined twice %s", header.Name)
			}
			currentHeaders[header.Name] = struct{}{}
			if strings.ToLower(header.Name) != header.Name {
				details["header"] = header.Name
				return details, fmt.Errorf("header names must be lower case %s", header.Name)
			}
		}
	}

	if spec.DataReliabilityEnabled && spec.Enabled {
		for _, tag := range []FUNCTION_TAG{FUNCTION_TAG_GET_BLOCKNUM, FUNCTION_TAG_GET_BLOCK_BY_NUM} {
			if found := functionTags[tag]; !found {
				return details, fmt.Errorf("missing tagged functions for hash comparison: %s", tag)
			}
		}
	}

	return details, nil
}

func (spec *Spec) CombineCollections(parentsCollections map[CollectionData][]*ApiCollection) error {
	collectionDataList := make([]CollectionData, 0)
	// Populate the keys slice with the map keys
	for key := range parentsCollections {
		collectionDataList = append(collectionDataList, key)
	}
	// sort the slice so the order is deterministic
	sort.Slice(collectionDataList, func(i, j int) bool {
		return collectionDataList[i].String() < collectionDataList[j].String()
	})

	for _, collectionData := range collectionDataList {
		collectionsToCombine := parentsCollections[collectionData]
		if len(collectionsToCombine) == 0 {
			return fmt.Errorf("collection with length 0 %v", collectionData)
		}
		var combined *ApiCollection
		var others []*ApiCollection
		for i := 0; i < len(collectionsToCombine); i++ {
			combined = collectionsToCombine[i]
			others = collectionsToCombine[:i]
			others = append(others, collectionsToCombine[i+1:]...)
			if combined.Enabled {
				break
			}
		}
		if !combined.Enabled {
			// no collections enabled to combine, we skip this
			continue
		}
		err := combined.CombineWithOthers(others, false, false)
		if err != nil {
			return err
		}
		spec.ApiCollections = append(spec.ApiCollections, combined)
	}
	return nil
}

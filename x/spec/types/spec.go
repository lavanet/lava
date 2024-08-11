package types

import (
	fmt "fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/v2/utils"
	commontypes "github.com/lavanet/lava/v2/utils/common/types"
)

const (
	minCU                        = 1
	ContributorPrecision         = 100000 // Can't be 0!
	maxContributorsPercentageStr = "0.8"
	maxParsersPerApi             = 100
)

func (spec Spec) ValidateSpec(maxCU uint64) (map[string]string, error) {
	details := map[string]string{"spec": spec.Name, "status": strconv.FormatBool(spec.Enabled), "chainID": spec.Index}
	functionTagsAll := map[FUNCTION_TAG]bool{}

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

	ok := commontypes.ValidateString(spec.Index, commontypes.INDEX_RESTRICTIONS, nil)
	if !ok {
		return details, fmt.Errorf("spec index can be letters and numbers only %s", spec.Index)
	}

	if len(spec.Identity) > stakingtypes.MaxIdentityLength {
		return details, fmt.Errorf("spec identity should not be longer than %d. Identity: %s", stakingtypes.MaxIdentityLength, spec.Index)
	}

	for _, char := range spec.Name {
		if !unicode.IsLower(char) && char != ' ' {
			return details, fmt.Errorf("spec name must contain lowercase characters only")
		}
	}

	if spec.ReliabilityThreshold == 0 {
		return details, fmt.Errorf("ReliabilityThreshold can't be zero")
	}
	if len(spec.Contributor) > 0 {
		for _, contributorAddr := range spec.Contributor {
			_, err := sdk.AccAddressFromBech32(contributorAddr)
			if err != nil {
				return details, fmt.Errorf("spec contributor is not a valid account address %s in list: %s", contributorAddr, strings.Join(spec.Contributor, ","))
			}
		}
	}

	if spec.ContributorPercentage != nil && (spec.ContributorPercentage.GT(math.LegacyMustNewDecFromStr(maxContributorsPercentageStr)) || (spec.ContributorPercentage.LT(math.LegacyMustNewDecFromStr(strconv.FormatFloat(1.0/ContributorPrecision, 'f', -1, 64))))) {
		return details, fmt.Errorf("spec contributor percentage must be in the range [%s - %s]", math.LegacyMustNewDecFromStr(strconv.FormatFloat(1.0/ContributorPrecision, 'f', -1, 64)).String(), maxContributorsPercentageStr)
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

	if !spec.MinStakeProvider.IsValid() || !spec.MinStakeProvider.IsPositive() {
		return details, fmt.Errorf("MinStakeProvider can't be zero")
	}

	for _, apiCollection := range spec.ApiCollections {
		functionTags := map[FUNCTION_TAG]bool{}
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
			functionTagsAll[parsing.FunctionTag] = true
			if parsing.ResultParsing.Encoding != "" {
				if _, ok := availavleEncodings[parsing.ResultParsing.Encoding]; !ok {
					return details, fmt.Errorf("unsupported api encoding %s in apiCollection %v ", parsing.ResultParsing.Encoding, apiCollection.CollectionData)
				}
			}
			if parsing.FunctionTag == FUNCTION_TAG_GET_BLOCK_BY_NUM {
				if !strings.Contains(parsing.FunctionTemplate, "%") {
					return details, fmt.Errorf("function tag FUNCTION_TAG_GET_BLOCK_BY_NUM does not contain %%d")
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
			if strings.Contains(api.Name, " ") {
				details["api"] = api.Name
				return details, fmt.Errorf("api name includes a space character %s", api.Name)
			}
			parsers := api.GetParsers()
			if len(parsers) > 0 {
				for idx, parser := range parsers {
					if parser.ParsePath == "" || parser.ParseType == PARSER_TYPE_NO_PARSER {
						details["parser_index"] = strconv.Itoa(idx)
						details["api"] = api.Name
						return details, fmt.Errorf("invalid parser in api %s index %d", api.Name, idx)
					}
				}
			}

			if len(parsers) > maxParsersPerApi {
				details["api"] = api.Name
				return details, fmt.Errorf("invalid api %s too many parsers %d", api.Name, len(parsers))
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

		// get the spec's extension names list
		extensionsNames := map[string]struct{}{}
		for _, extension := range apiCollection.Extensions {
			extensionsNames[extension.Name] = struct{}{}
		}
		if len(extensionsNames) > 0 {
			// validate verifications
			for _, verification := range apiCollection.Verifications {
				for _, parseValue := range verification.Values {
					_, found := extensionsNames[parseValue.Extension]
					if parseValue.Extension != "" && !found {
						return details, utils.LavaFormatWarning("verification's extension not found in extension list", fmt.Errorf("spec verification validation failed"),
							utils.LogAttr("verification_extension", parseValue.Extension),
						)
					}
				}
				if verification.ParseDirective.FunctionTag != FUNCTION_TAG_VERIFICATION {
					if !functionTags[verification.ParseDirective.FunctionTag] {
						return details, utils.LavaFormatWarning("verification's function tag not found in the parse directives", fmt.Errorf("spec verification validation failed"),
							utils.LogAttr("verification_tag", verification.ParseDirective.FunctionTag),
						)
					}
				}
			}
		}
	}

	if spec.DataReliabilityEnabled && spec.Enabled {
		for _, tag := range []FUNCTION_TAG{FUNCTION_TAG_GET_BLOCKNUM, FUNCTION_TAG_GET_BLOCK_BY_NUM} {
			if found := functionTagsAll[tag]; !found {
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
	sort.SliceStable(collectionDataList, func(i, j int) bool {
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

func (spec *Spec) ServicesMap() (addons, extensions map[string]struct{}) {
	addons = map[string]struct{}{}
	extensions = map[string]struct{}{}
	for _, apiCollection := range spec.ApiCollections {
		addons[apiCollection.CollectionData.AddOn] = struct{}{}
		for _, extension := range apiCollection.Extensions {
			extensions[extension.Name] = struct{}{}
		}
	}
	return
}

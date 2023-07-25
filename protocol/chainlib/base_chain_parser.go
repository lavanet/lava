package chainlib

import (
	"errors"
	"regexp"
	"strings"
	"sync"

	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type BaseChainParser struct {
	taggedApis     map[spectypes.FUNCTION_TAG]TaggedContainer
	spec           spectypes.Spec
	rwLock         sync.RWMutex
	serverApis     map[ApiKey]ApiContainer
	apiCollections map[CollectionKey]*spectypes.ApiCollection
	headers        map[ApiKey]*spectypes.Header
	verifications  map[RouterKey][]VerificationContainer
}

func (bcp *BaseChainParser) HandleHeaders(metadata []pairingtypes.Metadata, apiCollection *spectypes.ApiCollection, headersDirection spectypes.Header_HeaderType) (filteredHeaders []pairingtypes.Metadata, overwriteRequestedBlock string, ignoredMetadata []pairingtypes.Metadata) {
	bcp.rwLock.RLock()
	defer bcp.rwLock.RUnlock()
	if len(metadata) == 0 {
		return []pairingtypes.Metadata{}, "", []pairingtypes.Metadata{}
	}
	retMeatadata := []pairingtypes.Metadata{}
	for _, header := range metadata {
		headerName := strings.ToLower(header.Name)
		apiKey := ApiKey{Name: headerName, ConnectionType: apiCollection.CollectionData.Type}
		headerDirective, ok := bcp.headers[apiKey]
		if !ok {
			// this header is not handled
			continue
		}
		if headerDirective.Kind == headersDirection || headerDirective.Kind == spectypes.Header_pass_both {
			retMeatadata = append(retMeatadata, header)
			if headerDirective.FunctionTag == spectypes.FUNCTION_TAG_SET_LATEST_IN_METADATA {
				// this header sets the latest requested block
				overwriteRequestedBlock = header.Value
			}
		} else if headerDirective.Kind == spectypes.Header_pass_ignore {
			ignoredMetadata = append(ignoredMetadata, header)
		}
	}
	return retMeatadata, overwriteRequestedBlock, ignoredMetadata
}

func (bcp *BaseChainParser) isAddon(addon string) bool {
	for collectionKey := range bcp.apiCollections {
		if collectionKey.Addon == addon {
			return true
		}
	}
	return false
}

func (bcp *BaseChainParser) separateAddonsExtensions(supported []string) (addons []string, extensions []string) {
	for _, addon := range supported {
		if bcp.isAddon(addon) {
			addons = append(addons, addon)
		} else {
			if addon == "" {
				continue
			}
			extensions = append(extensions, addon)
		}
	}
	return addons, extensions
}

// gets all verifications for an endpoint supporting multiple addons and extensions
func (bcp *BaseChainParser) GetVerifications(supported []string) (retVerifications []VerificationContainer) {
	// addons will contains extensions and addons,
	// extensions must exist in all verifications, addons must be split because they are separated
	addons, extensions := bcp.separateAddonsExtensions(supported)
	addons = append(addons, "") // always add the empty addon
	for _, addon := range addons {
		routerKey := NewRouterKey(append(extensions, addon))
		verifications, ok := bcp.verifications[routerKey]
		if ok {
			retVerifications = append(retVerifications, verifications...)
		}
	}
	return
}

func (bcp *BaseChainParser) Construct(spec spectypes.Spec, taggedApis map[spectypes.FUNCTION_TAG]TaggedContainer, serverApis map[ApiKey]ApiContainer, apiCollections map[CollectionKey]*spectypes.ApiCollection, headers map[ApiKey]*spectypes.Header, verifications map[RouterKey][]VerificationContainer) {
	bcp.spec = spec
	bcp.serverApis = serverApis
	bcp.taggedApis = taggedApis
	bcp.headers = headers
	bcp.apiCollections = apiCollections
	bcp.verifications = verifications
}

func (bcp *BaseChainParser) GetParsingByTag(tag spectypes.FUNCTION_TAG) (parsing *spectypes.ParseDirective, collectionData *spectypes.CollectionData, existed bool) {
	bcp.rwLock.RLock()
	defer bcp.rwLock.RUnlock()

	val, ok := bcp.taggedApis[tag]
	if !ok {
		return nil, nil, false
	}
	return val.Parsing, &val.ApiCollection.CollectionData, ok
}

// getSupportedApi fetches service api from spec by name
func (apip *BaseChainParser) getSupportedApi(name string, connectionType string) (*ApiContainer, error) {
	// Guard that the GrpcChainParser instance exists
	if apip == nil {
		return nil, errors.New("ChainParser not defined")
	}

	// Acquire read lock
	apip.rwLock.RLock()
	defer apip.rwLock.RUnlock()

	// Fetch server api by name
	apiCont, ok := apip.serverApis[ApiKey{
		Name:           name,
		ConnectionType: connectionType,
	}]

	// Return an error if spec does not exist
	if !ok {
		return nil, utils.LavaFormatError("api not supported", nil, utils.Attribute{Key: "name", Value: name}, utils.Attribute{Key: "connectionType", Value: connectionType})
	}

	// Return an error if api is disabled
	if !apiCont.api.Enabled {
		return nil, utils.LavaFormatError("api is disabled", nil, utils.Attribute{Key: "name", Value: name}, utils.Attribute{Key: "connectionType", Value: connectionType})
	}

	return &apiCont, nil
}

// getSupportedApi fetches service api from spec by name
func (apip *BaseChainParser) getApiCollection(connectionType string, internalPath string, addon string) (*spectypes.ApiCollection, error) {
	// Guard that the GrpcChainParser instance exists
	if apip == nil {
		return nil, errors.New("ChainParser not defined")
	}

	// Acquire read lock
	apip.rwLock.RLock()
	defer apip.rwLock.RUnlock()

	// Fetch server api by name
	api, ok := apip.apiCollections[CollectionKey{
		ConnectionType: connectionType,
		InternalPath:   internalPath,
		Addon:          addon,
	}]

	// Return an error if spec does not exist
	if !ok {
		return nil, utils.LavaFormatError("api not supported", nil, utils.Attribute{Key: "connectionType", Value: connectionType})
	}

	// Return an error if api is disabled
	if !api.Enabled {
		return nil, utils.LavaFormatError("api is disabled", nil, utils.Attribute{Key: "connectionType", Value: connectionType})
	}

	return api, nil
}

func getServiceApis(spec spectypes.Spec, rpcInterface string) (retServerApis map[ApiKey]ApiContainer, retTaggedApis map[spectypes.FUNCTION_TAG]TaggedContainer, retApiCollections map[CollectionKey]*spectypes.ApiCollection, retHeaders map[ApiKey]*spectypes.Header, retVerifications map[RouterKey][]VerificationContainer) {
	serverApis := map[ApiKey]ApiContainer{}
	taggedApis := map[spectypes.FUNCTION_TAG]TaggedContainer{}
	headers := map[ApiKey]*spectypes.Header{}
	apiCollections := map[CollectionKey]*spectypes.ApiCollection{}
	verifications := map[RouterKey][]VerificationContainer{}
	if spec.Enabled {
		for _, apiCollection := range spec.ApiCollections {
			if !apiCollection.Enabled {
				continue
			}
			if apiCollection.CollectionData.ApiInterface != rpcInterface {
				continue
			}
			collectionKey := CollectionKey{
				ConnectionType: apiCollection.CollectionData.Type,
				InternalPath:   apiCollection.CollectionData.InternalPath,
				Addon:          apiCollection.CollectionData.AddOn,
			}
			for _, parsing := range apiCollection.ParseDirectives {
				taggedApis[parsing.FunctionTag] = TaggedContainer{
					Parsing:       parsing,
					ApiCollection: apiCollection,
				}
			}

			for _, api := range apiCollection.Apis {
				if !api.Enabled {
					continue
				}
				//
				// TODO: find a better spot for this (more optimized, precompile regex, etc)
				if rpcInterface == spectypes.APIInterfaceRest {
					re := regexp.MustCompile(`{[^}]+}`)
					processedName := string(re.ReplaceAll([]byte(api.Name), []byte("replace-me-with-regex")))
					processedName = regexp.QuoteMeta(processedName)
					processedName = strings.ReplaceAll(processedName, "replace-me-with-regex", `[^\/\s]+`)
					serverApis[ApiKey{
						Name:           processedName,
						ConnectionType: collectionKey.ConnectionType,
					}] = ApiContainer{
						api:           api,
						collectionKey: collectionKey,
					}
				} else {
					serverApis[ApiKey{
						Name:           api.Name,
						ConnectionType: collectionKey.ConnectionType,
					}] = ApiContainer{
						api:           api,
						collectionKey: collectionKey,
					}
				}
			}
			for _, header := range apiCollection.Headers {
				headers[ApiKey{
					Name:           header.Name,
					ConnectionType: collectionKey.ConnectionType,
				}] = header
			}
			for _, verification := range apiCollection.Verifications {
				for _, parseValue := range verification.Values {

					extensions := strings.Split(parseValue.Extension, ",")
					// we are appending addons and extensions because chainRouter is routing using them
					routerKey := NewRouterKey(append([]string{apiCollection.CollectionData.AddOn}, extensions...))
					verCont := VerificationContainer{
						ConnectionType: apiCollection.CollectionData.Type,
						Name:           verification.Name,
						ParseDirective: *verification.ParseDirective,
						Value:          parseValue.ExpectedValue,
						Routing:        routerKey,
					}
					extensionVerifications, ok := verifications[routerKey]
					if !ok {
						extensionVerifications = []VerificationContainer{verCont}
					} else {
						extensionVerifications = append(extensionVerifications, verCont)
					}
					verifications[routerKey] = extensionVerifications
				}
			}
			apiCollections[collectionKey] = apiCollection
		}
	}
	return serverApis, taggedApis, apiCollections, headers, verifications
}

// matchSpecApiByName returns service api which match given name
func matchSpecApiByName(name string, connectionType string, serverApis map[ApiKey]ApiContainer) (*ApiContainer, bool) {
	// TODO: make it faster and better by not doing a regex instead using a better algorithm
	for apiName, api := range serverApis {
		re, err := regexp.Compile("^" + apiName.Name + "$")
		if err != nil {
			utils.LavaFormatError("regex Compile api", err, utils.Attribute{Key: "apiName", Value: apiName})
			continue
		}
		if re.MatchString(name) && apiName.ConnectionType == connectionType {
			return &api, true
		}
	}
	return nil, false
}

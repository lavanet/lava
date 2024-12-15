package chainlib

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/lavanet/lava/v4/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v4/protocol/common"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/utils/lavaslices"
	"github.com/lavanet/lava/v4/utils/maps"
	epochstorage "github.com/lavanet/lava/v4/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
)

var AllowMissingApisByDefault = true

type PolicyInf interface {
	GetSupportedAddons(specID string) (addons []string, err error)
	GetSupportedExtensions(specID string) (extensions []epochstorage.EndpointService, err error)
}

type InternalPath struct {
	Path           string
	Enabled        bool
	ApiInterface   string
	ConnectionType string
	Addon          string
}

type BaseChainParser struct {
	internalPaths   map[string]InternalPath
	taggedApis      map[spectypes.FUNCTION_TAG]TaggedContainer
	spec            spectypes.Spec
	rwLock          sync.RWMutex
	serverApis      map[ApiKey]ApiContainer
	apiCollections  map[CollectionKey]*spectypes.ApiCollection
	headers         map[ApiKey]*spectypes.Header
	verifications   map[VerificationKey]map[string][]VerificationContainer // map[VerificationKey]map[InternalPath][]VerificationContainer
	allowedAddons   map[string]bool
	extensionParser extensionslib.ExtensionParser
	active          bool
}

func (bcp *BaseChainParser) Activate() {
	bcp.active = true
}

func (bcp *BaseChainParser) Active() bool {
	return bcp.active
}

func (bcp *BaseChainParser) UpdateBlockTime(newBlockTime time.Duration) {
	bcp.rwLock.Lock()
	defer bcp.rwLock.Unlock()
	utils.LavaFormatInfo("chainParser updated block time", utils.Attribute{Key: "newTime", Value: newBlockTime}, utils.Attribute{Key: "oldTime", Value: time.Duration(bcp.spec.AverageBlockTime) * time.Millisecond}, utils.Attribute{Key: "specID", Value: bcp.spec.Index})
	bcp.spec.AverageBlockTime = newBlockTime.Milliseconds()
}

func (bcp *BaseChainParser) HandleHeaders(metadata []pairingtypes.Metadata, apiCollection *spectypes.ApiCollection, headersDirection spectypes.Header_HeaderType) (filteredHeaders []pairingtypes.Metadata, overwriteRequestedBlock string, ignoredMetadata []pairingtypes.Metadata) {
	bcp.rwLock.RLock()
	defer bcp.rwLock.RUnlock()
	if len(metadata) == 0 {
		return []pairingtypes.Metadata{}, "", []pairingtypes.Metadata{}
	}
	retMetadata := []pairingtypes.Metadata{}
	for _, header := range metadata {
		headerName := strings.ToLower(header.Name)
		apiKey := ApiKey{Name: headerName, ConnectionType: apiCollection.CollectionData.Type}
		headerDirective, ok := bcp.headers[apiKey]
		if !ok {
			// this header is not handled
			continue
		}
		if headerDirective.Kind == headersDirection || headerDirective.Kind == spectypes.Header_pass_both {
			retMetadata = append(retMetadata, header)
			if headerDirective.FunctionTag == spectypes.FUNCTION_TAG_SET_LATEST_IN_METADATA {
				// this header sets the latest requested block
				overwriteRequestedBlock = header.Value
			}
		} else if headerDirective.Kind == spectypes.Header_pass_ignore {
			ignoredMetadata = append(ignoredMetadata, header)
		}
	}
	return retMetadata, overwriteRequestedBlock, ignoredMetadata
}

func (bcp *BaseChainParser) isAddon(addon string) bool {
	_, ok := bcp.allowedAddons[addon]
	return ok
}

func (bcp *BaseChainParser) isExtension(extension string) bool {
	return bcp.extensionParser.AllowedExtension(extension)
}

// use while bcp locked.
func (bcp *BaseChainParser) validateAddons(nodeMessage *baseChainMessageContainer) error {
	var addon string
	if addon = GetAddon(nodeMessage); addon != "" { // check we have an addon
		if allowed := bcp.allowedAddons[addon]; !allowed { // check addon is allowed
			return utils.LavaFormatError("consumer policy does not allow addon", nil,
				utils.LogAttr("addon", addon),
			)
		}
	}
	// no addons to validate or validation completed successfully
	return nil
}

func (bcp *BaseChainParser) Validate(nodeMessage *baseChainMessageContainer) error {
	bcp.rwLock.RLock()
	defer bcp.rwLock.RUnlock()
	err := bcp.validateAddons(nodeMessage)
	// add more validations in the future here.
	return err
}

func (bcp *BaseChainParser) BuildMapFromPolicyQuery(policy PolicyInf, chainId string, apiInterface string) (map[string]struct{}, error) {
	addons, err := policy.GetSupportedAddons(chainId)
	if err != nil {
		return nil, err
	}
	extensions, err := policy.GetSupportedExtensions(chainId)
	if err != nil {
		return nil, err
	}
	services := make(map[string]struct{})
	for _, addon := range addons {
		services[addon] = struct{}{}
	}
	for _, consumerExtension := range extensions {
		// store only relevant apiInterface extensions
		if consumerExtension.ApiInterface == apiInterface {
			services[consumerExtension.Extension] = struct{}{}
		}
	}
	return services, nil
}

func (bcp *BaseChainParser) SetPolicyFromAddonAndExtensionMap(policyInformation map[string]struct{}) {
	bcp.rwLock.Lock()
	defer bcp.rwLock.Unlock()
	// reset the current one in case we configured it previously
	configuredExtensions := make(map[extensionslib.ExtensionKey]*spectypes.Extension)
	for collectionKey, apiCollection := range bcp.apiCollections {
		// manage extensions
		for _, extension := range apiCollection.Extensions {
			if extension.Name == "" {
				// skip empty extensions
				continue
			}
			if _, ok := policyInformation[extension.Name]; ok {
				extensionKey := extensionslib.ExtensionKey{
					Extension:      extension.Name,
					ConnectionType: collectionKey.ConnectionType,
					InternalPath:   collectionKey.InternalPath,
					Addon:          collectionKey.Addon,
				}
				configuredExtensions[extensionKey] = extension
			}
		}
	}
	bcp.extensionParser.SetConfiguredExtensions(configuredExtensions)
	// manage allowed addons
	for addon := range bcp.allowedAddons {
		_, bcp.allowedAddons[addon] = policyInformation[addon]
	}
}

// policy information contains all configured services (extensions and addons) allowed to be used by the consumer
func (bcp *BaseChainParser) SetPolicy(policy PolicyInf, chainId string, apiInterface string) error {
	policyInformation, err := bcp.BuildMapFromPolicyQuery(policy, chainId, apiInterface)
	if err != nil {
		return err
	}
	bcp.SetPolicyFromAddonAndExtensionMap(policyInformation)
	return nil
}

// this function errors if it meets a value that is neither a n addon or an extension
func (bcp *BaseChainParser) SeparateAddonsExtensions(supported []string) (addons, extensions []string, err error) {
	checked := map[string]struct{}{}
	for _, supportedToCheck := range supported {
		// ignore repeated occurrences
		if _, ok := checked[supportedToCheck]; ok {
			continue
		}
		checked[supportedToCheck] = struct{}{}

		if bcp.isAddon(supportedToCheck) {
			addons = append(addons, supportedToCheck)
		} else {
			if supportedToCheck == "" {
				continue
			}
			if bcp.isExtension(supportedToCheck) || supportedToCheck == WebSocketExtension {
				extensions = append(extensions, supportedToCheck)
				continue
			}
			// neither is an error
			err = utils.LavaFormatError("invalid supported to check, is neither an addon or an extension", err,
				utils.Attribute{Key: "spec", Value: bcp.spec.Index},
				utils.Attribute{Key: "supported", Value: supportedToCheck})
		}
	}
	return addons, extensions, err
}

// gets all verifications for an endpoint supporting multiple addons and extensions
func (bcp *BaseChainParser) GetVerifications(supported []string, internalPath string, apiInterface string) (retVerifications []VerificationContainer, err error) {
	// addons will contains extensions and addons,
	// extensions must exist in all verifications, addons must be split because they are separated
	addons, extensions, err := bcp.SeparateAddonsExtensions(supported)
	if err != nil {
		return nil, err
	}
	if len(extensions) == 0 {
		extensions = []string{""}
	}
	addons = append(addons, "") // always add the empty addon

	for _, addon := range addons {
		for _, extension := range extensions {
			verificationKey := VerificationKey{
				Extension: extension,
				Addon:     addon,
			}
			collectionVerifications, ok := bcp.verifications[verificationKey]
			if ok {
				if verifications, ok := collectionVerifications[internalPath]; ok {
					retVerifications = append(retVerifications, verifications...)
				}
			}
		}
	}
	return retVerifications, nil
}

func (bcp *BaseChainParser) Construct(spec spectypes.Spec, internalPaths map[string]InternalPath, taggedApis map[spectypes.FUNCTION_TAG]TaggedContainer,
	serverApis map[ApiKey]ApiContainer, apiCollections map[CollectionKey]*spectypes.ApiCollection, headers map[ApiKey]*spectypes.Header,
	verifications map[VerificationKey]map[string][]VerificationContainer,
) {
	bcp.spec = spec
	bcp.internalPaths = internalPaths
	bcp.serverApis = serverApis
	bcp.taggedApis = taggedApis
	bcp.headers = headers
	bcp.apiCollections = apiCollections
	bcp.verifications = verifications
	allowedAddons := map[string]bool{}
	allowedExtensions := map[string]struct{}{}
	for _, apiCollection := range apiCollections {
		for _, extension := range apiCollection.Extensions {
			allowedExtensions[extension.Name] = struct{}{}
		}
		// if addon was already existing (happens on spec update), use the existing policy, otherwise set it to false by default
		allowedAddons[apiCollection.CollectionData.AddOn] = bcp.allowedAddons[apiCollection.CollectionData.AddOn]
	}
	bcp.allowedAddons = allowedAddons

	bcp.extensionParser = extensionslib.NewExtensionParser(allowedExtensions, bcp.extensionParser.GetConfiguredExtensions())
}

func (bcp *BaseChainParser) GetParsingByTag(tag spectypes.FUNCTION_TAG) (parsing *spectypes.ParseDirective, apiCollection *spectypes.ApiCollection, existed bool) {
	bcp.rwLock.RLock()
	defer bcp.rwLock.RUnlock()

	val, ok := bcp.taggedApis[tag]
	if !ok {
		return nil, nil, false
	}
	return val.Parsing, val.ApiCollection, ok
}

func (bcp *BaseChainParser) IsTagInCollection(tag spectypes.FUNCTION_TAG, collectionKey CollectionKey) bool {
	bcp.rwLock.RLock()
	defer bcp.rwLock.RUnlock()

	apiCollection, ok := bcp.apiCollections[collectionKey]
	return ok && lavaslices.ContainsPredicate(apiCollection.ParseDirectives, func(elem *spectypes.ParseDirective) bool {
		return elem.FunctionTag == tag
	})
}

func (bcp *BaseChainParser) GetAllInternalPaths() []string {
	bcp.rwLock.RLock()
	defer bcp.rwLock.RUnlock()
	return lavaslices.Map(maps.ValuesSlice(bcp.internalPaths), func(internalPath InternalPath) string {
		return internalPath.Path
	})
}

func (bcp *BaseChainParser) IsInternalPathEnabled(internalPath string, apiInterface string, addon string) bool {
	bcp.rwLock.RLock()
	defer bcp.rwLock.RUnlock()
	internalPathObj, ok := bcp.internalPaths[internalPath]
	return ok && internalPathObj.Enabled && internalPathObj.ApiInterface == apiInterface && internalPathObj.Addon == addon
}

func (bcp *BaseChainParser) ExtensionParsing(addon string, parsedMessageArg *baseChainMessageContainer, extensionInfo extensionslib.ExtensionInfo) {
	if extensionInfo.ExtensionOverride == nil {
		// consumer side extension parsing. to set the extension based on the latest block and the request
		bcp.extensionParsingInner(addon, parsedMessageArg, extensionInfo.LatestBlock)
	} else {
		// this is used for provider parsing. as the provider needs to set the requested extension by the request.
		parsedMessageArg.OverrideExtensions(extensionInfo.ExtensionOverride, &bcp.extensionParser)
	}
	// in case we want to force extensions we can add additional extensions. this is used on consumer side with flags.
	if extensionInfo.AdditionalExtensions != nil {
		parsedMessageArg.OverrideExtensions(extensionInfo.AdditionalExtensions, &bcp.extensionParser)
	}
}

func (bcp *BaseChainParser) extensionParsingInner(addon string, parsedMessageArg *baseChainMessageContainer, latestBlock uint64) {
	bcp.rwLock.RLock()
	defer bcp.rwLock.RUnlock()
	bcp.extensionParser.ExtensionParsing(addon, parsedMessageArg, latestBlock)
}

func (apip *BaseChainParser) defaultApiContainer(apiKey ApiKey) (*ApiContainer, error) {
	// Guard that the GrpcChainParser instance exists
	if apip == nil {
		return nil, errors.New("ChainParser not defined")
	}
	utils.LavaFormatDebug("api not supported", utils.Attribute{Key: "apiKey", Value: apiKey})
	apiCont := &ApiContainer{
		api: &spectypes.Api{
			Enabled:           true,
			Name:              "Default-" + apiKey.Name,
			ComputeUnits:      20, // set 20 compute units by default
			ExtraComputeUnits: 0,
			Category:          spectypes.SpecCategory{},
			BlockParsing: spectypes.BlockParser{
				ParserFunc: spectypes.PARSER_FUNC_EMPTY,
			},
			TimeoutMs: 0,
			Parsers:   []spectypes.GenericParser{},
		},
		collectionKey: CollectionKey{
			ConnectionType: apiKey.ConnectionType,
			InternalPath:   apiKey.InternalPath,
			Addon:          "",
		},
	}

	return apiCont, nil
}

// getSupportedApi fetches service api from spec by name
func (apip *BaseChainParser) getSupportedApi(apiKey ApiKey) (*ApiContainer, error) {
	// Guard that the GrpcChainParser instance exists
	if apip == nil {
		return nil, errors.New("ChainParser not defined")
	}

	// Acquire read lock
	apip.rwLock.RLock()
	defer apip.rwLock.RUnlock()

	// Fetch server api by name
	apiCont, ok := apip.serverApis[apiKey]

	// Return an error if spec does not exist
	if !ok {
		if AllowMissingApisByDefault {
			return apip.defaultApiContainer(apiKey)
		}
		return nil, common.APINotSupportedError
	}

	// Return an error if api is disabled
	if !apiCont.api.Enabled {
		return nil, utils.LavaFormatInfo("api is disabled", utils.Attribute{Key: "apiKey", Value: apiKey})
	}

	return &apiCont, nil
}

func (apip *BaseChainParser) isValidInternalPath(path string) bool {
	if apip == nil || len(apip.internalPaths) == 0 {
		return false
	}
	_, ok := apip.internalPaths[path]
	return ok
}

// take an http request and direct it through the consumer
func (apip *BaseChainParser) ExtractDataFromRequest(request *http.Request) (url string, data string, connectionType string, metadata []pairingtypes.Metadata, err error) {
	// Extract relative URL path
	url = request.URL.Path
	// Extract connection type
	connectionType = request.Method

	// Extract metadata
	for key, values := range request.Header {
		for _, value := range values {
			metadata = append(metadata, pairingtypes.Metadata{
				Name:  key,
				Value: value,
			})
		}
	}

	// Extract data
	if request.Body != nil {
		bodyBytes, err := io.ReadAll(request.Body)
		if err != nil {
			return "", "", "", nil, err
		}
		data = string(bodyBytes)
	}

	return url, data, connectionType, metadata, nil
}

func (apip *BaseChainParser) SetResponseFromRelayResult(relayResult *common.RelayResult) (*http.Response, error) {
	if relayResult == nil {
		return nil, errors.New("relayResult is nil")
	}
	response := &http.Response{
		StatusCode: relayResult.StatusCode,
		Header:     make(http.Header),
	}

	for _, values := range relayResult.Reply.Metadata {
		response.Header.Add(values.Name, values.Value)
	}

	if relayResult.Reply != nil && relayResult.Reply.Data != nil {
		response.Body = io.NopCloser(strings.NewReader(string(relayResult.Reply.Data)))
	}

	return response, nil
}

// getSupportedApi fetches service api from spec by name
func (apip *BaseChainParser) getApiCollection(connectionType, internalPath, addon string) (*spectypes.ApiCollection, error) {
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
		utils.LavaFormatDebug("api not supported", utils.Attribute{Key: "connectionType", Value: connectionType})
		return nil, common.APINotSupportedError
	}

	// Return an error if api is disabled
	if !api.Enabled {
		return nil, utils.LavaFormatError("api is disabled", nil, utils.Attribute{Key: "connectionType", Value: connectionType})
	}

	return api, nil
}

func getServiceApis(
	spec spectypes.Spec,
	rpcInterface string,
) (
	retInternalPaths map[string]InternalPath,
	retServerApis map[ApiKey]ApiContainer,
	retTaggedApis map[spectypes.FUNCTION_TAG]TaggedContainer,
	retApiCollections map[CollectionKey]*spectypes.ApiCollection,
	retHeaders map[ApiKey]*spectypes.Header,
	retVerifications map[VerificationKey]map[string][]VerificationContainer,
) {
	retInternalPaths = map[string]InternalPath{}
	serverApis := map[ApiKey]ApiContainer{}
	taggedApis := map[spectypes.FUNCTION_TAG]TaggedContainer{}
	headers := map[ApiKey]*spectypes.Header{}
	apiCollections := map[CollectionKey]*spectypes.ApiCollection{}
	verifications := map[VerificationKey]map[string][]VerificationContainer{}
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

			// add as a valid internal path
			retInternalPaths[apiCollection.CollectionData.InternalPath] = InternalPath{
				Path:           apiCollection.CollectionData.InternalPath,
				Enabled:        apiCollection.Enabled,
				ApiInterface:   apiCollection.CollectionData.ApiInterface,
				ConnectionType: apiCollection.CollectionData.Type,
				Addon:          apiCollection.CollectionData.AddOn,
			}

			for _, parsing := range apiCollection.ParseDirectives {
				// We do this because some specs may have multiple parse directives
				// with the same tag - SUBSCRIBE (like in Solana).
				//
				// Since the function tag is not used for handling the subscription flow,
				// we can ignore the extra parse directives and take only the first one. The
				// subscription flow is handled by the consumer websocket manager and the chain router
				// that uses the api collection to fetch the correct parse directive.
				//
				// The only place the SUBSCRIBE tag is checked against the taggedApis map is in the chain parser with GetParsingByTag.
				// But there, we're not interested in the parse directive, only if the tag is present.
				if _, ok := taggedApis[parsing.FunctionTag]; !ok {
					taggedApis[parsing.FunctionTag] = TaggedContainer{
						Parsing:       parsing,
						ApiCollection: apiCollection,
					}
				}
			}

			for _, api := range apiCollection.Apis {
				if !api.Enabled {
					continue
				}

				// TODO: find a better spot for this (more optimized, precompile regex, etc)
				if rpcInterface == spectypes.APIInterfaceRest {
					re := regexp.MustCompile(`{[^}]+}`)
					processedName := string(re.ReplaceAll([]byte(api.Name), []byte("replace-me-with-regex")))
					processedName = regexp.QuoteMeta(processedName)
					processedName = strings.ReplaceAll(processedName, "replace-me-with-regex/", `[^\/\s]+/`)
					processedName = strings.ReplaceAll(processedName, "replace-me-with-regex", `[^\/\s]*`)
					serverApis[ApiKey{
						Name:           processedName,
						ConnectionType: collectionKey.ConnectionType,
					}] = ApiContainer{
						api:           api,
						collectionKey: collectionKey,
					}
				} else {
					// add another internal path entry so it can specifically be referenced
					if apiCollection.CollectionData.InternalPath != "" {
						serverApis[ApiKey{
							Name:           api.Name,
							ConnectionType: collectionKey.ConnectionType,
							InternalPath:   apiCollection.CollectionData.InternalPath,
						}] = ApiContainer{
							api:           api,
							collectionKey: collectionKey,
						}
						// if it does not exist set it
						if _, ok := serverApis[ApiKey{Name: api.Name, ConnectionType: collectionKey.ConnectionType}]; !ok {
							serverApis[ApiKey{
								Name:           api.Name,
								ConnectionType: collectionKey.ConnectionType,
							}] = ApiContainer{
								api:           api,
								collectionKey: collectionKey,
							}
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
			}
			for _, header := range apiCollection.Headers {
				headers[ApiKey{
					Name:           header.Name,
					ConnectionType: collectionKey.ConnectionType,
				}] = header
			}
			for _, verification := range apiCollection.Verifications {
				if verification.ParseDirective.FunctionTag != spectypes.FUNCTION_TAG_VERIFICATION {
					if _, ok := taggedApis[verification.ParseDirective.FunctionTag]; ok {
						verification.ParseDirective = taggedApis[verification.ParseDirective.FunctionTag].Parsing
					} else {
						utils.LavaFormatError("Bad verification definition", fmt.Errorf("verification function tag is not defined in the collections parse directives"), utils.LogAttr("function_tag", verification.ParseDirective.FunctionTag))
						continue
					}
				}

				for _, parseValue := range verification.Values {
					verificationKey := VerificationKey{
						Extension: parseValue.Extension,
						Addon:     apiCollection.CollectionData.AddOn,
					}

					verCont := VerificationContainer{
						InternalPath:    apiCollection.CollectionData.InternalPath,
						ConnectionType:  apiCollection.CollectionData.Type,
						Name:            verification.Name,
						ParseDirective:  *verification.ParseDirective,
						Value:           parseValue.ExpectedValue,
						LatestDistance:  parseValue.LatestDistance,
						VerificationKey: verificationKey,
						Severity:        parseValue.Severity,
					}

					internalPath := apiCollection.CollectionData.InternalPath
					if extensionVerifications, ok := verifications[verificationKey]; !ok {
						verifications[verificationKey] = map[string][]VerificationContainer{internalPath: {verCont}}
					} else if collectionVerifications, ok := extensionVerifications[internalPath]; !ok {
						verifications[verificationKey][internalPath] = []VerificationContainer{verCont}
					} else {
						verifications[verificationKey][internalPath] = append(collectionVerifications, verCont)
					}
				}
			}
			apiCollections[collectionKey] = apiCollection
		}
	}
	return retInternalPaths, serverApis, taggedApis, apiCollections, headers, verifications
}

func (bcp *BaseChainParser) ExtensionsParser() *extensionslib.ExtensionParser {
	return &bcp.extensionParser
}

// matchSpecApiByName returns service api which match given name
func matchSpecApiByName(name, connectionType string, serverApis map[ApiKey]ApiContainer) (*ApiContainer, bool) {
	// TODO: make it faster and better by not doing a regex instead using a better algorithm
	foundNameOnDifferentConnectionType := ""
	for apiName, api := range serverApis {
		re, err := regexp.Compile("^" + apiName.Name + "$")
		if err != nil {
			utils.LavaFormatError("regex Compile api", err, utils.Attribute{Key: "apiName", Value: apiName})
			continue
		}
		if re.MatchString(name) {
			if apiName.ConnectionType == connectionType {
				return &api, true
			} else {
				foundNameOnDifferentConnectionType = apiName.ConnectionType
			}
		}
	}
	if foundNameOnDifferentConnectionType != "" { // its hard to notice when we have an API on only one connection type.
		utils.LavaFormatWarning("API was found on a different connection type", nil,
			utils.Attribute{Key: "connection_type_found", Value: foundNameOnDifferentConnectionType},
			utils.Attribute{Key: "connection_type_requested", Value: connectionType},
			utils.LogAttr("requested_api", name),
		)
	}
	return nil, false
}

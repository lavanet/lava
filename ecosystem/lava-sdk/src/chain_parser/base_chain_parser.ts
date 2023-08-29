import {} from '../grpc_web_services/lavanet/lava/'

export class TaggedContainer {
	private Parsing: spectypes.ParseDirective
	private ApiCollection: spectypes.ApiCollection
}

export class BaseChainParser {
    private taggedApis      map[spectypes.FUNCTION_TAG]TaggedContainer
	private spec            spectypes.Spec
	private rwLock          sync.RWMutex
	private serverApis      map[ApiKey]ApiContainer
	private apiCollections  map[CollectionKey]*spectypes.ApiCollection
	private headers         map[ApiKey]*spectypes.Header
	private verifications   map[VerificationKey][]VerificationContainer
	private allowedAddons   map[string]struct{}
	private extensionParser extensionslib.ExtensionParser
}
declare class SDKErrors {
    static errAccountNotInitialized: Error;
    static errRelayerServiceNotInitialized: Error;
    static errLavaProvidersNotInitialized: Error;
    static errSessionNotInitialized: Error;
    static errMethodNotSupported: Error;
    static errChainIDUnsupported: Error;
    static errNetworkUnsupported: Error;
    static errRPCRelayMethodNotSupported: Error;
    static errPrivKeyAndBadgeNotInitialized: Error;
    static errPrivKeyAndBadgeBothInitialized: Error;
    static errRestRelayMethodNotSupported: Error;
}
export default SDKErrors;

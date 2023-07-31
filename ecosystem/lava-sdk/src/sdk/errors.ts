class SDKErrors {
  static errAccountNotInitialized: Error = new Error(
    "Account was not initialized"
  );
  static errRelayerServiceNotInitialized: Error = new Error(
    "Relayer service was not initialized"
  );
  static errLavaProvidersNotInitialized: Error = new Error(
    "Lava providers was not initialized"
  );
  static errSessionNotInitialized: Error = new Error(
    "Session was not initialized"
  );
  static errMethodNotSupported: Error = new Error("Method not supported");
  static errChainIDUnsupported: Error = new Error(
    "Invalid or unsupported chainID"
  );
  static errNetworkUnsupported: Error = new Error(
    "Invalid or unsupported network"
  );
  static errRPCRelayMethodNotSupported: Error = new Error(
    "SendRelay not supported if the SDK is initialized with rest rpcInterface, use sendRestRelay method"
  );
  static errPrivKeyAndBadgeNotInitialized: Error = new Error(
    "Consumer private key or badge was not initialized"
  );
  static errPrivKeyAndBadgeBothInitialized: Error = new Error(
    "Consumer private key and badge was both initialized"
  );
  static errRestRelayMethodNotSupported: Error = new Error(
    "SendRestRelay not supported if the SDK is initialized with RPC rpcInterface (tendermintRPC/jsonRPC), use sendRelay method"
  );
}

export default SDKErrors;

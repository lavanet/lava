class ProvidersErrors {
  static errLavaProvidersNotInitialized: Error = new Error(
    "Lava providers not initialized"
  );
  static errRelayerServiceNotInitialized: Error = new Error(
    "Relayer service was not initialized"
  );
  static errNoValidProvidersForCurrentEpoch: Error = new Error(
    "No valid providers for current epoch"
  );
  static errSpecNotFound: Error = new Error("Spec not found");
  static errApiNotFound: Error = new Error("API not found");
  static errMaxCuNotFound: Error = new Error("MaxCU not found");
  static errProvidersNotFound: Error = new Error("Providers not found");
  static errNoProviders: Error = new Error("No providers found");
  static errNoRelayer: Error = new Error("No relayer found");
  static errConfigNotValidJson: Error = new Error(
    "Pairing list config not valid json file"
  );
}

export default ProvidersErrors;

export class StateTrackerErrors {
  static errTimeTillNextEpochMissing: Error = new Error(
    "Time till next epoch is missing"
  );
}

export class ProvidersErrors {
  static errLavaProvidersNotInitialized: Error = new Error(
    "Lava providers not initialized"
  );
  static errProvidersNotFound: Error = new Error("Providers not found");
  static errProbeResponseUndefined: Error = new Error(
    "Probe response undefined"
  );
}

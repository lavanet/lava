"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
class StateTrackerErrors {
}
StateTrackerErrors.errLavaProvidersNotInitialized = new Error("Lava providers not initialized");
StateTrackerErrors.errRelayerServiceNotInitialized = new Error("Relayer service was not initialized");
StateTrackerErrors.errNoValidProvidersForCurrentEpoch = new Error("No valid providers for current epoch");
StateTrackerErrors.errSpecNotFound = new Error("Spec not found");
exports.default = StateTrackerErrors;

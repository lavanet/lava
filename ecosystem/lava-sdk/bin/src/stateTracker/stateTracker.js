"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.StateTracker = void 0;
const pairing_updater_1 = require("./pairing_updater");
const state_query_1 = require("./state_query");
class StateTracker {
    // Constructor for State Tracker
    constructor(badgeManager, pairingListConfig, relayer, chainIDRpcInterface, config, consumerSessionManagerMap // TODO add consumerSessionManagerMap type
    ) {
        // All variables in state tracker
        this.updaters = [];
        // Initialize State Query
        this.stateQuery = new state_query_1.StateQuery(badgeManager, pairingListConfig, chainIDRpcInterface, relayer, config);
        // Create Pairing Updater
        const pairingUpdater = new pairing_updater_1.PairingUpdater(this.stateQuery, consumerSessionManagerMap);
        // Register all updaters
        this.registerForUpdates(pairingUpdater);
        // Call executeUpdateOnNewEpoch method to start the process
        this.executeUpdateOnNewEpoch();
    }
    // registerForUpdates adds new updater
    registerForUpdates(updater) {
        this.updaters.push(updater);
    }
    // executeUpdateOnNewEpoch executes all updates on every new epoch
    executeUpdateOnNewEpoch() {
        return __awaiter(this, void 0, void 0, function* () {
            try {
                // Fetching all the info including the time_till_next_epoch
                const timeTillNextEpoch = yield this.stateQuery.fetchPairing();
                // Call update method on all registered updaters
                for (const updater of this.updaters) {
                    updater.update();
                }
                // Set up a timer to call this method again when the next epoch begins
                setTimeout(() => this.executeUpdateOnNewEpoch(), timeTillNextEpoch);
            }
            catch (error) {
                console.error("An error occurred during pairing processing:", error);
                // TODO fix DEFAULT_RETRY_INTERVAL
                const DEFAULT_RETRY_INTERVAL = 10000;
                setTimeout(() => this.executeUpdateOnNewEpoch(), DEFAULT_RETRY_INTERVAL);
            }
        });
    }
}
exports.StateTracker = StateTracker;

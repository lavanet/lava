import { PairingUpdater } from "./pairing_updater";
import { StateQuery } from "./state_query";
import { BadgeManager } from "../badge/fetchBadge";

// TODO we can make relayer not default
import Relayer from "../relayer/relayer";

// Updater interface
interface Updater {
  update(): void;
}

// ChainIDRpcInterface interface
export interface ChainIDRpcInterface {
  chainID: string;
  rpcInterface: string;
}

export interface Config {
  geolocation: string;
  network: string;
  accountAddress: string;
  debug: boolean;
}

export class StateTracker {
  // All variables in state tracker
  private updaters: Updater[] = [];
  private stateQuery: StateQuery;

  // Constructor for State Tracker
  constructor(
    badgeManager: BadgeManager,
    pairingListConfig: string,
    relayer: Relayer,
    chainIDRpcInterface: ChainIDRpcInterface[],
    config: Config,
    consumerSessionManagerMap: any // TODO add consumerSessionManagerMap type
  ) {
    // Initialize State Query
    this.stateQuery = new StateQuery(
      badgeManager,
      pairingListConfig,
      chainIDRpcInterface,
      relayer,
      config
    );

    // Create Pairing Updater
    const pairingUpdater = new PairingUpdater(
      this.stateQuery,
      consumerSessionManagerMap
    );

    // Register all updaters
    this.registerForUpdates(pairingUpdater);

    // Call executeUpdateOnNewEpoch method to start the process
    this.executeUpdateOnNewEpoch();
  }

  // registerForUpdates adds new updater
  private registerForUpdates(updater: Updater) {
    this.updaters.push(updater);
  }

  // executeUpdateOnNewEpoch executes all updates on every new epoch
  async executeUpdateOnNewEpoch(): Promise<void> {
    try {
      // Fetching all the info including the time_till_next_epoch
      const timeTillNextEpoch = await this.stateQuery.fetchPairing();

      // Call update method on all registered updaters
      for (const updater of this.updaters) {
        updater.update();
      }

      // Set up a timer to call this method again when the next epoch begins
      setTimeout(() => this.executeUpdateOnNewEpoch(), timeTillNextEpoch);
    } catch (error) {
      console.error("An error occurred during pairing processing:", error);

      // TODO fix DEFAULT_RETRY_INTERVAL
      const DEFAULT_RETRY_INTERVAL = 10000;
      setTimeout(() => this.executeUpdateOnNewEpoch(), DEFAULT_RETRY_INTERVAL);
    }
  }
}

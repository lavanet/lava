import { PairingUpdater } from "./updaters/pairing_updater";
import { StateChainQuery } from "./stateQuery/state_chain_query";
import { StateBadgeQuery } from "./stateQuery/state_badge_query";
import { BadgeManager } from "../badge/badgeManager";
import { debugPrint } from "../util/common";
import { StateQuery } from "./stateQuery/state_query";
import { Updater } from "./updaters/updater";
import Relayer from "../relayer/relayer";
import { ConsumerSessionsWithProvider } from "../lavasession/consumerTypes";

const DEFAULT_RETRY_INTERVAL = 10000;

// ChainIDRpcInterface
// TODO Move it to SDK class when we create it
export interface ChainIDRpcInterface {
  chainID: string;
  rpcInterface: string;
}

// Config interface
// TODO Move it to SDK class when we create it
export interface Config {
  geolocation: string;
  network: string;
  accountAddress: string;
  debug: boolean;
}

// ConsumerSessionManager interface
export interface ConsumerSessionManager {
  getRpcEndpoint(): string;
  updateAllProviders(
    epoch: number,
    providers: ConsumerSessionsWithProvider[]
  ): Promise<Error | undefined>;
}

// ConsumerSessionManagerList is an array of ConsumerSessionManager
export type ConsumerSessionManagerList = ConsumerSessionManager[];

// ConsumerSessionManagerMap is an map where key is chainID and value is ConsumerSessionManagerList
export type ConsumerSessionManagerMap = Map<string, ConsumerSessionManagerList>;

export class StateTracker {
  private updaters: Updater[] = []; // List of all registered updaters
  private stateQuery: StateQuery; // State Query instance
  private config: Config; // Config options

  // Constructor for State Tracker
  constructor(
    pairingListConfig: string,
    relayer: Relayer,
    chainIDRpcInterface: ChainIDRpcInterface[],
    config: Config,
    consumerSessionManagerMap: ConsumerSessionManagerMap,
    walletAddress: string,
    badgeManager?: BadgeManager
  ) {
    debugPrint(config.debug, "Initialization of State Tracker started");

    // Save config
    this.config = config;

    if (badgeManager != undefined) {
      this.stateQuery = new StateBadgeQuery(
        badgeManager,
        walletAddress,
        config,
        chainIDRpcInterface
      );
    } else {
      // Initialize State Query
      this.stateQuery = new StateChainQuery(
        pairingListConfig,
        chainIDRpcInterface,
        relayer,
        config
      );
    }

    // Create Pairing Updater
    const pairingUpdater = new PairingUpdater(
      this.stateQuery,
      consumerSessionManagerMap,
      config
    );

    // Register all updaters
    this.registerForUpdates(pairingUpdater);

    debugPrint(config.debug, "Pairing updater added");

    // Call executeUpdateOnNewEpoch method to start the process
    this.executeUpdateOnNewEpoch();
  }

  // executeUpdateOnNewEpoch executes all updates on every new epoch
  async executeUpdateOnNewEpoch(): Promise<void> {
    try {
      debugPrint(this.config.debug, "New epoch started, fetching pairing list");

      // Fetching all the info including the time_till_next_epoch
      const timeTillNextEpoch = await this.stateQuery.fetchPairing();

      debugPrint(
        this.config.debug,
        "Pairing list fetched, started new epoch in: " + timeTillNextEpoch
      );

      // Call update method on all registered updaters
      for (const updater of this.updaters) {
        updater.update();
      }

      // Set up a timer to call this method again when the next epoch begins
      setTimeout(
        () => this.executeUpdateOnNewEpoch(),
        timeTillNextEpoch * 1000
      );
    } catch (error) {
      console.error("An error occurred during pairing processing:", error);

      debugPrint(
        this.config.debug,
        "Retry fetching pairing list in: " + DEFAULT_RETRY_INTERVAL
      );

      // Retry fetching pairing list after DEFAULT_RETRY_INTERVAL
      setTimeout(() => this.executeUpdateOnNewEpoch(), DEFAULT_RETRY_INTERVAL);
    }
  }

  // registerForUpdates adds new updater
  private registerForUpdates(updater: Updater) {
    this.updaters.push(updater);
  }
}

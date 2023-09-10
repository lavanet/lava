import { StateChainQuery } from "./stateQuery/state_chain_query";
import { StateBadgeQuery } from "./stateQuery/state_badge_query";
import { BadgeManager } from "../badge/badgeManager";
import { Logger } from "../logger/logger";
import { StateQuery } from "./stateQuery/state_query";
import { Updater } from "./updaters/updater";
import { Relayer } from "../relayer/relayer";
import { AccountData } from "@cosmjs/proto-signing";
import { PairingResponse } from "../stateTracker/stateQuery/state_query";
import { PairingUpdater } from "./updaters/pairing_updater";
import { ConsumerSessionManager } from "../lavasession/consumerSessionManager";

const DEFAULT_RETRY_INTERVAL = 10000;

// Config interface
export interface Config {
  geolocation: string;
  network: string;
}

export class StateTracker {
  private updaters: Map<string, Updater>;
  private stateQuery: StateQuery;
  private timeTillNextEpoch = 0;

  // Constructor for State Tracker
  constructor(
    pairingListConfig: string,
    relayer: Relayer,
    chainIDs: string[],
    config: Config,
    account: AccountData,
    walletAddress: string,
    badgeManager?: BadgeManager
  ) {
    Logger.SetLogLevel(5);
    Logger.debug("Initialization of State Tracker started");

    this.updaters = new Map();
    if (badgeManager && badgeManager.isActive()) {
      this.stateQuery = new StateBadgeQuery(
        badgeManager,
        walletAddress,
        account,
        chainIDs,
        relayer
      );
    } else {
      // Initialize State Query
      this.stateQuery = new StateChainQuery(
        pairingListConfig,
        chainIDs,
        relayer,
        config,
        account
      );
    }

    // Create Pairing Updater
    const pairingUpdater = new PairingUpdater(this.stateQuery, config);

    // Register all updaters
    this.registerForUpdates(pairingUpdater, "pairingUpdater");

    Logger.debug("Pairing updater added");
  }

  getPairingResponse(chainId: string): PairingResponse | undefined {
    return this.stateQuery.getPairing(chainId);
  }

  async initialize() {
    Logger.debug("Initialization of State Tracker started");
    this.timeTillNextEpoch = await this.stateQuery.fetchPairing();
  }

  async startTracking() {
    Logger.debug("State Tracker started");
    // Call update method on all registered updaters
    for (let [key, updater] of this.updaters) {
      updater.update();
    }

    // Set up a timer to call this method again when the next epoch begins
    setTimeout(
      () => this.executeUpdateOnNewEpoch(),
      this.timeTillNextEpoch * 1000
    );
  }

  // executeUpdateOnNewEpoch executes all updates on every new epoch
  async executeUpdateOnNewEpoch(): Promise<void> {
    try {
      Logger.debug("New epoch started, fetching pairing list");

      // Fetching all the info including the time_till_next_epoch
      const timeTillNextEpoch = await this.stateQuery.fetchPairing();

      Logger.debug(
        "Pairing list fetched, started new epoch in: " + timeTillNextEpoch
      );

      // Call update method on all registered updaters
      for (let [key, updater] of this.updaters) {
        updater.update();
      }

      // Set up a timer to call this method again when the next epoch begins
      setTimeout(
        () => this.executeUpdateOnNewEpoch(),
        timeTillNextEpoch * 1000
      );
    } catch (error) {
      Logger.error("An error occurred during pairing processing:", error);

      Logger.debug("Retry fetching pairing list in: " + DEFAULT_RETRY_INTERVAL);

      // Retry fetching pairing list after DEFAULT_RETRY_INTERVAL
      setTimeout(() => this.executeUpdateOnNewEpoch(), DEFAULT_RETRY_INTERVAL);
    }
  }

  RegisterConsumerSessionManagerForPairingUpdates(
    consumerSessionManager: ConsumerSessionManager
  ) {
    const pairingUpdater = this.updaters.get("pairingUpdater");
    if (pairingUpdater == undefined) {
      return;
    }
    // TODO: change the updater interface to include only update method, and here use the instanceof and check if its a specific type of pairing Updater
    // then use the specific method registerPairing which is not an updater type but a pairingUpdater type.
    pairingUpdater.registerPairing(consumerSessionManager);
  }

  // registerForUpdates adds new updater
  private registerForUpdates(updater: Updater, name: string) {
    this.updaters.set(name, updater);
  }
}

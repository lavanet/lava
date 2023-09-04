import { PairingUpdater } from "./updaters/pairing_updater";
import { StateChainQuery } from "./stateQuery/state_chain_query";
import { StateBadgeQuery } from "./stateQuery/state_badge_query";
import { BadgeManager } from "../badge/badgeManager";
import { Logger } from "../logger/logger";
import { StateQuery } from "./stateQuery/state_query";
import { Updater } from "./updaters/updater";
import { Relayer } from "../relayer/relayer";
import { AccountData } from "@cosmjs/proto-signing";
import { RPCEndpoint } from "../lavasession/consumerTypes";
import { RandomProviderOptimizer } from "../lavasession/providerOptimizer";
import {
  ConsumerSessionManagersMap,
  ConsumerSessionManager,
} from "../lavasession/consumerSessionManager";
import { ChainIDRpcInterface } from "../sdk/sdk";

const DEFAULT_RETRY_INTERVAL = 10000;

// Config interface
export interface Config {
  geolocation: string;
  network: string;
}

export class StateTracker {
  private updaters: Updater[] = []; // List of all registered updaters
  private stateQuery: StateQuery; // State Query instance
  private config: Config; // Config options

  // Constructor for State Tracker
  constructor(
    pairingListConfig: string,
    relayer: Relayer,
    chainIDRpcInterfaces: ChainIDRpcInterface[],
    config: Config,
    account: AccountData,
    consumerSessionManagerMap: ConsumerSessionManagersMap,
    walletAddress: string,
    badgeManager?: BadgeManager
  ) {
    Logger.debug("Initialization of State Tracker started");

    // Save config
    this.config = config;

    if (badgeManager != undefined) {
      this.stateQuery = new StateBadgeQuery(
        badgeManager,
        walletAddress,
        config,
        account,
        chainIDRpcInterfaces,
        relayer
      );
    } else {
      // Initialize State Query
      this.stateQuery = new StateChainQuery(
        pairingListConfig,
        chainIDRpcInterfaces,
        relayer,
        config,
        account
      );
    }

    // Create Pairing Updater
    const pairingUpdater = new PairingUpdater(
      this.stateQuery,
      consumerSessionManagerMap,
      config,
      account
    );

    // Create Optimizer
    const optimizer = new RandomProviderOptimizer();

    // Register all pairirings
    for (const chainIDRpcInterface of chainIDRpcInterfaces) {
      const sessionManager = new ConsumerSessionManager(
        relayer,
        new RPCEndpoint(
          account.address,
          chainIDRpcInterface.chainID,
          chainIDRpcInterface.rpcInterface,
          config.geolocation
        ),
        optimizer
      );

      pairingUpdater.registerPairing(sessionManager);
    }

    // Register all updaters
    this.registerForUpdates(pairingUpdater);

    Logger.debug("Pairing updater added");
  }

  async initialize() {
    Logger.debug("Initialization of State Tracker started");
    await this.executeUpdateOnNewEpoch();
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

      Logger.debug("Retry fetching pairing list in: " + DEFAULT_RETRY_INTERVAL);

      // Retry fetching pairing list after DEFAULT_RETRY_INTERVAL
      setTimeout(() => this.executeUpdateOnNewEpoch(), DEFAULT_RETRY_INTERVAL);
    }
  }

  // registerForUpdates adds new updater
  private registerForUpdates(updater: Updater) {
    this.updaters.push(updater);
  }
}

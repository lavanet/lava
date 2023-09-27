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
import { RPCConsumerServer } from "../rpcconsumer/rpcconsumer_server";
import { Spec } from "../grpc_web_services/lavanet/lava/spec/spec_pb";

const DEFAULT_RETRY_INTERVAL = 10000;
// we are adding 10% to the epoch passing time so we dont race providers updates.
// we have overlap protecting us.
const DEFAULT_TIME_BACKOFF = 1000 * 1.1; // MS * 10%
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
    rpcConsumer: RPCConsumerServer | undefined,
    spec: Spec,
    account: AccountData,
    walletAddress: string,
    badgeManager?: BadgeManager
  ) {
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
      if (!rpcConsumer) {
        throw Logger.fatal(
          "No rpc consumer server provided in private key flow."
        );
      }
      this.stateQuery = new StateChainQuery(
        pairingListConfig,
        chainIDs,
        rpcConsumer,
        config,
        account,
        spec
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

    // Init state query
    await this.stateQuery.init();

    // Run all updaters
    // Only for chain query
    if (this.stateQuery instanceof StateChainQuery) {
      await this.update();
    }

    // Fetch Pairing
    this.timeTillNextEpoch = await this.stateQuery.fetchPairing();
  }

  async startTracking() {
    Logger.debug("State Tracker started");
    // update all consumer session managers with the provider lists after initialization.
    await this.update();
    // Set up a timer to call this method again when the next epoch begins
    setTimeout(
      () => this.executeUpdateOnNewEpoch(),
      this.timeTillNextEpoch * DEFAULT_TIME_BACKOFF
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

      await this.update();

      // Set up a timer to call this method again when the next epoch begins
      setTimeout(
        () => this.executeUpdateOnNewEpoch(),
        timeTillNextEpoch * DEFAULT_TIME_BACKOFF // we are adding 10% to the timeout to make sure we don't race providers
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

    if (!pairingUpdater) {
      throw new Error("Missing pairing updater");
    }

    if (!(pairingUpdater instanceof PairingUpdater)) {
      throw new Error("Invalid updater type");
    }

    pairingUpdater.registerPairing(consumerSessionManager);
  }

  private async update() {
    // Call update method on all registered updaters
    const promiseArray = [];
    for (const updater of this.updaters.values()) {
      promiseArray.push(updater.update());
    }
    await Promise.allSettled(promiseArray);
  }

  // registerForUpdates adds new updater
  private registerForUpdates(updater: Updater, name: string) {
    this.updaters.set(name, updater);
  }
}

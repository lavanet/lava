import { StateQuery } from "../stateQuery/state_query";
import { Logger } from "../../logger/logger";
import { RPCConsumerServer } from "../../consumer/rpcconsumer_server";

export class PairingUpdater {
  private stateQuery: StateQuery;
  private consumer: RPCConsumerServer;
  private chainIDs: string[];

  // Constructor for Pairing Updater
  constructor(
    stateQuery: StateQuery,
    consumer: RPCConsumerServer,
    chainIDs: string[]
  ) {
    Logger.debug("Initialization of Pairing Updater started");

    // Save arguments
    this.stateQuery = stateQuery;
    this.consumer = consumer;
    this.chainIDs = chainIDs;
  }

  // update updates pairing list on every consumer session manager
  public update() {
    Logger.debug("Start updating consumer session managers");

    for (const chainID of this.chainIDs) {
      Logger.debug("Updating pairing list for: ", chainID);

      // Fetch pairing list
      const pairing = this.stateQuery.getPairing(chainID);
      if (pairing == undefined) {
        Logger.debug("Failed fetching pairing list for: ", chainID);

        // TODO check how we handle this
        continue;
      }
      Logger.debug("Pairing list fetched: ", pairing);
      this.consumer.updateAllProviders(pairing);
    }
  }
}

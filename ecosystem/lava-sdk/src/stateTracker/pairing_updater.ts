// import state query
import { StateQuery } from "./state_query";

export class PairingUpdater {
  private stateQuery: StateQuery;
  private consumerSessionManagerMap: any;

  constructor(stateQuery: StateQuery, consumerSessionManagerMap: any) {
    // Save arguments
    this.stateQuery = stateQuery;
    this.consumerSessionManagerMap = consumerSessionManagerMap;
  }

  public update() {
    const keys = Object.keys(this.consumerSessionManagerMap);
    for (const chainID of keys) {
      const consumerSessionManagerList =
        this.consumerSessionManagerMap[chainID];
      // Your code using chainID and consumerSessionManagerList
      const pairing = this.stateQuery.getPairing(chainID);

      // Update each consumer session menager with matching pairing list
      for (const consumerSessionManager of consumerSessionManagerList) {
        this.consumerSessionManagerMap(pairing, consumerSessionManager);
      }
    }
  }

  // updateConsummerSessionManager filters pairing list and update consuemr session manager
  private updateConsummerSessionManager(
    pairing: any,
    consumerSessionManager: any
  ) {
    // Filter pairing list for specific consumer session manager
    const pairingListForThisCSM = this.filterPairingListByEndpoint(
      pairing,
      consumerSessionManager.RPCEndpoint()
    );

    // Update specific consumer session manager
    consumerSessionManager.UpdateAllProviders(pairingListForThisCSM);
  }

  // filterPairingListByEndpoint filters pairing list and return only the once for rpcInterface
  private filterPairingListByEndpoint(pairing: any, rpcInterface: string): any {
    return;
  }
}

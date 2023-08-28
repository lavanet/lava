import { StateQuery, PairingResponse } from "./state_query";
import { debugPrint, parseLong } from "../util/common";
import {
  Config,
  ConsumerSessionManager,
  ConsumerSessionManagerMap,
} from "./state_tracker";
// import {
//   ConsumerSessionWithProvider,
//   SingleConsumerSession,
//   Endpoint,
// } from "../types/types";
import {
  ConsumerSessionsWithProvider,
  SingleConsumerSession,
  Endpoint,
} from "../lavasession/consumerTypes";

export class PairingUpdater {
  private stateQuery: StateQuery; // State Query instance
  private config: Config; // Config options
  private consumerSessionManagerMap: ConsumerSessionManagerMap; // ConsumerSessionManagerMap instance

  // Constructor for Pairing Updater
  constructor(
    stateQuery: StateQuery,
    consumerSessionManagerMap: ConsumerSessionManagerMap,
    config: Config
  ) {
    debugPrint(config.debug, "Initialization of Pairing Updater started");

    // Save arguments
    this.stateQuery = stateQuery;
    this.config = config;
    this.consumerSessionManagerMap = consumerSessionManagerMap;
  }

  // update updates pairing list on every consumer session manager
  public update() {
    debugPrint(this.config.debug, "Start updating consumer session managers");
    this.consumerSessionManagerMap.forEach(
      (consumerSessionManagerList, chainID) => {
        debugPrint(this.config.debug, "Updating pairing list for: ", chainID);

        // Fetch pairing list
        const pairing = this.stateQuery.getPairing(chainID);
        if (pairing == undefined) {
          debugPrint(
            this.config.debug,
            "Failed fetching pairing list for: ",
            chainID
          );
        } else {
          debugPrint(this.config.debug, "Pairing list fetched: ", pairing);
        }

        // Update each consumer session manager with matching pairing list
        consumerSessionManagerList.forEach((consumerSessionManager) => {
          this.updateConsummerSessionManager(pairing, consumerSessionManager);
        });
      }
    );
  }

  // updateConsummerSessionManager filters pairing list and update consuemr session manager
  private updateConsummerSessionManager(
    pairing: PairingResponse | undefined,
    consumerSessionManager: ConsumerSessionManager
  ) {
    // If pairing undefined
    // update consumer session manager with empty provider list
    if (pairing == undefined) {
      consumerSessionManager.updateAllProviders(0, []);

      return;
    }

    // Filter pairing list for specific consumer session manager
    const pairingListForThisCSM = this.filterPairingListByEndpoint(
      pairing,
      consumerSessionManager.getRpcEndpoint()
    );

    // Update specific consumer session manager
    consumerSessionManager.updateAllProviders(
      pairing.currentEpoch,
      pairingListForThisCSM
    );
  }

  // filterPairingListByEndpoint filters pairing list and return only the once for rpcInterface
  private filterPairingListByEndpoint(
    pairing: PairingResponse,
    rpcInterface: string
  ): ConsumerSessionsWithProvider[] {
    // Initialize ConsumerSessionWithProvider array
    const pairingForSameGeolocation: Array<ConsumerSessionsWithProvider> = [];
    const pairingFromDifferentGeolocation: Array<ConsumerSessionsWithProvider> =
      [];
    // Iterate over providers to populate pairing list
    for (const provider of pairing.providers) {
      debugPrint(this.config.debug, "parsing provider", provider);

      // Skip providers with no endpoints
      if (provider.endpoints.length == 0) {
        continue;
      }

      // Initialize relevantEndpoints array
      const sameGeoEndpoints: Array<Endpoint> = [];
      const differntGeoEndpoints: Array<Endpoint> = [];

      // Only take into account endpoints that use the same api interface
      // And geolocation
      for (const endpoint of provider.endpoints) {
        if (!endpoint.apiInterfaces.includes(rpcInterface)) {
          continue;
        }
        const convertedEndpoint: Endpoint = {
          networkAddress: endpoint.iPPORT,
          enabled: true,
          connectionRefusals: 0,
          addons: new Set(endpoint.addons),
          extensions: new Set(endpoint.extensions),
        };
        if (
          parseLong(endpoint.geolocation) == Number(this.config.geolocation)
        ) {
          sameGeoEndpoints.push(convertedEndpoint); // set same geo location provider endpoint
        } else {
          differntGeoEndpoints.push(convertedEndpoint); // set different geo location provider endpoint
        }
      }

      // skip if we have no endpoints at all.
      if (sameGeoEndpoints.length == 0 && differntGeoEndpoints.length == 0) {
        debugPrint(this.config.debug, "No endpoints found");
        continue;
      }

      let sameGeoOptions = false; // if we have same geolocation options or not
      let endpointListToStore: Endpoint[] = differntGeoEndpoints;
      if (sameGeoEndpoints.length > 0) {
        sameGeoOptions = true;
        endpointListToStore = sameGeoEndpoints;
      }

      const newPairing = new ConsumerSessionsWithProvider(
        this.config.accountAddress,
        endpointListToStore,
        {},
        pairing.maxCu,
        pairing.currentEpoch
      );

      // Add newly created pairing in the pairing list
      if (sameGeoOptions) {
        pairingForSameGeolocation.push(newPairing);
      } else {
        pairingFromDifferentGeolocation.push(newPairing);
      }
    }

    if (
      pairingForSameGeolocation.length == 0 &&
      pairingFromDifferentGeolocation.length == 0
    ) {
      debugPrint(this.config.debug, "No relevant providers found");
    }

    // Return providers list [pairingForSameGeolocation,pairingFromDifferentGeolocation]
    return pairingForSameGeolocation.concat(pairingFromDifferentGeolocation);
  }
}

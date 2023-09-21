import { StateQuery } from "../stateQuery/state_query";
import { Logger } from "../../logger/logger";
import {
  ConsumerSessionManagersMap,
  ConsumerSessionManager,
} from "../../lavasession/consumerSessionManager";
import { ConsumerSessionsWithProvider } from "../../lavasession/consumerTypes";
import { PairingResponse } from "../stateQuery/state_query";
import { Config } from "../state_tracker";
import { Endpoint as ConsumerEndpoint } from "../../lavasession/consumerTypes";

export class PairingUpdater {
  private stateQuery: StateQuery;
  private consumerSessionManagerMap: ConsumerSessionManagersMap;
  private config: Config;

  // Constructor for Pairing Updater
  constructor(stateQuery: StateQuery, config: Config) {
    Logger.debug("Initialization of Pairing Updater started");

    // Save arguments
    this.stateQuery = stateQuery;
    this.consumerSessionManagerMap = new Map();
    this.config = config;
  }

  async registerPairing(
    consumerSessionManager: ConsumerSessionManager
  ): Promise<void | Error> {
    const chainID = consumerSessionManager.getRpcEndpoint().chainId;

    const consumerSessionsManagersList =
      this.consumerSessionManagerMap.get(chainID);

    if (!consumerSessionsManagersList) {
      this.consumerSessionManagerMap.set(chainID, [consumerSessionManager]);
      return;
    }

    consumerSessionsManagersList.push(consumerSessionManager);
    this.consumerSessionManagerMap.set(chainID, consumerSessionsManagersList);
  }

  // update updates pairing list on every consumer session manager
  public async update() {
    Logger.debug("Start updating consumer session managers");

    // Get all chainIDs from the map
    const chainIDs = Array.from(this.consumerSessionManagerMap.keys());

    for (const chainID of chainIDs) {
      const consumerSessionManagerList =
        this.consumerSessionManagerMap.get(chainID);

      if (consumerSessionManagerList == undefined) {
        Logger.debug("Consumer session manager udnefined: ", chainID);
        continue;
      }

      Logger.debug("Updating pairing list for: ", chainID);
      Logger.debug(
        "Number of CSM registered to this chainId: ",
        consumerSessionManagerList.length
      );

      // Fetch pairing list
      const pairing = this.stateQuery.getPairing(chainID);
      if (pairing == undefined) {
        Logger.debug("Failed fetching pairing list for: ", chainID);
      } else {
        Logger.debug(
          "Pairing list fetched: ",
          pairing.currentEpoch,
          pairing.providers
        );
      }

      const promiseArray = [];
      // Update each consumer session manager with matching pairing list
      for (const consumerSessionManager of consumerSessionManagerList) {
        promiseArray.push(
          this.updateConsumerSessionManager(pairing, consumerSessionManager)
        );
      }
      await Promise.allSettled(promiseArray);
    }
  }

  // updateConsummerSessionManager filters pairing list and update consuemr session manager
  private async updateConsumerSessionManager(
    pairing: PairingResponse | undefined,
    consumerSessionManager: ConsumerSessionManager
  ): Promise<void> {
    // If pairing undefined return + error
    if (pairing == undefined) {
      Logger.error("Pairing response is undefined");
      return;
    }

    // Filter pairing list for specific consumer session manager
    const pairingListForThisCSM = this.filterPairingListByEndpoint(
      pairing,
      consumerSessionManager.getRpcEndpoint().apiInterface
    );

    // Update specific consumer session manager
    await consumerSessionManager.updateAllProviders(
      pairing.currentEpoch,
      pairingListForThisCSM
    );

    return;
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
      Logger.debug("parsing provider", provider);
      // Skip providers with no endpoints
      if (provider.getEndpointsList().length == 0) {
        continue;
      }

      // Initialize relevantEndpoints array
      const sameGeoEndpoints: Array<ConsumerEndpoint> = [];
      const differntGeoEndpoints: Array<ConsumerEndpoint> = [];

      // Only take into account endpoints that use the same api interface
      // And geolocation
      for (const endpoint of provider.getEndpointsList()) {
        if (!endpoint.getApiInterfacesList().includes(rpcInterface)) {
          continue;
        }

        const consumerEndpoint: ConsumerEndpoint = {
          addons: new Set(endpoint.getAddonsList()),
          extensions: new Set(endpoint.getExtensionsList()),
          networkAddress: endpoint.getIpport(),
          enabled: true,
          connectionRefusals: 0,
        };

        if (endpoint.getGeolocation() == Number(this.config.geolocation)) {
          sameGeoEndpoints.push(consumerEndpoint); // set same geo location provider endpoint
        } else {
          differntGeoEndpoints.push(consumerEndpoint); // set different geo location provider endpoint
        }
      }

      // skip if we have no endpoints at all.
      if (sameGeoEndpoints.length == 0 && differntGeoEndpoints.length == 0) {
        Logger.debug("No endpoints found");
        continue;
      }

      let sameGeoOptions = false; // if we have same geolocation options or not
      let endpointListToStore: ConsumerEndpoint[] = differntGeoEndpoints;
      if (sameGeoEndpoints.length > 0) {
        sameGeoOptions = true;
        endpointListToStore = sameGeoEndpoints;
      }

      const newPairing = new ConsumerSessionsWithProvider(
        provider.getAddress(),
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
      Logger.debug("No relevant providers found");
    }
    Logger.debug(
      "providers initialized",
      "our geo",
      JSON.stringify(pairingForSameGeolocation),
      "other geo",
      JSON.stringify(pairingFromDifferentGeolocation)
    );
    // Return providers list [pairingForSameGeolocation,pairingFromDifferentGeolocation]
    return pairingForSameGeolocation.concat(pairingFromDifferentGeolocation);
  }
}

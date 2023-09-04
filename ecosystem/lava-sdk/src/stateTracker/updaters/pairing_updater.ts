import { StateQuery, PairingResponse } from "../stateQuery/state_query";
import { parseLong } from "../../util/common";
import { AccountData } from "@cosmjs/proto-signing";
import { Config } from "../state_tracker";
import { Logger } from "../../logger/logger";
import { ChainIDRpcInterface } from "../../sdk/sdk";
import {
  ConsumerSessionManager,
  ConsumerSessionManagersMap,
} from "../../lavasession/consumerSessionManager";
import {
  ConsumerSessionsWithProvider,
  Endpoint,
} from "../../lavasession/consumerTypes";

export class PairingUpdater {
  private stateQuery: StateQuery; // State Query instance
  private config: Config; // Config options
  private consumerSessionManagerMap: ConsumerSessionManagersMap; // ConsumerSessionManagerMap instance
  private account: AccountData;

  // Constructor for Pairing Updater
  constructor(
    stateQuery: StateQuery,
    consumerSessionManagerMap: ConsumerSessionManagersMap,
    config: Config,
    account: AccountData
  ) {
    Logger.debug("Initialization of Pairing Updater started");

    // Save arguments
    this.account = account;
    this.stateQuery = stateQuery;
    this.config = config;
    this.consumerSessionManagerMap = consumerSessionManagerMap;
  }

  public registerPairing(consumerSessionManager: ConsumerSessionManager) {
    const chainID = consumerSessionManager.getRpcEndpoint().chainId;

    if (this.consumerSessionManagerMap.has(chainID)) {
      // If the chainID exists, fetch the array and push the new manager to it.
      const managers = this.consumerSessionManagerMap.get(chainID);
      if (managers) {
        managers.push(consumerSessionManager);
      }
    } else {
      // If the chainID doesn't exist, initialize a new array with the manager as the first element.
      this.consumerSessionManagerMap.set(chainID, [consumerSessionManager]);
    }
  }

  // update updates pairing list on every consumer session manager
  public update() {
    Logger.debug("Start updating consumer session managers");
    this.consumerSessionManagerMap.forEach(
      (consumerSessionManagerList, chainID) => {
        Logger.debug("Updating pairing list for: ", chainID);

        // Fetch pairing list
        const pairing = this.stateQuery.getPairing(chainID);
        if (pairing == undefined) {
          Logger.debug("Failed fetching pairing list for: ", chainID);
        } else {
          Logger.debug("Pairing list fetched: ", pairing);
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
      consumerSessionManager.getRpcEndpoint().apiInterface
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
      Logger.debug("parsing provider", provider);
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
        const convertedEndpoint = {
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
        Logger.debug("No endpoints found");
        continue;
      }

      let sameGeoOptions = false; // if we have same geolocation options or not
      let endpointListToStore: Endpoint[] = differntGeoEndpoints;
      if (sameGeoEndpoints.length > 0) {
        sameGeoOptions = true;
        endpointListToStore = sameGeoEndpoints;
      }

      const newPairing = new ConsumerSessionsWithProvider(
        provider.address,
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

    // Return providers list [pairingForSameGeolocation,pairingFromDifferentGeolocation]
    return pairingForSameGeolocation.concat(pairingFromDifferentGeolocation);
  }
}

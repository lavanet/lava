import { PairingResponse } from "./state_query";
import {
  BadgeManager,
  TimoutFailureFetchingBadgeError,
} from "../../badge/badgeManager";
import { GenerateBadgeResponse } from "../../grpc_web_services/lavanet/lava/pairing/badges_pb";
import { StateTrackerErrors } from "../errors";
import { StakeEntry } from "../../codec/lavanet/lava/epochstorage/stake_entry";
import { Relayer } from "../../relayer/relayer";
import { AccountData } from "@cosmjs/proto-signing";
import { Logger } from "../../logger/logger";

export class StateBadgeQuery {
  private pairing: Map<string, PairingResponse | undefined>;
  private badgeManager: BadgeManager;
  private chainIDs: string[];
  private walletAddress: string;
  private relayer: Relayer;
  private account: AccountData;

  constructor(
    badgeManager: BadgeManager,
    walletAddress: string,
    account: AccountData,
    chainIDs: string[],
    relayer: Relayer
  ) {
    Logger.debug("Initialization of State Badge Query started");

    // Save arguments
    this.badgeManager = badgeManager;
    this.walletAddress = walletAddress;
    this.chainIDs = chainIDs;
    this.relayer = relayer;
    this.account = account;

    // Initialize pairing to an empty map
    this.pairing = new Map<string, PairingResponse>();

    Logger.debug("Initialization of State Badge Query ended");
  }

  // fetchPairing fetches pairing for all chainIDs we support
  public async fetchPairing(): Promise<number> {
    Logger.debug("Fetching pairing started");
    let timeLeftToNextPairing;
    for (const chainID of this.chainIDs) {
      const badgeResponse = await this.fetchNewBadge(chainID);

      const badge = badgeResponse.getBadge();
      if (badge == undefined) {
        this.pairing.set(chainID, undefined);

        continue;
      }

      this.relayer.setBadge(badge);

      (this.account as any).address = badgeResponse.getBadgeSignerAddress();

      const pairingResponse = badgeResponse.getGetPairingResponse();

      if (pairingResponse == undefined) {
        this.pairing.set(chainID, undefined);

        continue;
      }

      // Parse time till next epoch
      timeLeftToNextPairing = pairingResponse.getTimeLeftToNextPairing();

      // Generate StakeEntry
      const stakeEntry: Array<StakeEntry> = [];

      for (const provider of pairingResponse.getProvidersList()) {
        const providerObject = provider.toObject() as any;

        // Rename 'endpointsList' to 'endpoints' and process its attributes
        if (providerObject.endpointsList) {
          providerObject.endpoints = providerObject.endpointsList.map(
            (endpoint: any) => {
              // Process the endpoint attributes if needed
              const processedEndpoint =
                this.removeListSuffixFromAttributes(endpoint);

              // Convert the 'ipport' attribute to 'iPPORT'
              if (processedEndpoint.ipport) {
                processedEndpoint.iPPORT = processedEndpoint.ipport;
                delete processedEndpoint.ipport;
              }

              return processedEndpoint;
            }
          );
          delete providerObject.endpointsList;
        }

        stakeEntry.push(providerObject);
      }

      // Save pairing response for chainID
      this.pairing.set(chainID, undefined);
    }

    // If timeLeftToNextPairing undefined return an error
    if (timeLeftToNextPairing == undefined) {
      throw StateTrackerErrors.errTimeTillNextEpochMissing;
    }

    Logger.debug("Fetching pairing ended");

    return timeLeftToNextPairing;
  }

  // getPairing return pairing list for specific chainID
  public getPairing(chainID: string): PairingResponse | undefined {
    // Return pairing for the specific chainId from the map
    return this.pairing.get(chainID);
  }

  // Helper function to rename attributes by removing the 'List' suffix
  private removeListSuffixFromAttributes(obj: any): any {
    const newObj: any = {};
    for (const key in obj) {
      const newKey = key.replace("List", "");
      newObj[newKey] = obj[key];
    }
    return newObj;
  }

  private async fetchNewBadge(chainID: string): Promise<GenerateBadgeResponse> {
    if (this.badgeManager == undefined) {
      throw Error("Badge undefined");
    }

    const badgeResponse = await this.badgeManager.fetchBadge(
      this.walletAddress,
      chainID
    );

    if (badgeResponse instanceof Error) {
      throw TimoutFailureFetchingBadgeError;
    }

    return badgeResponse;
  }
}

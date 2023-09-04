import { PairingResponse } from "./state_query";
import {
  BadgeManager,
  TimoutFailureFetchingBadgeError,
} from "../../badge/badgeManager";
import { debugPrint } from "../../util/common";
import { Config } from "../state_tracker";
import { ChainIDRpcInterface } from "../../sdk/sdk";
import { GenerateBadgeResponse } from "../../grpc_web_services/lavanet/lava/pairing/badges_pb";
import { StateTrackerErrors } from "../errors";
import { StakeEntry } from "../../codec/lavanet/lava/epochstorage/stake_entry";
import Relayer from "../../relayer/relayer";
import { AccountData } from "@cosmjs/proto-signing";

export class StateBadgeQuery {
  private pairing: Map<string, PairingResponse>; // Pairing is a map where key is chainID and value is PairingResponse
  private badgeManager: BadgeManager;
  private chainIDRpcInterfaces: ChainIDRpcInterface[]; // Array of {chainID, rpcInterface} pairs
  private walletAddress: string;
  private config: Config;
  private relayer: Relayer;
  private account: AccountData;

  constructor(
    badgeManager: BadgeManager,
    walletAddress: string,
    config: Config,
    account: AccountData,
    chainIdRpcInterfaces: ChainIDRpcInterface[],
    relayer: Relayer
  ) {
    debugPrint(config.debug, "Initialization of State Badge Query started");

    // Save arguments
    this.badgeManager = badgeManager;
    this.walletAddress = walletAddress;
    this.config = config;
    this.chainIDRpcInterfaces = chainIdRpcInterfaces;
    this.relayer = relayer;
    this.account = account;

    // Initialize pairing to an empty map
    this.pairing = new Map<string, PairingResponse>();

    debugPrint(config.debug, "Initialization of State Badge Query ended");
  }

  // fetchPairing fetches pairing for all chainIDs we support
  public async fetchPairing(): Promise<number> {
    debugPrint(this.config.debug, "Fetching pairing started");
    let timeLeftToNextPairing;
    for (const chainIDRpcInterface of this.chainIDRpcInterfaces) {
      const badgeResponse = await this.fetchNewBadge(
        chainIDRpcInterface.chainID
      );

      const badge = badgeResponse.getBadge();
      if (badge == undefined) {
        this.pairing.set(chainIDRpcInterface.chainID, {
          providers: [],
          maxCu: -1,
          currentEpoch: -1,
        });

        continue;
      }

      this.relayer.setBadge(badge);

      (this.account as any).address = badgeResponse.getBadgeSignerAddress();

      const pairingResponse = badgeResponse.getGetPairingResponse();

      if (pairingResponse == undefined) {
        this.pairing.set(chainIDRpcInterface.chainID, {
          providers: [],
          maxCu: -1,
          currentEpoch: -1,
        });

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
      this.pairing.set(chainIDRpcInterface.chainID, {
        providers: stakeEntry,
        maxCu: badge.getCuAllocation(),
        currentEpoch: pairingResponse.getCurrentEpoch(),
      });
    }

    // If timeLeftToNextPairing undefined return an error
    if (timeLeftToNextPairing == undefined) {
      throw StateTrackerErrors.errTimeTillNextEpochMissing;
    }

    debugPrint(this.config.debug, "Fetching pairing ended");

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

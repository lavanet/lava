import { PairingResponse } from "./state_query";
import { GenerateBadgeResponse } from "../../grpc_web_services/lavanet/lava/pairing/badges_pb";
import { StateTrackerErrors } from "../errors";
import { Relayer } from "../../relayer/relayer";
import { AccountData } from "@cosmjs/proto-signing";
import { Logger } from "../../logger/logger";
import {
  BadgeManager,
  TimoutFailureFetchingBadgeError,
} from "../../badge/badgeManager";

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
      if (badgeResponse == undefined) {
        this.pairing.set(chainID, undefined);

        continue;
      }

      const badge = badgeResponse.getBadge();
      if (badge == undefined) {
        this.pairing.set(chainID, undefined);

        continue;
      }

      this.relayer.setBadge(badge);

      (this.account as any).address = badgeResponse.getBadgeSignerAddress();

      const pairingResponse = badgeResponse.getGetPairingResponse();
      const specResponse = badgeResponse.getSpec();

      if (pairingResponse == undefined || specResponse == undefined) {
        this.pairing.set(chainID, undefined);

        continue;
      }

      // Parse time till next epoch
      timeLeftToNextPairing = pairingResponse.getTimeLeftToNextPairing();

      // Generate StakeEntry
      const stakeEntry = pairingResponse.getProvidersList();

      // Save pairing response for chainID
      this.pairing.set(chainID, {
        providers: stakeEntry,
        maxCu: badge.getCuAllocation(),
        currentEpoch: pairingResponse.getCurrentEpoch(),
        spec: specResponse,
      });
    }

    // If timeLeftToNextPairing undefined return an error
    if (timeLeftToNextPairing == undefined) {
      throw StateTrackerErrors.errTimeTillNextEpochMissing;
    }

    Logger.debug("Fetching pairing ended");

    return timeLeftToNextPairing;
  }

  public async init(): Promise<void> {
    return;
  }

  // getPairing return pairing list for specific chainID
  public getPairing(chainID: string): PairingResponse | undefined {
    // Return pairing for the specific chainId from the map
    return this.pairing.get(chainID);
  }

  private async fetchNewBadge(
    chainID: string
  ): Promise<GenerateBadgeResponse | undefined> {
    try {
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
    } catch (err) {
      throw Logger.fatal("Failed fetching badge", err);
    }
  }
}

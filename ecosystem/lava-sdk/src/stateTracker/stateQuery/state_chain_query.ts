import { DEFAULT_LAVA_PAIRING_LIST } from "../../config/default";
import { Config } from "../state_tracker";
import { fetchLavaPairing } from "../../util/lavaPairing";
import { StateTrackerErrors } from "../errors";
import { PairingResponse } from "./state_query";
import { AccountData } from "@cosmjs/proto-signing";
import Long from "long";
import {
  base64ToUint8Array,
  generateRPCData,
  parseLong,
  generateRandomInt,
} from "../../util/common";
import {
  QueryGetPairingRequest,
  QuerySdkPairingResponse,
} from "../../codec/lavanet/lava/pairing/query";
import { Relayer } from "../../relayer/relayer";
import { ProvidersErrors } from "../errors";
import {
  ConsumerSessionsWithProvider,
  SingleConsumerSession,
  Endpoint,
} from "../../lavasession/consumerTypes";
import { Logger } from "../../logger/logger";

const lavaChainID = "LAV1";
const lavaRPCInterface = "tendermintrpc";

export class StateChainQuery {
  private pairingListConfig: string;
  private relayer: Relayer;
  private chainIDs: string[];
  private lavaProviders: ConsumerSessionsWithProvider[];
  private config: Config;
  private pairing: Map<string, PairingResponse | undefined>;
  private account: AccountData;

  constructor(
    pairingListConfig: string,
    chainIDs: string[],
    relayer: Relayer,
    config: Config,
    account: AccountData
  ) {
    Logger.debug("Initialization of State Chain Query started");

    // Save arguments
    this.pairingListConfig = pairingListConfig;
    this.chainIDs = chainIDs;
    this.relayer = relayer;
    this.config = config;
    this.account = account;

    // Assign lavaProviders to an empty array
    this.lavaProviders = [];

    // Initialize pairing to an empty map
    this.pairing = new Map<string, PairingResponse>();

    Logger.debug("Initialization of State Chain Query ended");
  }

  // fetchPairing fetches pairing for all chainIDs we support
  public async fetchPairing(): Promise<number> {
    try {
      Logger.debug("Fetching pairing started");
      // Save time till next epoch
      let timeLeftToNextPairing;

      // fetch lava providers
      await this.fetchLavaProviders(this.pairingListConfig);

      // Make sure lava providers are initialized
      if (this.lavaProviders == null) {
        throw ProvidersErrors.errLavaProvidersNotInitialized;
      }

      // Reset pairing
      this.pairing = new Map<string, PairingResponse>();

      // Fetch latest block using probe
      const latestNumber = await this.getLatestBlockFromProviders(
        this.relayer,
        this.lavaProviders,
        lavaChainID,
        lavaRPCInterface
      );

      // Update latest block in lava pairing
      for (const consumerSessionWithProvider of this.lavaProviders) {
        consumerSessionWithProvider.setPairingEpoch(latestNumber);
      }

      // Iterate over chain and construct pairing
      for (const chainID of this.chainIDs) {
        // Fetch pairing for specified chainID
        const pairingResponse = await this.getPairingFromChain(
          {
            chainID: chainID,
            client: this.account.address,
          },
          10
        );

        // If pairing is undefined set to empty object
        if (
          pairingResponse == undefined ||
          pairingResponse.pairing == undefined ||
          pairingResponse.spec == undefined
        ) {
          this.pairing.set(chainID, undefined);

          continue;
        }

        // Parse time till next epoch
        timeLeftToNextPairing = parseLong(
          pairingResponse.pairing.timeLeftToNextPairing
        );

        // Save pairing response for chainID
        this.pairing.set(chainID, {
          providers: pairingResponse.pairing.providers,
          maxCu: parseLong(pairingResponse.maxCu),
          currentEpoch: latestNumber,
          spec: pairingResponse.spec,
        });
      }

      // If timeLeftToNextPairing undefined return an error
      if (timeLeftToNextPairing == undefined) {
        throw StateTrackerErrors.errTimeTillNextEpochMissing;
      }

      Logger.debug("Fetching pairing ended");

      // Return timeLeftToNextPairing
      return timeLeftToNextPairing;
    } catch (err) {
      throw err;
    }
  }

  // getPairing return pairing list for specific chainID
  public getPairing(chainID: string): PairingResponse | undefined {
    // Return pairing for the specific chainId from the map
    return this.pairing.get(chainID);
  }

  //fetchLavaProviders fetches lava providers from different sources
  private async fetchLavaProviders(
    pairingListConfig: string
  ): Promise<ConsumerSessionsWithProvider[]> {
    try {
      Logger.debug("Fetching lava providers started");

      // If we have providers return them
      if (this.lavaProviders.length != 0) {
        Logger.debug("Return already saved providers");
        return this.lavaProviders;
      }

      // Else if pairingListConfig exists use it to fetch lava providers from local file
      if (pairingListConfig != "") {
        const pairingList = await this.fetchLocalLavaPairingList(
          pairingListConfig
        );

        const providers = this.constructLavaPairing(pairingList);

        // Construct lava providers from pairing list and return it
        return providers;
      }

      // Fetch pairing from default lava pairing list
      const pairingList = await this.fetchDefaultLavaPairingList();

      const providers = this.constructLavaPairing(pairingList);

      // Construct lava providers from pairing list and return it
      return providers;
    } catch (err) {
      throw err;
    }
  }

  // getPairingFromChain fetch pairing response from lava providers
  private async getPairingFromChain(
    request: QueryGetPairingRequest,
    relayCu: number
  ): Promise<QuerySdkPairingResponse> {
    try {
      Logger.debug("Get pairing for:" + request.chainID + " started");
      // Encode request
      const requestData = QueryGetPairingRequest.encode(request).finish();

      // Create hex from data
      const hexData = Buffer.from(requestData).toString("hex");

      // Init send relay options
      const sendRelayOptions = {
        data: generateRPCData("abci_query", [
          "/lavanet.lava.pairing.Query/SdkPairing",
          hexData,
          "0",
          false,
        ]),
        url: "",
        connectionType: "",
      };

      // Send relay to all providers and return first response
      const jsonResponse = await this.relayer.SendRelayToAllProvidersAndRace(
        this.lavaProviders,
        sendRelayOptions,
        relayCu,
        lavaRPCInterface,
        lavaChainID
      );

      if (jsonResponse.result.response.value == null) {
        // If response is null log the error
        Logger.error(
          "ERROR, failed to fetch pairing for spec: " +
            request.chainID +
            ",error: " +
            jsonResponse.result.response.log
        );

        // Return empty object
        // We do not want to return error because it will stop the state tracker for other chains
        return {
          pairing: undefined,
          spec: undefined,
          maxCu: Long.fromNumber(-1),
        };
      }

      // Decode response
      const byteArrayResponse = base64ToUint8Array(
        jsonResponse.result.response.value
      );
      const decodedResponse = QuerySdkPairingResponse.decode(byteArrayResponse);

      // If response undefined throw an error
      if (
        decodedResponse.pairing == undefined ||
        decodedResponse.pairing.providers == undefined
      ) {
        throw ProvidersErrors.errProvidersNotFound;
      }

      Logger.debug("Get pairing for:" + request.chainID + " ended");
      // Return decoded response
      return decodedResponse;
    } catch (err) {
      // Console log the error
      console.error(err);

      // Return empty object
      // We do not want to return error because it will stop the state tracker for other chains
      return {
        pairing: undefined,
        spec: undefined,
        maxCu: Long.fromNumber(-1),
      };
    }
  }

  // getLatestBlockFromProviders tries to fetch latest block using probe
  private async getLatestBlockFromProviders(
    relayer: Relayer,
    providers: ConsumerSessionsWithProvider[],
    chainID: string,
    rpcInterface: string
  ): Promise<number> {
    Logger.debug("Get latest block from providers started");

    let lastProbeResponse = null;

    // Iterate over providers and return first successfull probe response
    for (let i = 0; i < providers.length; i++) {
      try {
        // Send probe request
        const probeResponse = await relayer.probeProvider(
          this.lavaProviders[i].sessions[0].endpoint.networkAddress,
          rpcInterface,
          chainID
        );

        // If no error save response and break
        lastProbeResponse = probeResponse;

        break;
      } catch (err) {
        // If error is instance of Error
        if (err instanceof Error) {
          // Store the relay response
          lastProbeResponse = err;
        }
      }
    }

    // If last provider returned an error
    // throw it
    if (lastProbeResponse instanceof Error) {
      throw lastProbeResponse;
    }

    // If probe response does not exists return an error
    // This should never happen
    if (lastProbeResponse == undefined) {
      throw ProvidersErrors.errProbeResponseUndefined;
    }

    Logger.debug(
      "Get latest block from providers ended",
      "latest block " + lastProbeResponse.getLavaEpoch()
    );

    // Return latest block from probe response
    return lastProbeResponse.getLavaEpoch();
  }

  // fetchLocalLavaPairingList uses local pairingList.json file to load lava providers
  private async fetchLocalLavaPairingList(path: string): Promise<any> {
    Logger.debug("Fetch pairing list from local config");

    try {
      const data = await fetchLavaPairing(path);
      return this.validatePairingData(data);
    } catch (err) {
      Logger.debug("Error happened in fetchLocalLavaPairingList", err);
      throw err;
    }
  }

  // fetchLocalLavaPairingList fetch lava pairing from seed providers list
  private async fetchDefaultLavaPairingList(): Promise<any> {
    Logger.debug("Fetch pairing list from seed providers in github");

    // Fetch lava providers from seed list
    const response = await fetch(DEFAULT_LAVA_PAIRING_LIST);

    // Validate response
    if (!response.ok) {
      throw Logger.fatal(
        `Unable to fetch pairing list: ${response.statusText}`
      );
    }

    try {
      // Parse response
      const data = await response.json();

      return this.validatePairingData(data);
    } catch (error) {
      throw Logger.fatal(
        "Error happened in fetchDefaultLavaPairingList",
        error
      );
    }
  }

  // constructLavaPairing constructs consumer session with provider list from pairing list
  private constructLavaPairing(
    pairingList: any
  ): ConsumerSessionsWithProvider[] {
    try {
      // Initialize ConsumerSessionWithProvider array
      const pairing: Array<ConsumerSessionsWithProvider> = [];

      for (const provider of pairingList) {
        const endpoint: Endpoint = {
          networkAddress: provider.rpcAddress,
          enabled: true,
          connectionRefusals: 0,
          addons: new Set<string>(),
          extensions: new Set<string>(),
        };

        // Create a new pairing object
        const newPairing = new ConsumerSessionsWithProvider(
          provider.publicAddress,
          [],
          {},
          1000,
          0
        );

        const randomSessionId = generateRandomInt();
        const singleConsumerSession = new SingleConsumerSession(
          randomSessionId,
          newPairing,
          endpoint
        );

        newPairing.sessions[0] = singleConsumerSession;

        // Add newly created pairing in the pairing list
        pairing.push(newPairing);
      }

      // Save lava providers
      this.lavaProviders = pairing;

      return pairing;
    } catch (err) {
      throw err;
    }
  }

  // validatePairingData validates pairing data
  private validatePairingData(data: any): any {
    if (data[this.config.network] == undefined) {
      throw Logger.fatal(
        `Unsupported network (${
          this.config.network
        }), supported networks: ${Object.keys(data)}`
      );
    }

    if (data[this.config.network][this.config.geolocation] == undefined) {
      throw Logger.fatal(
        `Unsupported geolocation (${this.config.geolocation}) for network (${
          this.config.network
        }). Supported geolocations: ${Object.keys(data[this.config.network])}`
      );
    }
    return data[this.config.network][this.config.geolocation];
  }
}

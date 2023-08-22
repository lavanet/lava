import {
  DEFAULT_LAVA_PAIRING_LIST,
  BOOT_RETRY_ATTEMPTS,
} from "../config/default";
import { BadgeManager } from "../badge/fetchBadge";
import { ChainIDRpcInterface, Config } from "./stateTracker";
import { fetchLavaPairing } from "../util/lavaPairing";
import { debugPrint } from "../util/common";

// TODO we can make relayer not default
import Relayer from "../relayer/relayer";

// TODO remove provider error when we refactor
import ProvidersErrors from "../lavaOverLava/errors";

// TODO once we refactor consumer session manager we should define new type
import {
  ConsumerSessionWithProvider,
  SingleConsumerSession,
  Endpoint,
} from "../types/types";

export class StateQuery {
  private badgeManager: BadgeManager;
  private pairingListConfig: string;
  private relayer: Relayer;
  private chainIDRpcInterfaces: ChainIDRpcInterface[];
  private lavaProviders: ConsumerSessionWithProvider[];
  private config: Config;

  // pairing is going to be a map with chainId as key and an array of ConsumerSessionWithProvider as value
  private pairing: Map<string, ConsumerSessionWithProvider[]>;

  // Constructor for state tracker
  constructor(
    badgeManager: BadgeManager,
    pairingListConfig: string,
    chainIdRpcInterfaces: ChainIDRpcInterface[],
    relayer: Relayer,
    config: Config
  ) {
    // Save arguments
    this.badgeManager = badgeManager;
    this.pairingListConfig = pairingListConfig;
    this.chainIDRpcInterfaces = chainIdRpcInterfaces;
    this.relayer = relayer;
    this.config = config;

    // Assign lavaProviders to an empty array
    this.lavaProviders = [];

    // Initialize pairing to an empty map
    this.pairing = new Map<string, ConsumerSessionWithProvider[]>();
  }

  //fetchLavaProviders fetches lava providers from different sources
  private async fetchLavaProviders(
    badgeManager: BadgeManager,
    pairingListConfig: string
  ): Promise<ConsumerSessionWithProvider[]> {
    // TODO once we refactor badge server, use it to fetch lava providers

    // If we have providers return them
    if (this.lavaProviders.length != 0) {
      return this.lavaProviders;
    }

    // Else if pairingListConfig exists use it to fetch lava providers from local file
    if (pairingListConfig != "") {
      const pairingList = await this.fetchLocalLavaPairingList(
        pairingListConfig
      );
      // Construct lava providers from pairing list and return it
      return this.constructLavaPairing(pairingList);
    }

    const pairingList = await this.fetchDefaultLavaPairingList();
    // Construct lava providers from pairing list and return it
    return this.constructLavaPairing(pairingList);
  }

  // fetchPairing fetches pairing for all chainIDs we support
  public async fetchPairing(): Promise<number> {
    // fetch lava providers
    await this.fetchLavaProviders(this.badgeManager, this.pairingListConfig);

    // Make sure lava providers are initialized
    if (this.lavaProviders == null) {
      throw ProvidersErrors.errLavaProvidersNotInitialized;
    }

    // Fetch latest block using probe
    const latestNumber = await this.getLatestBlockFromProviders(
      this.relayer,
      this.lavaProviders,
      "LAV1",
      "tendermintrpc"
    );

    this.pairing = new Map<string, ConsumerSessionWithProvider[]>();

    // For each chain
    for (const chainIDRpcInterface of this.chainIDRpcInterfaces) {
      // Send to all providers SdkPairing call
      // Create ConsumerSessionWithProvider[] from providers
      // Save for this chain providers
    }

    // After we have everything we will return timeTillNextEpoch and currentBlock
    return 3;
  }

  // getPairing return pairing list for specific chainID
  public getPairing(
    chainID: string
  ): ConsumerSessionWithProvider[] | undefined {
    // Return pairing for the specific chainId from the map
    return this.pairing.get(chainID);
  }

  // getLatestBlockFromProviders tries to fetch latest block using probe
  private async getLatestBlockFromProviders(
    relayer: Relayer,
    providers: ConsumerSessionWithProvider[],
    chainID: string,
    rpcInterface: string
  ): Promise<number> {
    let lastProbeResponse = null;
    for (let i = 0; i < providers.length; i++) {
      try {
        const probeResponse = await relayer.probeProvider(
          this.lavaProviders[i].Session.Endpoint.Addr,
          chainID,
          rpcInterface
        );

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

    if (lastProbeResponse instanceof Error) {
      throw lastProbeResponse;
    }

    if (lastProbeResponse == undefined) {
      throw ProvidersErrors.errProbeResponseUndefined;
    }

    return lastProbeResponse.getLatestBlock();
  }

  // validatePairingData validates pairing data
  private validatePairingData(data: any): any {
    if (data[this.config.network] == undefined) {
      throw new Error(
        `Unsupported network (${
          this.config.network
        }), supported networks: ${Object.keys(data)}`
      );
    }

    if (data[this.config.network][this.config.geolocation] == undefined) {
      throw new Error(
        `Unsupported geolocation (${this.config.geolocation}) for network (${
          this.config.network
        }). Supported geolocations: ${Object.keys(data[this.config.network])}`
      );
    }

    return data[this.config.network][this.config.geolocation];
  }

  // fetchLocalLavaPairingList uses local pairingList.json file to load lava providers
  private async fetchLocalLavaPairingList(path: string): Promise<any> {
    try {
      const data = await fetchLavaPairing(path);
      return this.validatePairingData(data);
    } catch (err) {
      throw err;
    }
  }

  // fetchLocalLavaPairingList fetch lava pairing from seed providers list
  private async fetchDefaultLavaPairingList(): Promise<any> {
    // Fetch lava providers from seed list
    const response = await fetch(DEFAULT_LAVA_PAIRING_LIST);

    // Validate response
    if (!response.ok) {
      throw new Error(`Unable to fetch pairing list: ${response.statusText}`);
    }

    try {
      // Parse response
      const data = await response.json();
      return this.validatePairingData(data);
    } catch (error) {
      throw error;
    }
  }

  // constructLavaPairing constructs consumer session with provider list from pairing list
  private constructLavaPairing(
    pairingList: any
  ): ConsumerSessionWithProvider[] {
    // Initialize ConsumerSessionWithProvider array
    const pairing: Array<ConsumerSessionWithProvider> = [];

    for (const provider of pairingList) {
      const singleConsumerSession = new SingleConsumerSession(
        0, // cuSum
        0, // latestRelayCuSum
        1, // relayNumber
        new Endpoint(provider.rpcAddress, true, 0),
        -1, //invalid epoch
        provider.publicAddress
      );

      // Create a new pairing object
      const newPairing = new ConsumerSessionWithProvider(
        this.config.accountAddress,
        [],
        singleConsumerSession,
        100000, // invalid max cu
        0, // used compute units
        false
      );

      // Add newly created pairing in the pairing list
      pairing.push(newPairing);
    }
    // Save lava providers
    this.lavaProviders = pairing;

    return pairing;
  }

  // SendRelayToAllProvidersAndRace sends relay to all lava providers and returns first response
  private async SendRelayToAllProvidersAndRace(
    providers: ConsumerSessionWithProvider[],
    options: any,
    relayCu: number,
    rpcInterface: string
  ): Promise<any> {
    let lastError;
    for (
      let retryAttempt = 0;
      retryAttempt < BOOT_RETRY_ATTEMPTS;
      retryAttempt++
    ) {
      const allRelays: Map<string, Promise<any>> = new Map();
      let response;
      for (const provider of providers) {
        const uniqueKey =
          provider.Session.ProviderAddress +
          String(Math.floor(Math.random() * 10000000));
        const providerRelayPromise = this.SendRelayWithRetry(
          options,
          provider,
          relayCu,
          rpcInterface
        )
          .then((result: any) => {
            debugPrint(this.config.debug, "response succeeded", result);
            response = result;
            allRelays.delete(uniqueKey);
          })
          .catch((err: any) => {
            debugPrint(
              this.config.debug,
              "one of the promises failed in SendRelayToAllProvidersAndRace reason:",
              err,
              uniqueKey
            );
            lastError = err;
            allRelays.delete(uniqueKey);
          });
        allRelays.set(uniqueKey, providerRelayPromise);
      }
      const promisesToWait = allRelays.size;
      for (let i = 0; i < promisesToWait; i++) {
        const returnedPromise = await Promise.race(allRelays);
        await returnedPromise[1];
        if (response) {
          debugPrint(
            this.config.debug,
            "SendRelayToAllProvidersAndRace, got response from one provider",
            response
          );
          return response;
        }
      }
      debugPrint(
        this.config.debug,
        "Failed all promises SendRelayToAllProvidersAndRace, trying again",
        retryAttempt,
        "out of",
        BOOT_RETRY_ATTEMPTS
      );
    }
    throw new Error(
      "Failed all promises SendRelayToAllProvidersAndRace: " + String(lastError)
    );
  }

  // SendRelayWithRetry tryes to send relay to provider and reties depending on errors
  private async SendRelayWithRetry(
    options: any,
    lavaRPCEndpoint: ConsumerSessionWithProvider,
    relayCu: number,
    rpcInterface: string
  ): Promise<any> {
    let response;
    if (this.relayer == null) {
      throw ProvidersErrors.errNoRelayer;
    }
    try {
      // For now we have hardcode relay cu
      response = await this.relayer.sendRelay(
        options,
        lavaRPCEndpoint,
        relayCu,
        rpcInterface
      );
    } catch (error) {
      // If error is instace of Error
      if (error instanceof Error) {
        // If error is not old blokc height throw and error
        // Extract current block height from error
        const currentBlockHeight = this.extractBlockNumberFromError(error);

        // If current block height equal nill throw an error
        if (currentBlockHeight == null) {
          throw error;
        }

        // Save current block height
        lavaRPCEndpoint.Session.PairingEpoch = parseInt(currentBlockHeight);

        // Retry same relay with added block height
        try {
          response = await this.relayer.sendRelay(
            options,
            lavaRPCEndpoint,
            relayCu,
            rpcInterface
          );
        } catch (error) {
          throw error;
        }
      }
    }

    // Validate that response is not undefined
    if (response == undefined) {
      return "";
    }

    // Decode response
    const dec = new TextDecoder();
    const decodedResponse = dec.decode(response.getData_asU8());

    // Parse response
    const jsonResponse = JSON.parse(decodedResponse);

    // Return response
    return jsonResponse;
  }

  // extractBlockNumberFromError extract block number from error if exists
  private extractBlockNumberFromError(error: Error): string | null {
    let currentBlockHeightRegex = /current epoch Value:(\d+)/;
    let match = error.message.match(currentBlockHeightRegex);

    // Retry with new error
    if (match == null) {
      currentBlockHeightRegex = /current epoch: (\d+)/; // older epoch parsing

      match = error.message.match(currentBlockHeightRegex);
      return match ? match[1] : null;
    }
    return match ? match[1] : null;
  }
}

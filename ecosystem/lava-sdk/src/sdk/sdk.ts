import { createWallet, createDynamicWallet } from "../wallet/wallet";
import SDKErrors from "./errors";
import { AccountData } from "@cosmjs/proto-signing";
import Relayer from "../relayer/relayer";
import { RelayReply } from "../grpc_web_services/lavanet/lava/pairing/relay_pb";
import {
  TimoutFailureFetchingBadgeError as TimeoutFailureFetchingBadgeError,
  BadgeOptions,
  BadgeManager,
} from "../badge/fetchBadge";
import { Badge } from "../grpc_web_services/lavanet/lava/pairing/relay_pb";
import { SessionManager, ConsumerSessionWithProvider } from "../types/types";
import {
  isValidChainID,
  fetchRpcInterface,
  validateRpcInterfaceWithChainID,
} from "../util/chains";
import { generateRPCData } from "../util/common";
import { LavaProviders } from "../lavaOverLava/providers";
import {
  LAVA_CHAIN_ID as LAVA_SPEC_ID,
  DEFAULT_LAVA_PAIRING_NETWORK,
  DEFAULT_GEOLOCATION,
  DEFAULT_LAVA_CHAINID,
} from "../config/default";
import { QueryShowAllChainsResponse } from "../codec/lavanet/lava/spec/query";
import { GenerateBadgeResponse } from "../grpc_web_services/lavanet/lava/pairing/badges_pb";
/**
 * Options for sending RPC relay.
 */
export interface SendRelayOptions {
  method: string; // Required: The RPC method to be called
  params: Array<any>; // Required: An array of parameters to be passed to the RPC method
}

/**
 * Options for sending Rest relay.
 */
export interface SendRestRelayOptions {
  method: string; // Required: The HTTP method to be used (e.g., "GET", "POST")
  url: string; // Required: The URL to send the request to
  // eslint-disable-next-line
  data?: Record<string, any>; // Optional: An object containing data to be sent in the request body (applicable for methods like "POST" and "PUT")
}

/**
 * Options for initializing the LavaSDK.
 */
export interface LavaSDKOptions {
  privateKey?: string; // Required: The private key of the staked Lava client for the specified chainID
  badge?: BadgeOptions; // Required: Public URL of badge server and ID of the project you want to connect. Remove privateKey if badge is enabled.
  chainID: string; // Required: The ID of the chain you want to query
  rpcInterface?: string; // Optional: The interface that will be used for sending relays
  pairingListConfig?: string; // Optional: The Lava pairing list config used for communicating with the Lava network
  network?: string; // Optional: The network from pairingListConfig to be used ["mainnet", "testnet"]
  geolocation?: string; // Optional: The geolocation to be used ["1" for North America, "2" for Europe ]
  lavaChainId?: string; // Optional: The Lava chain ID (default value for Lava Testnet)
  secure?: boolean; // Optional: communicates through https, this is a temporary flag that will be disabled once the chain will use https by default
  allowInsecureTransport?: boolean; // Optional: indicates to use a insecure transport when connecting the provider, this is used for testing purposes only and allows self-signed certificates to be used
  debug?: boolean; // Optional for debugging the LavaSDK mostly prints to speed up development
}

export class LavaSDK {
  private privKey: string;
  private walletAddress: string;
  private chainID: string;
  private rpcInterface: string;
  private network: string;
  private pairingListConfig: string;
  private geolocation: string;
  private lavaChainId: string;
  private badgeManager: BadgeManager;
  private currentEpochBadge: Badge | undefined; // The current badge is the badge for the current epoch

  private lavaProviders: LavaProviders | Error;
  private account: AccountData | Error;
  private relayer: Relayer | Error;
  private secure: boolean;
  private allowInsecureTransport: boolean;
  private debugMode: boolean;

  private activeSessionManager: SessionManager | Error;

  /**
   * Create Lava-SDK instance
   *
   * Use Lava-SDK for dAccess with a supported network. You can find a list of supported networks and their chain IDs at (url).
   *
   * @async
   * @param {LavaSDKOptions} options The options to use for initializing the LavaSDK.
   *
   * @returns A promise that resolves when the LavaSDK has been successfully initialized, returns LavaSDK object.
   */
  constructor(options: LavaSDKOptions) {
    // Extract attributes from options
    const { privateKey, badge, chainID, rpcInterface } = options;
    let { pairingListConfig, network, geolocation, lavaChainId } = options;

    // If network is not defined use default network
    network = network || DEFAULT_LAVA_PAIRING_NETWORK;

    // if lava chain id is not defined use default
    lavaChainId = lavaChainId || DEFAULT_LAVA_CHAINID;

    // If geolocation is not defined use default geolocation
    geolocation = geolocation || DEFAULT_GEOLOCATION;

    // If lava pairing config not defined set as empty
    pairingListConfig = pairingListConfig || "";

    if (!badge && !privateKey) {
      throw SDKErrors.errPrivKeyAndBadgeNotInitialized;
    } else if (badge && privateKey) {
      throw SDKErrors.errPrivKeyAndBadgeBothInitialized;
    }

    // Initialize local attributes
    this.debugMode = options.debug ? options.debug : false; // enabling debug prints mainly used for development / debugging
    this.secure = options.secure !== undefined ? options.secure : true;
    this.allowInsecureTransport = options.allowInsecureTransport
      ? options.allowInsecureTransport
      : false;
    this.debugPrint(
      "secure",
      this.secure,
      "allowInsecureTransport",
      this.allowInsecureTransport
    );
    this.chainID = chainID;
    this.rpcInterface = rpcInterface ? rpcInterface : "";
    this.privKey = privateKey ? privateKey : "";
    this.walletAddress = "";
    this.badgeManager = new BadgeManager(badge);
    this.network = network;
    this.geolocation = geolocation;
    this.lavaChainId = lavaChainId;
    this.pairingListConfig = pairingListConfig;
    this.account = SDKErrors.errAccountNotInitialized;
    this.relayer = SDKErrors.errRelayerServiceNotInitialized;
    this.lavaProviders = SDKErrors.errLavaProvidersNotInitialized;
    this.activeSessionManager = SDKErrors.errSessionNotInitialized;

    // Init sdk
    return (async (): Promise<LavaSDK> => {
      await this.init();

      return this;
    })() as unknown as LavaSDK;
  }

  /*
   * Initialize LavaSDK, by using :
   * let sdk = await LavaSDK.create({... options})
   */
  static async create(options: LavaSDKOptions): Promise<LavaSDK> {
    return await new LavaSDK(options);
  }

  static async generateKey() {
    const dynamicWallet = await createDynamicWallet();
    const lavaWallet = dynamicWallet.wallet;
    const accountData = await lavaWallet.getConsumerAccount();

    return {
      lavaAddress: accountData.address,
      privateKey: dynamicWallet.privKey,
      seedPhrase: dynamicWallet.seedPhrase,
    };
  }

  private debugPrint(message?: any, ...optionalParams: any[]) {
    this.debugMode && console.log(message, ...optionalParams);
  }

  private async fetchNewBadge(): Promise<GenerateBadgeResponse> {
    const badgeResponse = await this.badgeManager.fetchBadge(
      this.walletAddress
    );
    if (badgeResponse instanceof Error) {
      throw TimeoutFailureFetchingBadgeError;
    }
    return badgeResponse;
  }

  private async initLavaProviders(start: number) {
    if (this.account instanceof Error) {
      throw new Error("initLavaProviders failed: " + String(this.account));
    }
    // Create new instance of lava providers
    this.lavaProviders = await new LavaProviders({
      accountAddress: this.account.address,
      network: this.network,
      relayer: new Relayer(
        LAVA_SPEC_ID,
        this.privKey,
        this.lavaChainId,
        this.secure,
        this.allowInsecureTransport,
        this.currentEpochBadge
      ),
      geolocation: this.geolocation,
      debug: this.debugMode,
    });
    this.debugPrint("time took lava providers", performance.now() - start);

    // Init lava providers
    await this.lavaProviders.init(this.pairingListConfig);
    this.debugPrint("time took lava providers init", performance.now() - start);

    const parsedChainList: QueryShowAllChainsResponse =
      await this.lavaProviders.showAllChains();

    // Validate chainID
    if (!isValidChainID(this.chainID, parsedChainList)) {
      throw SDKErrors.errChainIDUnsupported;
    }
    this.debugPrint("time took ShowAllChains", performance.now() - start);

    // If rpc is not defined use default for specified chainID
    this.rpcInterface =
      this.rpcInterface || fetchRpcInterface(this.chainID, parsedChainList);
    this.debugPrint("time took fetchRpcInterface", performance.now() - start);

    // Validate rpc interface with chain id
    validateRpcInterfaceWithChainID(
      this.chainID,
      parsedChainList,
      this.rpcInterface
    );
    this.debugPrint(
      "time took validateRpcInterfaceWithChainID",
      performance.now() - start
    );

    // Get pairing list for current epoch
    this.activeSessionManager = await this.lavaProviders.getSession(
      this.chainID,
      this.rpcInterface,
      this.currentEpochBadge
    );

    this.debugPrint("time took getSession", performance.now() - start);
  }

  private async init() {
    const start = performance.now();
    if (this.badgeManager.isActive()) {
      const { wallet, privKey } = await createDynamicWallet();
      this.privKey = privKey;
      this.walletAddress = (await wallet.getConsumerAccount()).address;
      const badgeResponse = await this.fetchNewBadge();
      this.currentEpochBadge = badgeResponse.getBadge();
      const badgeSignerAddress = badgeResponse.getBadgeSignerAddress();
      this.account = {
        algo: "secp256k1",
        address: badgeSignerAddress,
        pubkey: new Uint8Array([]),
      };
      this.debugPrint(
        "time took to get badge from badge server",
        performance.now() - start
      );
      this.debugPrint("Badge:", badgeResponse);
      // this.debugPrint("badge", badge);
    } else {
      const wallet = await createWallet(this.privKey);
      this.account = await wallet.getConsumerAccount();
    }

    // Create relayer for querying network
    this.relayer = new Relayer(
      this.chainID,
      this.privKey,
      this.lavaChainId,
      this.secure,
      this.allowInsecureTransport,
      this.currentEpochBadge
    );

    await this.initLavaProviders(start);
  }

  private async handleRpcRelay(options: SendRelayOptions): Promise<string> {
    try {
      if (this.rpcInterface === "rest") {
        throw SDKErrors.errRPCRelayMethodNotSupported;
      }
      // Extract attributes from options
      const { method, params } = options;

      // Get cuSum for specified method
      const cuSum = this.getCuSumForMethod(method);

      const data = generateRPCData(method, params);

      // Check if relay was initialized
      if (this.relayer instanceof Error) {
        throw SDKErrors.errRelayerServiceNotInitialized;
      }

      // Construct send relay options
      const sendRelayOptions = {
        data: data,
        url: "",
        connectionType: this.rpcInterface === "jsonrpc" ? "POST" : "",
      };

      return await this.sendRelayWithRetries(sendRelayOptions, cuSum);
    } catch (err) {
      throw err;
    }
  }

  private async handleRestRelay(
    options: SendRestRelayOptions
  ): Promise<string> {
    try {
      if (this.rpcInterface !== "rest") {
        throw SDKErrors.errRestRelayMethodNotSupported;
      }

      // Extract attributes from options
      const { method, url, data } = options;

      // Get cuSum for specified method
      const cuSum = this.getCuSumForMethod(url);

      let query = "?";
      for (const key in data) {
        query = query + key + "=" + data[key] + "&";
      }

      // Construct send relay options
      const sendRelayOptions = {
        data: query,
        url: url,
        connectionType: method,
      };

      return await this.sendRelayWithRetries(sendRelayOptions, cuSum);
    } catch (err) {
      throw err;
    }
  }

  // sendRelayWithRetries iterates over provider list and tries to send relay
  // if no provider return result it will result the error from the last provider
  private async sendRelayWithRetries(options: any, cuSum: number) {
    // Check if relay was initialized
    if (this.relayer instanceof Error) {
      throw SDKErrors.errRelayerServiceNotInitialized;
    }
    let lastRelayResponse = null;
    // Fetching both pairing lists, one for our geolocation and the other for the other geo locations
    const [pairingList, extendedPairingList] =
      await this.getConsumerProviderSession();
    // because we iterate over the pairing list, we can merge then having pairingList first and extended 2nd
    pairingList.push(...extendedPairingList);
    if (pairingList.length == 0) {
      throw new Error(
        "sendRelayWithRetries couldn't find pairing list for this epoch."
      );
    }
    for (let i = 0; i < pairingList.length; i++) {
      try {
        // Send relay
        const relayResponse = await this.relayer.sendRelay(
          options,
          pairingList[i],
          cuSum,
          this.rpcInterface
        );

        // Return relay in json format
        return this.decodeRelayResponse(relayResponse);
      } catch (err) {
        // If error is instace of Error
        if (err instanceof Error) {
          // An error occurred during the sendRelay operation
          /*
          console.error(
            "Error during sendRelay: " + err.message,
            " from provider: " + pairingList[i].Session.Endpoint.Addr
          );
          */

          // Store the relay response
          lastRelayResponse = err;
        }
      }
    }

    // If an error occurred in all the operations, return the decoded response of the last operation
    throw lastRelayResponse;
  }

  /**
   * Send relay to network through providers.
   *
   * @async
   * @param options The options to use for sending relay.
   *
   * @returns A promise that resolves when the relay response has been returned, and returns a JSON string
   *
   */
  async sendRelay(
    options: SendRelayOptions | SendRestRelayOptions
  ): Promise<string> {
    if (this.isRest(options)) return await this.handleRestRelay(options);
    return await this.handleRpcRelay(options);
  }

  private decodeRelayResponse(relayResponse: RelayReply): string {
    // Decode relay response
    const dec = new TextDecoder();
    const decodedResponse = dec.decode(relayResponse.getData_asU8());

    return decodedResponse;
  }

  private getCuSumForMethod(method: string): number {
    // Check if activeSession was initialized
    if (this.activeSessionManager instanceof Error) {
      throw SDKErrors.errSessionNotInitialized;
    }
    // get cuSum for specified method
    const cuSum = this.activeSessionManager.getCuSumFromApi(
      method,
      this.chainID
    );

    // if cuSum is undefiend method does not exists in spec
    if (cuSum == undefined) {
      throw SDKErrors.errMethodNotSupported;
    }

    return cuSum;
  }

  // Return randomized pairing list, our geolocation index 0 and other geolocation index 1
  private async getConsumerProviderSession(): Promise<
    [ConsumerSessionWithProvider[], ConsumerSessionWithProvider[]]
  > {
    // Check if lava providers were initialized
    if (this.lavaProviders instanceof Error) {
      throw SDKErrors.errLavaProvidersNotInitialized;
    }

    // Check if state tracker was initialized
    if (this.account instanceof Error) {
      throw SDKErrors.errAccountNotInitialized;
    }

    // Check if activeSessionManager was initialized
    if (this.activeSessionManager instanceof Error) {
      throw SDKErrors.errSessionNotInitialized;
    }

    // Check if new epoch has started
    if (this.newEpochStarted()) {
      // fetch a new badge:
      if (this.badgeManager.isActive()) {
        const badgeResponse = await this.fetchNewBadge();
        this.currentEpochBadge = badgeResponse.getBadge();
        if (this.relayer instanceof Relayer) {
          this.relayer.setBadge(this.currentEpochBadge);
        }
        this.lavaProviders.updateLavaProvidersRelayersBadge(
          this.currentEpochBadge
        );
      }
      this.activeSessionManager = await this.lavaProviders.getSession(
        this.chainID,
        this.rpcInterface,
        this.currentEpochBadge
      );
    }

    // Return randomized pairing list, our geolocation index 0 and other geolocation index 1
    return [
      this.lavaProviders.pickRandomProviders(
        this.activeSessionManager.PairingList
      ),
      this.lavaProviders.pickRandomProviders(
        this.activeSessionManager.PairingListExtended
      ),
    ];
  }

  private newEpochStarted(): boolean {
    // Check if activeSession was initialized
    if (this.activeSessionManager instanceof Error) {
      throw SDKErrors.errSessionNotInitialized;
    }

    // Get current date and time
    const now = new Date();

    // Return if new epoch has started
    return now.getTime() > this.activeSessionManager.NextEpochStart.getTime();
  }

  private isRest(
    options: SendRelayOptions | SendRestRelayOptions
  ): options is SendRestRelayOptions {
    return (options as SendRestRelayOptions).url !== undefined;
  }
}

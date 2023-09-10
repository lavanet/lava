import SDKErrors from "./errors";
import { AccountData } from "@cosmjs/proto-signing";
import { Relayer } from "../relayer/relayer";
import { BadgeOptions, BadgeManager } from "../badge/badgeManager";
import {
  DEFAULT_LAVA_PAIRING_NETWORK,
  DEFAULT_GEOLOCATION,
  DEFAULT_LAVA_CHAINID,
} from "../config/default";
import { Logger, LogLevel } from "../logger/logger";
import { createWallet, createDynamicWallet } from "../wallet/wallet";
import { StateTracker } from "../stateTracker/state_tracker";
import { RPCConsumerServer } from "../consumer/rpcconsumer_server";
import { ConsumerSessionManager } from "../lavasession/consumerSessionManager";
import { RandomProviderOptimizer } from "../lavasession/providerOptimizer";
import { RPCEndpoint } from "../lavasession/consumerTypes";
import { getChainParser } from "../chainlib/common";

export type ChainIDsToInit = string | string[]; // chainId or an array of chain ids to initialize sdk for.
type RelayReceiver = string; // chainId + ApiInterface

/**
 * Options for initializing the LavaSDK.
 */
export interface LavaSDKOptions {
  privateKey?: string; // Required: The private key of the staked Lava client for the specified chainID
  badge?: BadgeOptions; // Required: Public URL of badge server and ID of the project you want to connect. Remove privateKey if badge is enabled.
  chainIDRpcInterface: ChainIDsToInit; // Required: The ID of the chain you want to query or an array of chain ids example "ETH1" | ["ETH1", "LAV1"]
  pairingListConfig?: string; // Optional: The Lava pairing list config used for communicating with the Lava network
  network?: string; // Optional: The network from pairingListConfig to be used ["mainnet", "testnet"]
  geolocation?: string; // Optional: The geolocation to be used ["1" for North America, "2" for Europe ]
  lavaChainId?: string; // Optional: The Lava chain ID (default value for Lava Testnet)
  secure?: boolean; // Optional: communicates through https, this is a temporary flag that will be disabled once the chain will use https by default
  allowInsecureTransport?: boolean; // Optional: indicates to use a insecure transport when connecting the provider, this is used for testing purposes only and allows self-signed certificates to be used
  logLevel?: string | LogLevel; // Optional for log level settings, "debug" | "info" | "warn" | "error" | "success" | "NoPrints"
  transport?: any; // Optional for transport settings if you would like to change the default transport settings. see utils/browser.ts for the current settings
}

export class LavaSDK {
  private privKey: string;
  private walletAddress: string;
  private network: string;
  private pairingListConfig: string;
  private geolocation: string;
  private lavaChainId: string;
  private badgeManager: BadgeManager;
  private account: AccountData | Error;
  private secure: boolean;
  private allowInsecureTransport: boolean;
  private chainIDRpcInterface: string[];
  private transport: any;
  private rpcConsumerServer: Map<string, RPCConsumerServer>;
  private relayer?: Relayer; // we setup the relayer in the init function as we require extra information

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
    const {
      privateKey,
      badge,
      chainIDRpcInterface,
      pairingListConfig,
      network,
      geolocation,
      lavaChainId,
    } = options;

    // Validate attributes
    if (!badge && !privateKey) {
      throw SDKErrors.errPrivKeyAndBadgeNotInitialized;
    } else if (badge && privateKey) {
      throw SDKErrors.errPrivKeyAndBadgeBothInitialized;
    }

    // Set log level
    Logger.SetLogLevel(options.logLevel);

    // Init attributes
    this.secure = options.secure !== undefined ? options.secure : true;
    this.allowInsecureTransport = options.allowInsecureTransport
      ? options.allowInsecureTransport
      : false;

    if (typeof chainIDRpcInterface == "string") {
      this.chainIDRpcInterface = [chainIDRpcInterface];
    } else {
      this.chainIDRpcInterface = chainIDRpcInterface;
    }
    this.privKey = privateKey ? privateKey : "";
    this.walletAddress = "";
    this.badgeManager = new BadgeManager(badge);
    this.network = network || DEFAULT_LAVA_PAIRING_NETWORK;
    this.geolocation = geolocation || DEFAULT_GEOLOCATION;
    this.lavaChainId = lavaChainId || DEFAULT_LAVA_CHAINID;
    this.pairingListConfig = pairingListConfig || "";
    this.account = SDKErrors.errAccountNotInitialized;
    this.transport = options.transport;

    this.rpcConsumerServer = new Map();
  }

  static async create(options: LavaSDKOptions): Promise<LavaSDK> {
    const sdkInstance = new LavaSDK(options);
    await sdkInstance.init();
    return sdkInstance;
  }

  public async init() {
    // Init wallet
    if (!this.badgeManager.isActive()) {
      const wallet = await createWallet(this.privKey);
      this.account = await wallet.getConsumerAccount();
    } else {
      const { wallet, privKey } = await createDynamicWallet();
      this.privKey = privKey;
      this.walletAddress = (await wallet.getConsumerAccount()).address;

      // We are updating this object when we fetch badge in state badge fetcher
      this.account = {
        algo: "secp256k1",
        address: "",
        pubkey: new Uint8Array([]),
      };
    }

    this.relayer = new Relayer({
      privKey: this.privKey,
      lavaChainId: this.lavaChainId,
      secure: this.secure,
      allowInsecureTransport: this.allowInsecureTransport,
      transport: this.transport,
    });

    // create provider optimizer
    const optimizer = new RandomProviderOptimizer();

    // Init state tracker
    const tracker = new StateTracker(
      this.pairingListConfig,
      this.relayer,
      this.chainIDRpcInterface,
      {
        geolocation: this.geolocation,
        network: this.network,
      },
      this.account,
      this.walletAddress,
      this.badgeManager
    );

    /// Fetch init state query
    await tracker.initialize();

    // init rpcconsumer servers
    for (const chainId of this.chainIDRpcInterface) {
      const pairingResponse = tracker.getPairingResponse(chainId);

      if (pairingResponse == undefined) {
        Logger.debug("No pairing list provided for chainID: ", chainId);
        continue;
      }

      for (const apiCollection of pairingResponse.spec.getApiCollectionsList()) {
        // Get api interface
        const apiInterface = apiCollection
          .getCollectionData()
          ?.getApiInterface();

        // Validate api interface
        if (apiInterface == undefined) {
          Logger.debug("No api interface in spec: ", chainId);
          continue;
        }

        // get rpc Endpoint
        const rpcEndpoint = new RPCEndpoint(
          "", // We do no need this in sdk as we are not opening any ports
          chainId,
          apiInterface,
          this.geolocation // This is also deprecated
        );

        // create consumer session manager
        const csm = new ConsumerSessionManager(
          this.relayer,
          rpcEndpoint,
          optimizer
        );

        // create chain parser
        const chainParse = getChainParser(apiInterface);

        // create rpc consumer server
        const rpcConsuemer = new RPCConsumerServer(
          this.relayer,
          csm,
          chainParse,
          this.geolocation
        );

        // save rpc consumer server in map
        this.rpcConsumerServer.set(chainId + apiInterface, rpcConsuemer);
      }
    }
  }
}

import SDKErrors from "./errors";
import { AccountData } from "@cosmjs/proto-signing";
import Relayer from "../relayer/relayer";
import { BadgeOptions, BadgeManager } from "../badge/badgeManager";
import {
  DEFAULT_LAVA_PAIRING_NETWORK,
  DEFAULT_GEOLOCATION,
  DEFAULT_LAVA_CHAINID,
} from "../config/default";
import { Logger, LogLevel } from "../logger/logger";
import { createWallet, createDynamicWallet } from "../wallet/wallet";
import { StateTracker } from "../stateTracker/state_tracker";
import { ConsumerSessionManagersMap } from "../lavasession/consumerSessionManager";

export interface ChainIDRpcInterface {
  chainID: string;
  rpcInterface: string;
}

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
  chainIDRpcInterface: ChainIDRpcInterface[]; // Required: The ID of the chain you want to query
  pairingListConfig?: string; // Optional: The Lava pairing list config used for communicating with the Lava network
  network?: string; // Optional: The network from pairingListConfig to be used ["mainnet", "testnet"]
  geolocation?: string; // Optional: The geolocation to be used ["1" for North America, "2" for Europe ]
  lavaChainId?: string; // Optional: The Lava chain ID (default value for Lava Testnet)
  secure?: boolean; // Optional: communicates through https, this is a temporary flag that will be disabled once the chain will use https by default
  allowInsecureTransport?: boolean; // Optional: indicates to use a insecure transport when connecting the provider, this is used for testing purposes only and allows self-signed certificates to be used
  logLevel?: string | LogLevel; // Optional for log level settings, "debug" | "info" | "warn" | "error" | "success" | "NoPrints"
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
  private chainIDRpcInterface: ChainIDRpcInterface[];
  private sessionsWithProviderMap: ConsumerSessionManagersMap;

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

    this.chainIDRpcInterface = chainIDRpcInterface;
    this.privKey = privateKey ? privateKey : "";
    this.walletAddress = "";
    this.badgeManager = new BadgeManager(badge);
    this.network = network || DEFAULT_LAVA_PAIRING_NETWORK;
    this.geolocation = geolocation || DEFAULT_GEOLOCATION;
    this.lavaChainId = lavaChainId || DEFAULT_LAVA_CHAINID;
    this.pairingListConfig = pairingListConfig || "";
    this.account = SDKErrors.errAccountNotInitialized;
    this.sessionsWithProviderMap = new Map();

    // Init sdk
    return (async (): Promise<LavaSDK> => {
      await this.init();

      return this;
    })() as unknown as LavaSDK;
  }

  private async init() {
    // Init relayer
    const relayer = new Relayer(
      this.privKey,
      this.lavaChainId,
      this.secure,
      this.allowInsecureTransport
    );

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

    // Init session manager map

    // Init state tracker
    const tracker = new StateTracker(
      this.pairingListConfig,
      relayer,
      this.chainIDRpcInterface,
      {
        geolocation: this.geolocation,
        network: this.network,
      },
      this.account,
      this.sessionsWithProviderMap,
      this.walletAddress,
      this.badgeManager
    );
    await tracker.initialize();
  }
}

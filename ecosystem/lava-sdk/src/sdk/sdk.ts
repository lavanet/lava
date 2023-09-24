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
import { RPCConsumerServer } from "../rpcconsumer/rpcconsumer_server";
import { ConsumerSessionManager } from "../lavasession/consumerSessionManager";
import { RandomProviderOptimizer } from "../lavasession/providerOptimizer";
import { RPCEndpoint } from "../lavasession/consumerTypes";
import { getChainParser } from "../chainlib/common";
import {
  APIInterfaceJsonRPC,
  APIInterfaceRest,
  APIInterfaceTendermintRPC,
  SendRelayOptions,
  SendRestRelayOptions,
} from "../chainlib/base_chain_parser";
import { FinalizationConsensus } from "../lavaprotocol/finalization_consensus";
import { getDefaultLavaSpec } from "../chainlib/default_lava_spec";

export type ChainIDsToInit = string | string[]; // chainId or an array of chain ids to initialize sdk for.
type RelayReceiver = string; // chainId + ApiInterface

/**
 * Options for initializing the LavaSDK.
 */
export interface LavaSDKOptions {
  privateKey?: string; // Required: The private key of the staked Lava client for the specified chainID
  badge?: BadgeOptions; // Required: Public URL of badge server and ID of the project you want to connect. Remove privateKey if badge is enabled.
  chainIds: ChainIDsToInit; // Required: The ID of the chain you want to query or an array of chain ids example "ETH1" | ["ETH1", "LAV1"]
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
  private rpcConsumerServerRouter: Map<RelayReceiver, RPCConsumerServer>; // routing the right chainID and apiInterface to rpc server
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
      chainIds: chainIDRpcInterface,
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

    this.rpcConsumerServerRouter = new Map();
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

    // Get default lava spec
    const spec = getDefaultLavaSpec();
    // create provider optimizer
    const optimizer = new RandomProviderOptimizer();

    let rpcConsumerServerLoL: RPCConsumerServer | undefined;
    let lavaTendermintRpcConsumerSessionManager:
      | ConsumerSessionManager
      | undefined;
    // if badge is not active set rpc consumer server for lava queries
    if (!this.badgeManager.isActive()) {
      const rpcEndpoint = new RPCEndpoint(
        "", // We do no need this in sdk as we are not opening any ports
        "LAV1",
        "tendermintrpc",
        this.geolocation // This is also deprecated
      );

      // create consumer session manager for lava tendermint
      lavaTendermintRpcConsumerSessionManager = new ConsumerSessionManager(
        this.relayer,
        rpcEndpoint,
        optimizer,
        {
          transport: this.transport,
          allowInsecureTransport: this.allowInsecureTransport,
        }
      );

      // Get chain parser for tendermintrpc
      const chainParse = getChainParser("tendermintrpc");

      // Init lava Spec
      chainParse.init(spec);

      const finalizationConsensus = new FinalizationConsensus();

      rpcConsumerServerLoL = new RPCConsumerServer(
        this.relayer,
        lavaTendermintRpcConsumerSessionManager,
        chainParse,
        this.geolocation,
        rpcEndpoint,
        this.lavaChainId,
        finalizationConsensus
      );
    }

    // Init state tracker
    const tracker = new StateTracker(
      this.pairingListConfig,
      this.relayer,
      this.chainIDRpcInterface,
      {
        geolocation: this.geolocation,
        network: this.network,
      },
      rpcConsumerServerLoL,
      spec,
      this.account,
      this.walletAddress,
      this.badgeManager
    );

    // Register LAVATendermint csm for update
    // If badge does not exists
    if (!this.badgeManager.isActive()) {
      if (!lavaTendermintRpcConsumerSessionManager) {
        throw Logger.fatal(
          "lavaTendermintRpcConsumerSessionManager is undefined in private key flow"
        );
      }

      tracker.RegisterConsumerSessionManagerForPairingUpdates(
        lavaTendermintRpcConsumerSessionManager
      );
    }

    // Fetch init state query
    await tracker.initialize();

    // init rpcconsumer servers
    for (const chainId of this.chainIDRpcInterface) {
      const pairingResponse = tracker.getPairingResponse(chainId);

      if (pairingResponse == undefined) {
        Logger.debug("No pairing list provided for chainID: ", chainId);
        continue;
      }
      const spec = pairingResponse.spec;
      const apiCollectionList = spec.getApiCollectionsList();
      for (const apiCollection of apiCollectionList) {
        // Get api interface
        if (!apiCollection.getEnabled()) {
          continue;
        }

        const apiInterface = apiCollection
          .getCollectionData()
          ?.getApiInterface();

        // Validate api interface
        if (apiInterface == undefined) {
          Logger.debug("No api interface in spec: ", chainId);
          continue;
        }

        if (apiInterface == "grpc") {
          Logger.debug("Skipping grpc for: ", chainId);
          continue;
        }

        // in case we have rest - POST + rest - GET collections this will prevent us from adding the same chainId and apiInterface twice
        if (
          !(
            this.getRpcConsumerServerRaw(chainId, apiInterface) instanceof Error
          )
        ) {
          continue;
        }

        // create chain parser
        const chainParser = getChainParser(apiInterface);
        chainParser.init(spec); // TODO: instead of init implement spec updater (update only when there was a spec change spec.getBlockLastUpdated())

        // set the existing rpc consumer server from the initialization instead of creating a new one and continue
        if (
          chainId == "LAV1" &&
          apiInterface == "tendermintrpc" &&
          rpcConsumerServerLoL
        ) {
          rpcConsumerServerLoL.setChainParser(chainParser);
          this.rpcConsumerServerRouter.set(
            this.getRouterKey(chainId, apiInterface),
            rpcConsumerServerLoL
          );
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
          optimizer,
          {
            transport: this.transport,
            allowInsecureTransport: this.allowInsecureTransport,
          }
        );

        tracker.RegisterConsumerSessionManagerForPairingUpdates(csm);

        // create finalization consensus
        const finalizationConsensus = new FinalizationConsensus();
        // TODO: when this is supported
        // tracker.RegisterFinalizationConsensusForUpdates(finalizationConsensus);

        // create rpc consumer server
        const rpcConsumerServer = new RPCConsumerServer(
          this.relayer,
          csm,
          chainParser,
          this.geolocation,
          rpcEndpoint,
          this.lavaChainId,
          finalizationConsensus
        );

        // save rpc consumer server in map
        this.rpcConsumerServerRouter.set(
          this.getRouterKey(chainId, apiInterface),
          rpcConsumerServer
        );
      }
    }
    await tracker.startTracking();
  }

  getRpcConsumerServer(
    options: SendRelayOptions | SendRestRelayOptions
  ): RPCConsumerServer | Error {
    const routerMap = this.rpcConsumerServerRouter;
    const chainID = options.chainId;
    if (routerMap.size == 1 && chainID == undefined) {
      const firstEntry = routerMap.values().next();
      if (firstEntry.done) {
        return new Error("returned empty routerMap");
      }
      return firstEntry.value;
    }
    const isRest = this.isRest(options);
    if (chainID == undefined) {
      let specId = "";
      let apiInterface = "";
      for (const rpcConsumerServer of routerMap.values()) {
        const supported = rpcConsumerServer.supportedChainAndApiInterface();
        if (specId != "" && specId != supported.specId) {
          return new Error(
            "optional chainID argument must be specified when initializing the lavaSDK with multiple chains"
          );
        }
        specId = supported.specId;
        if (isRest) {
          apiInterface = APIInterfaceRest;
          continue;
        }
        if (
          apiInterface != "" &&
          apiInterface != supported.apiInterface &&
          supported.apiInterface != APIInterfaceRest &&
          apiInterface != APIInterfaceRest
        ) {
          return new Error(
            "optional apiInterface argument must be specified when initializing the lavaSDK with a chain that has multiple apiInterfaces that support SendRelayOptions (tendermintrpc,jsonrpc)"
          );
        }
        apiInterface = supported.apiInterface;
      }
      return this.getRpcConsumerServerRaw(specId, apiInterface);
    } else {
      if (isRest || options.apiInterface != undefined) {
        let apiInterface: string;
        if (isRest) {
          apiInterface = APIInterfaceRest;
        } else if (options.apiInterface != undefined) {
          apiInterface = options.apiInterface;
        } else {
          return new Error("unreachable code");
        }
        return this.getRpcConsumerServerRaw(chainID, apiInterface);
      } else {
        // get here only if chainID is specified and apiInterface is not and it's not rest
        const jsonRpcConsumerServer = this.getRpcConsumerServerRaw(
          chainID,
          APIInterfaceJsonRPC
        );
        const tendermintRpcConsumerServer = this.getRpcConsumerServerRaw(
          chainID,
          APIInterfaceTendermintRPC
        );

        if (
          // check if it only has tendermintrpc
          jsonRpcConsumerServer instanceof Error &&
          tendermintRpcConsumerServer instanceof RPCConsumerServer
        ) {
          return tendermintRpcConsumerServer;
        } else if (
          // check if it only has jsonrpc
          tendermintRpcConsumerServer instanceof Error &&
          jsonRpcConsumerServer instanceof RPCConsumerServer
        ) {
          return jsonRpcConsumerServer;
        }
        return new Error(
          "optional apiInterface argument must be specified when initializing the lavaSDK with a chain that has multiple apiInterfaces that support SendRelayOptions (tendermintrpc,jsonrpc)"
        );
      }
    }
  }

  // the inner async function throws on relay error
  public async sendRelay(options: SendRelayOptions | SendRestRelayOptions) {
    const rpcConsumerServer = this.getRpcConsumerServer(options);
    if (rpcConsumerServer instanceof Error) {
      throw Logger.fatal(
        "Did not find relay receiver",
        rpcConsumerServer.message,
        "Check you initialized the chains properly",
        "Chain Requested",
        options?.chainId ?? JSON.stringify(this.rpcConsumerServerRouter.keys())
      );
    }
    const relayResult = rpcConsumerServer.sendRelay(options);
    return await relayResult.then((response) => {
      // // Decode response
      const reply = response.reply;
      if (reply == undefined) {
        throw new Error("empty reply");
      }
      const dec = new TextDecoder();
      const decodedResponse = dec.decode(reply.getData_asU8());
      // Parse response
      const jsonResponse = JSON.parse(decodedResponse);
      // Return response
      return jsonResponse;
    });
  }

  protected getRouterKey(chainId: string, apiInterface: string): RelayReceiver {
    return chainId + "," + apiInterface;
  }

  protected isRest(
    options: SendRelayOptions | SendRestRelayOptions
  ): options is SendRestRelayOptions {
    return "connectionType" in options; // how to check which options were given
  }

  private getRpcConsumerServerRaw(
    chainID: string,
    apiInterface: string
  ): RPCConsumerServer | Error {
    const routerMap = this.rpcConsumerServerRouter;
    const rpcConsumerServer = routerMap.get(
      this.getRouterKey(chainID, apiInterface)
    );
    if (rpcConsumerServer == undefined) {
      return new Error(
        "did not find rpcConsumerServer for " +
          this.getRouterKey(chainID, apiInterface)
      );
    }
    return rpcConsumerServer;
  }
}

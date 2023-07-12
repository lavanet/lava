"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.LavaSDK = void 0;
const wallet_1 = require("../wallet/wallet");
const errors_1 = __importDefault(require("./errors"));
const relayer_1 = __importDefault(require("../relayer/relayer"));
const fetchBadge_1 = require("../badge/fetchBadge");
const chains_1 = require("../util/chains");
const common_1 = require("../util/common");
const providers_1 = require("../lavaOverLava/providers");
const default_1 = require("../config/default");
class LavaSDK {
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
    constructor(options) {
        // Extract attributes from options
        const { privateKey, badge, chainID, rpcInterface } = options;
        let { pairingListConfig, network, geolocation, lavaChainId } = options;
        // If network is not defined use default network
        network = network || default_1.DEFAULT_LAVA_PAIRING_NETWORK;
        // if lava chain id is not defined use default
        lavaChainId = lavaChainId || default_1.DEFAULT_LAVA_CHAINID;
        // If geolocation is not defined use default geolocation
        geolocation = geolocation || default_1.DEFAULT_GEOLOCATION;
        // If lava pairing config not defined set as empty
        pairingListConfig = pairingListConfig || "";
        if (!badge && !privateKey) {
            throw errors_1.default.errPrivKeyAndBadgeNotInitialized;
        }
        else if (badge && privateKey) {
            throw errors_1.default.errPrivKeyAndBadgeBothInitialized;
        }
        // Initialize local attributes
        this.secure = options.secure ? options.secure : false;
        this.chainID = chainID;
        this.rpcInterface = rpcInterface ? rpcInterface : "";
        this.privKey = privateKey ? privateKey : "";
        this.walletAddress = "";
        this.badgeManager = new fetchBadge_1.BadgeManager(badge);
        this.network = network;
        this.geolocation = geolocation;
        this.lavaChainId = lavaChainId;
        this.pairingListConfig = pairingListConfig;
        this.account = errors_1.default.errAccountNotInitialized;
        this.relayer = errors_1.default.errRelayerServiceNotInitialized;
        this.lavaProviders = errors_1.default.errLavaProvidersNotInitialized;
        this.activeSessionManager = errors_1.default.errSessionNotInitialized;
        this.debugMode = options.debug ? options.debug : false; // enabling debug prints mainly used for development / debugging
        // Init sdk
        return (() => __awaiter(this, void 0, void 0, function* () {
            yield this.init();
            return this;
        }))();
    }
    /*
     * Initialize LavaSDK, by using :
     * let sdk = await LavaSDK.create({... options})
     */
    static create(options) {
        return __awaiter(this, void 0, void 0, function* () {
            return yield new LavaSDK(options);
        });
    }
    debugPrint(message, ...optionalParams) {
        if (this.debugMode) {
            console.log(message, ...optionalParams);
        }
    }
    fetchNewBadge() {
        return __awaiter(this, void 0, void 0, function* () {
            const badgeResponse = yield this.badgeManager.fetchBadge(this.walletAddress);
            if (badgeResponse instanceof Error) {
                throw fetchBadge_1.TimoutFailureFetchingBadgeError;
            }
            return badgeResponse;
        });
    }
    initLavaProviders(start) {
        return __awaiter(this, void 0, void 0, function* () {
            if (this.account instanceof Error) {
                throw new Error("initLavaProviders failed: " + String(this.account));
            }
            // Create new instance of lava providers
            this.lavaProviders = yield new providers_1.LavaProviders({
                accountAddress: this.account.address,
                network: this.network,
                relayer: new relayer_1.default(default_1.LAVA_CHAIN_ID, this.privKey, this.lavaChainId, this.secure, this.currentEpochBadge),
                geolocation: this.geolocation,
                debug: this.debugMode,
            });
            this.debugPrint("time took lava providers", performance.now() - start);
            // Init lava providers
            yield this.lavaProviders.init(this.pairingListConfig);
            this.debugPrint("time took lava providers init", performance.now() - start);
            const parsedChainList = yield this.lavaProviders.showAllChains();
            // Validate chainID
            if (!(0, chains_1.isValidChainID)(this.chainID, parsedChainList)) {
                throw errors_1.default.errChainIDUnsupported;
            }
            this.debugPrint("time took ShowAllChains", performance.now() - start);
            // If rpc is not defined use default for specified chainID
            this.rpcInterface =
                this.rpcInterface || (0, chains_1.fetchRpcInterface)(this.chainID, parsedChainList);
            this.debugPrint("time took fetchRpcInterface", performance.now() - start);
            // Validate rpc interface with chain id
            (0, chains_1.validateRpcInterfaceWithChainID)(this.chainID, parsedChainList, this.rpcInterface);
            this.debugPrint("time took validateRpcInterfaceWithChainID", performance.now() - start);
            // Get pairing list for current epoch
            this.activeSessionManager = yield this.lavaProviders.getSession(this.chainID, this.rpcInterface, this.currentEpochBadge);
            this.debugPrint("time took getSession", performance.now() - start);
        });
    }
    init() {
        return __awaiter(this, void 0, void 0, function* () {
            const start = performance.now();
            if (this.badgeManager.isActive()) {
                const { wallet, privKey } = yield (0, wallet_1.createDynamicWallet)();
                this.privKey = privKey;
                this.walletAddress = (yield wallet.getConsumerAccount()).address;
                const badgeResponse = yield this.fetchNewBadge();
                this.currentEpochBadge = badgeResponse.getBadge();
                const badgeSignerAddress = badgeResponse.getBadgeSignerAddress();
                this.account = {
                    algo: "secp256k1",
                    address: badgeSignerAddress,
                    pubkey: new Uint8Array([]),
                };
                this.debugPrint("time took to get badge from badge server", performance.now() - start);
                // this.debugPrint("badge", badge);
            }
            else {
                const wallet = yield (0, wallet_1.createWallet)(this.privKey);
                this.account = yield wallet.getConsumerAccount();
            }
            // Create relayer for querying network
            this.relayer = new relayer_1.default(this.chainID, this.privKey, this.lavaChainId, this.secure, this.currentEpochBadge);
            yield this.initLavaProviders(start);
        });
    }
    handleRpcRelay(options) {
        return __awaiter(this, void 0, void 0, function* () {
            try {
                if (this.rpcInterface === "rest") {
                    throw errors_1.default.errRPCRelayMethodNotSupported;
                }
                // Extract attributes from options
                const { method, params } = options;
                // Get pairing list
                const pairingList = yield this.getConsumerProviderSession();
                // Get cuSum for specified method
                const cuSum = this.getCuSumForMethod(method);
                const data = (0, common_1.generateRPCData)(method, params);
                // Check if relay was initialized
                if (this.relayer instanceof Error) {
                    throw errors_1.default.errRelayerServiceNotInitialized;
                }
                // Construct send relay options
                const sendRelayOptions = {
                    data: data,
                    url: "",
                    connectionType: this.rpcInterface === "jsonrpc" ? "POST" : "",
                };
                return yield this.sendRelayWithRetries(sendRelayOptions, pairingList, cuSum);
            }
            catch (err) {
                throw err;
            }
        });
    }
    handleRestRelay(options) {
        return __awaiter(this, void 0, void 0, function* () {
            try {
                if (this.rpcInterface !== "rest") {
                    throw errors_1.default.errRestRelayMethodNotSupported;
                }
                // Extract attributes from options
                const { method, url, data } = options;
                // Get pairing list
                const pairingList = yield this.getConsumerProviderSession();
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
                return yield this.sendRelayWithRetries(sendRelayOptions, pairingList, cuSum);
            }
            catch (err) {
                throw err;
            }
        });
    }
    // sendRelayWithRetries iterates over provider list and tries to send relay
    // if no provider return result it will result the error from the last provider
    sendRelayWithRetries(options, pairingList, cuSum) {
        return __awaiter(this, void 0, void 0, function* () {
            // Check if relay was initialized
            if (this.relayer instanceof Error) {
                throw errors_1.default.errRelayerServiceNotInitialized;
            }
            let lastRelayResponse = null;
            for (let i = 0; i <= pairingList.length; i++) {
                try {
                    // Send relay
                    const relayResponse = yield this.relayer.sendRelay(options, pairingList[i], cuSum, this.rpcInterface);
                    // Return relay in json format
                    return this.decodeRelayResponse(relayResponse);
                }
                catch (err) {
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
        });
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
    sendRelay(options) {
        return __awaiter(this, void 0, void 0, function* () {
            if (this.isRest(options))
                return yield this.handleRestRelay(options);
            return yield this.handleRpcRelay(options);
        });
    }
    decodeRelayResponse(relayResponse) {
        // Decode relay response
        const dec = new TextDecoder();
        const decodedResponse = dec.decode(relayResponse.getData_asU8());
        return decodedResponse;
    }
    getCuSumForMethod(method) {
        // Check if activeSession was initialized
        if (this.activeSessionManager instanceof Error) {
            throw errors_1.default.errSessionNotInitialized;
        }
        // get cuSum for specified method
        const cuSum = this.activeSessionManager.getCuSumFromApi(method, this.chainID);
        // if cuSum is undefiend method does not exists in spec
        if (cuSum == undefined) {
            throw errors_1.default.errMethodNotSupported;
        }
        return cuSum;
    }
    getConsumerProviderSession() {
        return __awaiter(this, void 0, void 0, function* () {
            // Check if lava providers were initialized
            if (this.lavaProviders instanceof Error) {
                throw errors_1.default.errLavaProvidersNotInitialized;
            }
            // Check if state tracker was initialized
            if (this.account instanceof Error) {
                throw errors_1.default.errAccountNotInitialized;
            }
            // Check if activeSessionManager was initialized
            if (this.activeSessionManager instanceof Error) {
                throw errors_1.default.errSessionNotInitialized;
            }
            // Check if new epoch has started
            if (this.newEpochStarted()) {
                // fetch a new badge:
                if (this.badgeManager.isActive()) {
                    const badgeResponse = yield this.fetchNewBadge();
                    this.currentEpochBadge = badgeResponse.getBadge();
                    if (this.relayer instanceof relayer_1.default) {
                        this.relayer.setBadge(this.currentEpochBadge);
                    }
                    this.lavaProviders.updateLavaProvidersRelayersBadge(this.currentEpochBadge);
                }
                this.activeSessionManager = yield this.lavaProviders.getSession(this.chainID, this.rpcInterface, this.currentEpochBadge);
            }
            // Return randomized pairing list
            return this.lavaProviders.pickRandomProviders(this.activeSessionManager.PairingList);
        });
    }
    newEpochStarted() {
        // Check if activeSession was initialized
        if (this.activeSessionManager instanceof Error) {
            throw errors_1.default.errSessionNotInitialized;
        }
        // Get current date and time
        const now = new Date();
        // Return if new epoch has started
        return now.getTime() > this.activeSessionManager.NextEpochStart.getTime();
    }
    isRest(options) {
        return options.url !== undefined;
    }
}
exports.LavaSDK = LavaSDK;

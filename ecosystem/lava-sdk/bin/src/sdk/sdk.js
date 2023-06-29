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
const providers_1 = require("../lavaOverLava/providers");
const default_1 = require("../config/default");
const query_1 = require("../codec/spec/query");
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
        this.base64ToUint8Array = (str) => {
            const buffer = Buffer.from(str, "base64");
            return new Uint8Array(buffer);
        };
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
        this.badge = badge ? badge : { badgeServerAddress: "", projectId: "" };
        this.network = network;
        this.geolocation = geolocation;
        this.lavaChainId = lavaChainId;
        this.pairingListConfig = pairingListConfig;
        this.account = errors_1.default.errAccountNotInitialized;
        this.relayer = errors_1.default.errRelayerServiceNotInitialized;
        this.lavaProviders = errors_1.default.errLavaProvidersNotInitialized;
        this.activeSessionManager = errors_1.default.errSessionNotInitialized;
        this.isBadge = Boolean(badge);
        // Init sdk
        return (() => __awaiter(this, void 0, void 0, function* () {
            yield this.init();
            return this;
        }))();
    }
    init() {
        return __awaiter(this, void 0, void 0, function* () {
            let wallet;
            let badge;
            if (this.isBadge) {
                const { wallet, privKey } = yield (0, wallet_1.createDynamicWallet)();
                this.privKey = privKey;
                const walletAddress = (yield wallet.getConsumerAccount()).address;
                const badgeResponse = yield (0, fetchBadge_1.fetchBadge)(this.badge.badgeServerAddress, walletAddress, this.badge.projectId);
                badge = badgeResponse.getBadge();
                const badgeSignerAddress = badgeResponse.getBadgeSignerAddress();
                this.account = {
                    algo: "secp256k1",
                    address: badgeSignerAddress,
                    pubkey: new Uint8Array([]),
                };
            }
            else {
                wallet = yield (0, wallet_1.createWallet)(this.privKey);
                this.account = yield wallet.getConsumerAccount();
            }
            // Init relayer for lava providers
            const lavaRelayer = new relayer_1.default(default_1.LAVA_CHAIN_ID, this.privKey, this.lavaChainId, this.secure, badge);
            // Create new instance of lava providers
            const lavaProviders = yield new providers_1.LavaProviders(this.account.address, this.network, lavaRelayer, this.geolocation);
            // Init lava providers
            yield lavaProviders.init(this.pairingListConfig);
            const sendRelayOptions = {
                data: this.generateRPCData("abci_query", [
                    "/lavanet.lava.spec.Query/ShowAllChains",
                    "",
                    "0",
                    false,
                ]),
                url: "",
                connectionType: "",
            };
            const info = yield lavaProviders.SendRelayWithRetry(sendRelayOptions, lavaProviders.GetNextLavaProvider(), 10, "tendermintrpc");
            const byteArrayResponse = this.base64ToUint8Array(info.result.response.value);
            const parsedChainList = query_1.QueryShowAllChainsResponse.decode(byteArrayResponse);
            // Validate chainID
            if (!(0, chains_1.isValidChainID)(this.chainID, parsedChainList)) {
                throw errors_1.default.errChainIDUnsupported;
            }
            // If rpc is not defined use default for specified chainID
            this.rpcInterface =
                this.rpcInterface || (0, chains_1.fetchRpcInterface)(this.chainID, parsedChainList);
            // Validate rpc interface with chain id
            (0, chains_1.validateRpcInterfaceWithChainID)(this.chainID, parsedChainList, this.rpcInterface);
            // Save lava providers as local attribute
            this.lavaProviders = lavaProviders;
            // Get pairing list for current epoch
            this.activeSessionManager = yield this.lavaProviders.getSession(this.chainID, this.rpcInterface);
            // Create relayer for querying network
            this.relayer = new relayer_1.default(this.chainID, this.privKey, this.lavaChainId, this.secure, badge);
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
                const data = this.generateRPCData(method, params);
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
    generateRPCData(method, params) {
        const stringifyMethod = JSON.stringify(method);
        const stringifyParam = JSON.stringify(params, (key, value) => {
            if (typeof value === "bigint") {
                return value.toString();
            }
            return value;
        });
        // TODO make id changable
        return ('{"jsonrpc": "2.0", "id": 1, "method": ' +
            stringifyMethod +
            ', "params": ' +
            stringifyParam +
            "}");
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
                this.activeSessionManager = yield this.lavaProviders.getSession(this.chainID, this.rpcInterface);
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

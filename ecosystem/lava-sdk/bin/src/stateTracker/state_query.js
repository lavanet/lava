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
exports.StateQuery = void 0;
const default_1 = require("../config/default");
const lavaPairing_1 = require("../util/lavaPairing");
const common_1 = require("../util/common");
// TODO remove provider error when we refactor
const errors_1 = __importDefault(require("../lavaOverLava/errors"));
// TODO once we refactor consumer session manager we should define new type
const types_1 = require("../types/types");
class StateQuery {
    // Constructor for state tracker
    constructor(badgeManager, pairingListConfig, chainIdRpcInterfaces, relayer, config) {
        // Save arguments
        this.badgeManager = badgeManager;
        this.pairingListConfig = pairingListConfig;
        this.chainIDRpcInterfaces = chainIdRpcInterfaces;
        this.relayer = relayer;
        this.config = config;
        // Assign lavaProviders to an empty array
        this.lavaProviders = [];
        // Initialize pairing to an empty map
        this.pairing = new Map();
    }
    //fetchLavaProviders fetches lava providers from different sources
    fetchLavaProviders(badgeManager, pairingListConfig) {
        return __awaiter(this, void 0, void 0, function* () {
            // TODO once we refactor badge server, use it to fetch lava providers
            // If we have providers return them
            if (this.lavaProviders.length != 0) {
                return this.lavaProviders;
            }
            // Else if pairingListConfig exists use it to fetch lava providers from local file
            if (pairingListConfig != "") {
                const pairingList = yield this.fetchLocalLavaPairingList(pairingListConfig);
                // Construct lava providers from pairing list and return it
                return this.constructLavaPairing(pairingList);
            }
            const pairingList = yield this.fetchDefaultLavaPairingList();
            // Construct lava providers from pairing list and return it
            return this.constructLavaPairing(pairingList);
        });
    }
    // fetchPairing fetches pairing for all chainIDs we support
    fetchPairing() {
        return __awaiter(this, void 0, void 0, function* () {
            // fetch lava providers
            yield this.fetchLavaProviders(this.badgeManager, this.pairingListConfig);
            // Make sure lava providers are initialized
            if (this.lavaProviders == null) {
                throw errors_1.default.errLavaProvidersNotInitialized;
            }
            // Fetch latest block using probe
            const latestNumber = yield this.getLatestBlockFromProviders(this.relayer, this.lavaProviders, "LAV1", "tendermintrpc");
            this.pairing = new Map();
            // For each chain
            for (const chainIDRpcInterface of this.chainIDRpcInterfaces) {
                // Send to all providers SdkPairing call
                // Create ConsumerSessionWithProvider[] from providers
                // Save for this chain providers
            }
            // After we have everything we will return timeTillNextEpoch and currentBlock
            return 3;
        });
    }
    // getPairing return pairing list for specific chainID
    getPairing(chainID) {
        // Return pairing for the specific chainId from the map
        return this.pairing.get(chainID);
    }
    // getLatestBlockFromProviders tries to fetch latest block using probe
    getLatestBlockFromProviders(relayer, providers, chainID, rpcInterface) {
        return __awaiter(this, void 0, void 0, function* () {
            let lastProbeResponse = null;
            for (let i = 0; i < providers.length; i++) {
                try {
                    const probeResponse = yield relayer.probeProvider(this.lavaProviders[i].Session.Endpoint.Addr, chainID, rpcInterface);
                    lastProbeResponse = probeResponse;
                    break;
                }
                catch (err) {
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
                throw errors_1.default.errProbeResponseUndefined;
            }
            return lastProbeResponse.getLatestBlock();
        });
    }
    // validatePairingData validates pairing data
    validatePairingData(data) {
        if (data[this.config.network] == undefined) {
            throw new Error(`Unsupported network (${this.config.network}), supported networks: ${Object.keys(data)}`);
        }
        if (data[this.config.network][this.config.geolocation] == undefined) {
            throw new Error(`Unsupported geolocation (${this.config.geolocation}) for network (${this.config.network}). Supported geolocations: ${Object.keys(data[this.config.network])}`);
        }
        return data[this.config.network][this.config.geolocation];
    }
    // fetchLocalLavaPairingList uses local pairingList.json file to load lava providers
    fetchLocalLavaPairingList(path) {
        return __awaiter(this, void 0, void 0, function* () {
            try {
                const data = yield (0, lavaPairing_1.fetchLavaPairing)(path);
                return this.validatePairingData(data);
            }
            catch (err) {
                throw err;
            }
        });
    }
    // fetchLocalLavaPairingList fetch lava pairing from seed providers list
    fetchDefaultLavaPairingList() {
        return __awaiter(this, void 0, void 0, function* () {
            // Fetch lava providers from seed list
            const response = yield fetch(default_1.DEFAULT_LAVA_PAIRING_LIST);
            // Validate response
            if (!response.ok) {
                throw new Error(`Unable to fetch pairing list: ${response.statusText}`);
            }
            try {
                // Parse response
                const data = yield response.json();
                return this.validatePairingData(data);
            }
            catch (error) {
                throw error;
            }
        });
    }
    // constructLavaPairing constructs consumer session with provider list from pairing list
    constructLavaPairing(pairingList) {
        // Initialize ConsumerSessionWithProvider array
        const pairing = [];
        for (const provider of pairingList) {
            const singleConsumerSession = new types_1.SingleConsumerSession(0, // cuSum
            0, // latestRelayCuSum
            1, // relayNumber
            new types_1.Endpoint(provider.rpcAddress, true, 0), -1, //invalid epoch
            provider.publicAddress);
            // Create a new pairing object
            const newPairing = new types_1.ConsumerSessionWithProvider(this.config.accountAddress, [], singleConsumerSession, 100000, // invalid max cu
            0, // used compute units
            false);
            // Add newly created pairing in the pairing list
            pairing.push(newPairing);
        }
        // Save lava providers
        this.lavaProviders = pairing;
        return pairing;
    }
    // SendRelayToAllProvidersAndRace sends relay to all lava providers and returns first response
    SendRelayToAllProvidersAndRace(providers, options, relayCu, rpcInterface) {
        return __awaiter(this, void 0, void 0, function* () {
            let lastError;
            for (let retryAttempt = 0; retryAttempt < default_1.BOOT_RETRY_ATTEMPTS; retryAttempt++) {
                const allRelays = new Map();
                let response;
                for (const provider of providers) {
                    const uniqueKey = provider.Session.ProviderAddress +
                        String(Math.floor(Math.random() * 10000000));
                    const providerRelayPromise = this.SendRelayWithRetry(options, provider, relayCu, rpcInterface)
                        .then((result) => {
                        (0, common_1.debugPrint)(this.config.debug, "response succeeded", result);
                        response = result;
                        allRelays.delete(uniqueKey);
                    })
                        .catch((err) => {
                        (0, common_1.debugPrint)(this.config.debug, "one of the promises failed in SendRelayToAllProvidersAndRace reason:", err, uniqueKey);
                        lastError = err;
                        allRelays.delete(uniqueKey);
                    });
                    allRelays.set(uniqueKey, providerRelayPromise);
                }
                const promisesToWait = allRelays.size;
                for (let i = 0; i < promisesToWait; i++) {
                    const returnedPromise = yield Promise.race(allRelays);
                    yield returnedPromise[1];
                    if (response) {
                        (0, common_1.debugPrint)(this.config.debug, "SendRelayToAllProvidersAndRace, got response from one provider", response);
                        return response;
                    }
                }
                (0, common_1.debugPrint)(this.config.debug, "Failed all promises SendRelayToAllProvidersAndRace, trying again", retryAttempt, "out of", default_1.BOOT_RETRY_ATTEMPTS);
            }
            throw new Error("Failed all promises SendRelayToAllProvidersAndRace: " + String(lastError));
        });
    }
    // SendRelayWithRetry tryes to send relay to provider and reties depending on errors
    SendRelayWithRetry(options, lavaRPCEndpoint, relayCu, rpcInterface) {
        return __awaiter(this, void 0, void 0, function* () {
            let response;
            if (this.relayer == null) {
                throw errors_1.default.errNoRelayer;
            }
            try {
                // For now we have hardcode relay cu
                response = yield this.relayer.sendRelay(options, lavaRPCEndpoint, relayCu, rpcInterface);
            }
            catch (error) {
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
                        response = yield this.relayer.sendRelay(options, lavaRPCEndpoint, relayCu, rpcInterface);
                    }
                    catch (error) {
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
        });
    }
    // extractBlockNumberFromError extract block number from error if exists
    extractBlockNumberFromError(error) {
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
exports.StateQuery = StateQuery;

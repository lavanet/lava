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
exports.LavaProviders = void 0;
const default_1 = require("../config/default");
const types_1 = require("../types/types");
const query_1 = require("../codec/pairing/query");
const query_2 = require("../codec/spec/query");
const lavaPairing_1 = require("../util/lavaPairing");
const errors_1 = __importDefault(require("./errors"));
const common_1 = require("../util/common");
const query_3 = require("../codec/spec/query");
const BOOT_RETRY_ATTEMPTS = 2;
class LavaProviders {
    constructor(options) {
        this.index = 0;
        this.providers = [];
        this.network = options.network;
        this.accountAddress = options.accountAddress;
        this.relayer = options.relayer;
        this.geolocation = options.geolocation;
        this.debugMode = options.debug ? options.debug : false;
    }
    updateLavaProvidersRelayersBadge(badge) {
        if (this.relayer) {
            this.relayer.setBadge(badge);
        }
    }
    init(pairingListConfig) {
        return __awaiter(this, void 0, void 0, function* () {
            let data;
            // if no pairing list config use default
            if (pairingListConfig == "") {
                data = yield this.initDefaultConfig();
            }
            else {
                // Else use local config file
                data = yield this.initLocalConfig(pairingListConfig);
            }
            // Initialize ConsumerSessionWithProvider array
            const pairing = [];
            for (const provider of data) {
                const singleConsumerSession = new types_1.SingleConsumerSession(0, // cuSum
                0, // latestRelayCuSum
                1, // relayNumber
                new types_1.Endpoint(provider.rpcAddress, true, 0), -1, //invalid epoch
                provider.publicAddress);
                // Create a new pairing object
                const newPairing = new types_1.ConsumerSessionWithProvider(this.accountAddress, [], singleConsumerSession, 100000, // invalid max cu
                0, // used compute units
                false);
                // Add newly created pairing in the pairing list
                pairing.push(newPairing);
            }
            // Save providers as local attribute
            this.providers = pairing;
        });
    }
    showAllChains() {
        return __awaiter(this, void 0, void 0, function* () {
            const sendRelayOptions = {
                data: (0, common_1.generateRPCData)("abci_query", [
                    "/lavanet.lava.spec.Query/ShowAllChains",
                    "",
                    "0",
                    false,
                ]),
                url: "",
                connectionType: "",
            };
            const info = yield this.SendRelayToAllProvidersAndRace(sendRelayOptions, 10, "tendermintrpc");
            const byteArrayResponse = (0, common_1.base64ToUint8Array)(info.result.response.value);
            const response = query_3.QueryShowAllChainsResponse.decode(byteArrayResponse);
            return response;
        });
    }
    initDefaultConfig() {
        return __awaiter(this, void 0, void 0, function* () {
            // Fetch config from github repo
            const response = yield fetch(default_1.DEFAULT_LAVA_PAIRING_LIST);
            // Validate response
            if (!response.ok) {
                throw new Error(`Unable to fetch pairing list: ${response.statusText}`);
            }
            try {
                // Parse response
                const data = yield response.json();
                if (data[this.network] == undefined) {
                    throw new Error(`Unsupported network (${this.network}), supported networks: ${Object.keys(data)}, seed pairing list used`);
                }
                if (data[this.network][this.geolocation] == undefined) {
                    throw new Error(`Unsupported geolocation (${this.geolocation}) for network (${this.network}). Supported geolocations: ${Object.keys(data[this.network])}, seed pairing list used`);
                }
                // Return data array
                return data[this.network][this.geolocation];
            }
            catch (error) {
                throw error;
            }
        });
    }
    initLocalConfig(path) {
        return __awaiter(this, void 0, void 0, function* () {
            try {
                const data = yield (0, lavaPairing_1.fetchLavaPairing)(path);
                if (data[this.network] == undefined) {
                    throw new Error(`Unsupported network (${this.network}), supported networks: ${Object.keys(data)}, local pairing list used`);
                }
                if (data[this.network][this.geolocation] == undefined) {
                    throw new Error(`Unsupported geolocation (${this.geolocation}) for network (${this.network}). Supported geolocations: ${Object.keys(data[this.network])}, local pairing list used`);
                }
                return data[this.network][this.geolocation];
            }
            catch (err) {
                throw err;
            }
        });
    }
    // GetLavaProviders returns lava providers list
    GetLavaProviders() {
        if (this.providers.length == 0) {
            throw errors_1.default.errNoProviders;
        }
        return this.providers;
    }
    // GetNextLavaProvider returns lava providers used for fetching epoch
    // in round-robin fashion
    GetNextLavaProvider() {
        if (this.providers.length == 0) {
            throw errors_1.default.errNoProviders;
        }
        const rpcAddress = this.providers[this.index];
        this.index = (this.index + 1) % this.providers.length;
        return rpcAddress;
    }
    // getSession returns providers for current epoch
    getSession(chainID, rpcInterface, badge) {
        return __awaiter(this, void 0, void 0, function* () {
            if (this.providers == null) {
                throw errors_1.default.errLavaProvidersNotInitialized;
            }
            // Create request for fetching api methods for LAV1
            const lavaApis = yield this.getServiceApis({ ChainID: "LAV1" }, "grpc", new Map([["lavanet.lava.spec.Query/Spec", 10]]));
            // Create request for getServiceApis method for chainID
            const apis = yield this.getServiceApis({ ChainID: chainID }, rpcInterface, lavaApis);
            // Create pairing request for getPairing method
            const pairingRequest = {
                chainID: chainID,
                client: this.accountAddress,
            };
            // Get pairing from the chain
            const pairingResponse = yield this.getPairingFromChain(pairingRequest, lavaApis);
            // Set when will next epoch start
            const nextEpochStart = new Date();
            nextEpochStart.setSeconds(nextEpochStart.getSeconds() +
                parseInt(pairingResponse.timeLeftToNextPairing));
            // Extract providers from pairing response
            const providers = pairingResponse.providers;
            // Initialize ConsumerSessionWithProvider array
            const pairing = [];
            // Create request for getting userEntity
            const userEntityRequest = {
                address: this.accountAddress,
                chainID: chainID,
                block: pairingResponse.currentEpoch,
            };
            // Fetch max compute units
            let maxcu;
            if (badge) {
                maxcu = badge.getCuAllocation();
            }
            else {
                maxcu = yield this.getMaxCuForUser(userEntityRequest, lavaApis);
            }
            // Iterate over providers to populate pairing list
            for (const provider of providers) {
                // Skip providers with no endpoints
                if (provider.endpoints.length == 0) {
                    continue;
                }
                // Initialize relevantEndpoints array
                const relevantEndpoints = [];
                // Only take into account endpoints that use the same api interface
                // And geolocation
                for (const endpoint of provider.endpoints) {
                    if (endpoint.useType == rpcInterface &&
                        endpoint.geolocation == this.geolocation) {
                        const convertedEndpoint = new types_1.Endpoint(endpoint.iPPORT, true, 0);
                        relevantEndpoints.push(convertedEndpoint);
                    }
                }
                // Skip providers with no relevant endpoints
                if (relevantEndpoints.length == 0) {
                    continue;
                }
                const singleConsumerSession = new types_1.SingleConsumerSession(0, // cuSum
                0, // latestRelayCuSum
                1, // relayNumber
                relevantEndpoints[0], parseInt(pairingResponse.currentEpoch), provider.address);
                // Create a new pairing object
                const newPairing = new types_1.ConsumerSessionWithProvider(this.accountAddress, relevantEndpoints, singleConsumerSession, maxcu, 0, // used compute units
                false);
                // Add newly created pairing in the pairing list
                pairing.push(newPairing);
            }
            // Create session object
            const sessionManager = new types_1.SessionManager(pairing, nextEpochStart, apis);
            return sessionManager;
        });
    }
    debugPrint(message, ...optionalParams) {
        if (this.debugMode) {
            console.log(message, ...optionalParams);
        }
    }
    pickRandomProviders(providers) {
        // Remove providers which does not match criteria
        this.debugPrint("pickRandomProviders Providers list", providers);
        const validProviders = providers.filter((item) => item.MaxComputeUnits > item.UsedComputeUnits);
        if (validProviders.length === 0) {
            throw errors_1.default.errNoValidProvidersForCurrentEpoch;
        }
        // Fisher-Yates shuffle
        for (let i = validProviders.length - 1; i > 0; i--) {
            const j = Math.floor(Math.random() * (i + 1));
            [validProviders[i], validProviders[j]] = [
                validProviders[j],
                validProviders[i],
            ];
        }
        return validProviders;
    }
    pickRandomProvider(providers) {
        this.debugPrint("pickRandomProvider Providers list", providers);
        // Remove providers which does not match criteria
        const validProviders = providers.filter((item) => item.MaxComputeUnits > item.UsedComputeUnits);
        if (validProviders.length === 0) {
            throw errors_1.default.errNoValidProvidersForCurrentEpoch;
        }
        // Pick random provider
        const random = Math.floor(Math.random() * validProviders.length);
        return validProviders[random];
    }
    getPairingFromChain(request, lavaApis) {
        return __awaiter(this, void 0, void 0, function* () {
            const requestData = query_1.QueryGetPairingRequest.encode(request).finish();
            const hexData = Buffer.from(requestData).toString("hex");
            const sendRelayOptions = {
                data: (0, common_1.generateRPCData)("abci_query", [
                    "/lavanet.lava.pairing.Query/GetPairing",
                    hexData,
                    "0",
                    false,
                ]),
                url: "",
                connectionType: "",
            };
            const relayCu = lavaApis.get("lavanet.lava.pairing.Query/GetPairing");
            if (relayCu == undefined) {
                throw errors_1.default.errApiNotFound;
            }
            const jsonResponse = yield this.SendRelayToAllProvidersAndRace(sendRelayOptions, relayCu, "tendermintrpc");
            const byteArrayResponse = (0, common_1.base64ToUint8Array)(jsonResponse.result.response.value);
            const decodedResponse = query_1.QueryGetPairingResponse.decode(byteArrayResponse);
            if (decodedResponse.providers == undefined) {
                throw errors_1.default.errProvidersNotFound;
            }
            return decodedResponse;
        });
    }
    getMaxCuForUser(request, lavaApis) {
        return __awaiter(this, void 0, void 0, function* () {
            const requestData = query_1.QueryUserEntryRequest.encode(request).finish();
            const hexData = Buffer.from(requestData).toString("hex");
            const sendRelayOptions = {
                data: (0, common_1.generateRPCData)("abci_query", [
                    "/lavanet.lava.pairing.Query/UserEntry",
                    hexData,
                    "0",
                    false,
                ]),
                url: "",
                connectionType: "",
            };
            const relayCu = lavaApis.get("lavanet.lava.pairing.Query/UserEntry");
            if (relayCu == undefined) {
                throw errors_1.default.errApiNotFound;
            }
            const jsonResponse = yield this.SendRelayToAllProvidersAndRace(sendRelayOptions, relayCu, "tendermintrpc");
            const byteArrayResponse = (0, common_1.base64ToUint8Array)(jsonResponse.result.response.value);
            const response = query_1.QueryUserEntryResponse.decode(byteArrayResponse);
            if (response.maxCU == undefined) {
                throw errors_1.default.errMaxCuNotFound;
            }
            // return maxCu from userEntry
            return response.maxCU.low;
        });
    }
    getServiceApis(request, rpcInterface, lavaApis) {
        return __awaiter(this, void 0, void 0, function* () {
            const requestData = query_2.QueryGetSpecRequest.encode(request).finish();
            const hexData = Buffer.from(requestData).toString("hex");
            const sendRelayOptions = {
                data: (0, common_1.generateRPCData)("abci_query", [
                    "/lavanet.lava.spec.Query/Spec",
                    hexData,
                    "0",
                    false,
                ]),
                url: "",
                connectionType: "",
            };
            const relayCu = lavaApis.get("lavanet.lava.spec.Query/Spec");
            if (relayCu == undefined) {
                throw errors_1.default.errApiNotFound;
            }
            const jsonResponse = yield this.SendRelayToAllProvidersAndRace(sendRelayOptions, relayCu, "tendermintrpc");
            const byteArrayResponse = (0, common_1.base64ToUint8Array)(jsonResponse.result.response.value);
            const response = query_2.QueryGetSpecResponse.decode(byteArrayResponse);
            if (response.Spec == undefined) {
                throw errors_1.default.errSpecNotFound;
            }
            const apis = new Map();
            // Extract apis from response
            for (const element of response.Spec.apis) {
                for (const apiInterface of element.apiInterfaces) {
                    // Skip if interface which does not match
                    if (apiInterface.interface != rpcInterface)
                        continue;
                    if (apiInterface.interface == "rest") {
                        // handle REST apis
                        const name = this.convertRestApiName(element.name);
                        apis.set(name, element.computeUnits.low);
                    }
                    else {
                        // Handle RPC apis
                        apis.set(element.name, element.computeUnits.low);
                    }
                }
            }
            return apis;
        });
    }
    convertRestApiName(name) {
        const regex = /\{\s*[^}]+\s*\}/g;
        return name.replace(regex, "[^/s]+");
    }
    SendRelayToAllProvidersAndRace(options, relayCu, rpcInterface) {
        return __awaiter(this, void 0, void 0, function* () {
            let lastError;
            for (let retryAttempt = 0; retryAttempt < BOOT_RETRY_ATTEMPTS; retryAttempt++) {
                const allRelays = new Map();
                let response;
                for (const provider of this.GetLavaProviders()) {
                    const uniqueKey = provider.Session.ProviderAddress +
                        String(Math.floor(Math.random() * 10000000));
                    const providerRelayPromise = this.SendRelayWithRetry(options, provider, relayCu, rpcInterface)
                        .then((result) => {
                        this.debugPrint("response succeeded", result);
                        response = result;
                        allRelays.delete(uniqueKey);
                    })
                        .catch((err) => {
                        this.debugPrint("one of the promises failed in SendRelayToAllProvidersAndRace reason:", err, uniqueKey);
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
                        this.debugPrint("SendRelayToAllProvidersAndRace, got response from one provider", response);
                        return response;
                    }
                }
                this.debugPrint("Failed all promises SendRelayToAllProvidersAndRace, trying again", retryAttempt, "out of", BOOT_RETRY_ATTEMPTS);
            }
            throw new Error("Failed all promises SendRelayToAllProvidersAndRace: " + String(lastError));
        });
    }
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
exports.LavaProviders = LavaProviders;

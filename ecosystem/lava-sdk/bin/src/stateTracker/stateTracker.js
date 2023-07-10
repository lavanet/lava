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
exports.StateTracker = void 0;
const types_1 = require("../types/types");
const errors_1 = __importDefault(require("./errors"));
const errors_2 = __importDefault(require("./errors"));
class StateTracker {
    constructor(lavaProviders, relayer) {
        this.lavaProviders = lavaProviders;
        this.relayer = relayer;
    }
    // Get session return providers for current epoch
    getSession(account, chainID, rpcInterface) {
        return __awaiter(this, void 0, void 0, function* () {
            try {
                if (this.lavaProviders == null) {
                    throw errors_1.default.errLavaProvidersNotInitialized;
                }
                const lavaRPCEndpoint = this.lavaProviders.getNextProvider();
                console.log("Fetching pairing list from ", lavaRPCEndpoint.Session.Endpoint);
                // Create request for getServiceApis method
                const apis = yield this.getServiceApis(lavaRPCEndpoint, chainID, rpcInterface);
                // Create pairing request for getPairing method
                const pairingRequest = {
                    chainID: chainID,
                    client: account.address,
                };
                // Get pairing from the chain
                const pairingResponse = yield this.getPairingFromChain(lavaRPCEndpoint, pairingRequest);
                // Set when will next epoch start
                const nextEpochStart = new Date();
                nextEpochStart.setSeconds(nextEpochStart.getSeconds() +
                    parseInt(pairingResponse.timeLeftToNextPairing));
                // Extract providers from pairing response
                const providers = pairingResponse.providers;
                // Initialize ConsumerSessionWithProvider array
                const pairing = [];
                // create request for getting userEntity
                const userEntityRequest = {
                    address: account.address,
                    chainID: chainID,
                    block: pairingResponse.currentEpoch,
                };
                // fetch max compute units
                const maxcu = yield this.getMaxCuForUser(lavaRPCEndpoint, userEntityRequest);
                //Iterate over providers to populate pairing list
                for (const provider of providers) {
                    // Skip providers with no endpoints
                    if (provider.endpoints.length == 0) {
                        continue;
                    }
                    // Initialize relevantEndpoints array
                    const relevantEndpoints = [];
                    //only take into account endpoints that use the same api interface
                    for (const endpoint of provider.endpoints) {
                        if (endpoint.useType == rpcInterface) {
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
                    const newPairing = new types_1.ConsumerSessionWithProvider(account.address, relevantEndpoints, singleConsumerSession, maxcu, 0, // used compute units
                    false);
                    // Add newly created pairing in the pairing list
                    pairing.push(newPairing);
                }
                // Create session object
                const sessionManager = new types_1.SessionManager(pairing, nextEpochStart, apis);
                return sessionManager;
            }
            catch (err) {
                throw err;
            }
        });
    }
    pickRandomProvider(providers) {
        // Remove providers which does not match criteria
        const validProviders = providers.filter((item) => item.MaxComputeUnits > item.UsedComputeUnits);
        if (validProviders.length === 0) {
            throw errors_2.default.errNoValidProvidersForCurrentEpoch;
        }
        // Pick random provider
        const random = Math.floor(Math.random() * validProviders.length);
        return validProviders[random];
    }
    getPairingFromChain(lavaRPCEndpoint, request) {
        return __awaiter(this, void 0, void 0, function* () {
            const options = {
                connectionType: "GET",
                url: "/lavanet/lava/pairing/get_pairing/" +
                    request.chainID +
                    "/" +
                    request.client,
                data: "",
            };
            const jsonResponse = yield this.sendRelayWithRetry(options, lavaRPCEndpoint);
            return jsonResponse;
        });
    }
    getMaxCuForUser(lavaRPCEndpoint, request) {
        return __awaiter(this, void 0, void 0, function* () {
            const options = {
                connectionType: "GET",
                url: "/lavanet/lava/pairing/user_entry/" +
                    request.address +
                    "/" +
                    request.chainID,
                data: "?block=" + request.block,
            };
            const jsonResponse = yield this.sendRelayWithRetry(options, lavaRPCEndpoint);
            // return maxCu from userEntry
            return parseInt(jsonResponse.maxCU);
        });
    }
    getServiceApis(lavaRPCEndpoint, chainID, rpcInterface) {
        return __awaiter(this, void 0, void 0, function* () {
            const options = {
                connectionType: "GET",
                url: "/lavanet/lava/spec/spec/" + chainID,
                data: "",
            };
            const jsonResponse = yield this.sendRelayWithRetry(options, lavaRPCEndpoint);
            if (jsonResponse.Spec == undefined) {
                throw errors_1.default.errSpecNotFound;
            }
            const apis = new Map();
            // Extract apis from response
            for (const element of jsonResponse.Spec.apis) {
                for (const apiInterface of element.api_interfaces) {
                    // Skip if interface which does not match
                    if (apiInterface.interface != rpcInterface)
                        continue;
                    if (apiInterface.interface == "rest") {
                        // handle REST apis
                        const name = this.convertRestApiName(element.name);
                        apis.set(name, parseInt(element.compute_units));
                    }
                    else {
                        // Handle RPC apis
                        apis.set(element.name, parseInt(element.compute_units));
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
    sendRelayWithRetry(options, // TODO add type
    lavaRPCEndpoint) {
        var _a, _b;
        return __awaiter(this, void 0, void 0, function* () {
            var response;
            // TODO make sure relayer is not null
            try {
                response = yield ((_a = this.relayer) === null || _a === void 0 ? void 0 : _a.sendRelay(options, lavaRPCEndpoint, 10));
            }
            catch (error) {
                if (error instanceof Error) {
                    if (error.message.startsWith("user reported very old lava block height")) {
                        const currentBlockHeightRegex = /current epoch block:(\d+)/;
                        const match = error.message.match(currentBlockHeightRegex);
                        const currentBlockHeight = match ? match[1] : null;
                        if (currentBlockHeight != null) {
                            lavaRPCEndpoint.Session.PairingEpoch = parseInt(currentBlockHeight);
                            response = yield ((_b = this.relayer) === null || _b === void 0 ? void 0 : _b.sendRelay(options, lavaRPCEndpoint, 10));
                        }
                    }
                    else {
                        throw error;
                    }
                }
                else {
                    throw error;
                }
            }
            if (response == undefined) {
                return "";
            }
            const dec = new TextDecoder();
            const decodedResponse = dec.decode(response.getData_asU8());
            const jsonResponse = JSON.parse(decodedResponse);
            return jsonResponse;
        });
    }
}
exports.StateTracker = StateTracker;

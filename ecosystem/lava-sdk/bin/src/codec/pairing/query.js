"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.QueryClientImpl = exports.QueryAccountInfoResponse = exports.QueryStaticProvidersListResponse = exports.QueryStaticProvidersListRequest = exports.QueryUserEntryResponse = exports.QueryUserEntryRequest = exports.QueryAllEpochPaymentsResponse = exports.QueryAllEpochPaymentsRequest = exports.QueryGetEpochPaymentsResponse = exports.QueryGetEpochPaymentsRequest = exports.QueryAllProviderPaymentStorageResponse = exports.QueryAllProviderPaymentStorageRequest = exports.QueryGetProviderPaymentStorageResponse = exports.QueryGetProviderPaymentStorageRequest = exports.QueryAllUniquePaymentStorageClientProviderResponse = exports.QueryAllUniquePaymentStorageClientProviderRequest = exports.QueryGetUniquePaymentStorageClientProviderResponse = exports.QueryGetUniquePaymentStorageClientProviderRequest = exports.QueryVerifyPairingResponse = exports.QueryVerifyPairingRequest = exports.QueryGetPairingResponse = exports.QueryGetPairingRequest = exports.QueryProvidersResponse = exports.QueryProvidersRequest = exports.QueryParamsResponse = exports.QueryParamsRequest = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const pagination_1 = require("../cosmos/base/query/v1beta1/pagination");
const stake_entry_1 = require("../epochstorage/stake_entry");
const project_1 = require("../projects/project");
const subscription_1 = require("../subscription/subscription");
const epoch_payments_1 = require("./epoch_payments");
const params_1 = require("./params");
const provider_payment_storage_1 = require("./provider_payment_storage");
const unique_payment_storage_client_provider_1 = require("./unique_payment_storage_client_provider");
exports.protobufPackage = "lavanet.lava.pairing";
function createBaseQueryParamsRequest() {
    return {};
}
exports.QueryParamsRequest = {
    encode(_, writer = minimal_1.default.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryParamsRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(_) {
        return {};
    },
    toJSON(_) {
        const obj = {};
        return obj;
    },
    create(base) {
        return exports.QueryParamsRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(_) {
        const message = createBaseQueryParamsRequest();
        return message;
    },
};
function createBaseQueryParamsResponse() {
    return { params: undefined };
}
exports.QueryParamsResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.params !== undefined) {
            params_1.Params.encode(message.params, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryParamsResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.params = params_1.Params.decode(reader, reader.uint32());
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return { params: isSet(object.params) ? params_1.Params.fromJSON(object.params) : undefined };
    },
    toJSON(message) {
        const obj = {};
        message.params !== undefined && (obj.params = message.params ? params_1.Params.toJSON(message.params) : undefined);
        return obj;
    },
    create(base) {
        return exports.QueryParamsResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseQueryParamsResponse();
        message.params = (object.params !== undefined && object.params !== null)
            ? params_1.Params.fromPartial(object.params)
            : undefined;
        return message;
    },
};
function createBaseQueryProvidersRequest() {
    return { chainID: "", showFrozen: false };
}
exports.QueryProvidersRequest = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.chainID !== "") {
            writer.uint32(10).string(message.chainID);
        }
        if (message.showFrozen === true) {
            writer.uint32(16).bool(message.showFrozen);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryProvidersRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.chainID = reader.string();
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.showFrozen = reader.bool();
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            chainID: isSet(object.chainID) ? String(object.chainID) : "",
            showFrozen: isSet(object.showFrozen) ? Boolean(object.showFrozen) : false,
        };
    },
    toJSON(message) {
        const obj = {};
        message.chainID !== undefined && (obj.chainID = message.chainID);
        message.showFrozen !== undefined && (obj.showFrozen = message.showFrozen);
        return obj;
    },
    create(base) {
        return exports.QueryProvidersRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseQueryProvidersRequest();
        message.chainID = (_a = object.chainID) !== null && _a !== void 0 ? _a : "";
        message.showFrozen = (_b = object.showFrozen) !== null && _b !== void 0 ? _b : false;
        return message;
    },
};
function createBaseQueryProvidersResponse() {
    return { stakeEntry: [], output: "" };
}
exports.QueryProvidersResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        for (const v of message.stakeEntry) {
            stake_entry_1.StakeEntry.encode(v, writer.uint32(10).fork()).ldelim();
        }
        if (message.output !== "") {
            writer.uint32(18).string(message.output);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryProvidersResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.stakeEntry.push(stake_entry_1.StakeEntry.decode(reader, reader.uint32()));
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.output = reader.string();
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            stakeEntry: Array.isArray(object === null || object === void 0 ? void 0 : object.stakeEntry) ? object.stakeEntry.map((e) => stake_entry_1.StakeEntry.fromJSON(e)) : [],
            output: isSet(object.output) ? String(object.output) : "",
        };
    },
    toJSON(message) {
        const obj = {};
        if (message.stakeEntry) {
            obj.stakeEntry = message.stakeEntry.map((e) => e ? stake_entry_1.StakeEntry.toJSON(e) : undefined);
        }
        else {
            obj.stakeEntry = [];
        }
        message.output !== undefined && (obj.output = message.output);
        return obj;
    },
    create(base) {
        return exports.QueryProvidersResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseQueryProvidersResponse();
        message.stakeEntry = ((_a = object.stakeEntry) === null || _a === void 0 ? void 0 : _a.map((e) => stake_entry_1.StakeEntry.fromPartial(e))) || [];
        message.output = (_b = object.output) !== null && _b !== void 0 ? _b : "";
        return message;
    },
};
function createBaseQueryGetPairingRequest() {
    return { chainID: "", client: "" };
}
exports.QueryGetPairingRequest = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.chainID !== "") {
            writer.uint32(10).string(message.chainID);
        }
        if (message.client !== "") {
            writer.uint32(18).string(message.client);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryGetPairingRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.chainID = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.client = reader.string();
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            chainID: isSet(object.chainID) ? String(object.chainID) : "",
            client: isSet(object.client) ? String(object.client) : "",
        };
    },
    toJSON(message) {
        const obj = {};
        message.chainID !== undefined && (obj.chainID = message.chainID);
        message.client !== undefined && (obj.client = message.client);
        return obj;
    },
    create(base) {
        return exports.QueryGetPairingRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseQueryGetPairingRequest();
        message.chainID = (_a = object.chainID) !== null && _a !== void 0 ? _a : "";
        message.client = (_b = object.client) !== null && _b !== void 0 ? _b : "";
        return message;
    },
};
function createBaseQueryGetPairingResponse() {
    return {
        providers: [],
        currentEpoch: long_1.default.UZERO,
        timeLeftToNextPairing: long_1.default.UZERO,
        specLastUpdatedBlock: long_1.default.UZERO,
        blockOfNextPairing: long_1.default.UZERO,
    };
}
exports.QueryGetPairingResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        for (const v of message.providers) {
            stake_entry_1.StakeEntry.encode(v, writer.uint32(10).fork()).ldelim();
        }
        if (!message.currentEpoch.isZero()) {
            writer.uint32(16).uint64(message.currentEpoch);
        }
        if (!message.timeLeftToNextPairing.isZero()) {
            writer.uint32(24).uint64(message.timeLeftToNextPairing);
        }
        if (!message.specLastUpdatedBlock.isZero()) {
            writer.uint32(32).uint64(message.specLastUpdatedBlock);
        }
        if (!message.blockOfNextPairing.isZero()) {
            writer.uint32(40).uint64(message.blockOfNextPairing);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryGetPairingResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.providers.push(stake_entry_1.StakeEntry.decode(reader, reader.uint32()));
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.currentEpoch = reader.uint64();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.timeLeftToNextPairing = reader.uint64();
                    continue;
                case 4:
                    if (tag != 32) {
                        break;
                    }
                    message.specLastUpdatedBlock = reader.uint64();
                    continue;
                case 5:
                    if (tag != 40) {
                        break;
                    }
                    message.blockOfNextPairing = reader.uint64();
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            providers: Array.isArray(object === null || object === void 0 ? void 0 : object.providers) ? object.providers.map((e) => stake_entry_1.StakeEntry.fromJSON(e)) : [],
            currentEpoch: isSet(object.currentEpoch) ? long_1.default.fromValue(object.currentEpoch) : long_1.default.UZERO,
            timeLeftToNextPairing: isSet(object.timeLeftToNextPairing)
                ? long_1.default.fromValue(object.timeLeftToNextPairing)
                : long_1.default.UZERO,
            specLastUpdatedBlock: isSet(object.specLastUpdatedBlock)
                ? long_1.default.fromValue(object.specLastUpdatedBlock)
                : long_1.default.UZERO,
            blockOfNextPairing: isSet(object.blockOfNextPairing) ? long_1.default.fromValue(object.blockOfNextPairing) : long_1.default.UZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        if (message.providers) {
            obj.providers = message.providers.map((e) => e ? stake_entry_1.StakeEntry.toJSON(e) : undefined);
        }
        else {
            obj.providers = [];
        }
        message.currentEpoch !== undefined && (obj.currentEpoch = (message.currentEpoch || long_1.default.UZERO).toString());
        message.timeLeftToNextPairing !== undefined &&
            (obj.timeLeftToNextPairing = (message.timeLeftToNextPairing || long_1.default.UZERO).toString());
        message.specLastUpdatedBlock !== undefined &&
            (obj.specLastUpdatedBlock = (message.specLastUpdatedBlock || long_1.default.UZERO).toString());
        message.blockOfNextPairing !== undefined &&
            (obj.blockOfNextPairing = (message.blockOfNextPairing || long_1.default.UZERO).toString());
        return obj;
    },
    create(base) {
        return exports.QueryGetPairingResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseQueryGetPairingResponse();
        message.providers = ((_a = object.providers) === null || _a === void 0 ? void 0 : _a.map((e) => stake_entry_1.StakeEntry.fromPartial(e))) || [];
        message.currentEpoch = (object.currentEpoch !== undefined && object.currentEpoch !== null)
            ? long_1.default.fromValue(object.currentEpoch)
            : long_1.default.UZERO;
        message.timeLeftToNextPairing =
            (object.timeLeftToNextPairing !== undefined && object.timeLeftToNextPairing !== null)
                ? long_1.default.fromValue(object.timeLeftToNextPairing)
                : long_1.default.UZERO;
        message.specLastUpdatedBlock = (object.specLastUpdatedBlock !== undefined && object.specLastUpdatedBlock !== null)
            ? long_1.default.fromValue(object.specLastUpdatedBlock)
            : long_1.default.UZERO;
        message.blockOfNextPairing = (object.blockOfNextPairing !== undefined && object.blockOfNextPairing !== null)
            ? long_1.default.fromValue(object.blockOfNextPairing)
            : long_1.default.UZERO;
        return message;
    },
};
function createBaseQueryVerifyPairingRequest() {
    return { chainID: "", client: "", provider: "", block: long_1.default.UZERO };
}
exports.QueryVerifyPairingRequest = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.chainID !== "") {
            writer.uint32(10).string(message.chainID);
        }
        if (message.client !== "") {
            writer.uint32(18).string(message.client);
        }
        if (message.provider !== "") {
            writer.uint32(26).string(message.provider);
        }
        if (!message.block.isZero()) {
            writer.uint32(32).uint64(message.block);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryVerifyPairingRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.chainID = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.client = reader.string();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.provider = reader.string();
                    continue;
                case 4:
                    if (tag != 32) {
                        break;
                    }
                    message.block = reader.uint64();
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            chainID: isSet(object.chainID) ? String(object.chainID) : "",
            client: isSet(object.client) ? String(object.client) : "",
            provider: isSet(object.provider) ? String(object.provider) : "",
            block: isSet(object.block) ? long_1.default.fromValue(object.block) : long_1.default.UZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.chainID !== undefined && (obj.chainID = message.chainID);
        message.client !== undefined && (obj.client = message.client);
        message.provider !== undefined && (obj.provider = message.provider);
        message.block !== undefined && (obj.block = (message.block || long_1.default.UZERO).toString());
        return obj;
    },
    create(base) {
        return exports.QueryVerifyPairingRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBaseQueryVerifyPairingRequest();
        message.chainID = (_a = object.chainID) !== null && _a !== void 0 ? _a : "";
        message.client = (_b = object.client) !== null && _b !== void 0 ? _b : "";
        message.provider = (_c = object.provider) !== null && _c !== void 0 ? _c : "";
        message.block = (object.block !== undefined && object.block !== null) ? long_1.default.fromValue(object.block) : long_1.default.UZERO;
        return message;
    },
};
function createBaseQueryVerifyPairingResponse() {
    return { valid: false, pairedProviders: long_1.default.UZERO, cuPerEpoch: long_1.default.UZERO };
}
exports.QueryVerifyPairingResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.valid === true) {
            writer.uint32(8).bool(message.valid);
        }
        if (!message.pairedProviders.isZero()) {
            writer.uint32(24).uint64(message.pairedProviders);
        }
        if (!message.cuPerEpoch.isZero()) {
            writer.uint32(32).uint64(message.cuPerEpoch);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryVerifyPairingResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.valid = reader.bool();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.pairedProviders = reader.uint64();
                    continue;
                case 4:
                    if (tag != 32) {
                        break;
                    }
                    message.cuPerEpoch = reader.uint64();
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            valid: isSet(object.valid) ? Boolean(object.valid) : false,
            pairedProviders: isSet(object.pairedProviders) ? long_1.default.fromValue(object.pairedProviders) : long_1.default.UZERO,
            cuPerEpoch: isSet(object.cuPerEpoch) ? long_1.default.fromValue(object.cuPerEpoch) : long_1.default.UZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.valid !== undefined && (obj.valid = message.valid);
        message.pairedProviders !== undefined && (obj.pairedProviders = (message.pairedProviders || long_1.default.UZERO).toString());
        message.cuPerEpoch !== undefined && (obj.cuPerEpoch = (message.cuPerEpoch || long_1.default.UZERO).toString());
        return obj;
    },
    create(base) {
        return exports.QueryVerifyPairingResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseQueryVerifyPairingResponse();
        message.valid = (_a = object.valid) !== null && _a !== void 0 ? _a : false;
        message.pairedProviders = (object.pairedProviders !== undefined && object.pairedProviders !== null)
            ? long_1.default.fromValue(object.pairedProviders)
            : long_1.default.UZERO;
        message.cuPerEpoch = (object.cuPerEpoch !== undefined && object.cuPerEpoch !== null)
            ? long_1.default.fromValue(object.cuPerEpoch)
            : long_1.default.UZERO;
        return message;
    },
};
function createBaseQueryGetUniquePaymentStorageClientProviderRequest() {
    return { index: "" };
}
exports.QueryGetUniquePaymentStorageClientProviderRequest = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryGetUniquePaymentStorageClientProviderRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.index = reader.string();
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return { index: isSet(object.index) ? String(object.index) : "" };
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = message.index);
        return obj;
    },
    create(base) {
        return exports.QueryGetUniquePaymentStorageClientProviderRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseQueryGetUniquePaymentStorageClientProviderRequest();
        message.index = (_a = object.index) !== null && _a !== void 0 ? _a : "";
        return message;
    },
};
function createBaseQueryGetUniquePaymentStorageClientProviderResponse() {
    return { uniquePaymentStorageClientProvider: undefined };
}
exports.QueryGetUniquePaymentStorageClientProviderResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.uniquePaymentStorageClientProvider !== undefined) {
            unique_payment_storage_client_provider_1.UniquePaymentStorageClientProvider.encode(message.uniquePaymentStorageClientProvider, writer.uint32(10).fork())
                .ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryGetUniquePaymentStorageClientProviderResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.uniquePaymentStorageClientProvider = unique_payment_storage_client_provider_1.UniquePaymentStorageClientProvider.decode(reader, reader.uint32());
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            uniquePaymentStorageClientProvider: isSet(object.uniquePaymentStorageClientProvider)
                ? unique_payment_storage_client_provider_1.UniquePaymentStorageClientProvider.fromJSON(object.uniquePaymentStorageClientProvider)
                : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.uniquePaymentStorageClientProvider !== undefined &&
            (obj.uniquePaymentStorageClientProvider = message.uniquePaymentStorageClientProvider
                ? unique_payment_storage_client_provider_1.UniquePaymentStorageClientProvider.toJSON(message.uniquePaymentStorageClientProvider)
                : undefined);
        return obj;
    },
    create(base) {
        return exports.QueryGetUniquePaymentStorageClientProviderResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseQueryGetUniquePaymentStorageClientProviderResponse();
        message.uniquePaymentStorageClientProvider =
            (object.uniquePaymentStorageClientProvider !== undefined && object.uniquePaymentStorageClientProvider !== null)
                ? unique_payment_storage_client_provider_1.UniquePaymentStorageClientProvider.fromPartial(object.uniquePaymentStorageClientProvider)
                : undefined;
        return message;
    },
};
function createBaseQueryAllUniquePaymentStorageClientProviderRequest() {
    return { pagination: undefined };
}
exports.QueryAllUniquePaymentStorageClientProviderRequest = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.pagination !== undefined) {
            pagination_1.PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryAllUniquePaymentStorageClientProviderRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.pagination = pagination_1.PageRequest.decode(reader, reader.uint32());
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return { pagination: isSet(object.pagination) ? pagination_1.PageRequest.fromJSON(object.pagination) : undefined };
    },
    toJSON(message) {
        const obj = {};
        message.pagination !== undefined &&
            (obj.pagination = message.pagination ? pagination_1.PageRequest.toJSON(message.pagination) : undefined);
        return obj;
    },
    create(base) {
        return exports.QueryAllUniquePaymentStorageClientProviderRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseQueryAllUniquePaymentStorageClientProviderRequest();
        message.pagination = (object.pagination !== undefined && object.pagination !== null)
            ? pagination_1.PageRequest.fromPartial(object.pagination)
            : undefined;
        return message;
    },
};
function createBaseQueryAllUniquePaymentStorageClientProviderResponse() {
    return { uniquePaymentStorageClientProvider: [], pagination: undefined };
}
exports.QueryAllUniquePaymentStorageClientProviderResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        for (const v of message.uniquePaymentStorageClientProvider) {
            unique_payment_storage_client_provider_1.UniquePaymentStorageClientProvider.encode(v, writer.uint32(10).fork()).ldelim();
        }
        if (message.pagination !== undefined) {
            pagination_1.PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryAllUniquePaymentStorageClientProviderResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.uniquePaymentStorageClientProvider.push(unique_payment_storage_client_provider_1.UniquePaymentStorageClientProvider.decode(reader, reader.uint32()));
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.pagination = pagination_1.PageResponse.decode(reader, reader.uint32());
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            uniquePaymentStorageClientProvider: Array.isArray(object === null || object === void 0 ? void 0 : object.uniquePaymentStorageClientProvider)
                ? object.uniquePaymentStorageClientProvider.map((e) => unique_payment_storage_client_provider_1.UniquePaymentStorageClientProvider.fromJSON(e))
                : [],
            pagination: isSet(object.pagination) ? pagination_1.PageResponse.fromJSON(object.pagination) : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        if (message.uniquePaymentStorageClientProvider) {
            obj.uniquePaymentStorageClientProvider = message.uniquePaymentStorageClientProvider.map((e) => e ? unique_payment_storage_client_provider_1.UniquePaymentStorageClientProvider.toJSON(e) : undefined);
        }
        else {
            obj.uniquePaymentStorageClientProvider = [];
        }
        message.pagination !== undefined &&
            (obj.pagination = message.pagination ? pagination_1.PageResponse.toJSON(message.pagination) : undefined);
        return obj;
    },
    create(base) {
        return exports.QueryAllUniquePaymentStorageClientProviderResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseQueryAllUniquePaymentStorageClientProviderResponse();
        message.uniquePaymentStorageClientProvider =
            ((_a = object.uniquePaymentStorageClientProvider) === null || _a === void 0 ? void 0 : _a.map((e) => unique_payment_storage_client_provider_1.UniquePaymentStorageClientProvider.fromPartial(e))) || [];
        message.pagination = (object.pagination !== undefined && object.pagination !== null)
            ? pagination_1.PageResponse.fromPartial(object.pagination)
            : undefined;
        return message;
    },
};
function createBaseQueryGetProviderPaymentStorageRequest() {
    return { index: "" };
}
exports.QueryGetProviderPaymentStorageRequest = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryGetProviderPaymentStorageRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.index = reader.string();
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return { index: isSet(object.index) ? String(object.index) : "" };
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = message.index);
        return obj;
    },
    create(base) {
        return exports.QueryGetProviderPaymentStorageRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseQueryGetProviderPaymentStorageRequest();
        message.index = (_a = object.index) !== null && _a !== void 0 ? _a : "";
        return message;
    },
};
function createBaseQueryGetProviderPaymentStorageResponse() {
    return { providerPaymentStorage: undefined };
}
exports.QueryGetProviderPaymentStorageResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.providerPaymentStorage !== undefined) {
            provider_payment_storage_1.ProviderPaymentStorage.encode(message.providerPaymentStorage, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryGetProviderPaymentStorageResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.providerPaymentStorage = provider_payment_storage_1.ProviderPaymentStorage.decode(reader, reader.uint32());
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            providerPaymentStorage: isSet(object.providerPaymentStorage)
                ? provider_payment_storage_1.ProviderPaymentStorage.fromJSON(object.providerPaymentStorage)
                : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.providerPaymentStorage !== undefined && (obj.providerPaymentStorage = message.providerPaymentStorage
            ? provider_payment_storage_1.ProviderPaymentStorage.toJSON(message.providerPaymentStorage)
            : undefined);
        return obj;
    },
    create(base) {
        return exports.QueryGetProviderPaymentStorageResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseQueryGetProviderPaymentStorageResponse();
        message.providerPaymentStorage =
            (object.providerPaymentStorage !== undefined && object.providerPaymentStorage !== null)
                ? provider_payment_storage_1.ProviderPaymentStorage.fromPartial(object.providerPaymentStorage)
                : undefined;
        return message;
    },
};
function createBaseQueryAllProviderPaymentStorageRequest() {
    return { pagination: undefined };
}
exports.QueryAllProviderPaymentStorageRequest = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.pagination !== undefined) {
            pagination_1.PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryAllProviderPaymentStorageRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.pagination = pagination_1.PageRequest.decode(reader, reader.uint32());
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return { pagination: isSet(object.pagination) ? pagination_1.PageRequest.fromJSON(object.pagination) : undefined };
    },
    toJSON(message) {
        const obj = {};
        message.pagination !== undefined &&
            (obj.pagination = message.pagination ? pagination_1.PageRequest.toJSON(message.pagination) : undefined);
        return obj;
    },
    create(base) {
        return exports.QueryAllProviderPaymentStorageRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseQueryAllProviderPaymentStorageRequest();
        message.pagination = (object.pagination !== undefined && object.pagination !== null)
            ? pagination_1.PageRequest.fromPartial(object.pagination)
            : undefined;
        return message;
    },
};
function createBaseQueryAllProviderPaymentStorageResponse() {
    return { providerPaymentStorage: [], pagination: undefined };
}
exports.QueryAllProviderPaymentStorageResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        for (const v of message.providerPaymentStorage) {
            provider_payment_storage_1.ProviderPaymentStorage.encode(v, writer.uint32(10).fork()).ldelim();
        }
        if (message.pagination !== undefined) {
            pagination_1.PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryAllProviderPaymentStorageResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.providerPaymentStorage.push(provider_payment_storage_1.ProviderPaymentStorage.decode(reader, reader.uint32()));
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.pagination = pagination_1.PageResponse.decode(reader, reader.uint32());
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            providerPaymentStorage: Array.isArray(object === null || object === void 0 ? void 0 : object.providerPaymentStorage)
                ? object.providerPaymentStorage.map((e) => provider_payment_storage_1.ProviderPaymentStorage.fromJSON(e))
                : [],
            pagination: isSet(object.pagination) ? pagination_1.PageResponse.fromJSON(object.pagination) : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        if (message.providerPaymentStorage) {
            obj.providerPaymentStorage = message.providerPaymentStorage.map((e) => e ? provider_payment_storage_1.ProviderPaymentStorage.toJSON(e) : undefined);
        }
        else {
            obj.providerPaymentStorage = [];
        }
        message.pagination !== undefined &&
            (obj.pagination = message.pagination ? pagination_1.PageResponse.toJSON(message.pagination) : undefined);
        return obj;
    },
    create(base) {
        return exports.QueryAllProviderPaymentStorageResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseQueryAllProviderPaymentStorageResponse();
        message.providerPaymentStorage = ((_a = object.providerPaymentStorage) === null || _a === void 0 ? void 0 : _a.map((e) => provider_payment_storage_1.ProviderPaymentStorage.fromPartial(e))) ||
            [];
        message.pagination = (object.pagination !== undefined && object.pagination !== null)
            ? pagination_1.PageResponse.fromPartial(object.pagination)
            : undefined;
        return message;
    },
};
function createBaseQueryGetEpochPaymentsRequest() {
    return { index: "" };
}
exports.QueryGetEpochPaymentsRequest = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryGetEpochPaymentsRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.index = reader.string();
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return { index: isSet(object.index) ? String(object.index) : "" };
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = message.index);
        return obj;
    },
    create(base) {
        return exports.QueryGetEpochPaymentsRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseQueryGetEpochPaymentsRequest();
        message.index = (_a = object.index) !== null && _a !== void 0 ? _a : "";
        return message;
    },
};
function createBaseQueryGetEpochPaymentsResponse() {
    return { epochPayments: undefined };
}
exports.QueryGetEpochPaymentsResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.epochPayments !== undefined) {
            epoch_payments_1.EpochPayments.encode(message.epochPayments, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryGetEpochPaymentsResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.epochPayments = epoch_payments_1.EpochPayments.decode(reader, reader.uint32());
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return { epochPayments: isSet(object.epochPayments) ? epoch_payments_1.EpochPayments.fromJSON(object.epochPayments) : undefined };
    },
    toJSON(message) {
        const obj = {};
        message.epochPayments !== undefined &&
            (obj.epochPayments = message.epochPayments ? epoch_payments_1.EpochPayments.toJSON(message.epochPayments) : undefined);
        return obj;
    },
    create(base) {
        return exports.QueryGetEpochPaymentsResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseQueryGetEpochPaymentsResponse();
        message.epochPayments = (object.epochPayments !== undefined && object.epochPayments !== null)
            ? epoch_payments_1.EpochPayments.fromPartial(object.epochPayments)
            : undefined;
        return message;
    },
};
function createBaseQueryAllEpochPaymentsRequest() {
    return { pagination: undefined };
}
exports.QueryAllEpochPaymentsRequest = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.pagination !== undefined) {
            pagination_1.PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryAllEpochPaymentsRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.pagination = pagination_1.PageRequest.decode(reader, reader.uint32());
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return { pagination: isSet(object.pagination) ? pagination_1.PageRequest.fromJSON(object.pagination) : undefined };
    },
    toJSON(message) {
        const obj = {};
        message.pagination !== undefined &&
            (obj.pagination = message.pagination ? pagination_1.PageRequest.toJSON(message.pagination) : undefined);
        return obj;
    },
    create(base) {
        return exports.QueryAllEpochPaymentsRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseQueryAllEpochPaymentsRequest();
        message.pagination = (object.pagination !== undefined && object.pagination !== null)
            ? pagination_1.PageRequest.fromPartial(object.pagination)
            : undefined;
        return message;
    },
};
function createBaseQueryAllEpochPaymentsResponse() {
    return { epochPayments: [], pagination: undefined };
}
exports.QueryAllEpochPaymentsResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        for (const v of message.epochPayments) {
            epoch_payments_1.EpochPayments.encode(v, writer.uint32(10).fork()).ldelim();
        }
        if (message.pagination !== undefined) {
            pagination_1.PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryAllEpochPaymentsResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.epochPayments.push(epoch_payments_1.EpochPayments.decode(reader, reader.uint32()));
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.pagination = pagination_1.PageResponse.decode(reader, reader.uint32());
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            epochPayments: Array.isArray(object === null || object === void 0 ? void 0 : object.epochPayments)
                ? object.epochPayments.map((e) => epoch_payments_1.EpochPayments.fromJSON(e))
                : [],
            pagination: isSet(object.pagination) ? pagination_1.PageResponse.fromJSON(object.pagination) : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        if (message.epochPayments) {
            obj.epochPayments = message.epochPayments.map((e) => e ? epoch_payments_1.EpochPayments.toJSON(e) : undefined);
        }
        else {
            obj.epochPayments = [];
        }
        message.pagination !== undefined &&
            (obj.pagination = message.pagination ? pagination_1.PageResponse.toJSON(message.pagination) : undefined);
        return obj;
    },
    create(base) {
        return exports.QueryAllEpochPaymentsResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseQueryAllEpochPaymentsResponse();
        message.epochPayments = ((_a = object.epochPayments) === null || _a === void 0 ? void 0 : _a.map((e) => epoch_payments_1.EpochPayments.fromPartial(e))) || [];
        message.pagination = (object.pagination !== undefined && object.pagination !== null)
            ? pagination_1.PageResponse.fromPartial(object.pagination)
            : undefined;
        return message;
    },
};
function createBaseQueryUserEntryRequest() {
    return { address: "", chainID: "", block: long_1.default.UZERO };
}
exports.QueryUserEntryRequest = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.address !== "") {
            writer.uint32(10).string(message.address);
        }
        if (message.chainID !== "") {
            writer.uint32(18).string(message.chainID);
        }
        if (!message.block.isZero()) {
            writer.uint32(24).uint64(message.block);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryUserEntryRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.address = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.chainID = reader.string();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.block = reader.uint64();
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            address: isSet(object.address) ? String(object.address) : "",
            chainID: isSet(object.chainID) ? String(object.chainID) : "",
            block: isSet(object.block) ? long_1.default.fromValue(object.block) : long_1.default.UZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.address !== undefined && (obj.address = message.address);
        message.chainID !== undefined && (obj.chainID = message.chainID);
        message.block !== undefined && (obj.block = (message.block || long_1.default.UZERO).toString());
        return obj;
    },
    create(base) {
        return exports.QueryUserEntryRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseQueryUserEntryRequest();
        message.address = (_a = object.address) !== null && _a !== void 0 ? _a : "";
        message.chainID = (_b = object.chainID) !== null && _b !== void 0 ? _b : "";
        message.block = (object.block !== undefined && object.block !== null) ? long_1.default.fromValue(object.block) : long_1.default.UZERO;
        return message;
    },
};
function createBaseQueryUserEntryResponse() {
    return { consumer: undefined, maxCU: long_1.default.UZERO };
}
exports.QueryUserEntryResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.consumer !== undefined) {
            stake_entry_1.StakeEntry.encode(message.consumer, writer.uint32(10).fork()).ldelim();
        }
        if (!message.maxCU.isZero()) {
            writer.uint32(16).uint64(message.maxCU);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryUserEntryResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.consumer = stake_entry_1.StakeEntry.decode(reader, reader.uint32());
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.maxCU = reader.uint64();
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            consumer: isSet(object.consumer) ? stake_entry_1.StakeEntry.fromJSON(object.consumer) : undefined,
            maxCU: isSet(object.maxCU) ? long_1.default.fromValue(object.maxCU) : long_1.default.UZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.consumer !== undefined &&
            (obj.consumer = message.consumer ? stake_entry_1.StakeEntry.toJSON(message.consumer) : undefined);
        message.maxCU !== undefined && (obj.maxCU = (message.maxCU || long_1.default.UZERO).toString());
        return obj;
    },
    create(base) {
        return exports.QueryUserEntryResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseQueryUserEntryResponse();
        message.consumer = (object.consumer !== undefined && object.consumer !== null)
            ? stake_entry_1.StakeEntry.fromPartial(object.consumer)
            : undefined;
        message.maxCU = (object.maxCU !== undefined && object.maxCU !== null) ? long_1.default.fromValue(object.maxCU) : long_1.default.UZERO;
        return message;
    },
};
function createBaseQueryStaticProvidersListRequest() {
    return { chainID: "" };
}
exports.QueryStaticProvidersListRequest = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.chainID !== "") {
            writer.uint32(10).string(message.chainID);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryStaticProvidersListRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.chainID = reader.string();
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return { chainID: isSet(object.chainID) ? String(object.chainID) : "" };
    },
    toJSON(message) {
        const obj = {};
        message.chainID !== undefined && (obj.chainID = message.chainID);
        return obj;
    },
    create(base) {
        return exports.QueryStaticProvidersListRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseQueryStaticProvidersListRequest();
        message.chainID = (_a = object.chainID) !== null && _a !== void 0 ? _a : "";
        return message;
    },
};
function createBaseQueryStaticProvidersListResponse() {
    return { providers: [] };
}
exports.QueryStaticProvidersListResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        for (const v of message.providers) {
            stake_entry_1.StakeEntry.encode(v, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryStaticProvidersListResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.providers.push(stake_entry_1.StakeEntry.decode(reader, reader.uint32()));
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            providers: Array.isArray(object === null || object === void 0 ? void 0 : object.providers) ? object.providers.map((e) => stake_entry_1.StakeEntry.fromJSON(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        if (message.providers) {
            obj.providers = message.providers.map((e) => e ? stake_entry_1.StakeEntry.toJSON(e) : undefined);
        }
        else {
            obj.providers = [];
        }
        return obj;
    },
    create(base) {
        return exports.QueryStaticProvidersListResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseQueryStaticProvidersListResponse();
        message.providers = ((_a = object.providers) === null || _a === void 0 ? void 0 : _a.map((e) => stake_entry_1.StakeEntry.fromPartial(e))) || [];
        return message;
    },
};
function createBaseQueryAccountInfoResponse() {
    return { provider: [], frozen: [], consumer: [], unstaked: [], subscription: undefined, project: undefined };
}
exports.QueryAccountInfoResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        for (const v of message.provider) {
            stake_entry_1.StakeEntry.encode(v, writer.uint32(10).fork()).ldelim();
        }
        for (const v of message.frozen) {
            stake_entry_1.StakeEntry.encode(v, writer.uint32(18).fork()).ldelim();
        }
        for (const v of message.consumer) {
            stake_entry_1.StakeEntry.encode(v, writer.uint32(26).fork()).ldelim();
        }
        for (const v of message.unstaked) {
            stake_entry_1.StakeEntry.encode(v, writer.uint32(34).fork()).ldelim();
        }
        if (message.subscription !== undefined) {
            subscription_1.Subscription.encode(message.subscription, writer.uint32(42).fork()).ldelim();
        }
        if (message.project !== undefined) {
            project_1.Project.encode(message.project, writer.uint32(50).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryAccountInfoResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.provider.push(stake_entry_1.StakeEntry.decode(reader, reader.uint32()));
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.frozen.push(stake_entry_1.StakeEntry.decode(reader, reader.uint32()));
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.consumer.push(stake_entry_1.StakeEntry.decode(reader, reader.uint32()));
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.unstaked.push(stake_entry_1.StakeEntry.decode(reader, reader.uint32()));
                    continue;
                case 5:
                    if (tag != 42) {
                        break;
                    }
                    message.subscription = subscription_1.Subscription.decode(reader, reader.uint32());
                    continue;
                case 6:
                    if (tag != 50) {
                        break;
                    }
                    message.project = project_1.Project.decode(reader, reader.uint32());
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            provider: Array.isArray(object === null || object === void 0 ? void 0 : object.provider) ? object.provider.map((e) => stake_entry_1.StakeEntry.fromJSON(e)) : [],
            frozen: Array.isArray(object === null || object === void 0 ? void 0 : object.frozen) ? object.frozen.map((e) => stake_entry_1.StakeEntry.fromJSON(e)) : [],
            consumer: Array.isArray(object === null || object === void 0 ? void 0 : object.consumer) ? object.consumer.map((e) => stake_entry_1.StakeEntry.fromJSON(e)) : [],
            unstaked: Array.isArray(object === null || object === void 0 ? void 0 : object.unstaked) ? object.unstaked.map((e) => stake_entry_1.StakeEntry.fromJSON(e)) : [],
            subscription: isSet(object.subscription) ? subscription_1.Subscription.fromJSON(object.subscription) : undefined,
            project: isSet(object.project) ? project_1.Project.fromJSON(object.project) : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        if (message.provider) {
            obj.provider = message.provider.map((e) => e ? stake_entry_1.StakeEntry.toJSON(e) : undefined);
        }
        else {
            obj.provider = [];
        }
        if (message.frozen) {
            obj.frozen = message.frozen.map((e) => e ? stake_entry_1.StakeEntry.toJSON(e) : undefined);
        }
        else {
            obj.frozen = [];
        }
        if (message.consumer) {
            obj.consumer = message.consumer.map((e) => e ? stake_entry_1.StakeEntry.toJSON(e) : undefined);
        }
        else {
            obj.consumer = [];
        }
        if (message.unstaked) {
            obj.unstaked = message.unstaked.map((e) => e ? stake_entry_1.StakeEntry.toJSON(e) : undefined);
        }
        else {
            obj.unstaked = [];
        }
        message.subscription !== undefined &&
            (obj.subscription = message.subscription ? subscription_1.Subscription.toJSON(message.subscription) : undefined);
        message.project !== undefined && (obj.project = message.project ? project_1.Project.toJSON(message.project) : undefined);
        return obj;
    },
    create(base) {
        return exports.QueryAccountInfoResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d;
        const message = createBaseQueryAccountInfoResponse();
        message.provider = ((_a = object.provider) === null || _a === void 0 ? void 0 : _a.map((e) => stake_entry_1.StakeEntry.fromPartial(e))) || [];
        message.frozen = ((_b = object.frozen) === null || _b === void 0 ? void 0 : _b.map((e) => stake_entry_1.StakeEntry.fromPartial(e))) || [];
        message.consumer = ((_c = object.consumer) === null || _c === void 0 ? void 0 : _c.map((e) => stake_entry_1.StakeEntry.fromPartial(e))) || [];
        message.unstaked = ((_d = object.unstaked) === null || _d === void 0 ? void 0 : _d.map((e) => stake_entry_1.StakeEntry.fromPartial(e))) || [];
        message.subscription = (object.subscription !== undefined && object.subscription !== null)
            ? subscription_1.Subscription.fromPartial(object.subscription)
            : undefined;
        message.project = (object.project !== undefined && object.project !== null)
            ? project_1.Project.fromPartial(object.project)
            : undefined;
        return message;
    },
};
class QueryClientImpl {
    constructor(rpc, opts) {
        this.service = (opts === null || opts === void 0 ? void 0 : opts.service) || "lavanet.lava.pairing.Query";
        this.rpc = rpc;
        this.Params = this.Params.bind(this);
        this.Providers = this.Providers.bind(this);
        this.GetPairing = this.GetPairing.bind(this);
        this.VerifyPairing = this.VerifyPairing.bind(this);
        this.UniquePaymentStorageClientProvider = this.UniquePaymentStorageClientProvider.bind(this);
        this.UniquePaymentStorageClientProviderAll = this.UniquePaymentStorageClientProviderAll.bind(this);
        this.ProviderPaymentStorage = this.ProviderPaymentStorage.bind(this);
        this.ProviderPaymentStorageAll = this.ProviderPaymentStorageAll.bind(this);
        this.EpochPayments = this.EpochPayments.bind(this);
        this.EpochPaymentsAll = this.EpochPaymentsAll.bind(this);
        this.UserEntry = this.UserEntry.bind(this);
        this.StaticProvidersList = this.StaticProvidersList.bind(this);
    }
    Params(request) {
        const data = exports.QueryParamsRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "Params", data);
        return promise.then((data) => exports.QueryParamsResponse.decode(minimal_1.default.Reader.create(data)));
    }
    Providers(request) {
        const data = exports.QueryProvidersRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "Providers", data);
        return promise.then((data) => exports.QueryProvidersResponse.decode(minimal_1.default.Reader.create(data)));
    }
    GetPairing(request) {
        const data = exports.QueryGetPairingRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "GetPairing", data);
        return promise.then((data) => exports.QueryGetPairingResponse.decode(minimal_1.default.Reader.create(data)));
    }
    VerifyPairing(request) {
        const data = exports.QueryVerifyPairingRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "VerifyPairing", data);
        return promise.then((data) => exports.QueryVerifyPairingResponse.decode(minimal_1.default.Reader.create(data)));
    }
    UniquePaymentStorageClientProvider(request) {
        const data = exports.QueryGetUniquePaymentStorageClientProviderRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "UniquePaymentStorageClientProvider", data);
        return promise.then((data) => exports.QueryGetUniquePaymentStorageClientProviderResponse.decode(minimal_1.default.Reader.create(data)));
    }
    UniquePaymentStorageClientProviderAll(request) {
        const data = exports.QueryAllUniquePaymentStorageClientProviderRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "UniquePaymentStorageClientProviderAll", data);
        return promise.then((data) => exports.QueryAllUniquePaymentStorageClientProviderResponse.decode(minimal_1.default.Reader.create(data)));
    }
    ProviderPaymentStorage(request) {
        const data = exports.QueryGetProviderPaymentStorageRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "ProviderPaymentStorage", data);
        return promise.then((data) => exports.QueryGetProviderPaymentStorageResponse.decode(minimal_1.default.Reader.create(data)));
    }
    ProviderPaymentStorageAll(request) {
        const data = exports.QueryAllProviderPaymentStorageRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "ProviderPaymentStorageAll", data);
        return promise.then((data) => exports.QueryAllProviderPaymentStorageResponse.decode(minimal_1.default.Reader.create(data)));
    }
    EpochPayments(request) {
        const data = exports.QueryGetEpochPaymentsRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "EpochPayments", data);
        return promise.then((data) => exports.QueryGetEpochPaymentsResponse.decode(minimal_1.default.Reader.create(data)));
    }
    EpochPaymentsAll(request) {
        const data = exports.QueryAllEpochPaymentsRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "EpochPaymentsAll", data);
        return promise.then((data) => exports.QueryAllEpochPaymentsResponse.decode(minimal_1.default.Reader.create(data)));
    }
    UserEntry(request) {
        const data = exports.QueryUserEntryRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "UserEntry", data);
        return promise.then((data) => exports.QueryUserEntryResponse.decode(minimal_1.default.Reader.create(data)));
    }
    StaticProvidersList(request) {
        const data = exports.QueryStaticProvidersListRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "StaticProvidersList", data);
        return promise.then((data) => exports.QueryStaticProvidersListResponse.decode(minimal_1.default.Reader.create(data)));
    }
}
exports.QueryClientImpl = QueryClientImpl;
if (minimal_1.default.util.Long !== long_1.default) {
    minimal_1.default.util.Long = long_1.default;
    minimal_1.default.configure();
}
function isSet(value) {
    return value !== null && value !== undefined;
}

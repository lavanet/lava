"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.GenesisState = exports.BadgeUsedCu = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const fixationEntry_1 = require("../common/fixationEntry");
const epoch_payments_1 = require("./epoch_payments");
const params_1 = require("./params");
const provider_payment_storage_1 = require("./provider_payment_storage");
const unique_payment_storage_client_provider_1 = require("./unique_payment_storage_client_provider");
exports.protobufPackage = "lavanet.lava.pairing";
function createBaseBadgeUsedCu() {
    return { badgeUsedCuKey: new Uint8Array(), usedCu: long_1.default.UZERO };
}
exports.BadgeUsedCu = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.badgeUsedCuKey.length !== 0) {
            writer.uint32(10).bytes(message.badgeUsedCuKey);
        }
        if (!message.usedCu.isZero()) {
            writer.uint32(16).uint64(message.usedCu);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseBadgeUsedCu();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.badgeUsedCuKey = reader.bytes();
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.usedCu = reader.uint64();
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
            badgeUsedCuKey: isSet(object.badgeUsedCuKey) ? bytesFromBase64(object.badgeUsedCuKey) : new Uint8Array(),
            usedCu: isSet(object.usedCu) ? long_1.default.fromValue(object.usedCu) : long_1.default.UZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.badgeUsedCuKey !== undefined &&
            (obj.badgeUsedCuKey = base64FromBytes(message.badgeUsedCuKey !== undefined ? message.badgeUsedCuKey : new Uint8Array()));
        message.usedCu !== undefined && (obj.usedCu = (message.usedCu || long_1.default.UZERO).toString());
        return obj;
    },
    create(base) {
        return exports.BadgeUsedCu.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseBadgeUsedCu();
        message.badgeUsedCuKey = (_a = object.badgeUsedCuKey) !== null && _a !== void 0 ? _a : new Uint8Array();
        message.usedCu = (object.usedCu !== undefined && object.usedCu !== null)
            ? long_1.default.fromValue(object.usedCu)
            : long_1.default.UZERO;
        return message;
    },
};
function createBaseGenesisState() {
    return {
        params: undefined,
        uniquePaymentStorageClientProviderList: [],
        providerPaymentStorageList: [],
        epochPaymentsList: [],
        badgeUsedCuList: [],
        badgesTS: [],
    };
}
exports.GenesisState = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.params !== undefined) {
            params_1.Params.encode(message.params, writer.uint32(10).fork()).ldelim();
        }
        for (const v of message.uniquePaymentStorageClientProviderList) {
            unique_payment_storage_client_provider_1.UniquePaymentStorageClientProvider.encode(v, writer.uint32(18).fork()).ldelim();
        }
        for (const v of message.providerPaymentStorageList) {
            provider_payment_storage_1.ProviderPaymentStorage.encode(v, writer.uint32(26).fork()).ldelim();
        }
        for (const v of message.epochPaymentsList) {
            epoch_payments_1.EpochPayments.encode(v, writer.uint32(34).fork()).ldelim();
        }
        for (const v of message.badgeUsedCuList) {
            exports.BadgeUsedCu.encode(v, writer.uint32(42).fork()).ldelim();
        }
        for (const v of message.badgesTS) {
            fixationEntry_1.RawMessage.encode(v, writer.uint32(50).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseGenesisState();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.params = params_1.Params.decode(reader, reader.uint32());
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.uniquePaymentStorageClientProviderList.push(unique_payment_storage_client_provider_1.UniquePaymentStorageClientProvider.decode(reader, reader.uint32()));
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.providerPaymentStorageList.push(provider_payment_storage_1.ProviderPaymentStorage.decode(reader, reader.uint32()));
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.epochPaymentsList.push(epoch_payments_1.EpochPayments.decode(reader, reader.uint32()));
                    continue;
                case 5:
                    if (tag != 42) {
                        break;
                    }
                    message.badgeUsedCuList.push(exports.BadgeUsedCu.decode(reader, reader.uint32()));
                    continue;
                case 6:
                    if (tag != 50) {
                        break;
                    }
                    message.badgesTS.push(fixationEntry_1.RawMessage.decode(reader, reader.uint32()));
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
            params: isSet(object.params) ? params_1.Params.fromJSON(object.params) : undefined,
            uniquePaymentStorageClientProviderList: Array.isArray(object === null || object === void 0 ? void 0 : object.uniquePaymentStorageClientProviderList)
                ? object.uniquePaymentStorageClientProviderList.map((e) => unique_payment_storage_client_provider_1.UniquePaymentStorageClientProvider.fromJSON(e))
                : [],
            providerPaymentStorageList: Array.isArray(object === null || object === void 0 ? void 0 : object.providerPaymentStorageList)
                ? object.providerPaymentStorageList.map((e) => provider_payment_storage_1.ProviderPaymentStorage.fromJSON(e))
                : [],
            epochPaymentsList: Array.isArray(object === null || object === void 0 ? void 0 : object.epochPaymentsList)
                ? object.epochPaymentsList.map((e) => epoch_payments_1.EpochPayments.fromJSON(e))
                : [],
            badgeUsedCuList: Array.isArray(object === null || object === void 0 ? void 0 : object.badgeUsedCuList)
                ? object.badgeUsedCuList.map((e) => exports.BadgeUsedCu.fromJSON(e))
                : [],
            badgesTS: Array.isArray(object === null || object === void 0 ? void 0 : object.badgesTS) ? object.badgesTS.map((e) => fixationEntry_1.RawMessage.fromJSON(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.params !== undefined && (obj.params = message.params ? params_1.Params.toJSON(message.params) : undefined);
        if (message.uniquePaymentStorageClientProviderList) {
            obj.uniquePaymentStorageClientProviderList = message.uniquePaymentStorageClientProviderList.map((e) => e ? unique_payment_storage_client_provider_1.UniquePaymentStorageClientProvider.toJSON(e) : undefined);
        }
        else {
            obj.uniquePaymentStorageClientProviderList = [];
        }
        if (message.providerPaymentStorageList) {
            obj.providerPaymentStorageList = message.providerPaymentStorageList.map((e) => e ? provider_payment_storage_1.ProviderPaymentStorage.toJSON(e) : undefined);
        }
        else {
            obj.providerPaymentStorageList = [];
        }
        if (message.epochPaymentsList) {
            obj.epochPaymentsList = message.epochPaymentsList.map((e) => e ? epoch_payments_1.EpochPayments.toJSON(e) : undefined);
        }
        else {
            obj.epochPaymentsList = [];
        }
        if (message.badgeUsedCuList) {
            obj.badgeUsedCuList = message.badgeUsedCuList.map((e) => e ? exports.BadgeUsedCu.toJSON(e) : undefined);
        }
        else {
            obj.badgeUsedCuList = [];
        }
        if (message.badgesTS) {
            obj.badgesTS = message.badgesTS.map((e) => e ? fixationEntry_1.RawMessage.toJSON(e) : undefined);
        }
        else {
            obj.badgesTS = [];
        }
        return obj;
    },
    create(base) {
        return exports.GenesisState.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d, _e;
        const message = createBaseGenesisState();
        message.params = (object.params !== undefined && object.params !== null)
            ? params_1.Params.fromPartial(object.params)
            : undefined;
        message.uniquePaymentStorageClientProviderList =
            ((_a = object.uniquePaymentStorageClientProviderList) === null || _a === void 0 ? void 0 : _a.map((e) => unique_payment_storage_client_provider_1.UniquePaymentStorageClientProvider.fromPartial(e))) ||
                [];
        message.providerPaymentStorageList =
            ((_b = object.providerPaymentStorageList) === null || _b === void 0 ? void 0 : _b.map((e) => provider_payment_storage_1.ProviderPaymentStorage.fromPartial(e))) || [];
        message.epochPaymentsList = ((_c = object.epochPaymentsList) === null || _c === void 0 ? void 0 : _c.map((e) => epoch_payments_1.EpochPayments.fromPartial(e))) || [];
        message.badgeUsedCuList = ((_d = object.badgeUsedCuList) === null || _d === void 0 ? void 0 : _d.map((e) => exports.BadgeUsedCu.fromPartial(e))) || [];
        message.badgesTS = ((_e = object.badgesTS) === null || _e === void 0 ? void 0 : _e.map((e) => fixationEntry_1.RawMessage.fromPartial(e))) || [];
        return message;
    },
};
var tsProtoGlobalThis = (() => {
    if (typeof globalThis !== "undefined") {
        return globalThis;
    }
    if (typeof self !== "undefined") {
        return self;
    }
    if (typeof window !== "undefined") {
        return window;
    }
    if (typeof global !== "undefined") {
        return global;
    }
    throw "Unable to locate global object";
})();
function bytesFromBase64(b64) {
    if (tsProtoGlobalThis.Buffer) {
        return Uint8Array.from(tsProtoGlobalThis.Buffer.from(b64, "base64"));
    }
    else {
        const bin = tsProtoGlobalThis.atob(b64);
        const arr = new Uint8Array(bin.length);
        for (let i = 0; i < bin.length; ++i) {
            arr[i] = bin.charCodeAt(i);
        }
        return arr;
    }
}
function base64FromBytes(arr) {
    if (tsProtoGlobalThis.Buffer) {
        return tsProtoGlobalThis.Buffer.from(arr).toString("base64");
    }
    else {
        const bin = [];
        arr.forEach((byte) => {
            bin.push(String.fromCharCode(byte));
        });
        return tsProtoGlobalThis.btoa(bin.join(""));
    }
}
if (minimal_1.default.util.Long !== long_1.default) {
    minimal_1.default.util.Long = long_1.default;
    minimal_1.default.configure();
}
function isSet(value) {
    return value !== null && value !== undefined;
}

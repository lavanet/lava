"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Spec = exports.spec_ProvidersTypesToJSON = exports.spec_ProvidersTypesFromJSON = exports.Spec_ProvidersTypes = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const coin_1 = require("../cosmos/base/v1beta1/coin");
const api_collection_1 = require("./api_collection");
exports.protobufPackage = "lavanet.lava.spec";
var Spec_ProvidersTypes;
(function (Spec_ProvidersTypes) {
    Spec_ProvidersTypes[Spec_ProvidersTypes["dynamic"] = 0] = "dynamic";
    Spec_ProvidersTypes[Spec_ProvidersTypes["static"] = 1] = "static";
    Spec_ProvidersTypes[Spec_ProvidersTypes["UNRECOGNIZED"] = -1] = "UNRECOGNIZED";
})(Spec_ProvidersTypes = exports.Spec_ProvidersTypes || (exports.Spec_ProvidersTypes = {}));
function spec_ProvidersTypesFromJSON(object) {
    switch (object) {
        case 0:
        case "dynamic":
            return Spec_ProvidersTypes.dynamic;
        case 1:
        case "static":
            return Spec_ProvidersTypes.static;
        case -1:
        case "UNRECOGNIZED":
        default:
            return Spec_ProvidersTypes.UNRECOGNIZED;
    }
}
exports.spec_ProvidersTypesFromJSON = spec_ProvidersTypesFromJSON;
function spec_ProvidersTypesToJSON(object) {
    switch (object) {
        case Spec_ProvidersTypes.dynamic:
            return "dynamic";
        case Spec_ProvidersTypes.static:
            return "static";
        case Spec_ProvidersTypes.UNRECOGNIZED:
        default:
            return "UNRECOGNIZED";
    }
}
exports.spec_ProvidersTypesToJSON = spec_ProvidersTypesToJSON;
function createBaseSpec() {
    return {
        index: "",
        name: "",
        enabled: false,
        reliabilityThreshold: 0,
        dataReliabilityEnabled: false,
        blockDistanceForFinalizedData: 0,
        blocksInFinalizationProof: 0,
        averageBlockTime: long_1.default.ZERO,
        allowedBlockLagForQosSync: long_1.default.ZERO,
        blockLastUpdated: long_1.default.UZERO,
        minStakeProvider: undefined,
        minStakeClient: undefined,
        providersTypes: 0,
        imports: [],
        apiCollections: [],
    };
}
exports.Spec = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        if (message.name !== "") {
            writer.uint32(18).string(message.name);
        }
        if (message.enabled === true) {
            writer.uint32(32).bool(message.enabled);
        }
        if (message.reliabilityThreshold !== 0) {
            writer.uint32(40).uint32(message.reliabilityThreshold);
        }
        if (message.dataReliabilityEnabled === true) {
            writer.uint32(48).bool(message.dataReliabilityEnabled);
        }
        if (message.blockDistanceForFinalizedData !== 0) {
            writer.uint32(56).uint32(message.blockDistanceForFinalizedData);
        }
        if (message.blocksInFinalizationProof !== 0) {
            writer.uint32(64).uint32(message.blocksInFinalizationProof);
        }
        if (!message.averageBlockTime.isZero()) {
            writer.uint32(72).int64(message.averageBlockTime);
        }
        if (!message.allowedBlockLagForQosSync.isZero()) {
            writer.uint32(80).int64(message.allowedBlockLagForQosSync);
        }
        if (!message.blockLastUpdated.isZero()) {
            writer.uint32(88).uint64(message.blockLastUpdated);
        }
        if (message.minStakeProvider !== undefined) {
            coin_1.Coin.encode(message.minStakeProvider, writer.uint32(98).fork()).ldelim();
        }
        if (message.minStakeClient !== undefined) {
            coin_1.Coin.encode(message.minStakeClient, writer.uint32(106).fork()).ldelim();
        }
        if (message.providersTypes !== 0) {
            writer.uint32(112).int32(message.providersTypes);
        }
        for (const v of message.imports) {
            writer.uint32(122).string(v);
        }
        for (const v of message.apiCollections) {
            api_collection_1.ApiCollection.encode(v, writer.uint32(130).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseSpec();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.index = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.name = reader.string();
                    continue;
                case 4:
                    if (tag != 32) {
                        break;
                    }
                    message.enabled = reader.bool();
                    continue;
                case 5:
                    if (tag != 40) {
                        break;
                    }
                    message.reliabilityThreshold = reader.uint32();
                    continue;
                case 6:
                    if (tag != 48) {
                        break;
                    }
                    message.dataReliabilityEnabled = reader.bool();
                    continue;
                case 7:
                    if (tag != 56) {
                        break;
                    }
                    message.blockDistanceForFinalizedData = reader.uint32();
                    continue;
                case 8:
                    if (tag != 64) {
                        break;
                    }
                    message.blocksInFinalizationProof = reader.uint32();
                    continue;
                case 9:
                    if (tag != 72) {
                        break;
                    }
                    message.averageBlockTime = reader.int64();
                    continue;
                case 10:
                    if (tag != 80) {
                        break;
                    }
                    message.allowedBlockLagForQosSync = reader.int64();
                    continue;
                case 11:
                    if (tag != 88) {
                        break;
                    }
                    message.blockLastUpdated = reader.uint64();
                    continue;
                case 12:
                    if (tag != 98) {
                        break;
                    }
                    message.minStakeProvider = coin_1.Coin.decode(reader, reader.uint32());
                    continue;
                case 13:
                    if (tag != 106) {
                        break;
                    }
                    message.minStakeClient = coin_1.Coin.decode(reader, reader.uint32());
                    continue;
                case 14:
                    if (tag != 112) {
                        break;
                    }
                    message.providersTypes = reader.int32();
                    continue;
                case 15:
                    if (tag != 122) {
                        break;
                    }
                    message.imports.push(reader.string());
                    continue;
                case 16:
                    if (tag != 130) {
                        break;
                    }
                    message.apiCollections.push(api_collection_1.ApiCollection.decode(reader, reader.uint32()));
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
            index: isSet(object.index) ? String(object.index) : "",
            name: isSet(object.name) ? String(object.name) : "",
            enabled: isSet(object.enabled) ? Boolean(object.enabled) : false,
            reliabilityThreshold: isSet(object.reliabilityThreshold) ? Number(object.reliabilityThreshold) : 0,
            dataReliabilityEnabled: isSet(object.dataReliabilityEnabled) ? Boolean(object.dataReliabilityEnabled) : false,
            blockDistanceForFinalizedData: isSet(object.blockDistanceForFinalizedData)
                ? Number(object.blockDistanceForFinalizedData)
                : 0,
            blocksInFinalizationProof: isSet(object.blocksInFinalizationProof) ? Number(object.blocksInFinalizationProof) : 0,
            averageBlockTime: isSet(object.averageBlockTime) ? long_1.default.fromValue(object.averageBlockTime) : long_1.default.ZERO,
            allowedBlockLagForQosSync: isSet(object.allowedBlockLagForQosSync)
                ? long_1.default.fromValue(object.allowedBlockLagForQosSync)
                : long_1.default.ZERO,
            blockLastUpdated: isSet(object.blockLastUpdated) ? long_1.default.fromValue(object.blockLastUpdated) : long_1.default.UZERO,
            minStakeProvider: isSet(object.minStakeProvider) ? coin_1.Coin.fromJSON(object.minStakeProvider) : undefined,
            minStakeClient: isSet(object.minStakeClient) ? coin_1.Coin.fromJSON(object.minStakeClient) : undefined,
            providersTypes: isSet(object.providersTypes) ? spec_ProvidersTypesFromJSON(object.providersTypes) : 0,
            imports: Array.isArray(object === null || object === void 0 ? void 0 : object.imports) ? object.imports.map((e) => String(e)) : [],
            apiCollections: Array.isArray(object === null || object === void 0 ? void 0 : object.apiCollections)
                ? object.apiCollections.map((e) => api_collection_1.ApiCollection.fromJSON(e))
                : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = message.index);
        message.name !== undefined && (obj.name = message.name);
        message.enabled !== undefined && (obj.enabled = message.enabled);
        message.reliabilityThreshold !== undefined && (obj.reliabilityThreshold = Math.round(message.reliabilityThreshold));
        message.dataReliabilityEnabled !== undefined && (obj.dataReliabilityEnabled = message.dataReliabilityEnabled);
        message.blockDistanceForFinalizedData !== undefined &&
            (obj.blockDistanceForFinalizedData = Math.round(message.blockDistanceForFinalizedData));
        message.blocksInFinalizationProof !== undefined &&
            (obj.blocksInFinalizationProof = Math.round(message.blocksInFinalizationProof));
        message.averageBlockTime !== undefined &&
            (obj.averageBlockTime = (message.averageBlockTime || long_1.default.ZERO).toString());
        message.allowedBlockLagForQosSync !== undefined &&
            (obj.allowedBlockLagForQosSync = (message.allowedBlockLagForQosSync || long_1.default.ZERO).toString());
        message.blockLastUpdated !== undefined &&
            (obj.blockLastUpdated = (message.blockLastUpdated || long_1.default.UZERO).toString());
        message.minStakeProvider !== undefined &&
            (obj.minStakeProvider = message.minStakeProvider ? coin_1.Coin.toJSON(message.minStakeProvider) : undefined);
        message.minStakeClient !== undefined &&
            (obj.minStakeClient = message.minStakeClient ? coin_1.Coin.toJSON(message.minStakeClient) : undefined);
        message.providersTypes !== undefined && (obj.providersTypes = spec_ProvidersTypesToJSON(message.providersTypes));
        if (message.imports) {
            obj.imports = message.imports.map((e) => e);
        }
        else {
            obj.imports = [];
        }
        if (message.apiCollections) {
            obj.apiCollections = message.apiCollections.map((e) => e ? api_collection_1.ApiCollection.toJSON(e) : undefined);
        }
        else {
            obj.apiCollections = [];
        }
        return obj;
    },
    create(base) {
        return exports.Spec.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d, _e, _f, _g, _h, _j, _k;
        const message = createBaseSpec();
        message.index = (_a = object.index) !== null && _a !== void 0 ? _a : "";
        message.name = (_b = object.name) !== null && _b !== void 0 ? _b : "";
        message.enabled = (_c = object.enabled) !== null && _c !== void 0 ? _c : false;
        message.reliabilityThreshold = (_d = object.reliabilityThreshold) !== null && _d !== void 0 ? _d : 0;
        message.dataReliabilityEnabled = (_e = object.dataReliabilityEnabled) !== null && _e !== void 0 ? _e : false;
        message.blockDistanceForFinalizedData = (_f = object.blockDistanceForFinalizedData) !== null && _f !== void 0 ? _f : 0;
        message.blocksInFinalizationProof = (_g = object.blocksInFinalizationProof) !== null && _g !== void 0 ? _g : 0;
        message.averageBlockTime = (object.averageBlockTime !== undefined && object.averageBlockTime !== null)
            ? long_1.default.fromValue(object.averageBlockTime)
            : long_1.default.ZERO;
        message.allowedBlockLagForQosSync =
            (object.allowedBlockLagForQosSync !== undefined && object.allowedBlockLagForQosSync !== null)
                ? long_1.default.fromValue(object.allowedBlockLagForQosSync)
                : long_1.default.ZERO;
        message.blockLastUpdated = (object.blockLastUpdated !== undefined && object.blockLastUpdated !== null)
            ? long_1.default.fromValue(object.blockLastUpdated)
            : long_1.default.UZERO;
        message.minStakeProvider = (object.minStakeProvider !== undefined && object.minStakeProvider !== null)
            ? coin_1.Coin.fromPartial(object.minStakeProvider)
            : undefined;
        message.minStakeClient = (object.minStakeClient !== undefined && object.minStakeClient !== null)
            ? coin_1.Coin.fromPartial(object.minStakeClient)
            : undefined;
        message.providersTypes = (_h = object.providersTypes) !== null && _h !== void 0 ? _h : 0;
        message.imports = ((_j = object.imports) === null || _j === void 0 ? void 0 : _j.map((e) => e)) || [];
        message.apiCollections = ((_k = object.apiCollections) === null || _k === void 0 ? void 0 : _k.map((e) => api_collection_1.ApiCollection.fromPartial(e))) || [];
        return message;
    },
};
if (minimal_1.default.util.Long !== long_1.default) {
    minimal_1.default.util.Long = long_1.default;
    minimal_1.default.configure();
}
function isSet(value) {
    return value !== null && value !== undefined;
}

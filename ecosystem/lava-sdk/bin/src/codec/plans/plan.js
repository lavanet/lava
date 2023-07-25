"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.ChainPolicy = exports.Policy = exports.Plan = exports.geolocationToJSON = exports.geolocationFromJSON = exports.Geolocation = exports.selectedProvidersModeToJSON = exports.selectedProvidersModeFromJSON = exports.selectedProvidersMode = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const coin_1 = require("../cosmos/base/v1beta1/coin");
exports.protobufPackage = "lavanet.lava.plans";
/** the enum below determines the pairing algorithm's behaviour with the selected providers feature */
var selectedProvidersMode;
(function (selectedProvidersMode) {
    /** ALLOWED - no providers restrictions */
    selectedProvidersMode[selectedProvidersMode["ALLOWED"] = 0] = "ALLOWED";
    /** MIXED - use the selected providers mixed with randomly chosen providers */
    selectedProvidersMode[selectedProvidersMode["MIXED"] = 1] = "MIXED";
    /** EXCLUSIVE - use only the selected providers */
    selectedProvidersMode[selectedProvidersMode["EXCLUSIVE"] = 2] = "EXCLUSIVE";
    /** DISABLED - selected providers feature is disabled */
    selectedProvidersMode[selectedProvidersMode["DISABLED"] = 3] = "DISABLED";
    selectedProvidersMode[selectedProvidersMode["UNRECOGNIZED"] = -1] = "UNRECOGNIZED";
})(selectedProvidersMode = exports.selectedProvidersMode || (exports.selectedProvidersMode = {}));
function selectedProvidersModeFromJSON(object) {
    switch (object) {
        case 0:
        case "ALLOWED":
            return selectedProvidersMode.ALLOWED;
        case 1:
        case "MIXED":
            return selectedProvidersMode.MIXED;
        case 2:
        case "EXCLUSIVE":
            return selectedProvidersMode.EXCLUSIVE;
        case 3:
        case "DISABLED":
            return selectedProvidersMode.DISABLED;
        case -1:
        case "UNRECOGNIZED":
        default:
            return selectedProvidersMode.UNRECOGNIZED;
    }
}
exports.selectedProvidersModeFromJSON = selectedProvidersModeFromJSON;
function selectedProvidersModeToJSON(object) {
    switch (object) {
        case selectedProvidersMode.ALLOWED:
            return "ALLOWED";
        case selectedProvidersMode.MIXED:
            return "MIXED";
        case selectedProvidersMode.EXCLUSIVE:
            return "EXCLUSIVE";
        case selectedProvidersMode.DISABLED:
            return "DISABLED";
        case selectedProvidersMode.UNRECOGNIZED:
        default:
            return "UNRECOGNIZED";
    }
}
exports.selectedProvidersModeToJSON = selectedProvidersModeToJSON;
/**
 * The geolocation values are encoded as bits in a bitmask, with two special values:
 * GLS is set to 0 so it will be restrictive with the AND operator.
 * GL is set to -1 so it will be permissive with the AND operator.
 */
var Geolocation;
(function (Geolocation) {
    /** GLS - Global-strict */
    Geolocation[Geolocation["GLS"] = 0] = "GLS";
    /** USC - US-Center */
    Geolocation[Geolocation["USC"] = 1] = "USC";
    Geolocation[Geolocation["EU"] = 2] = "EU";
    /** USE - US-East */
    Geolocation[Geolocation["USE"] = 4] = "USE";
    /** USW - US-West */
    Geolocation[Geolocation["USW"] = 8] = "USW";
    Geolocation[Geolocation["AF"] = 16] = "AF";
    Geolocation[Geolocation["AS"] = 32] = "AS";
    /** AU - (includes NZ) */
    Geolocation[Geolocation["AU"] = 64] = "AU";
    /** GL - Global */
    Geolocation[Geolocation["GL"] = 65535] = "GL";
    Geolocation[Geolocation["UNRECOGNIZED"] = -1] = "UNRECOGNIZED";
})(Geolocation = exports.Geolocation || (exports.Geolocation = {}));
function geolocationFromJSON(object) {
    switch (object) {
        case 0:
        case "GLS":
            return Geolocation.GLS;
        case 1:
        case "USC":
            return Geolocation.USC;
        case 2:
        case "EU":
            return Geolocation.EU;
        case 4:
        case "USE":
            return Geolocation.USE;
        case 8:
        case "USW":
            return Geolocation.USW;
        case 16:
        case "AF":
            return Geolocation.AF;
        case 32:
        case "AS":
            return Geolocation.AS;
        case 64:
        case "AU":
            return Geolocation.AU;
        case 65535:
        case "GL":
            return Geolocation.GL;
        case -1:
        case "UNRECOGNIZED":
        default:
            return Geolocation.UNRECOGNIZED;
    }
}
exports.geolocationFromJSON = geolocationFromJSON;
function geolocationToJSON(object) {
    switch (object) {
        case Geolocation.GLS:
            return "GLS";
        case Geolocation.USC:
            return "USC";
        case Geolocation.EU:
            return "EU";
        case Geolocation.USE:
            return "USE";
        case Geolocation.USW:
            return "USW";
        case Geolocation.AF:
            return "AF";
        case Geolocation.AS:
            return "AS";
        case Geolocation.AU:
            return "AU";
        case Geolocation.GL:
            return "GL";
        case Geolocation.UNRECOGNIZED:
        default:
            return "UNRECOGNIZED";
    }
}
exports.geolocationToJSON = geolocationToJSON;
function createBasePlan() {
    return {
        index: "",
        block: long_1.default.UZERO,
        price: undefined,
        allowOveruse: false,
        overuseRate: long_1.default.UZERO,
        description: "",
        type: "",
        annualDiscountPercentage: long_1.default.UZERO,
        planPolicy: undefined,
    };
}
exports.Plan = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        if (!message.block.isZero()) {
            writer.uint32(24).uint64(message.block);
        }
        if (message.price !== undefined) {
            coin_1.Coin.encode(message.price, writer.uint32(34).fork()).ldelim();
        }
        if (message.allowOveruse === true) {
            writer.uint32(64).bool(message.allowOveruse);
        }
        if (!message.overuseRate.isZero()) {
            writer.uint32(72).uint64(message.overuseRate);
        }
        if (message.description !== "") {
            writer.uint32(90).string(message.description);
        }
        if (message.type !== "") {
            writer.uint32(98).string(message.type);
        }
        if (!message.annualDiscountPercentage.isZero()) {
            writer.uint32(104).uint64(message.annualDiscountPercentage);
        }
        if (message.planPolicy !== undefined) {
            exports.Policy.encode(message.planPolicy, writer.uint32(114).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBasePlan();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.index = reader.string();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.block = reader.uint64();
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.price = coin_1.Coin.decode(reader, reader.uint32());
                    continue;
                case 8:
                    if (tag != 64) {
                        break;
                    }
                    message.allowOveruse = reader.bool();
                    continue;
                case 9:
                    if (tag != 72) {
                        break;
                    }
                    message.overuseRate = reader.uint64();
                    continue;
                case 11:
                    if (tag != 90) {
                        break;
                    }
                    message.description = reader.string();
                    continue;
                case 12:
                    if (tag != 98) {
                        break;
                    }
                    message.type = reader.string();
                    continue;
                case 13:
                    if (tag != 104) {
                        break;
                    }
                    message.annualDiscountPercentage = reader.uint64();
                    continue;
                case 14:
                    if (tag != 114) {
                        break;
                    }
                    message.planPolicy = exports.Policy.decode(reader, reader.uint32());
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
            block: isSet(object.block) ? long_1.default.fromValue(object.block) : long_1.default.UZERO,
            price: isSet(object.price) ? coin_1.Coin.fromJSON(object.price) : undefined,
            allowOveruse: isSet(object.allowOveruse) ? Boolean(object.allowOveruse) : false,
            overuseRate: isSet(object.overuseRate) ? long_1.default.fromValue(object.overuseRate) : long_1.default.UZERO,
            description: isSet(object.description) ? String(object.description) : "",
            type: isSet(object.type) ? String(object.type) : "",
            annualDiscountPercentage: isSet(object.annualDiscountPercentage)
                ? long_1.default.fromValue(object.annualDiscountPercentage)
                : long_1.default.UZERO,
            planPolicy: isSet(object.planPolicy) ? exports.Policy.fromJSON(object.planPolicy) : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = message.index);
        message.block !== undefined && (obj.block = (message.block || long_1.default.UZERO).toString());
        message.price !== undefined && (obj.price = message.price ? coin_1.Coin.toJSON(message.price) : undefined);
        message.allowOveruse !== undefined && (obj.allowOveruse = message.allowOveruse);
        message.overuseRate !== undefined && (obj.overuseRate = (message.overuseRate || long_1.default.UZERO).toString());
        message.description !== undefined && (obj.description = message.description);
        message.type !== undefined && (obj.type = message.type);
        message.annualDiscountPercentage !== undefined &&
            (obj.annualDiscountPercentage = (message.annualDiscountPercentage || long_1.default.UZERO).toString());
        message.planPolicy !== undefined &&
            (obj.planPolicy = message.planPolicy ? exports.Policy.toJSON(message.planPolicy) : undefined);
        return obj;
    },
    create(base) {
        return exports.Plan.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d;
        const message = createBasePlan();
        message.index = (_a = object.index) !== null && _a !== void 0 ? _a : "";
        message.block = (object.block !== undefined && object.block !== null) ? long_1.default.fromValue(object.block) : long_1.default.UZERO;
        message.price = (object.price !== undefined && object.price !== null) ? coin_1.Coin.fromPartial(object.price) : undefined;
        message.allowOveruse = (_b = object.allowOveruse) !== null && _b !== void 0 ? _b : false;
        message.overuseRate = (object.overuseRate !== undefined && object.overuseRate !== null)
            ? long_1.default.fromValue(object.overuseRate)
            : long_1.default.UZERO;
        message.description = (_c = object.description) !== null && _c !== void 0 ? _c : "";
        message.type = (_d = object.type) !== null && _d !== void 0 ? _d : "";
        message.annualDiscountPercentage =
            (object.annualDiscountPercentage !== undefined && object.annualDiscountPercentage !== null)
                ? long_1.default.fromValue(object.annualDiscountPercentage)
                : long_1.default.UZERO;
        message.planPolicy = (object.planPolicy !== undefined && object.planPolicy !== null)
            ? exports.Policy.fromPartial(object.planPolicy)
            : undefined;
        return message;
    },
};
function createBasePolicy() {
    return {
        chainPolicies: [],
        geolocationProfile: long_1.default.UZERO,
        totalCuLimit: long_1.default.UZERO,
        epochCuLimit: long_1.default.UZERO,
        maxProvidersToPair: long_1.default.UZERO,
        selectedProvidersMode: 0,
        selectedProviders: [],
    };
}
exports.Policy = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        for (const v of message.chainPolicies) {
            exports.ChainPolicy.encode(v, writer.uint32(10).fork()).ldelim();
        }
        if (!message.geolocationProfile.isZero()) {
            writer.uint32(16).uint64(message.geolocationProfile);
        }
        if (!message.totalCuLimit.isZero()) {
            writer.uint32(24).uint64(message.totalCuLimit);
        }
        if (!message.epochCuLimit.isZero()) {
            writer.uint32(32).uint64(message.epochCuLimit);
        }
        if (!message.maxProvidersToPair.isZero()) {
            writer.uint32(40).uint64(message.maxProvidersToPair);
        }
        if (message.selectedProvidersMode !== 0) {
            writer.uint32(48).int32(message.selectedProvidersMode);
        }
        for (const v of message.selectedProviders) {
            writer.uint32(58).string(v);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBasePolicy();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.chainPolicies.push(exports.ChainPolicy.decode(reader, reader.uint32()));
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.geolocationProfile = reader.uint64();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.totalCuLimit = reader.uint64();
                    continue;
                case 4:
                    if (tag != 32) {
                        break;
                    }
                    message.epochCuLimit = reader.uint64();
                    continue;
                case 5:
                    if (tag != 40) {
                        break;
                    }
                    message.maxProvidersToPair = reader.uint64();
                    continue;
                case 6:
                    if (tag != 48) {
                        break;
                    }
                    message.selectedProvidersMode = reader.int32();
                    continue;
                case 7:
                    if (tag != 58) {
                        break;
                    }
                    message.selectedProviders.push(reader.string());
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
            chainPolicies: Array.isArray(object === null || object === void 0 ? void 0 : object.chainPolicies)
                ? object.chainPolicies.map((e) => exports.ChainPolicy.fromJSON(e))
                : [],
            geolocationProfile: isSet(object.geolocationProfile) ? long_1.default.fromValue(object.geolocationProfile) : long_1.default.UZERO,
            totalCuLimit: isSet(object.totalCuLimit) ? long_1.default.fromValue(object.totalCuLimit) : long_1.default.UZERO,
            epochCuLimit: isSet(object.epochCuLimit) ? long_1.default.fromValue(object.epochCuLimit) : long_1.default.UZERO,
            maxProvidersToPair: isSet(object.maxProvidersToPair) ? long_1.default.fromValue(object.maxProvidersToPair) : long_1.default.UZERO,
            selectedProvidersMode: isSet(object.selectedProvidersMode)
                ? selectedProvidersModeFromJSON(object.selectedProvidersMode)
                : 0,
            selectedProviders: Array.isArray(object === null || object === void 0 ? void 0 : object.selectedProviders)
                ? object.selectedProviders.map((e) => String(e))
                : [],
        };
    },
    toJSON(message) {
        const obj = {};
        if (message.chainPolicies) {
            obj.chainPolicies = message.chainPolicies.map((e) => e ? exports.ChainPolicy.toJSON(e) : undefined);
        }
        else {
            obj.chainPolicies = [];
        }
        message.geolocationProfile !== undefined &&
            (obj.geolocationProfile = (message.geolocationProfile || long_1.default.UZERO).toString());
        message.totalCuLimit !== undefined && (obj.totalCuLimit = (message.totalCuLimit || long_1.default.UZERO).toString());
        message.epochCuLimit !== undefined && (obj.epochCuLimit = (message.epochCuLimit || long_1.default.UZERO).toString());
        message.maxProvidersToPair !== undefined &&
            (obj.maxProvidersToPair = (message.maxProvidersToPair || long_1.default.UZERO).toString());
        message.selectedProvidersMode !== undefined &&
            (obj.selectedProvidersMode = selectedProvidersModeToJSON(message.selectedProvidersMode));
        if (message.selectedProviders) {
            obj.selectedProviders = message.selectedProviders.map((e) => e);
        }
        else {
            obj.selectedProviders = [];
        }
        return obj;
    },
    create(base) {
        return exports.Policy.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBasePolicy();
        message.chainPolicies = ((_a = object.chainPolicies) === null || _a === void 0 ? void 0 : _a.map((e) => exports.ChainPolicy.fromPartial(e))) || [];
        message.geolocationProfile = (object.geolocationProfile !== undefined && object.geolocationProfile !== null)
            ? long_1.default.fromValue(object.geolocationProfile)
            : long_1.default.UZERO;
        message.totalCuLimit = (object.totalCuLimit !== undefined && object.totalCuLimit !== null)
            ? long_1.default.fromValue(object.totalCuLimit)
            : long_1.default.UZERO;
        message.epochCuLimit = (object.epochCuLimit !== undefined && object.epochCuLimit !== null)
            ? long_1.default.fromValue(object.epochCuLimit)
            : long_1.default.UZERO;
        message.maxProvidersToPair = (object.maxProvidersToPair !== undefined && object.maxProvidersToPair !== null)
            ? long_1.default.fromValue(object.maxProvidersToPair)
            : long_1.default.UZERO;
        message.selectedProvidersMode = (_b = object.selectedProvidersMode) !== null && _b !== void 0 ? _b : 0;
        message.selectedProviders = ((_c = object.selectedProviders) === null || _c === void 0 ? void 0 : _c.map((e) => e)) || [];
        return message;
    },
};
function createBaseChainPolicy() {
    return { chainId: "", apis: [] };
}
exports.ChainPolicy = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.chainId !== "") {
            writer.uint32(10).string(message.chainId);
        }
        for (const v of message.apis) {
            writer.uint32(18).string(v);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseChainPolicy();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.chainId = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.apis.push(reader.string());
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
            chainId: isSet(object.chainId) ? String(object.chainId) : "",
            apis: Array.isArray(object === null || object === void 0 ? void 0 : object.apis) ? object.apis.map((e) => String(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.chainId !== undefined && (obj.chainId = message.chainId);
        if (message.apis) {
            obj.apis = message.apis.map((e) => e);
        }
        else {
            obj.apis = [];
        }
        return obj;
    },
    create(base) {
        return exports.ChainPolicy.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseChainPolicy();
        message.chainId = (_a = object.chainId) !== null && _a !== void 0 ? _a : "";
        message.apis = ((_b = object.apis) === null || _b === void 0 ? void 0 : _b.map((e) => e)) || [];
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

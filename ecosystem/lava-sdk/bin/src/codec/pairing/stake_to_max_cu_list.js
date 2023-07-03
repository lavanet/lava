"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.StakeToMaxCU = exports.StakeToMaxCUList = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const coin_1 = require("../cosmos/base/v1beta1/coin");
exports.protobufPackage = "lavanet.lava.pairing";
function createBaseStakeToMaxCUList() {
    return { List: [] };
}
exports.StakeToMaxCUList = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        for (const v of message.List) {
            exports.StakeToMaxCU.encode(v, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseStakeToMaxCUList();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag !== 10) {
                        break;
                    }
                    message.List.push(exports.StakeToMaxCU.decode(reader, reader.uint32()));
                    continue;
            }
            if ((tag & 7) === 4 || tag === 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return { List: Array.isArray(object === null || object === void 0 ? void 0 : object.List) ? object.List.map((e) => exports.StakeToMaxCU.fromJSON(e)) : [] };
    },
    toJSON(message) {
        const obj = {};
        if (message.List) {
            obj.List = message.List.map((e) => e ? exports.StakeToMaxCU.toJSON(e) : undefined);
        }
        else {
            obj.List = [];
        }
        return obj;
    },
    create(base) {
        return exports.StakeToMaxCUList.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseStakeToMaxCUList();
        message.List = ((_a = object.List) === null || _a === void 0 ? void 0 : _a.map((e) => exports.StakeToMaxCU.fromPartial(e))) || [];
        return message;
    },
};
function createBaseStakeToMaxCU() {
    return { StakeThreshold: undefined, MaxComputeUnits: long_1.default.UZERO };
}
exports.StakeToMaxCU = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.StakeThreshold !== undefined) {
            coin_1.Coin.encode(message.StakeThreshold, writer.uint32(10).fork()).ldelim();
        }
        if (!message.MaxComputeUnits.isZero()) {
            writer.uint32(16).uint64(message.MaxComputeUnits);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseStakeToMaxCU();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag !== 10) {
                        break;
                    }
                    message.StakeThreshold = coin_1.Coin.decode(reader, reader.uint32());
                    continue;
                case 2:
                    if (tag !== 16) {
                        break;
                    }
                    message.MaxComputeUnits = reader.uint64();
                    continue;
            }
            if ((tag & 7) === 4 || tag === 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            StakeThreshold: isSet(object.StakeThreshold) ? coin_1.Coin.fromJSON(object.StakeThreshold) : undefined,
            MaxComputeUnits: isSet(object.MaxComputeUnits) ? long_1.default.fromValue(object.MaxComputeUnits) : long_1.default.UZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.StakeThreshold !== undefined &&
            (obj.StakeThreshold = message.StakeThreshold ? coin_1.Coin.toJSON(message.StakeThreshold) : undefined);
        message.MaxComputeUnits !== undefined && (obj.MaxComputeUnits = (message.MaxComputeUnits || long_1.default.UZERO).toString());
        return obj;
    },
    create(base) {
        return exports.StakeToMaxCU.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseStakeToMaxCU();
        message.StakeThreshold = (object.StakeThreshold !== undefined && object.StakeThreshold !== null)
            ? coin_1.Coin.fromPartial(object.StakeThreshold)
            : undefined;
        message.MaxComputeUnits = (object.MaxComputeUnits !== undefined && object.MaxComputeUnits !== null)
            ? long_1.default.fromValue(object.MaxComputeUnits)
            : long_1.default.UZERO;
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

"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.StakeEntry = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const coin_1 = require("../cosmos/base/v1beta1/coin");
const endpoint_1 = require("./endpoint");
exports.protobufPackage = "lavanet.lava.epochstorage";
function createBaseStakeEntry() {
    return {
        stake: undefined,
        address: "",
        stakeAppliedBlock: long_1.default.UZERO,
        endpoints: [],
        geolocation: long_1.default.UZERO,
        chain: "",
        moniker: "",
    };
}
exports.StakeEntry = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.stake !== undefined) {
            coin_1.Coin.encode(message.stake, writer.uint32(10).fork()).ldelim();
        }
        if (message.address !== "") {
            writer.uint32(18).string(message.address);
        }
        if (!message.stakeAppliedBlock.isZero()) {
            writer.uint32(24).uint64(message.stakeAppliedBlock);
        }
        for (const v of message.endpoints) {
            endpoint_1.Endpoint.encode(v, writer.uint32(34).fork()).ldelim();
        }
        if (!message.geolocation.isZero()) {
            writer.uint32(40).uint64(message.geolocation);
        }
        if (message.chain !== "") {
            writer.uint32(50).string(message.chain);
        }
        if (message.moniker !== "") {
            writer.uint32(66).string(message.moniker);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseStakeEntry();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.stake = coin_1.Coin.decode(reader, reader.uint32());
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.address = reader.string();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.stakeAppliedBlock = reader.uint64();
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.endpoints.push(endpoint_1.Endpoint.decode(reader, reader.uint32()));
                    continue;
                case 5:
                    if (tag != 40) {
                        break;
                    }
                    message.geolocation = reader.uint64();
                    continue;
                case 6:
                    if (tag != 50) {
                        break;
                    }
                    message.chain = reader.string();
                    continue;
                case 8:
                    if (tag != 66) {
                        break;
                    }
                    message.moniker = reader.string();
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
            stake: isSet(object.stake) ? coin_1.Coin.fromJSON(object.stake) : undefined,
            address: isSet(object.address) ? String(object.address) : "",
            stakeAppliedBlock: isSet(object.stakeAppliedBlock) ? long_1.default.fromValue(object.stakeAppliedBlock) : long_1.default.UZERO,
            endpoints: Array.isArray(object === null || object === void 0 ? void 0 : object.endpoints) ? object.endpoints.map((e) => endpoint_1.Endpoint.fromJSON(e)) : [],
            geolocation: isSet(object.geolocation) ? long_1.default.fromValue(object.geolocation) : long_1.default.UZERO,
            chain: isSet(object.chain) ? String(object.chain) : "",
            moniker: isSet(object.moniker) ? String(object.moniker) : "",
        };
    },
    toJSON(message) {
        const obj = {};
        message.stake !== undefined && (obj.stake = message.stake ? coin_1.Coin.toJSON(message.stake) : undefined);
        message.address !== undefined && (obj.address = message.address);
        message.stakeAppliedBlock !== undefined &&
            (obj.stakeAppliedBlock = (message.stakeAppliedBlock || long_1.default.UZERO).toString());
        if (message.endpoints) {
            obj.endpoints = message.endpoints.map((e) => e ? endpoint_1.Endpoint.toJSON(e) : undefined);
        }
        else {
            obj.endpoints = [];
        }
        message.geolocation !== undefined && (obj.geolocation = (message.geolocation || long_1.default.UZERO).toString());
        message.chain !== undefined && (obj.chain = message.chain);
        message.moniker !== undefined && (obj.moniker = message.moniker);
        return obj;
    },
    create(base) {
        return exports.StakeEntry.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d;
        const message = createBaseStakeEntry();
        message.stake = (object.stake !== undefined && object.stake !== null) ? coin_1.Coin.fromPartial(object.stake) : undefined;
        message.address = (_a = object.address) !== null && _a !== void 0 ? _a : "";
        message.stakeAppliedBlock = (object.stakeAppliedBlock !== undefined && object.stakeAppliedBlock !== null)
            ? long_1.default.fromValue(object.stakeAppliedBlock)
            : long_1.default.UZERO;
        message.endpoints = ((_b = object.endpoints) === null || _b === void 0 ? void 0 : _b.map((e) => endpoint_1.Endpoint.fromPartial(e))) || [];
        message.geolocation = (object.geolocation !== undefined && object.geolocation !== null)
            ? long_1.default.fromValue(object.geolocation)
            : long_1.default.UZERO;
        message.chain = (_c = object.chain) !== null && _c !== void 0 ? _c : "";
        message.moniker = (_d = object.moniker) !== null && _d !== void 0 ? _d : "";
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

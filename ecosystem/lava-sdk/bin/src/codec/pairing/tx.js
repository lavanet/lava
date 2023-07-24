"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.MsgClientImpl = exports.MsgUnfreezeProviderResponse = exports.MsgUnfreezeProvider = exports.MsgFreezeProviderResponse = exports.MsgFreezeProvider = exports.MsgRelayPaymentResponse = exports.MsgRelayPayment = exports.MsgUnstakeProviderResponse = exports.MsgUnstakeProvider = exports.MsgStakeProviderResponse = exports.MsgStakeProvider = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const coin_1 = require("../cosmos/base/v1beta1/coin");
const endpoint_1 = require("../epochstorage/endpoint");
const relay_1 = require("./relay");
exports.protobufPackage = "lavanet.lava.pairing";
function createBaseMsgStakeProvider() {
    return { creator: "", chainID: "", amount: undefined, endpoints: [], geolocation: long_1.default.UZERO, moniker: "" };
}
exports.MsgStakeProvider = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.creator !== "") {
            writer.uint32(10).string(message.creator);
        }
        if (message.chainID !== "") {
            writer.uint32(18).string(message.chainID);
        }
        if (message.amount !== undefined) {
            coin_1.Coin.encode(message.amount, writer.uint32(26).fork()).ldelim();
        }
        for (const v of message.endpoints) {
            endpoint_1.Endpoint.encode(v, writer.uint32(34).fork()).ldelim();
        }
        if (!message.geolocation.isZero()) {
            writer.uint32(40).uint64(message.geolocation);
        }
        if (message.moniker !== "") {
            writer.uint32(50).string(message.moniker);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgStakeProvider();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.creator = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.chainID = reader.string();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.amount = coin_1.Coin.decode(reader, reader.uint32());
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
            creator: isSet(object.creator) ? String(object.creator) : "",
            chainID: isSet(object.chainID) ? String(object.chainID) : "",
            amount: isSet(object.amount) ? coin_1.Coin.fromJSON(object.amount) : undefined,
            endpoints: Array.isArray(object === null || object === void 0 ? void 0 : object.endpoints) ? object.endpoints.map((e) => endpoint_1.Endpoint.fromJSON(e)) : [],
            geolocation: isSet(object.geolocation) ? long_1.default.fromValue(object.geolocation) : long_1.default.UZERO,
            moniker: isSet(object.moniker) ? String(object.moniker) : "",
        };
    },
    toJSON(message) {
        const obj = {};
        message.creator !== undefined && (obj.creator = message.creator);
        message.chainID !== undefined && (obj.chainID = message.chainID);
        message.amount !== undefined && (obj.amount = message.amount ? coin_1.Coin.toJSON(message.amount) : undefined);
        if (message.endpoints) {
            obj.endpoints = message.endpoints.map((e) => e ? endpoint_1.Endpoint.toJSON(e) : undefined);
        }
        else {
            obj.endpoints = [];
        }
        message.geolocation !== undefined && (obj.geolocation = (message.geolocation || long_1.default.UZERO).toString());
        message.moniker !== undefined && (obj.moniker = message.moniker);
        return obj;
    },
    create(base) {
        return exports.MsgStakeProvider.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d;
        const message = createBaseMsgStakeProvider();
        message.creator = (_a = object.creator) !== null && _a !== void 0 ? _a : "";
        message.chainID = (_b = object.chainID) !== null && _b !== void 0 ? _b : "";
        message.amount = (object.amount !== undefined && object.amount !== null)
            ? coin_1.Coin.fromPartial(object.amount)
            : undefined;
        message.endpoints = ((_c = object.endpoints) === null || _c === void 0 ? void 0 : _c.map((e) => endpoint_1.Endpoint.fromPartial(e))) || [];
        message.geolocation = (object.geolocation !== undefined && object.geolocation !== null)
            ? long_1.default.fromValue(object.geolocation)
            : long_1.default.UZERO;
        message.moniker = (_d = object.moniker) !== null && _d !== void 0 ? _d : "";
        return message;
    },
};
function createBaseMsgStakeProviderResponse() {
    return {};
}
exports.MsgStakeProviderResponse = {
    encode(_, writer = minimal_1.default.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgStakeProviderResponse();
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
        return exports.MsgStakeProviderResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(_) {
        const message = createBaseMsgStakeProviderResponse();
        return message;
    },
};
function createBaseMsgUnstakeProvider() {
    return { creator: "", chainID: "" };
}
exports.MsgUnstakeProvider = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.creator !== "") {
            writer.uint32(10).string(message.creator);
        }
        if (message.chainID !== "") {
            writer.uint32(18).string(message.chainID);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgUnstakeProvider();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.creator = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
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
        return {
            creator: isSet(object.creator) ? String(object.creator) : "",
            chainID: isSet(object.chainID) ? String(object.chainID) : "",
        };
    },
    toJSON(message) {
        const obj = {};
        message.creator !== undefined && (obj.creator = message.creator);
        message.chainID !== undefined && (obj.chainID = message.chainID);
        return obj;
    },
    create(base) {
        return exports.MsgUnstakeProvider.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseMsgUnstakeProvider();
        message.creator = (_a = object.creator) !== null && _a !== void 0 ? _a : "";
        message.chainID = (_b = object.chainID) !== null && _b !== void 0 ? _b : "";
        return message;
    },
};
function createBaseMsgUnstakeProviderResponse() {
    return {};
}
exports.MsgUnstakeProviderResponse = {
    encode(_, writer = minimal_1.default.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgUnstakeProviderResponse();
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
        return exports.MsgUnstakeProviderResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(_) {
        const message = createBaseMsgUnstakeProviderResponse();
        return message;
    },
};
function createBaseMsgRelayPayment() {
    return { creator: "", relays: [], descriptionString: "" };
}
exports.MsgRelayPayment = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.creator !== "") {
            writer.uint32(10).string(message.creator);
        }
        for (const v of message.relays) {
            relay_1.RelaySession.encode(v, writer.uint32(18).fork()).ldelim();
        }
        if (message.descriptionString !== "") {
            writer.uint32(34).string(message.descriptionString);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgRelayPayment();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.creator = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.relays.push(relay_1.RelaySession.decode(reader, reader.uint32()));
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.descriptionString = reader.string();
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
            creator: isSet(object.creator) ? String(object.creator) : "",
            relays: Array.isArray(object === null || object === void 0 ? void 0 : object.relays) ? object.relays.map((e) => relay_1.RelaySession.fromJSON(e)) : [],
            descriptionString: isSet(object.descriptionString) ? String(object.descriptionString) : "",
        };
    },
    toJSON(message) {
        const obj = {};
        message.creator !== undefined && (obj.creator = message.creator);
        if (message.relays) {
            obj.relays = message.relays.map((e) => e ? relay_1.RelaySession.toJSON(e) : undefined);
        }
        else {
            obj.relays = [];
        }
        message.descriptionString !== undefined && (obj.descriptionString = message.descriptionString);
        return obj;
    },
    create(base) {
        return exports.MsgRelayPayment.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBaseMsgRelayPayment();
        message.creator = (_a = object.creator) !== null && _a !== void 0 ? _a : "";
        message.relays = ((_b = object.relays) === null || _b === void 0 ? void 0 : _b.map((e) => relay_1.RelaySession.fromPartial(e))) || [];
        message.descriptionString = (_c = object.descriptionString) !== null && _c !== void 0 ? _c : "";
        return message;
    },
};
function createBaseMsgRelayPaymentResponse() {
    return {};
}
exports.MsgRelayPaymentResponse = {
    encode(_, writer = minimal_1.default.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgRelayPaymentResponse();
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
        return exports.MsgRelayPaymentResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(_) {
        const message = createBaseMsgRelayPaymentResponse();
        return message;
    },
};
function createBaseMsgFreezeProvider() {
    return { creator: "", chainIds: [], reason: "" };
}
exports.MsgFreezeProvider = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.creator !== "") {
            writer.uint32(10).string(message.creator);
        }
        for (const v of message.chainIds) {
            writer.uint32(18).string(v);
        }
        if (message.reason !== "") {
            writer.uint32(26).string(message.reason);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgFreezeProvider();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.creator = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.chainIds.push(reader.string());
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.reason = reader.string();
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
            creator: isSet(object.creator) ? String(object.creator) : "",
            chainIds: Array.isArray(object === null || object === void 0 ? void 0 : object.chainIds) ? object.chainIds.map((e) => String(e)) : [],
            reason: isSet(object.reason) ? String(object.reason) : "",
        };
    },
    toJSON(message) {
        const obj = {};
        message.creator !== undefined && (obj.creator = message.creator);
        if (message.chainIds) {
            obj.chainIds = message.chainIds.map((e) => e);
        }
        else {
            obj.chainIds = [];
        }
        message.reason !== undefined && (obj.reason = message.reason);
        return obj;
    },
    create(base) {
        return exports.MsgFreezeProvider.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBaseMsgFreezeProvider();
        message.creator = (_a = object.creator) !== null && _a !== void 0 ? _a : "";
        message.chainIds = ((_b = object.chainIds) === null || _b === void 0 ? void 0 : _b.map((e) => e)) || [];
        message.reason = (_c = object.reason) !== null && _c !== void 0 ? _c : "";
        return message;
    },
};
function createBaseMsgFreezeProviderResponse() {
    return {};
}
exports.MsgFreezeProviderResponse = {
    encode(_, writer = minimal_1.default.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgFreezeProviderResponse();
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
        return exports.MsgFreezeProviderResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(_) {
        const message = createBaseMsgFreezeProviderResponse();
        return message;
    },
};
function createBaseMsgUnfreezeProvider() {
    return { creator: "", chainIds: [] };
}
exports.MsgUnfreezeProvider = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.creator !== "") {
            writer.uint32(10).string(message.creator);
        }
        for (const v of message.chainIds) {
            writer.uint32(18).string(v);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgUnfreezeProvider();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.creator = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.chainIds.push(reader.string());
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
            creator: isSet(object.creator) ? String(object.creator) : "",
            chainIds: Array.isArray(object === null || object === void 0 ? void 0 : object.chainIds) ? object.chainIds.map((e) => String(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.creator !== undefined && (obj.creator = message.creator);
        if (message.chainIds) {
            obj.chainIds = message.chainIds.map((e) => e);
        }
        else {
            obj.chainIds = [];
        }
        return obj;
    },
    create(base) {
        return exports.MsgUnfreezeProvider.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseMsgUnfreezeProvider();
        message.creator = (_a = object.creator) !== null && _a !== void 0 ? _a : "";
        message.chainIds = ((_b = object.chainIds) === null || _b === void 0 ? void 0 : _b.map((e) => e)) || [];
        return message;
    },
};
function createBaseMsgUnfreezeProviderResponse() {
    return {};
}
exports.MsgUnfreezeProviderResponse = {
    encode(_, writer = minimal_1.default.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgUnfreezeProviderResponse();
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
        return exports.MsgUnfreezeProviderResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(_) {
        const message = createBaseMsgUnfreezeProviderResponse();
        return message;
    },
};
class MsgClientImpl {
    constructor(rpc, opts) {
        this.service = (opts === null || opts === void 0 ? void 0 : opts.service) || "lavanet.lava.pairing.Msg";
        this.rpc = rpc;
        this.StakeProvider = this.StakeProvider.bind(this);
        this.UnstakeProvider = this.UnstakeProvider.bind(this);
        this.RelayPayment = this.RelayPayment.bind(this);
        this.FreezeProvider = this.FreezeProvider.bind(this);
        this.UnfreezeProvider = this.UnfreezeProvider.bind(this);
    }
    StakeProvider(request) {
        const data = exports.MsgStakeProvider.encode(request).finish();
        const promise = this.rpc.request(this.service, "StakeProvider", data);
        return promise.then((data) => exports.MsgStakeProviderResponse.decode(minimal_1.default.Reader.create(data)));
    }
    UnstakeProvider(request) {
        const data = exports.MsgUnstakeProvider.encode(request).finish();
        const promise = this.rpc.request(this.service, "UnstakeProvider", data);
        return promise.then((data) => exports.MsgUnstakeProviderResponse.decode(minimal_1.default.Reader.create(data)));
    }
    RelayPayment(request) {
        const data = exports.MsgRelayPayment.encode(request).finish();
        const promise = this.rpc.request(this.service, "RelayPayment", data);
        return promise.then((data) => exports.MsgRelayPaymentResponse.decode(minimal_1.default.Reader.create(data)));
    }
    FreezeProvider(request) {
        const data = exports.MsgFreezeProvider.encode(request).finish();
        const promise = this.rpc.request(this.service, "FreezeProvider", data);
        return promise.then((data) => exports.MsgFreezeProviderResponse.decode(minimal_1.default.Reader.create(data)));
    }
    UnfreezeProvider(request) {
        const data = exports.MsgUnfreezeProvider.encode(request).finish();
        const promise = this.rpc.request(this.service, "UnfreezeProvider", data);
        return promise.then((data) => exports.MsgUnfreezeProviderResponse.decode(minimal_1.default.Reader.create(data)));
    }
}
exports.MsgClientImpl = MsgClientImpl;
if (minimal_1.default.util.Long !== long_1.default) {
    minimal_1.default.util.Long = long_1.default;
    minimal_1.default.configure();
}
function isSet(value) {
    return value !== null && value !== undefined;
}

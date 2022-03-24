/* eslint-disable */
import { Reader, util, configure, Writer } from "protobufjs/minimal";
import * as Long from "long";
export const protobufPackage = "lavanet.lava.servicer";
const baseRelayRequest = {
    specId: 0,
    apiId: 0,
    sessionId: 0,
    cuSum: 0,
    servicer: "",
    blockHeight: 0,
};
export const RelayRequest = {
    encode(message, writer = Writer.create()) {
        if (message.specId !== 0) {
            writer.uint32(8).uint32(message.specId);
        }
        if (message.apiId !== 0) {
            writer.uint32(16).uint32(message.apiId);
        }
        if (message.sessionId !== 0) {
            writer.uint32(24).uint64(message.sessionId);
        }
        if (message.cuSum !== 0) {
            writer.uint32(32).uint64(message.cuSum);
        }
        if (message.data.length !== 0) {
            writer.uint32(42).bytes(message.data);
        }
        if (message.sig.length !== 0) {
            writer.uint32(50).bytes(message.sig);
        }
        if (message.servicer !== "") {
            writer.uint32(58).string(message.servicer);
        }
        if (message.blockHeight !== 0) {
            writer.uint32(64).int64(message.blockHeight);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseRelayRequest };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.specId = reader.uint32();
                    break;
                case 2:
                    message.apiId = reader.uint32();
                    break;
                case 3:
                    message.sessionId = longToNumber(reader.uint64());
                    break;
                case 4:
                    message.cuSum = longToNumber(reader.uint64());
                    break;
                case 5:
                    message.data = reader.bytes();
                    break;
                case 6:
                    message.sig = reader.bytes();
                    break;
                case 7:
                    message.servicer = reader.string();
                    break;
                case 8:
                    message.blockHeight = longToNumber(reader.int64());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = { ...baseRelayRequest };
        if (object.specId !== undefined && object.specId !== null) {
            message.specId = Number(object.specId);
        }
        else {
            message.specId = 0;
        }
        if (object.apiId !== undefined && object.apiId !== null) {
            message.apiId = Number(object.apiId);
        }
        else {
            message.apiId = 0;
        }
        if (object.sessionId !== undefined && object.sessionId !== null) {
            message.sessionId = Number(object.sessionId);
        }
        else {
            message.sessionId = 0;
        }
        if (object.cuSum !== undefined && object.cuSum !== null) {
            message.cuSum = Number(object.cuSum);
        }
        else {
            message.cuSum = 0;
        }
        if (object.data !== undefined && object.data !== null) {
            message.data = bytesFromBase64(object.data);
        }
        if (object.sig !== undefined && object.sig !== null) {
            message.sig = bytesFromBase64(object.sig);
        }
        if (object.servicer !== undefined && object.servicer !== null) {
            message.servicer = String(object.servicer);
        }
        else {
            message.servicer = "";
        }
        if (object.blockHeight !== undefined && object.blockHeight !== null) {
            message.blockHeight = Number(object.blockHeight);
        }
        else {
            message.blockHeight = 0;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.specId !== undefined && (obj.specId = message.specId);
        message.apiId !== undefined && (obj.apiId = message.apiId);
        message.sessionId !== undefined && (obj.sessionId = message.sessionId);
        message.cuSum !== undefined && (obj.cuSum = message.cuSum);
        message.data !== undefined &&
            (obj.data = base64FromBytes(message.data !== undefined ? message.data : new Uint8Array()));
        message.sig !== undefined &&
            (obj.sig = base64FromBytes(message.sig !== undefined ? message.sig : new Uint8Array()));
        message.servicer !== undefined && (obj.servicer = message.servicer);
        message.blockHeight !== undefined &&
            (obj.blockHeight = message.blockHeight);
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseRelayRequest };
        if (object.specId !== undefined && object.specId !== null) {
            message.specId = object.specId;
        }
        else {
            message.specId = 0;
        }
        if (object.apiId !== undefined && object.apiId !== null) {
            message.apiId = object.apiId;
        }
        else {
            message.apiId = 0;
        }
        if (object.sessionId !== undefined && object.sessionId !== null) {
            message.sessionId = object.sessionId;
        }
        else {
            message.sessionId = 0;
        }
        if (object.cuSum !== undefined && object.cuSum !== null) {
            message.cuSum = object.cuSum;
        }
        else {
            message.cuSum = 0;
        }
        if (object.data !== undefined && object.data !== null) {
            message.data = object.data;
        }
        else {
            message.data = new Uint8Array();
        }
        if (object.sig !== undefined && object.sig !== null) {
            message.sig = object.sig;
        }
        else {
            message.sig = new Uint8Array();
        }
        if (object.servicer !== undefined && object.servicer !== null) {
            message.servicer = object.servicer;
        }
        else {
            message.servicer = "";
        }
        if (object.blockHeight !== undefined && object.blockHeight !== null) {
            message.blockHeight = object.blockHeight;
        }
        else {
            message.blockHeight = 0;
        }
        return message;
    },
};
const baseRelayReply = {};
export const RelayReply = {
    encode(message, writer = Writer.create()) {
        if (message.data.length !== 0) {
            writer.uint32(10).bytes(message.data);
        }
        if (message.sig.length !== 0) {
            writer.uint32(18).bytes(message.sig);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseRelayReply };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.data = reader.bytes();
                    break;
                case 2:
                    message.sig = reader.bytes();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = { ...baseRelayReply };
        if (object.data !== undefined && object.data !== null) {
            message.data = bytesFromBase64(object.data);
        }
        if (object.sig !== undefined && object.sig !== null) {
            message.sig = bytesFromBase64(object.sig);
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.data !== undefined &&
            (obj.data = base64FromBytes(message.data !== undefined ? message.data : new Uint8Array()));
        message.sig !== undefined &&
            (obj.sig = base64FromBytes(message.sig !== undefined ? message.sig : new Uint8Array()));
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseRelayReply };
        if (object.data !== undefined && object.data !== null) {
            message.data = object.data;
        }
        else {
            message.data = new Uint8Array();
        }
        if (object.sig !== undefined && object.sig !== null) {
            message.sig = object.sig;
        }
        else {
            message.sig = new Uint8Array();
        }
        return message;
    },
};
export class RelayerClientImpl {
    constructor(rpc) {
        this.rpc = rpc;
    }
    Relay(request) {
        const data = RelayRequest.encode(request).finish();
        const promise = this.rpc.request("lavanet.lava.servicer.Relayer", "Relay", data);
        return promise.then((data) => RelayReply.decode(new Reader(data)));
    }
}
var globalThis = (() => {
    if (typeof globalThis !== "undefined")
        return globalThis;
    if (typeof self !== "undefined")
        return self;
    if (typeof window !== "undefined")
        return window;
    if (typeof global !== "undefined")
        return global;
    throw "Unable to locate global object";
})();
const atob = globalThis.atob ||
    ((b64) => globalThis.Buffer.from(b64, "base64").toString("binary"));
function bytesFromBase64(b64) {
    const bin = atob(b64);
    const arr = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; ++i) {
        arr[i] = bin.charCodeAt(i);
    }
    return arr;
}
const btoa = globalThis.btoa ||
    ((bin) => globalThis.Buffer.from(bin, "binary").toString("base64"));
function base64FromBytes(arr) {
    const bin = [];
    for (let i = 0; i < arr.byteLength; ++i) {
        bin.push(String.fromCharCode(arr[i]));
    }
    return btoa(bin.join(""));
}
function longToNumber(long) {
    if (long.gt(Number.MAX_SAFE_INTEGER)) {
        throw new globalThis.Error("Value is larger than Number.MAX_SAFE_INTEGER");
    }
    return long.toNumber();
}
if (util.Long !== Long) {
    util.Long = Long;
    configure();
}

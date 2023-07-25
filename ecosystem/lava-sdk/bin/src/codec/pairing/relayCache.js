"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.RelayerCacheClientImpl = exports.RelayCacheSet = exports.RelayCacheGet = exports.CacheUsage = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const empty_1 = require("../google/protobuf/empty");
const relay_1 = require("./relay");
exports.protobufPackage = "lavanet.lava.pairing";
function createBaseCacheUsage() {
    return { CacheHits: long_1.default.UZERO, CacheMisses: long_1.default.UZERO };
}
exports.CacheUsage = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (!message.CacheHits.isZero()) {
            writer.uint32(8).uint64(message.CacheHits);
        }
        if (!message.CacheMisses.isZero()) {
            writer.uint32(16).uint64(message.CacheMisses);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseCacheUsage();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.CacheHits = reader.uint64();
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.CacheMisses = reader.uint64();
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
            CacheHits: isSet(object.CacheHits) ? long_1.default.fromValue(object.CacheHits) : long_1.default.UZERO,
            CacheMisses: isSet(object.CacheMisses) ? long_1.default.fromValue(object.CacheMisses) : long_1.default.UZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.CacheHits !== undefined && (obj.CacheHits = (message.CacheHits || long_1.default.UZERO).toString());
        message.CacheMisses !== undefined && (obj.CacheMisses = (message.CacheMisses || long_1.default.UZERO).toString());
        return obj;
    },
    create(base) {
        return exports.CacheUsage.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseCacheUsage();
        message.CacheHits = (object.CacheHits !== undefined && object.CacheHits !== null)
            ? long_1.default.fromValue(object.CacheHits)
            : long_1.default.UZERO;
        message.CacheMisses = (object.CacheMisses !== undefined && object.CacheMisses !== null)
            ? long_1.default.fromValue(object.CacheMisses)
            : long_1.default.UZERO;
        return message;
    },
};
function createBaseRelayCacheGet() {
    return { request: undefined, apiInterface: "", blockHash: new Uint8Array(), chainID: "", finalized: false };
}
exports.RelayCacheGet = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.request !== undefined) {
            relay_1.RelayRequest.encode(message.request, writer.uint32(10).fork()).ldelim();
        }
        if (message.apiInterface !== "") {
            writer.uint32(18).string(message.apiInterface);
        }
        if (message.blockHash.length !== 0) {
            writer.uint32(26).bytes(message.blockHash);
        }
        if (message.chainID !== "") {
            writer.uint32(34).string(message.chainID);
        }
        if (message.finalized === true) {
            writer.uint32(40).bool(message.finalized);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseRelayCacheGet();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.request = relay_1.RelayRequest.decode(reader, reader.uint32());
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.apiInterface = reader.string();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.blockHash = reader.bytes();
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.chainID = reader.string();
                    continue;
                case 5:
                    if (tag != 40) {
                        break;
                    }
                    message.finalized = reader.bool();
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
            request: isSet(object.request) ? relay_1.RelayRequest.fromJSON(object.request) : undefined,
            apiInterface: isSet(object.apiInterface) ? String(object.apiInterface) : "",
            blockHash: isSet(object.blockHash) ? bytesFromBase64(object.blockHash) : new Uint8Array(),
            chainID: isSet(object.chainID) ? String(object.chainID) : "",
            finalized: isSet(object.finalized) ? Boolean(object.finalized) : false,
        };
    },
    toJSON(message) {
        const obj = {};
        message.request !== undefined && (obj.request = message.request ? relay_1.RelayRequest.toJSON(message.request) : undefined);
        message.apiInterface !== undefined && (obj.apiInterface = message.apiInterface);
        message.blockHash !== undefined &&
            (obj.blockHash = base64FromBytes(message.blockHash !== undefined ? message.blockHash : new Uint8Array()));
        message.chainID !== undefined && (obj.chainID = message.chainID);
        message.finalized !== undefined && (obj.finalized = message.finalized);
        return obj;
    },
    create(base) {
        return exports.RelayCacheGet.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d;
        const message = createBaseRelayCacheGet();
        message.request = (object.request !== undefined && object.request !== null)
            ? relay_1.RelayRequest.fromPartial(object.request)
            : undefined;
        message.apiInterface = (_a = object.apiInterface) !== null && _a !== void 0 ? _a : "";
        message.blockHash = (_b = object.blockHash) !== null && _b !== void 0 ? _b : new Uint8Array();
        message.chainID = (_c = object.chainID) !== null && _c !== void 0 ? _c : "";
        message.finalized = (_d = object.finalized) !== null && _d !== void 0 ? _d : false;
        return message;
    },
};
function createBaseRelayCacheSet() {
    return {
        request: undefined,
        apiInterface: "",
        blockHash: new Uint8Array(),
        chainID: "",
        bucketID: "",
        response: undefined,
        finalized: false,
    };
}
exports.RelayCacheSet = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.request !== undefined) {
            relay_1.RelayRequest.encode(message.request, writer.uint32(10).fork()).ldelim();
        }
        if (message.apiInterface !== "") {
            writer.uint32(18).string(message.apiInterface);
        }
        if (message.blockHash.length !== 0) {
            writer.uint32(26).bytes(message.blockHash);
        }
        if (message.chainID !== "") {
            writer.uint32(34).string(message.chainID);
        }
        if (message.bucketID !== "") {
            writer.uint32(42).string(message.bucketID);
        }
        if (message.response !== undefined) {
            relay_1.RelayReply.encode(message.response, writer.uint32(50).fork()).ldelim();
        }
        if (message.finalized === true) {
            writer.uint32(56).bool(message.finalized);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseRelayCacheSet();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.request = relay_1.RelayRequest.decode(reader, reader.uint32());
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.apiInterface = reader.string();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.blockHash = reader.bytes();
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.chainID = reader.string();
                    continue;
                case 5:
                    if (tag != 42) {
                        break;
                    }
                    message.bucketID = reader.string();
                    continue;
                case 6:
                    if (tag != 50) {
                        break;
                    }
                    message.response = relay_1.RelayReply.decode(reader, reader.uint32());
                    continue;
                case 7:
                    if (tag != 56) {
                        break;
                    }
                    message.finalized = reader.bool();
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
            request: isSet(object.request) ? relay_1.RelayRequest.fromJSON(object.request) : undefined,
            apiInterface: isSet(object.apiInterface) ? String(object.apiInterface) : "",
            blockHash: isSet(object.blockHash) ? bytesFromBase64(object.blockHash) : new Uint8Array(),
            chainID: isSet(object.chainID) ? String(object.chainID) : "",
            bucketID: isSet(object.bucketID) ? String(object.bucketID) : "",
            response: isSet(object.response) ? relay_1.RelayReply.fromJSON(object.response) : undefined,
            finalized: isSet(object.finalized) ? Boolean(object.finalized) : false,
        };
    },
    toJSON(message) {
        const obj = {};
        message.request !== undefined && (obj.request = message.request ? relay_1.RelayRequest.toJSON(message.request) : undefined);
        message.apiInterface !== undefined && (obj.apiInterface = message.apiInterface);
        message.blockHash !== undefined &&
            (obj.blockHash = base64FromBytes(message.blockHash !== undefined ? message.blockHash : new Uint8Array()));
        message.chainID !== undefined && (obj.chainID = message.chainID);
        message.bucketID !== undefined && (obj.bucketID = message.bucketID);
        message.response !== undefined &&
            (obj.response = message.response ? relay_1.RelayReply.toJSON(message.response) : undefined);
        message.finalized !== undefined && (obj.finalized = message.finalized);
        return obj;
    },
    create(base) {
        return exports.RelayCacheSet.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d, _e;
        const message = createBaseRelayCacheSet();
        message.request = (object.request !== undefined && object.request !== null)
            ? relay_1.RelayRequest.fromPartial(object.request)
            : undefined;
        message.apiInterface = (_a = object.apiInterface) !== null && _a !== void 0 ? _a : "";
        message.blockHash = (_b = object.blockHash) !== null && _b !== void 0 ? _b : new Uint8Array();
        message.chainID = (_c = object.chainID) !== null && _c !== void 0 ? _c : "";
        message.bucketID = (_d = object.bucketID) !== null && _d !== void 0 ? _d : "";
        message.response = (object.response !== undefined && object.response !== null)
            ? relay_1.RelayReply.fromPartial(object.response)
            : undefined;
        message.finalized = (_e = object.finalized) !== null && _e !== void 0 ? _e : false;
        return message;
    },
};
class RelayerCacheClientImpl {
    constructor(rpc, opts) {
        this.service = (opts === null || opts === void 0 ? void 0 : opts.service) || "lavanet.lava.pairing.RelayerCache";
        this.rpc = rpc;
        this.GetRelay = this.GetRelay.bind(this);
        this.SetRelay = this.SetRelay.bind(this);
        this.Health = this.Health.bind(this);
    }
    GetRelay(request) {
        const data = exports.RelayCacheGet.encode(request).finish();
        const promise = this.rpc.request(this.service, "GetRelay", data);
        return promise.then((data) => relay_1.RelayReply.decode(minimal_1.default.Reader.create(data)));
    }
    SetRelay(request) {
        const data = exports.RelayCacheSet.encode(request).finish();
        const promise = this.rpc.request(this.service, "SetRelay", data);
        return promise.then((data) => empty_1.Empty.decode(minimal_1.default.Reader.create(data)));
    }
    Health(request) {
        const data = empty_1.Empty.encode(request).finish();
        const promise = this.rpc.request(this.service, "Health", data);
        return promise.then((data) => exports.CacheUsage.decode(minimal_1.default.Reader.create(data)));
    }
}
exports.RelayerCacheClientImpl = RelayerCacheClientImpl;
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

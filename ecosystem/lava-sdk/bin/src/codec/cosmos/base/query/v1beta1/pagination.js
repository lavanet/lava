"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.PageResponse = exports.PageRequest = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
exports.protobufPackage = "cosmos.base.query.v1beta1";
function createBasePageRequest() {
    return { key: new Uint8Array(), offset: long_1.default.UZERO, limit: long_1.default.UZERO, countTotal: false, reverse: false };
}
exports.PageRequest = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.key.length !== 0) {
            writer.uint32(10).bytes(message.key);
        }
        if (!message.offset.isZero()) {
            writer.uint32(16).uint64(message.offset);
        }
        if (!message.limit.isZero()) {
            writer.uint32(24).uint64(message.limit);
        }
        if (message.countTotal === true) {
            writer.uint32(32).bool(message.countTotal);
        }
        if (message.reverse === true) {
            writer.uint32(40).bool(message.reverse);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBasePageRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.key = reader.bytes();
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.offset = reader.uint64();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.limit = reader.uint64();
                    continue;
                case 4:
                    if (tag != 32) {
                        break;
                    }
                    message.countTotal = reader.bool();
                    continue;
                case 5:
                    if (tag != 40) {
                        break;
                    }
                    message.reverse = reader.bool();
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
            key: isSet(object.key) ? bytesFromBase64(object.key) : new Uint8Array(),
            offset: isSet(object.offset) ? long_1.default.fromValue(object.offset) : long_1.default.UZERO,
            limit: isSet(object.limit) ? long_1.default.fromValue(object.limit) : long_1.default.UZERO,
            countTotal: isSet(object.countTotal) ? Boolean(object.countTotal) : false,
            reverse: isSet(object.reverse) ? Boolean(object.reverse) : false,
        };
    },
    toJSON(message) {
        const obj = {};
        message.key !== undefined &&
            (obj.key = base64FromBytes(message.key !== undefined ? message.key : new Uint8Array()));
        message.offset !== undefined && (obj.offset = (message.offset || long_1.default.UZERO).toString());
        message.limit !== undefined && (obj.limit = (message.limit || long_1.default.UZERO).toString());
        message.countTotal !== undefined && (obj.countTotal = message.countTotal);
        message.reverse !== undefined && (obj.reverse = message.reverse);
        return obj;
    },
    create(base) {
        return exports.PageRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBasePageRequest();
        message.key = (_a = object.key) !== null && _a !== void 0 ? _a : new Uint8Array();
        message.offset = (object.offset !== undefined && object.offset !== null)
            ? long_1.default.fromValue(object.offset)
            : long_1.default.UZERO;
        message.limit = (object.limit !== undefined && object.limit !== null) ? long_1.default.fromValue(object.limit) : long_1.default.UZERO;
        message.countTotal = (_b = object.countTotal) !== null && _b !== void 0 ? _b : false;
        message.reverse = (_c = object.reverse) !== null && _c !== void 0 ? _c : false;
        return message;
    },
};
function createBasePageResponse() {
    return { nextKey: new Uint8Array(), total: long_1.default.UZERO };
}
exports.PageResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.nextKey.length !== 0) {
            writer.uint32(10).bytes(message.nextKey);
        }
        if (!message.total.isZero()) {
            writer.uint32(16).uint64(message.total);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBasePageResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.nextKey = reader.bytes();
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.total = reader.uint64();
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
            nextKey: isSet(object.nextKey) ? bytesFromBase64(object.nextKey) : new Uint8Array(),
            total: isSet(object.total) ? long_1.default.fromValue(object.total) : long_1.default.UZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.nextKey !== undefined &&
            (obj.nextKey = base64FromBytes(message.nextKey !== undefined ? message.nextKey : new Uint8Array()));
        message.total !== undefined && (obj.total = (message.total || long_1.default.UZERO).toString());
        return obj;
    },
    create(base) {
        return exports.PageResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBasePageResponse();
        message.nextKey = (_a = object.nextKey) !== null && _a !== void 0 ? _a : new Uint8Array();
        message.total = (object.total !== undefined && object.total !== null) ? long_1.default.fromValue(object.total) : long_1.default.UZERO;
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

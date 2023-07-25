"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.RawMessage = exports.Entry = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
exports.protobufPackage = "lavanet.lava.common";
function createBaseEntry() {
    return {
        index: "",
        block: long_1.default.UZERO,
        staleAt: long_1.default.UZERO,
        refcount: long_1.default.UZERO,
        data: new Uint8Array(),
        deleteAt: long_1.default.UZERO,
    };
}
exports.Entry = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        if (!message.block.isZero()) {
            writer.uint32(16).uint64(message.block);
        }
        if (!message.staleAt.isZero()) {
            writer.uint32(24).uint64(message.staleAt);
        }
        if (!message.refcount.isZero()) {
            writer.uint32(32).uint64(message.refcount);
        }
        if (message.data.length !== 0) {
            writer.uint32(42).bytes(message.data);
        }
        if (!message.deleteAt.isZero()) {
            writer.uint32(48).uint64(message.deleteAt);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseEntry();
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
                    if (tag != 16) {
                        break;
                    }
                    message.block = reader.uint64();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.staleAt = reader.uint64();
                    continue;
                case 4:
                    if (tag != 32) {
                        break;
                    }
                    message.refcount = reader.uint64();
                    continue;
                case 5:
                    if (tag != 42) {
                        break;
                    }
                    message.data = reader.bytes();
                    continue;
                case 6:
                    if (tag != 48) {
                        break;
                    }
                    message.deleteAt = reader.uint64();
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
            staleAt: isSet(object.staleAt) ? long_1.default.fromValue(object.staleAt) : long_1.default.UZERO,
            refcount: isSet(object.refcount) ? long_1.default.fromValue(object.refcount) : long_1.default.UZERO,
            data: isSet(object.data) ? bytesFromBase64(object.data) : new Uint8Array(),
            deleteAt: isSet(object.deleteAt) ? long_1.default.fromValue(object.deleteAt) : long_1.default.UZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = message.index);
        message.block !== undefined && (obj.block = (message.block || long_1.default.UZERO).toString());
        message.staleAt !== undefined && (obj.staleAt = (message.staleAt || long_1.default.UZERO).toString());
        message.refcount !== undefined && (obj.refcount = (message.refcount || long_1.default.UZERO).toString());
        message.data !== undefined &&
            (obj.data = base64FromBytes(message.data !== undefined ? message.data : new Uint8Array()));
        message.deleteAt !== undefined && (obj.deleteAt = (message.deleteAt || long_1.default.UZERO).toString());
        return obj;
    },
    create(base) {
        return exports.Entry.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseEntry();
        message.index = (_a = object.index) !== null && _a !== void 0 ? _a : "";
        message.block = (object.block !== undefined && object.block !== null) ? long_1.default.fromValue(object.block) : long_1.default.UZERO;
        message.staleAt = (object.staleAt !== undefined && object.staleAt !== null)
            ? long_1.default.fromValue(object.staleAt)
            : long_1.default.UZERO;
        message.refcount = (object.refcount !== undefined && object.refcount !== null)
            ? long_1.default.fromValue(object.refcount)
            : long_1.default.UZERO;
        message.data = (_b = object.data) !== null && _b !== void 0 ? _b : new Uint8Array();
        message.deleteAt = (object.deleteAt !== undefined && object.deleteAt !== null)
            ? long_1.default.fromValue(object.deleteAt)
            : long_1.default.UZERO;
        return message;
    },
};
function createBaseRawMessage() {
    return { key: new Uint8Array(), value: new Uint8Array() };
}
exports.RawMessage = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.key.length !== 0) {
            writer.uint32(10).bytes(message.key);
        }
        if (message.value.length !== 0) {
            writer.uint32(18).bytes(message.value);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseRawMessage();
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
                    if (tag != 18) {
                        break;
                    }
                    message.value = reader.bytes();
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
            value: isSet(object.value) ? bytesFromBase64(object.value) : new Uint8Array(),
        };
    },
    toJSON(message) {
        const obj = {};
        message.key !== undefined &&
            (obj.key = base64FromBytes(message.key !== undefined ? message.key : new Uint8Array()));
        message.value !== undefined &&
            (obj.value = base64FromBytes(message.value !== undefined ? message.value : new Uint8Array()));
        return obj;
    },
    create(base) {
        return exports.RawMessage.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseRawMessage();
        message.key = (_a = object.key) !== null && _a !== void 0 ? _a : new Uint8Array();
        message.value = (_b = object.value) !== null && _b !== void 0 ? _b : new Uint8Array();
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

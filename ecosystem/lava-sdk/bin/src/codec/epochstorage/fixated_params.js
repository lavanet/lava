"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.FixatedParams = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
exports.protobufPackage = "lavanet.lava.epochstorage";
function createBaseFixatedParams() {
    return { index: "", parameter: new Uint8Array(), fixationBlock: long_1.default.UZERO };
}
exports.FixatedParams = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        if (message.parameter.length !== 0) {
            writer.uint32(18).bytes(message.parameter);
        }
        if (!message.fixationBlock.isZero()) {
            writer.uint32(24).uint64(message.fixationBlock);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseFixatedParams();
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
                    message.parameter = reader.bytes();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.fixationBlock = reader.uint64();
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
            parameter: isSet(object.parameter) ? bytesFromBase64(object.parameter) : new Uint8Array(),
            fixationBlock: isSet(object.fixationBlock) ? long_1.default.fromValue(object.fixationBlock) : long_1.default.UZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = message.index);
        message.parameter !== undefined &&
            (obj.parameter = base64FromBytes(message.parameter !== undefined ? message.parameter : new Uint8Array()));
        message.fixationBlock !== undefined && (obj.fixationBlock = (message.fixationBlock || long_1.default.UZERO).toString());
        return obj;
    },
    create(base) {
        return exports.FixatedParams.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseFixatedParams();
        message.index = (_a = object.index) !== null && _a !== void 0 ? _a : "";
        message.parameter = (_b = object.parameter) !== null && _b !== void 0 ? _b : new Uint8Array();
        message.fixationBlock = (object.fixationBlock !== undefined && object.fixationBlock !== null)
            ? long_1.default.fromValue(object.fixationBlock)
            : long_1.default.UZERO;
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

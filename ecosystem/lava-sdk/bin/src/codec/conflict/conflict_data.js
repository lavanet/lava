"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.FinalizationConflict = exports.ReplyMetadata = exports.ConflictRelayData = exports.ResponseConflict = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const relay_1 = require("../pairing/relay");
exports.protobufPackage = "lavanet.lava.conflict";
function createBaseResponseConflict() {
    return { conflictRelayData0: undefined, conflictRelayData1: undefined };
}
exports.ResponseConflict = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.conflictRelayData0 !== undefined) {
            exports.ConflictRelayData.encode(message.conflictRelayData0, writer.uint32(10).fork()).ldelim();
        }
        if (message.conflictRelayData1 !== undefined) {
            exports.ConflictRelayData.encode(message.conflictRelayData1, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseResponseConflict();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.conflictRelayData0 = exports.ConflictRelayData.decode(reader, reader.uint32());
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.conflictRelayData1 = exports.ConflictRelayData.decode(reader, reader.uint32());
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
            conflictRelayData0: isSet(object.conflictRelayData0)
                ? exports.ConflictRelayData.fromJSON(object.conflictRelayData0)
                : undefined,
            conflictRelayData1: isSet(object.conflictRelayData1)
                ? exports.ConflictRelayData.fromJSON(object.conflictRelayData1)
                : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.conflictRelayData0 !== undefined && (obj.conflictRelayData0 = message.conflictRelayData0
            ? exports.ConflictRelayData.toJSON(message.conflictRelayData0)
            : undefined);
        message.conflictRelayData1 !== undefined && (obj.conflictRelayData1 = message.conflictRelayData1
            ? exports.ConflictRelayData.toJSON(message.conflictRelayData1)
            : undefined);
        return obj;
    },
    create(base) {
        return exports.ResponseConflict.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseResponseConflict();
        message.conflictRelayData0 = (object.conflictRelayData0 !== undefined && object.conflictRelayData0 !== null)
            ? exports.ConflictRelayData.fromPartial(object.conflictRelayData0)
            : undefined;
        message.conflictRelayData1 = (object.conflictRelayData1 !== undefined && object.conflictRelayData1 !== null)
            ? exports.ConflictRelayData.fromPartial(object.conflictRelayData1)
            : undefined;
        return message;
    },
};
function createBaseConflictRelayData() {
    return { request: undefined, reply: undefined };
}
exports.ConflictRelayData = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.request !== undefined) {
            relay_1.RelayRequest.encode(message.request, writer.uint32(10).fork()).ldelim();
        }
        if (message.reply !== undefined) {
            exports.ReplyMetadata.encode(message.reply, writer.uint32(26).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseConflictRelayData();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.request = relay_1.RelayRequest.decode(reader, reader.uint32());
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.reply = exports.ReplyMetadata.decode(reader, reader.uint32());
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
            reply: isSet(object.reply) ? exports.ReplyMetadata.fromJSON(object.reply) : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.request !== undefined && (obj.request = message.request ? relay_1.RelayRequest.toJSON(message.request) : undefined);
        message.reply !== undefined && (obj.reply = message.reply ? exports.ReplyMetadata.toJSON(message.reply) : undefined);
        return obj;
    },
    create(base) {
        return exports.ConflictRelayData.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseConflictRelayData();
        message.request = (object.request !== undefined && object.request !== null)
            ? relay_1.RelayRequest.fromPartial(object.request)
            : undefined;
        message.reply = (object.reply !== undefined && object.reply !== null)
            ? exports.ReplyMetadata.fromPartial(object.reply)
            : undefined;
        return message;
    },
};
function createBaseReplyMetadata() {
    return {
        hashAllDataHash: new Uint8Array(),
        sig: new Uint8Array(),
        latestBlock: long_1.default.ZERO,
        finalizedBlocksHashes: new Uint8Array(),
        sigBlocks: new Uint8Array(),
    };
}
exports.ReplyMetadata = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.hashAllDataHash.length !== 0) {
            writer.uint32(10).bytes(message.hashAllDataHash);
        }
        if (message.sig.length !== 0) {
            writer.uint32(18).bytes(message.sig);
        }
        if (!message.latestBlock.isZero()) {
            writer.uint32(24).int64(message.latestBlock);
        }
        if (message.finalizedBlocksHashes.length !== 0) {
            writer.uint32(34).bytes(message.finalizedBlocksHashes);
        }
        if (message.sigBlocks.length !== 0) {
            writer.uint32(42).bytes(message.sigBlocks);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseReplyMetadata();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.hashAllDataHash = reader.bytes();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.sig = reader.bytes();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.latestBlock = reader.int64();
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.finalizedBlocksHashes = reader.bytes();
                    continue;
                case 5:
                    if (tag != 42) {
                        break;
                    }
                    message.sigBlocks = reader.bytes();
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
            hashAllDataHash: isSet(object.hashAllDataHash) ? bytesFromBase64(object.hashAllDataHash) : new Uint8Array(),
            sig: isSet(object.sig) ? bytesFromBase64(object.sig) : new Uint8Array(),
            latestBlock: isSet(object.latestBlock) ? long_1.default.fromValue(object.latestBlock) : long_1.default.ZERO,
            finalizedBlocksHashes: isSet(object.finalizedBlocksHashes)
                ? bytesFromBase64(object.finalizedBlocksHashes)
                : new Uint8Array(),
            sigBlocks: isSet(object.sigBlocks) ? bytesFromBase64(object.sigBlocks) : new Uint8Array(),
        };
    },
    toJSON(message) {
        const obj = {};
        message.hashAllDataHash !== undefined &&
            (obj.hashAllDataHash = base64FromBytes(message.hashAllDataHash !== undefined ? message.hashAllDataHash : new Uint8Array()));
        message.sig !== undefined &&
            (obj.sig = base64FromBytes(message.sig !== undefined ? message.sig : new Uint8Array()));
        message.latestBlock !== undefined && (obj.latestBlock = (message.latestBlock || long_1.default.ZERO).toString());
        message.finalizedBlocksHashes !== undefined &&
            (obj.finalizedBlocksHashes = base64FromBytes(message.finalizedBlocksHashes !== undefined ? message.finalizedBlocksHashes : new Uint8Array()));
        message.sigBlocks !== undefined &&
            (obj.sigBlocks = base64FromBytes(message.sigBlocks !== undefined ? message.sigBlocks : new Uint8Array()));
        return obj;
    },
    create(base) {
        return exports.ReplyMetadata.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d;
        const message = createBaseReplyMetadata();
        message.hashAllDataHash = (_a = object.hashAllDataHash) !== null && _a !== void 0 ? _a : new Uint8Array();
        message.sig = (_b = object.sig) !== null && _b !== void 0 ? _b : new Uint8Array();
        message.latestBlock = (object.latestBlock !== undefined && object.latestBlock !== null)
            ? long_1.default.fromValue(object.latestBlock)
            : long_1.default.ZERO;
        message.finalizedBlocksHashes = (_c = object.finalizedBlocksHashes) !== null && _c !== void 0 ? _c : new Uint8Array();
        message.sigBlocks = (_d = object.sigBlocks) !== null && _d !== void 0 ? _d : new Uint8Array();
        return message;
    },
};
function createBaseFinalizationConflict() {
    return { relayReply0: undefined, relayReply1: undefined };
}
exports.FinalizationConflict = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.relayReply0 !== undefined) {
            relay_1.RelayReply.encode(message.relayReply0, writer.uint32(10).fork()).ldelim();
        }
        if (message.relayReply1 !== undefined) {
            relay_1.RelayReply.encode(message.relayReply1, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseFinalizationConflict();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.relayReply0 = relay_1.RelayReply.decode(reader, reader.uint32());
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.relayReply1 = relay_1.RelayReply.decode(reader, reader.uint32());
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
            relayReply0: isSet(object.relayReply0) ? relay_1.RelayReply.fromJSON(object.relayReply0) : undefined,
            relayReply1: isSet(object.relayReply1) ? relay_1.RelayReply.fromJSON(object.relayReply1) : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.relayReply0 !== undefined &&
            (obj.relayReply0 = message.relayReply0 ? relay_1.RelayReply.toJSON(message.relayReply0) : undefined);
        message.relayReply1 !== undefined &&
            (obj.relayReply1 = message.relayReply1 ? relay_1.RelayReply.toJSON(message.relayReply1) : undefined);
        return obj;
    },
    create(base) {
        return exports.FinalizationConflict.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseFinalizationConflict();
        message.relayReply0 = (object.relayReply0 !== undefined && object.relayReply0 !== null)
            ? relay_1.RelayReply.fromPartial(object.relayReply0)
            : undefined;
        message.relayReply1 = (object.relayReply1 !== undefined && object.relayReply1 !== null)
            ? relay_1.RelayReply.fromPartial(object.relayReply1)
            : undefined;
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

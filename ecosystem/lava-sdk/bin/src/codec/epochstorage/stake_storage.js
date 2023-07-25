"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.StakeStorage = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const stake_entry_1 = require("./stake_entry");
exports.protobufPackage = "lavanet.lava.epochstorage";
function createBaseStakeStorage() {
    return { index: "", stakeEntries: [], epochBlockHash: new Uint8Array() };
}
exports.StakeStorage = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        for (const v of message.stakeEntries) {
            stake_entry_1.StakeEntry.encode(v, writer.uint32(18).fork()).ldelim();
        }
        if (message.epochBlockHash.length !== 0) {
            writer.uint32(26).bytes(message.epochBlockHash);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseStakeStorage();
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
                    message.stakeEntries.push(stake_entry_1.StakeEntry.decode(reader, reader.uint32()));
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.epochBlockHash = reader.bytes();
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
            stakeEntries: Array.isArray(object === null || object === void 0 ? void 0 : object.stakeEntries)
                ? object.stakeEntries.map((e) => stake_entry_1.StakeEntry.fromJSON(e))
                : [],
            epochBlockHash: isSet(object.epochBlockHash) ? bytesFromBase64(object.epochBlockHash) : new Uint8Array(),
        };
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = message.index);
        if (message.stakeEntries) {
            obj.stakeEntries = message.stakeEntries.map((e) => e ? stake_entry_1.StakeEntry.toJSON(e) : undefined);
        }
        else {
            obj.stakeEntries = [];
        }
        message.epochBlockHash !== undefined &&
            (obj.epochBlockHash = base64FromBytes(message.epochBlockHash !== undefined ? message.epochBlockHash : new Uint8Array()));
        return obj;
    },
    create(base) {
        return exports.StakeStorage.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBaseStakeStorage();
        message.index = (_a = object.index) !== null && _a !== void 0 ? _a : "";
        message.stakeEntries = ((_b = object.stakeEntries) === null || _b === void 0 ? void 0 : _b.map((e) => stake_entry_1.StakeEntry.fromPartial(e))) || [];
        message.epochBlockHash = (_c = object.epochBlockHash) !== null && _c !== void 0 ? _c : new Uint8Array();
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

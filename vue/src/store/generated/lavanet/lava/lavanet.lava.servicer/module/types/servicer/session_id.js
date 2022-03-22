/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";
export const protobufPackage = "lavanet.lava.servicer";
const baseSessionID = { num: 0 };
export const SessionID = {
    encode(message, writer = Writer.create()) {
        if (message.num !== 0) {
            writer.uint32(8).uint64(message.num);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseSessionID };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.num = longToNumber(reader.uint64());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = { ...baseSessionID };
        if (object.num !== undefined && object.num !== null) {
            message.num = Number(object.num);
        }
        else {
            message.num = 0;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.num !== undefined && (obj.num = message.num);
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseSessionID };
        if (object.num !== undefined && object.num !== null) {
            message.num = object.num;
        }
        else {
            message.num = 0;
        }
        return message;
    },
};
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

/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";
import { UserStake } from "../user/user_stake";
export const protobufPackage = "lavanet.lava.user";
const baseUnstakingUsersAllSpecs = { id: 0 };
export const UnstakingUsersAllSpecs = {
    encode(message, writer = Writer.create()) {
        if (message.id !== 0) {
            writer.uint32(8).uint64(message.id);
        }
        if (message.unstaking !== undefined) {
            UserStake.encode(message.unstaking, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseUnstakingUsersAllSpecs };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.id = longToNumber(reader.uint64());
                    break;
                case 2:
                    message.unstaking = UserStake.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = { ...baseUnstakingUsersAllSpecs };
        if (object.id !== undefined && object.id !== null) {
            message.id = Number(object.id);
        }
        else {
            message.id = 0;
        }
        if (object.unstaking !== undefined && object.unstaking !== null) {
            message.unstaking = UserStake.fromJSON(object.unstaking);
        }
        else {
            message.unstaking = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.id !== undefined && (obj.id = message.id);
        message.unstaking !== undefined &&
            (obj.unstaking = message.unstaking
                ? UserStake.toJSON(message.unstaking)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseUnstakingUsersAllSpecs };
        if (object.id !== undefined && object.id !== null) {
            message.id = object.id;
        }
        else {
            message.id = 0;
        }
        if (object.unstaking !== undefined && object.unstaking !== null) {
            message.unstaking = UserStake.fromPartial(object.unstaking);
        }
        else {
            message.unstaking = undefined;
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

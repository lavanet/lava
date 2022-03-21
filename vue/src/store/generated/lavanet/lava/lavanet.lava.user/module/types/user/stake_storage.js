/* eslint-disable */
import { UserStake } from "../user/user_stake";
import { Writer, Reader } from "protobufjs/minimal";
export const protobufPackage = "lavanet.lava.user";
const baseStakeStorage = {};
export const StakeStorage = {
    encode(message, writer = Writer.create()) {
        if (message.stakedUsers !== undefined) {
            UserStake.encode(message.stakedUsers, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseStakeStorage };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.stakedUsers = UserStake.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = { ...baseStakeStorage };
        if (object.stakedUsers !== undefined && object.stakedUsers !== null) {
            message.stakedUsers = UserStake.fromJSON(object.stakedUsers);
        }
        else {
            message.stakedUsers = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.stakedUsers !== undefined &&
            (obj.stakedUsers = message.stakedUsers
                ? UserStake.toJSON(message.stakedUsers)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseStakeStorage };
        if (object.stakedUsers !== undefined && object.stakedUsers !== null) {
            message.stakedUsers = UserStake.fromPartial(object.stakedUsers);
        }
        else {
            message.stakedUsers = undefined;
        }
        return message;
    },
};

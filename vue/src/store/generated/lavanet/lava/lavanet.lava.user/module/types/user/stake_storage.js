/* eslint-disable */
import { UserStake } from "../user/user_stake";
import { Writer, Reader } from "protobufjs/minimal";
export const protobufPackage = "lavanet.lava.user";
const baseStakeStorage = {};
export const StakeStorage = {
    encode(message, writer = Writer.create()) {
        for (const v of message.stakedUsers) {
            UserStake.encode(v, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseStakeStorage };
        message.stakedUsers = [];
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.stakedUsers.push(UserStake.decode(reader, reader.uint32()));
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
        message.stakedUsers = [];
        if (object.stakedUsers !== undefined && object.stakedUsers !== null) {
            for (const e of object.stakedUsers) {
                message.stakedUsers.push(UserStake.fromJSON(e));
            }
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        if (message.stakedUsers) {
            obj.stakedUsers = message.stakedUsers.map((e) => e ? UserStake.toJSON(e) : undefined);
        }
        else {
            obj.stakedUsers = [];
        }
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseStakeStorage };
        message.stakedUsers = [];
        if (object.stakedUsers !== undefined && object.stakedUsers !== null) {
            for (const e of object.stakedUsers) {
                message.stakedUsers.push(UserStake.fromPartial(e));
            }
        }
        return message;
    },
};

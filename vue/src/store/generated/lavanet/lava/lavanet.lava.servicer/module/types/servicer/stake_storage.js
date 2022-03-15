/* eslint-disable */
import { StakeMap } from "../servicer/stake_map";
import { Writer, Reader } from "protobufjs/minimal";
export const protobufPackage = "lavanet.lava.servicer";
const baseStakeStorage = {};
export const StakeStorage = {
    encode(message, writer = Writer.create()) {
        for (const v of message.staked) {
            StakeMap.encode(v, writer.uint32(10).fork()).ldelim();
        }
        for (const v of message.unstaking) {
            StakeMap.encode(v, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseStakeStorage };
        message.staked = [];
        message.unstaking = [];
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.staked.push(StakeMap.decode(reader, reader.uint32()));
                    break;
                case 2:
                    message.unstaking.push(StakeMap.decode(reader, reader.uint32()));
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
        message.staked = [];
        message.unstaking = [];
        if (object.staked !== undefined && object.staked !== null) {
            for (const e of object.staked) {
                message.staked.push(StakeMap.fromJSON(e));
            }
        }
        if (object.unstaking !== undefined && object.unstaking !== null) {
            for (const e of object.unstaking) {
                message.unstaking.push(StakeMap.fromJSON(e));
            }
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        if (message.staked) {
            obj.staked = message.staked.map((e) => e ? StakeMap.toJSON(e) : undefined);
        }
        else {
            obj.staked = [];
        }
        if (message.unstaking) {
            obj.unstaking = message.unstaking.map((e) => e ? StakeMap.toJSON(e) : undefined);
        }
        else {
            obj.unstaking = [];
        }
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseStakeStorage };
        message.staked = [];
        message.unstaking = [];
        if (object.staked !== undefined && object.staked !== null) {
            for (const e of object.staked) {
                message.staked.push(StakeMap.fromPartial(e));
            }
        }
        if (object.unstaking !== undefined && object.unstaking !== null) {
            for (const e of object.unstaking) {
                message.unstaking.push(StakeMap.fromPartial(e));
            }
        }
        return message;
    },
};

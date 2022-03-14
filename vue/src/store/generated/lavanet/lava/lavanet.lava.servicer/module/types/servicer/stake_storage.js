/* eslint-disable */
import { StakeMap } from "../servicer/stake_map";
import { Writer, Reader } from "protobufjs/minimal";
export const protobufPackage = "lavanet.lava.servicer";
const baseStakeStorage = {};
export const StakeStorage = {
    encode(message, writer = Writer.create()) {
        if (message.staked !== undefined) {
            StakeMap.encode(message.staked, writer.uint32(10).fork()).ldelim();
        }
        if (message.unstaking !== undefined) {
            StakeMap.encode(message.unstaking, writer.uint32(18).fork()).ldelim();
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
                    message.staked = StakeMap.decode(reader, reader.uint32());
                    break;
                case 2:
                    message.unstaking = StakeMap.decode(reader, reader.uint32());
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
        if (object.staked !== undefined && object.staked !== null) {
            message.staked = StakeMap.fromJSON(object.staked);
        }
        else {
            message.staked = undefined;
        }
        if (object.unstaking !== undefined && object.unstaking !== null) {
            message.unstaking = StakeMap.fromJSON(object.unstaking);
        }
        else {
            message.unstaking = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.staked !== undefined &&
            (obj.staked = message.staked
                ? StakeMap.toJSON(message.staked)
                : undefined);
        message.unstaking !== undefined &&
            (obj.unstaking = message.unstaking
                ? StakeMap.toJSON(message.unstaking)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseStakeStorage };
        if (object.staked !== undefined && object.staked !== null) {
            message.staked = StakeMap.fromPartial(object.staked);
        }
        else {
            message.staked = undefined;
        }
        if (object.unstaking !== undefined && object.unstaking !== null) {
            message.unstaking = StakeMap.fromPartial(object.unstaking);
        }
        else {
            message.unstaking = undefined;
        }
        return message;
    },
};

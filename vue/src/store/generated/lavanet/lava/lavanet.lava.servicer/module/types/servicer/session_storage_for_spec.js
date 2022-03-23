/* eslint-disable */
import { StakeStorage } from "../servicer/stake_storage";
import { Writer, Reader } from "protobufjs/minimal";
export const protobufPackage = "lavanet.lava.servicer";
const baseSessionStorageForSpec = { index: "" };
export const SessionStorageForSpec = {
    encode(message, writer = Writer.create()) {
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        if (message.stakeStorage !== undefined) {
            StakeStorage.encode(message.stakeStorage, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseSessionStorageForSpec };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.index = reader.string();
                    break;
                case 2:
                    message.stakeStorage = StakeStorage.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = { ...baseSessionStorageForSpec };
        if (object.index !== undefined && object.index !== null) {
            message.index = String(object.index);
        }
        else {
            message.index = "";
        }
        if (object.stakeStorage !== undefined && object.stakeStorage !== null) {
            message.stakeStorage = StakeStorage.fromJSON(object.stakeStorage);
        }
        else {
            message.stakeStorage = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = message.index);
        message.stakeStorage !== undefined &&
            (obj.stakeStorage = message.stakeStorage
                ? StakeStorage.toJSON(message.stakeStorage)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseSessionStorageForSpec };
        if (object.index !== undefined && object.index !== null) {
            message.index = object.index;
        }
        else {
            message.index = "";
        }
        if (object.stakeStorage !== undefined && object.stakeStorage !== null) {
            message.stakeStorage = StakeStorage.fromPartial(object.stakeStorage);
        }
        else {
            message.stakeStorage = undefined;
        }
        return message;
    },
};

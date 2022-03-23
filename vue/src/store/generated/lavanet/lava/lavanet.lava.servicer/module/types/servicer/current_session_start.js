/* eslint-disable */
import { BlockNum } from "../servicer/block_num";
import { Writer, Reader } from "protobufjs/minimal";
export const protobufPackage = "lavanet.lava.servicer";
const baseCurrentSessionStart = {};
export const CurrentSessionStart = {
    encode(message, writer = Writer.create()) {
        if (message.block !== undefined) {
            BlockNum.encode(message.block, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseCurrentSessionStart };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.block = BlockNum.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = { ...baseCurrentSessionStart };
        if (object.block !== undefined && object.block !== null) {
            message.block = BlockNum.fromJSON(object.block);
        }
        else {
            message.block = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.block !== undefined &&
            (obj.block = message.block ? BlockNum.toJSON(message.block) : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseCurrentSessionStart };
        if (object.block !== undefined && object.block !== null) {
            message.block = BlockNum.fromPartial(object.block);
        }
        else {
            message.block = undefined;
        }
        return message;
    },
};

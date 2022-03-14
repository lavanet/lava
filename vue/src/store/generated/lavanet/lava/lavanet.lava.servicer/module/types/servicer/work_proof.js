/* eslint-disable */
import { Writer, Reader } from "protobufjs/minimal";
export const protobufPackage = "lavanet.lava.servicer";
const baseWorkProof = { data: "" };
export const WorkProof = {
    encode(message, writer = Writer.create()) {
        if (message.data !== "") {
            writer.uint32(10).string(message.data);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseWorkProof };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.data = reader.string();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = { ...baseWorkProof };
        if (object.data !== undefined && object.data !== null) {
            message.data = String(object.data);
        }
        else {
            message.data = "";
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.data !== undefined && (obj.data = message.data);
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseWorkProof };
        if (object.data !== undefined && object.data !== null) {
            message.data = object.data;
        }
        else {
            message.data = "";
        }
        return message;
    },
};

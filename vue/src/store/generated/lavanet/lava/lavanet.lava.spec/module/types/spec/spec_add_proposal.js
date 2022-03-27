/* eslint-disable */
import { Spec } from "../spec/spec";
import { Writer, Reader } from "protobufjs/minimal";
export const protobufPackage = "lavanet.lava.spec";
const baseSpecAddProposal = { title: "", description: "" };
export const SpecAddProposal = {
    encode(message, writer = Writer.create()) {
        if (message.title !== "") {
            writer.uint32(10).string(message.title);
        }
        if (message.description !== "") {
            writer.uint32(18).string(message.description);
        }
        for (const v of message.specs) {
            Spec.encode(v, writer.uint32(26).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseSpecAddProposal };
        message.specs = [];
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.title = reader.string();
                    break;
                case 2:
                    message.description = reader.string();
                    break;
                case 3:
                    message.specs.push(Spec.decode(reader, reader.uint32()));
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = { ...baseSpecAddProposal };
        message.specs = [];
        if (object.title !== undefined && object.title !== null) {
            message.title = String(object.title);
        }
        else {
            message.title = "";
        }
        if (object.description !== undefined && object.description !== null) {
            message.description = String(object.description);
        }
        else {
            message.description = "";
        }
        if (object.specs !== undefined && object.specs !== null) {
            for (const e of object.specs) {
                message.specs.push(Spec.fromJSON(e));
            }
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.title !== undefined && (obj.title = message.title);
        message.description !== undefined &&
            (obj.description = message.description);
        if (message.specs) {
            obj.specs = message.specs.map((e) => (e ? Spec.toJSON(e) : undefined));
        }
        else {
            obj.specs = [];
        }
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseSpecAddProposal };
        message.specs = [];
        if (object.title !== undefined && object.title !== null) {
            message.title = object.title;
        }
        else {
            message.title = "";
        }
        if (object.description !== undefined && object.description !== null) {
            message.description = object.description;
        }
        else {
            message.description = "";
        }
        if (object.specs !== undefined && object.specs !== null) {
            for (const e of object.specs) {
                message.specs.push(Spec.fromPartial(e));
            }
        }
        return message;
    },
};

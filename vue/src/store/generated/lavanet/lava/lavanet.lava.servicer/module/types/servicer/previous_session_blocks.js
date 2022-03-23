/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";
import { BlockNum } from "../servicer/block_num";
export const protobufPackage = "lavanet.lava.servicer";
const basePreviousSessionBlocks = { blocksNum: 0, overlapBlocks: 0 };
export const PreviousSessionBlocks = {
    encode(message, writer = Writer.create()) {
        if (message.blocksNum !== 0) {
            writer.uint32(8).uint64(message.blocksNum);
        }
        if (message.changeBlock !== undefined) {
            BlockNum.encode(message.changeBlock, writer.uint32(18).fork()).ldelim();
        }
        if (message.overlapBlocks !== 0) {
            writer.uint32(24).uint64(message.overlapBlocks);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...basePreviousSessionBlocks };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.blocksNum = longToNumber(reader.uint64());
                    break;
                case 2:
                    message.changeBlock = BlockNum.decode(reader, reader.uint32());
                    break;
                case 3:
                    message.overlapBlocks = longToNumber(reader.uint64());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = { ...basePreviousSessionBlocks };
        if (object.blocksNum !== undefined && object.blocksNum !== null) {
            message.blocksNum = Number(object.blocksNum);
        }
        else {
            message.blocksNum = 0;
        }
        if (object.changeBlock !== undefined && object.changeBlock !== null) {
            message.changeBlock = BlockNum.fromJSON(object.changeBlock);
        }
        else {
            message.changeBlock = undefined;
        }
        if (object.overlapBlocks !== undefined && object.overlapBlocks !== null) {
            message.overlapBlocks = Number(object.overlapBlocks);
        }
        else {
            message.overlapBlocks = 0;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.blocksNum !== undefined && (obj.blocksNum = message.blocksNum);
        message.changeBlock !== undefined &&
            (obj.changeBlock = message.changeBlock
                ? BlockNum.toJSON(message.changeBlock)
                : undefined);
        message.overlapBlocks !== undefined &&
            (obj.overlapBlocks = message.overlapBlocks);
        return obj;
    },
    fromPartial(object) {
        const message = { ...basePreviousSessionBlocks };
        if (object.blocksNum !== undefined && object.blocksNum !== null) {
            message.blocksNum = object.blocksNum;
        }
        else {
            message.blocksNum = 0;
        }
        if (object.changeBlock !== undefined && object.changeBlock !== null) {
            message.changeBlock = BlockNum.fromPartial(object.changeBlock);
        }
        else {
            message.changeBlock = undefined;
        }
        if (object.overlapBlocks !== undefined && object.overlapBlocks !== null) {
            message.overlapBlocks = object.overlapBlocks;
        }
        else {
            message.overlapBlocks = 0;
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

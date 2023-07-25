"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Params = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
exports.protobufPackage = "lavanet.lava.epochstorage";
function createBaseParams() {
    return {
        unstakeHoldBlocks: long_1.default.UZERO,
        epochBlocks: long_1.default.UZERO,
        epochsToSave: long_1.default.UZERO,
        latestParamChange: long_1.default.UZERO,
        unstakeHoldBlocksStatic: long_1.default.UZERO,
    };
}
exports.Params = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (!message.unstakeHoldBlocks.isZero()) {
            writer.uint32(8).uint64(message.unstakeHoldBlocks);
        }
        if (!message.epochBlocks.isZero()) {
            writer.uint32(16).uint64(message.epochBlocks);
        }
        if (!message.epochsToSave.isZero()) {
            writer.uint32(24).uint64(message.epochsToSave);
        }
        if (!message.latestParamChange.isZero()) {
            writer.uint32(32).uint64(message.latestParamChange);
        }
        if (!message.unstakeHoldBlocksStatic.isZero()) {
            writer.uint32(40).uint64(message.unstakeHoldBlocksStatic);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseParams();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.unstakeHoldBlocks = reader.uint64();
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.epochBlocks = reader.uint64();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.epochsToSave = reader.uint64();
                    continue;
                case 4:
                    if (tag != 32) {
                        break;
                    }
                    message.latestParamChange = reader.uint64();
                    continue;
                case 5:
                    if (tag != 40) {
                        break;
                    }
                    message.unstakeHoldBlocksStatic = reader.uint64();
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            unstakeHoldBlocks: isSet(object.unstakeHoldBlocks) ? long_1.default.fromValue(object.unstakeHoldBlocks) : long_1.default.UZERO,
            epochBlocks: isSet(object.epochBlocks) ? long_1.default.fromValue(object.epochBlocks) : long_1.default.UZERO,
            epochsToSave: isSet(object.epochsToSave) ? long_1.default.fromValue(object.epochsToSave) : long_1.default.UZERO,
            latestParamChange: isSet(object.latestParamChange) ? long_1.default.fromValue(object.latestParamChange) : long_1.default.UZERO,
            unstakeHoldBlocksStatic: isSet(object.unstakeHoldBlocksStatic)
                ? long_1.default.fromValue(object.unstakeHoldBlocksStatic)
                : long_1.default.UZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.unstakeHoldBlocks !== undefined &&
            (obj.unstakeHoldBlocks = (message.unstakeHoldBlocks || long_1.default.UZERO).toString());
        message.epochBlocks !== undefined && (obj.epochBlocks = (message.epochBlocks || long_1.default.UZERO).toString());
        message.epochsToSave !== undefined && (obj.epochsToSave = (message.epochsToSave || long_1.default.UZERO).toString());
        message.latestParamChange !== undefined &&
            (obj.latestParamChange = (message.latestParamChange || long_1.default.UZERO).toString());
        message.unstakeHoldBlocksStatic !== undefined &&
            (obj.unstakeHoldBlocksStatic = (message.unstakeHoldBlocksStatic || long_1.default.UZERO).toString());
        return obj;
    },
    create(base) {
        return exports.Params.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseParams();
        message.unstakeHoldBlocks = (object.unstakeHoldBlocks !== undefined && object.unstakeHoldBlocks !== null)
            ? long_1.default.fromValue(object.unstakeHoldBlocks)
            : long_1.default.UZERO;
        message.epochBlocks = (object.epochBlocks !== undefined && object.epochBlocks !== null)
            ? long_1.default.fromValue(object.epochBlocks)
            : long_1.default.UZERO;
        message.epochsToSave = (object.epochsToSave !== undefined && object.epochsToSave !== null)
            ? long_1.default.fromValue(object.epochsToSave)
            : long_1.default.UZERO;
        message.latestParamChange = (object.latestParamChange !== undefined && object.latestParamChange !== null)
            ? long_1.default.fromValue(object.latestParamChange)
            : long_1.default.UZERO;
        message.unstakeHoldBlocksStatic =
            (object.unstakeHoldBlocksStatic !== undefined && object.unstakeHoldBlocksStatic !== null)
                ? long_1.default.fromValue(object.unstakeHoldBlocksStatic)
                : long_1.default.UZERO;
        return message;
    },
};
if (minimal_1.default.util.Long !== long_1.default) {
    minimal_1.default.util.Long = long_1.default;
    minimal_1.default.configure();
}
function isSet(value) {
    return value !== null && value !== undefined;
}

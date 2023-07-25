"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.EpochDetails = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
exports.protobufPackage = "lavanet.lava.epochstorage";
function createBaseEpochDetails() {
    return { startBlock: long_1.default.UZERO, earliestStart: long_1.default.UZERO, deletedEpochs: [] };
}
exports.EpochDetails = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (!message.startBlock.isZero()) {
            writer.uint32(8).uint64(message.startBlock);
        }
        if (!message.earliestStart.isZero()) {
            writer.uint32(16).uint64(message.earliestStart);
        }
        writer.uint32(26).fork();
        for (const v of message.deletedEpochs) {
            writer.uint64(v);
        }
        writer.ldelim();
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseEpochDetails();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.startBlock = reader.uint64();
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.earliestStart = reader.uint64();
                    continue;
                case 3:
                    if (tag == 24) {
                        message.deletedEpochs.push(reader.uint64());
                        continue;
                    }
                    if (tag == 26) {
                        const end2 = reader.uint32() + reader.pos;
                        while (reader.pos < end2) {
                            message.deletedEpochs.push(reader.uint64());
                        }
                        continue;
                    }
                    break;
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
            startBlock: isSet(object.startBlock) ? long_1.default.fromValue(object.startBlock) : long_1.default.UZERO,
            earliestStart: isSet(object.earliestStart) ? long_1.default.fromValue(object.earliestStart) : long_1.default.UZERO,
            deletedEpochs: Array.isArray(object === null || object === void 0 ? void 0 : object.deletedEpochs)
                ? object.deletedEpochs.map((e) => long_1.default.fromValue(e))
                : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.startBlock !== undefined && (obj.startBlock = (message.startBlock || long_1.default.UZERO).toString());
        message.earliestStart !== undefined && (obj.earliestStart = (message.earliestStart || long_1.default.UZERO).toString());
        if (message.deletedEpochs) {
            obj.deletedEpochs = message.deletedEpochs.map((e) => (e || long_1.default.UZERO).toString());
        }
        else {
            obj.deletedEpochs = [];
        }
        return obj;
    },
    create(base) {
        return exports.EpochDetails.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseEpochDetails();
        message.startBlock = (object.startBlock !== undefined && object.startBlock !== null)
            ? long_1.default.fromValue(object.startBlock)
            : long_1.default.UZERO;
        message.earliestStart = (object.earliestStart !== undefined && object.earliestStart !== null)
            ? long_1.default.fromValue(object.earliestStart)
            : long_1.default.UZERO;
        message.deletedEpochs = ((_a = object.deletedEpochs) === null || _a === void 0 ? void 0 : _a.map((e) => long_1.default.fromValue(e))) || [];
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

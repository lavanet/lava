"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Params = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
exports.protobufPackage = "lavanet.lava.spec";
function createBaseParams() {
    return { geolocationCount: long_1.default.UZERO, maxCU: long_1.default.UZERO };
}
exports.Params = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (!message.geolocationCount.isZero()) {
            writer.uint32(8).uint64(message.geolocationCount);
        }
        if (!message.maxCU.isZero()) {
            writer.uint32(16).uint64(message.maxCU);
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
                    message.geolocationCount = reader.uint64();
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.maxCU = reader.uint64();
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
            geolocationCount: isSet(object.geolocationCount) ? long_1.default.fromValue(object.geolocationCount) : long_1.default.UZERO,
            maxCU: isSet(object.maxCU) ? long_1.default.fromValue(object.maxCU) : long_1.default.UZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.geolocationCount !== undefined &&
            (obj.geolocationCount = (message.geolocationCount || long_1.default.UZERO).toString());
        message.maxCU !== undefined && (obj.maxCU = (message.maxCU || long_1.default.UZERO).toString());
        return obj;
    },
    create(base) {
        return exports.Params.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseParams();
        message.geolocationCount = (object.geolocationCount !== undefined && object.geolocationCount !== null)
            ? long_1.default.fromValue(object.geolocationCount)
            : long_1.default.UZERO;
        message.maxCU = (object.maxCU !== undefined && object.maxCU !== null) ? long_1.default.fromValue(object.maxCU) : long_1.default.UZERO;
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

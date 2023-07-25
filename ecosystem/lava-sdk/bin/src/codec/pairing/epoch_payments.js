"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.EpochPayments = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
exports.protobufPackage = "lavanet.lava.pairing";
function createBaseEpochPayments() {
    return { index: "", providerPaymentStorageKeys: [] };
}
exports.EpochPayments = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        for (const v of message.providerPaymentStorageKeys) {
            writer.uint32(26).string(v);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseEpochPayments();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.index = reader.string();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.providerPaymentStorageKeys.push(reader.string());
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
            index: isSet(object.index) ? String(object.index) : "",
            providerPaymentStorageKeys: Array.isArray(object === null || object === void 0 ? void 0 : object.providerPaymentStorageKeys)
                ? object.providerPaymentStorageKeys.map((e) => String(e))
                : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = message.index);
        if (message.providerPaymentStorageKeys) {
            obj.providerPaymentStorageKeys = message.providerPaymentStorageKeys.map((e) => e);
        }
        else {
            obj.providerPaymentStorageKeys = [];
        }
        return obj;
    },
    create(base) {
        return exports.EpochPayments.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseEpochPayments();
        message.index = (_a = object.index) !== null && _a !== void 0 ? _a : "";
        message.providerPaymentStorageKeys = ((_b = object.providerPaymentStorageKeys) === null || _b === void 0 ? void 0 : _b.map((e) => e)) || [];
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

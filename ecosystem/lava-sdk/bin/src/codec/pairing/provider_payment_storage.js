"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.ProviderPaymentStorage = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
exports.protobufPackage = "lavanet.lava.pairing";
function createBaseProviderPaymentStorage() {
    return { index: "", epoch: long_1.default.UZERO, uniquePaymentStorageClientProviderKeys: [], complainersTotalCu: long_1.default.UZERO };
}
exports.ProviderPaymentStorage = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        if (!message.epoch.isZero()) {
            writer.uint32(24).uint64(message.epoch);
        }
        for (const v of message.uniquePaymentStorageClientProviderKeys) {
            writer.uint32(42).string(v);
        }
        if (!message.complainersTotalCu.isZero()) {
            writer.uint32(48).uint64(message.complainersTotalCu);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseProviderPaymentStorage();
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
                    if (tag != 24) {
                        break;
                    }
                    message.epoch = reader.uint64();
                    continue;
                case 5:
                    if (tag != 42) {
                        break;
                    }
                    message.uniquePaymentStorageClientProviderKeys.push(reader.string());
                    continue;
                case 6:
                    if (tag != 48) {
                        break;
                    }
                    message.complainersTotalCu = reader.uint64();
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
            epoch: isSet(object.epoch) ? long_1.default.fromValue(object.epoch) : long_1.default.UZERO,
            uniquePaymentStorageClientProviderKeys: Array.isArray(object === null || object === void 0 ? void 0 : object.uniquePaymentStorageClientProviderKeys)
                ? object.uniquePaymentStorageClientProviderKeys.map((e) => String(e))
                : [],
            complainersTotalCu: isSet(object.complainersTotalCu) ? long_1.default.fromValue(object.complainersTotalCu) : long_1.default.UZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = message.index);
        message.epoch !== undefined && (obj.epoch = (message.epoch || long_1.default.UZERO).toString());
        if (message.uniquePaymentStorageClientProviderKeys) {
            obj.uniquePaymentStorageClientProviderKeys = message.uniquePaymentStorageClientProviderKeys.map((e) => e);
        }
        else {
            obj.uniquePaymentStorageClientProviderKeys = [];
        }
        message.complainersTotalCu !== undefined &&
            (obj.complainersTotalCu = (message.complainersTotalCu || long_1.default.UZERO).toString());
        return obj;
    },
    create(base) {
        return exports.ProviderPaymentStorage.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseProviderPaymentStorage();
        message.index = (_a = object.index) !== null && _a !== void 0 ? _a : "";
        message.epoch = (object.epoch !== undefined && object.epoch !== null) ? long_1.default.fromValue(object.epoch) : long_1.default.UZERO;
        message.uniquePaymentStorageClientProviderKeys = ((_b = object.uniquePaymentStorageClientProviderKeys) === null || _b === void 0 ? void 0 : _b.map((e) => e)) || [];
        message.complainersTotalCu = (object.complainersTotalCu !== undefined && object.complainersTotalCu !== null)
            ? long_1.default.fromValue(object.complainersTotalCu)
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

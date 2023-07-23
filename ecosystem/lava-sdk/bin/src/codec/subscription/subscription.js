"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Subscription = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
exports.protobufPackage = "lavanet.lava.subscription";
function createBaseSubscription() {
    return {
        creator: "",
        consumer: "",
        block: long_1.default.UZERO,
        planIndex: "",
        planBlock: long_1.default.UZERO,
        durationTotal: long_1.default.UZERO,
        durationLeft: long_1.default.UZERO,
        monthExpiryTime: long_1.default.UZERO,
        monthCuTotal: long_1.default.UZERO,
        monthCuLeft: long_1.default.UZERO,
    };
}
exports.Subscription = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.creator !== "") {
            writer.uint32(10).string(message.creator);
        }
        if (message.consumer !== "") {
            writer.uint32(18).string(message.consumer);
        }
        if (!message.block.isZero()) {
            writer.uint32(24).uint64(message.block);
        }
        if (message.planIndex !== "") {
            writer.uint32(34).string(message.planIndex);
        }
        if (!message.planBlock.isZero()) {
            writer.uint32(40).uint64(message.planBlock);
        }
        if (!message.durationTotal.isZero()) {
            writer.uint32(48).uint64(message.durationTotal);
        }
        if (!message.durationLeft.isZero()) {
            writer.uint32(56).uint64(message.durationLeft);
        }
        if (!message.monthExpiryTime.isZero()) {
            writer.uint32(64).uint64(message.monthExpiryTime);
        }
        if (!message.monthCuTotal.isZero()) {
            writer.uint32(80).uint64(message.monthCuTotal);
        }
        if (!message.monthCuLeft.isZero()) {
            writer.uint32(88).uint64(message.monthCuLeft);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseSubscription();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.creator = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.consumer = reader.string();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.block = reader.uint64();
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.planIndex = reader.string();
                    continue;
                case 5:
                    if (tag != 40) {
                        break;
                    }
                    message.planBlock = reader.uint64();
                    continue;
                case 6:
                    if (tag != 48) {
                        break;
                    }
                    message.durationTotal = reader.uint64();
                    continue;
                case 7:
                    if (tag != 56) {
                        break;
                    }
                    message.durationLeft = reader.uint64();
                    continue;
                case 8:
                    if (tag != 64) {
                        break;
                    }
                    message.monthExpiryTime = reader.uint64();
                    continue;
                case 10:
                    if (tag != 80) {
                        break;
                    }
                    message.monthCuTotal = reader.uint64();
                    continue;
                case 11:
                    if (tag != 88) {
                        break;
                    }
                    message.monthCuLeft = reader.uint64();
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
            creator: isSet(object.creator) ? String(object.creator) : "",
            consumer: isSet(object.consumer) ? String(object.consumer) : "",
            block: isSet(object.block) ? long_1.default.fromValue(object.block) : long_1.default.UZERO,
            planIndex: isSet(object.planIndex) ? String(object.planIndex) : "",
            planBlock: isSet(object.planBlock) ? long_1.default.fromValue(object.planBlock) : long_1.default.UZERO,
            durationTotal: isSet(object.durationTotal) ? long_1.default.fromValue(object.durationTotal) : long_1.default.UZERO,
            durationLeft: isSet(object.durationLeft) ? long_1.default.fromValue(object.durationLeft) : long_1.default.UZERO,
            monthExpiryTime: isSet(object.monthExpiryTime) ? long_1.default.fromValue(object.monthExpiryTime) : long_1.default.UZERO,
            monthCuTotal: isSet(object.monthCuTotal) ? long_1.default.fromValue(object.monthCuTotal) : long_1.default.UZERO,
            monthCuLeft: isSet(object.monthCuLeft) ? long_1.default.fromValue(object.monthCuLeft) : long_1.default.UZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.creator !== undefined && (obj.creator = message.creator);
        message.consumer !== undefined && (obj.consumer = message.consumer);
        message.block !== undefined && (obj.block = (message.block || long_1.default.UZERO).toString());
        message.planIndex !== undefined && (obj.planIndex = message.planIndex);
        message.planBlock !== undefined && (obj.planBlock = (message.planBlock || long_1.default.UZERO).toString());
        message.durationTotal !== undefined && (obj.durationTotal = (message.durationTotal || long_1.default.UZERO).toString());
        message.durationLeft !== undefined && (obj.durationLeft = (message.durationLeft || long_1.default.UZERO).toString());
        message.monthExpiryTime !== undefined && (obj.monthExpiryTime = (message.monthExpiryTime || long_1.default.UZERO).toString());
        message.monthCuTotal !== undefined && (obj.monthCuTotal = (message.monthCuTotal || long_1.default.UZERO).toString());
        message.monthCuLeft !== undefined && (obj.monthCuLeft = (message.monthCuLeft || long_1.default.UZERO).toString());
        return obj;
    },
    create(base) {
        return exports.Subscription.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBaseSubscription();
        message.creator = (_a = object.creator) !== null && _a !== void 0 ? _a : "";
        message.consumer = (_b = object.consumer) !== null && _b !== void 0 ? _b : "";
        message.block = (object.block !== undefined && object.block !== null) ? long_1.default.fromValue(object.block) : long_1.default.UZERO;
        message.planIndex = (_c = object.planIndex) !== null && _c !== void 0 ? _c : "";
        message.planBlock = (object.planBlock !== undefined && object.planBlock !== null)
            ? long_1.default.fromValue(object.planBlock)
            : long_1.default.UZERO;
        message.durationTotal = (object.durationTotal !== undefined && object.durationTotal !== null)
            ? long_1.default.fromValue(object.durationTotal)
            : long_1.default.UZERO;
        message.durationLeft = (object.durationLeft !== undefined && object.durationLeft !== null)
            ? long_1.default.fromValue(object.durationLeft)
            : long_1.default.UZERO;
        message.monthExpiryTime = (object.monthExpiryTime !== undefined && object.monthExpiryTime !== null)
            ? long_1.default.fromValue(object.monthExpiryTime)
            : long_1.default.UZERO;
        message.monthCuTotal = (object.monthCuTotal !== undefined && object.monthCuTotal !== null)
            ? long_1.default.fromValue(object.monthCuTotal)
            : long_1.default.UZERO;
        message.monthCuLeft = (object.monthCuLeft !== undefined && object.monthCuLeft !== null)
            ? long_1.default.fromValue(object.monthCuLeft)
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

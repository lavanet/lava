"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Plan = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const coin_1 = require("../cosmos/base/v1beta1/coin");
const project_1 = require("../projects/project");
exports.protobufPackage = "lavanet.lava.plans";
function createBasePlan() {
    return {
        index: "",
        block: long_1.default.UZERO,
        price: undefined,
        allowOveruse: false,
        overuseRate: long_1.default.UZERO,
        description: "",
        type: "",
        annualDiscountPercentage: long_1.default.UZERO,
        planPolicy: undefined,
    };
}
exports.Plan = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        if (!message.block.isZero()) {
            writer.uint32(24).uint64(message.block);
        }
        if (message.price !== undefined) {
            coin_1.Coin.encode(message.price, writer.uint32(34).fork()).ldelim();
        }
        if (message.allowOveruse === true) {
            writer.uint32(64).bool(message.allowOveruse);
        }
        if (!message.overuseRate.isZero()) {
            writer.uint32(72).uint64(message.overuseRate);
        }
        if (message.description !== "") {
            writer.uint32(90).string(message.description);
        }
        if (message.type !== "") {
            writer.uint32(98).string(message.type);
        }
        if (!message.annualDiscountPercentage.isZero()) {
            writer.uint32(104).uint64(message.annualDiscountPercentage);
        }
        if (message.planPolicy !== undefined) {
            project_1.Policy.encode(message.planPolicy, writer.uint32(114).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBasePlan();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag !== 10) {
                        break;
                    }
                    message.index = reader.string();
                    continue;
                case 3:
                    if (tag !== 24) {
                        break;
                    }
                    message.block = reader.uint64();
                    continue;
                case 4:
                    if (tag !== 34) {
                        break;
                    }
                    message.price = coin_1.Coin.decode(reader, reader.uint32());
                    continue;
                case 8:
                    if (tag !== 64) {
                        break;
                    }
                    message.allowOveruse = reader.bool();
                    continue;
                case 9:
                    if (tag !== 72) {
                        break;
                    }
                    message.overuseRate = reader.uint64();
                    continue;
                case 11:
                    if (tag !== 90) {
                        break;
                    }
                    message.description = reader.string();
                    continue;
                case 12:
                    if (tag !== 98) {
                        break;
                    }
                    message.type = reader.string();
                    continue;
                case 13:
                    if (tag !== 104) {
                        break;
                    }
                    message.annualDiscountPercentage = reader.uint64();
                    continue;
                case 14:
                    if (tag !== 114) {
                        break;
                    }
                    message.planPolicy = project_1.Policy.decode(reader, reader.uint32());
                    continue;
            }
            if ((tag & 7) === 4 || tag === 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            index: isSet(object.index) ? String(object.index) : "",
            block: isSet(object.block) ? long_1.default.fromValue(object.block) : long_1.default.UZERO,
            price: isSet(object.price) ? coin_1.Coin.fromJSON(object.price) : undefined,
            allowOveruse: isSet(object.allowOveruse) ? Boolean(object.allowOveruse) : false,
            overuseRate: isSet(object.overuseRate) ? long_1.default.fromValue(object.overuseRate) : long_1.default.UZERO,
            description: isSet(object.description) ? String(object.description) : "",
            type: isSet(object.type) ? String(object.type) : "",
            annualDiscountPercentage: isSet(object.annualDiscountPercentage)
                ? long_1.default.fromValue(object.annualDiscountPercentage)
                : long_1.default.UZERO,
            planPolicy: isSet(object.planPolicy) ? project_1.Policy.fromJSON(object.planPolicy) : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = message.index);
        message.block !== undefined && (obj.block = (message.block || long_1.default.UZERO).toString());
        message.price !== undefined && (obj.price = message.price ? coin_1.Coin.toJSON(message.price) : undefined);
        message.allowOveruse !== undefined && (obj.allowOveruse = message.allowOveruse);
        message.overuseRate !== undefined && (obj.overuseRate = (message.overuseRate || long_1.default.UZERO).toString());
        message.description !== undefined && (obj.description = message.description);
        message.type !== undefined && (obj.type = message.type);
        message.annualDiscountPercentage !== undefined &&
            (obj.annualDiscountPercentage = (message.annualDiscountPercentage || long_1.default.UZERO).toString());
        message.planPolicy !== undefined &&
            (obj.planPolicy = message.planPolicy ? project_1.Policy.toJSON(message.planPolicy) : undefined);
        return obj;
    },
    create(base) {
        return exports.Plan.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d;
        const message = createBasePlan();
        message.index = (_a = object.index) !== null && _a !== void 0 ? _a : "";
        message.block = (object.block !== undefined && object.block !== null) ? long_1.default.fromValue(object.block) : long_1.default.UZERO;
        message.price = (object.price !== undefined && object.price !== null) ? coin_1.Coin.fromPartial(object.price) : undefined;
        message.allowOveruse = (_b = object.allowOveruse) !== null && _b !== void 0 ? _b : false;
        message.overuseRate = (object.overuseRate !== undefined && object.overuseRate !== null)
            ? long_1.default.fromValue(object.overuseRate)
            : long_1.default.UZERO;
        message.description = (_c = object.description) !== null && _c !== void 0 ? _c : "";
        message.type = (_d = object.type) !== null && _d !== void 0 ? _d : "";
        message.annualDiscountPercentage =
            (object.annualDiscountPercentage !== undefined && object.annualDiscountPercentage !== null)
                ? long_1.default.fromValue(object.annualDiscountPercentage)
                : long_1.default.UZERO;
        message.planPolicy = (object.planPolicy !== undefined && object.planPolicy !== null)
            ? project_1.Policy.fromPartial(object.planPolicy)
            : undefined;
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

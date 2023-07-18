"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Params = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
exports.protobufPackage = "lavanet.lava.pairing";
function createBaseParams() {
    return {
        mintCoinsPerCU: "",
        fraudStakeSlashingFactor: "",
        fraudSlashingAmount: long_1.default.UZERO,
        epochBlocksOverlap: long_1.default.UZERO,
        unpayLimit: "",
        slashLimit: "",
        dataReliabilityReward: "",
        QoSWeight: "",
        recommendedEpochNumToCollectPayment: long_1.default.UZERO,
    };
}
exports.Params = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.mintCoinsPerCU !== "") {
            writer.uint32(26).string(message.mintCoinsPerCU);
        }
        if (message.fraudStakeSlashingFactor !== "") {
            writer.uint32(42).string(message.fraudStakeSlashingFactor);
        }
        if (!message.fraudSlashingAmount.isZero()) {
            writer.uint32(48).uint64(message.fraudSlashingAmount);
        }
        if (!message.epochBlocksOverlap.isZero()) {
            writer.uint32(64).uint64(message.epochBlocksOverlap);
        }
        if (message.unpayLimit !== "") {
            writer.uint32(82).string(message.unpayLimit);
        }
        if (message.slashLimit !== "") {
            writer.uint32(90).string(message.slashLimit);
        }
        if (message.dataReliabilityReward !== "") {
            writer.uint32(98).string(message.dataReliabilityReward);
        }
        if (message.QoSWeight !== "") {
            writer.uint32(106).string(message.QoSWeight);
        }
        if (!message.recommendedEpochNumToCollectPayment.isZero()) {
            writer.uint32(112).uint64(message.recommendedEpochNumToCollectPayment);
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
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.mintCoinsPerCU = reader.string();
                    continue;
                case 5:
                    if (tag != 42) {
                        break;
                    }
                    message.fraudStakeSlashingFactor = reader.string();
                    continue;
                case 6:
                    if (tag != 48) {
                        break;
                    }
                    message.fraudSlashingAmount = reader.uint64();
                    continue;
                case 8:
                    if (tag != 64) {
                        break;
                    }
                    message.epochBlocksOverlap = reader.uint64();
                    continue;
                case 10:
                    if (tag != 82) {
                        break;
                    }
                    message.unpayLimit = reader.string();
                    continue;
                case 11:
                    if (tag != 90) {
                        break;
                    }
                    message.slashLimit = reader.string();
                    continue;
                case 12:
                    if (tag != 98) {
                        break;
                    }
                    message.dataReliabilityReward = reader.string();
                    continue;
                case 13:
                    if (tag != 106) {
                        break;
                    }
                    message.QoSWeight = reader.string();
                    continue;
                case 14:
                    if (tag != 112) {
                        break;
                    }
                    message.recommendedEpochNumToCollectPayment = reader.uint64();
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
            mintCoinsPerCU: isSet(object.mintCoinsPerCU) ? String(object.mintCoinsPerCU) : "",
            fraudStakeSlashingFactor: isSet(object.fraudStakeSlashingFactor) ? String(object.fraudStakeSlashingFactor) : "",
            fraudSlashingAmount: isSet(object.fraudSlashingAmount) ? long_1.default.fromValue(object.fraudSlashingAmount) : long_1.default.UZERO,
            epochBlocksOverlap: isSet(object.epochBlocksOverlap) ? long_1.default.fromValue(object.epochBlocksOverlap) : long_1.default.UZERO,
            unpayLimit: isSet(object.unpayLimit) ? String(object.unpayLimit) : "",
            slashLimit: isSet(object.slashLimit) ? String(object.slashLimit) : "",
            dataReliabilityReward: isSet(object.dataReliabilityReward) ? String(object.dataReliabilityReward) : "",
            QoSWeight: isSet(object.QoSWeight) ? String(object.QoSWeight) : "",
            recommendedEpochNumToCollectPayment: isSet(object.recommendedEpochNumToCollectPayment)
                ? long_1.default.fromValue(object.recommendedEpochNumToCollectPayment)
                : long_1.default.UZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.mintCoinsPerCU !== undefined && (obj.mintCoinsPerCU = message.mintCoinsPerCU);
        message.fraudStakeSlashingFactor !== undefined && (obj.fraudStakeSlashingFactor = message.fraudStakeSlashingFactor);
        message.fraudSlashingAmount !== undefined &&
            (obj.fraudSlashingAmount = (message.fraudSlashingAmount || long_1.default.UZERO).toString());
        message.epochBlocksOverlap !== undefined &&
            (obj.epochBlocksOverlap = (message.epochBlocksOverlap || long_1.default.UZERO).toString());
        message.unpayLimit !== undefined && (obj.unpayLimit = message.unpayLimit);
        message.slashLimit !== undefined && (obj.slashLimit = message.slashLimit);
        message.dataReliabilityReward !== undefined && (obj.dataReliabilityReward = message.dataReliabilityReward);
        message.QoSWeight !== undefined && (obj.QoSWeight = message.QoSWeight);
        message.recommendedEpochNumToCollectPayment !== undefined &&
            (obj.recommendedEpochNumToCollectPayment = (message.recommendedEpochNumToCollectPayment || long_1.default.UZERO)
                .toString());
        return obj;
    },
    create(base) {
        return exports.Params.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d, _e, _f;
        const message = createBaseParams();
        message.mintCoinsPerCU = (_a = object.mintCoinsPerCU) !== null && _a !== void 0 ? _a : "";
        message.fraudStakeSlashingFactor = (_b = object.fraudStakeSlashingFactor) !== null && _b !== void 0 ? _b : "";
        message.fraudSlashingAmount = (object.fraudSlashingAmount !== undefined && object.fraudSlashingAmount !== null)
            ? long_1.default.fromValue(object.fraudSlashingAmount)
            : long_1.default.UZERO;
        message.epochBlocksOverlap = (object.epochBlocksOverlap !== undefined && object.epochBlocksOverlap !== null)
            ? long_1.default.fromValue(object.epochBlocksOverlap)
            : long_1.default.UZERO;
        message.unpayLimit = (_c = object.unpayLimit) !== null && _c !== void 0 ? _c : "";
        message.slashLimit = (_d = object.slashLimit) !== null && _d !== void 0 ? _d : "";
        message.dataReliabilityReward = (_e = object.dataReliabilityReward) !== null && _e !== void 0 ? _e : "";
        message.QoSWeight = (_f = object.QoSWeight) !== null && _f !== void 0 ? _f : "";
        message.recommendedEpochNumToCollectPayment =
            (object.recommendedEpochNumToCollectPayment !== undefined && object.recommendedEpochNumToCollectPayment !== null)
                ? long_1.default.fromValue(object.recommendedEpochNumToCollectPayment)
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

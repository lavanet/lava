"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Rewards = exports.Params = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
exports.protobufPackage = "lavanet.lava.conflict";
function createBaseParams() {
    return { majorityPercent: "", voteStartSpan: long_1.default.UZERO, votePeriod: long_1.default.UZERO, Rewards: undefined };
}
exports.Params = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.majorityPercent !== "") {
            writer.uint32(10).string(message.majorityPercent);
        }
        if (!message.voteStartSpan.isZero()) {
            writer.uint32(16).uint64(message.voteStartSpan);
        }
        if (!message.votePeriod.isZero()) {
            writer.uint32(24).uint64(message.votePeriod);
        }
        if (message.Rewards !== undefined) {
            exports.Rewards.encode(message.Rewards, writer.uint32(34).fork()).ldelim();
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
                    if (tag != 10) {
                        break;
                    }
                    message.majorityPercent = reader.string();
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.voteStartSpan = reader.uint64();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.votePeriod = reader.uint64();
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.Rewards = exports.Rewards.decode(reader, reader.uint32());
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
            majorityPercent: isSet(object.majorityPercent) ? String(object.majorityPercent) : "",
            voteStartSpan: isSet(object.voteStartSpan) ? long_1.default.fromValue(object.voteStartSpan) : long_1.default.UZERO,
            votePeriod: isSet(object.votePeriod) ? long_1.default.fromValue(object.votePeriod) : long_1.default.UZERO,
            Rewards: isSet(object.Rewards) ? exports.Rewards.fromJSON(object.Rewards) : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.majorityPercent !== undefined && (obj.majorityPercent = message.majorityPercent);
        message.voteStartSpan !== undefined && (obj.voteStartSpan = (message.voteStartSpan || long_1.default.UZERO).toString());
        message.votePeriod !== undefined && (obj.votePeriod = (message.votePeriod || long_1.default.UZERO).toString());
        message.Rewards !== undefined && (obj.Rewards = message.Rewards ? exports.Rewards.toJSON(message.Rewards) : undefined);
        return obj;
    },
    create(base) {
        return exports.Params.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseParams();
        message.majorityPercent = (_a = object.majorityPercent) !== null && _a !== void 0 ? _a : "";
        message.voteStartSpan = (object.voteStartSpan !== undefined && object.voteStartSpan !== null)
            ? long_1.default.fromValue(object.voteStartSpan)
            : long_1.default.UZERO;
        message.votePeriod = (object.votePeriod !== undefined && object.votePeriod !== null)
            ? long_1.default.fromValue(object.votePeriod)
            : long_1.default.UZERO;
        message.Rewards = (object.Rewards !== undefined && object.Rewards !== null)
            ? exports.Rewards.fromPartial(object.Rewards)
            : undefined;
        return message;
    },
};
function createBaseRewards() {
    return { winnerRewardPercent: "", clientRewardPercent: "", votersRewardPercent: "" };
}
exports.Rewards = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.winnerRewardPercent !== "") {
            writer.uint32(10).string(message.winnerRewardPercent);
        }
        if (message.clientRewardPercent !== "") {
            writer.uint32(18).string(message.clientRewardPercent);
        }
        if (message.votersRewardPercent !== "") {
            writer.uint32(26).string(message.votersRewardPercent);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseRewards();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.winnerRewardPercent = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.clientRewardPercent = reader.string();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.votersRewardPercent = reader.string();
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
            winnerRewardPercent: isSet(object.winnerRewardPercent) ? String(object.winnerRewardPercent) : "",
            clientRewardPercent: isSet(object.clientRewardPercent) ? String(object.clientRewardPercent) : "",
            votersRewardPercent: isSet(object.votersRewardPercent) ? String(object.votersRewardPercent) : "",
        };
    },
    toJSON(message) {
        const obj = {};
        message.winnerRewardPercent !== undefined && (obj.winnerRewardPercent = message.winnerRewardPercent);
        message.clientRewardPercent !== undefined && (obj.clientRewardPercent = message.clientRewardPercent);
        message.votersRewardPercent !== undefined && (obj.votersRewardPercent = message.votersRewardPercent);
        return obj;
    },
    create(base) {
        return exports.Rewards.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBaseRewards();
        message.winnerRewardPercent = (_a = object.winnerRewardPercent) !== null && _a !== void 0 ? _a : "";
        message.clientRewardPercent = (_b = object.clientRewardPercent) !== null && _b !== void 0 ? _b : "";
        message.votersRewardPercent = (_c = object.votersRewardPercent) !== null && _c !== void 0 ? _c : "";
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

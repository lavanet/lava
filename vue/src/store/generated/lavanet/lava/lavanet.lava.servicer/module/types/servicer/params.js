/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";
export const protobufPackage = "lavanet.lava.servicer";
const baseParams = {
    minStake: 0,
    coinsPerCU: 0,
    unstakeHoldBlocks: 0,
    fraudStakeSlashingFactor: 0,
    fraudSlashingAmount: 0,
    servicersToPairCount: 0,
    sessionBlocks: 0,
    sessionsToSave: 0,
    sessionBlocksOverlap: 0,
};
export const Params = {
    encode(message, writer = Writer.create()) {
        if (message.minStake !== 0) {
            writer.uint32(8).uint64(message.minStake);
        }
        if (message.coinsPerCU !== 0) {
            writer.uint32(16).uint64(message.coinsPerCU);
        }
        if (message.unstakeHoldBlocks !== 0) {
            writer.uint32(24).uint64(message.unstakeHoldBlocks);
        }
        if (message.fraudStakeSlashingFactor !== 0) {
            writer.uint32(32).uint64(message.fraudStakeSlashingFactor);
        }
        if (message.fraudSlashingAmount !== 0) {
            writer.uint32(40).uint64(message.fraudSlashingAmount);
        }
        if (message.servicersToPairCount !== 0) {
            writer.uint32(48).uint64(message.servicersToPairCount);
        }
        if (message.sessionBlocks !== 0) {
            writer.uint32(56).uint64(message.sessionBlocks);
        }
        if (message.sessionsToSave !== 0) {
            writer.uint32(64).uint64(message.sessionsToSave);
        }
        if (message.sessionBlocksOverlap !== 0) {
            writer.uint32(72).uint64(message.sessionBlocksOverlap);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseParams };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.minStake = longToNumber(reader.uint64());
                    break;
                case 2:
                    message.coinsPerCU = longToNumber(reader.uint64());
                    break;
                case 3:
                    message.unstakeHoldBlocks = longToNumber(reader.uint64());
                    break;
                case 4:
                    message.fraudStakeSlashingFactor = longToNumber(reader.uint64());
                    break;
                case 5:
                    message.fraudSlashingAmount = longToNumber(reader.uint64());
                    break;
                case 6:
                    message.servicersToPairCount = longToNumber(reader.uint64());
                    break;
                case 7:
                    message.sessionBlocks = longToNumber(reader.uint64());
                    break;
                case 8:
                    message.sessionsToSave = longToNumber(reader.uint64());
                    break;
                case 9:
                    message.sessionBlocksOverlap = longToNumber(reader.uint64());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = { ...baseParams };
        if (object.minStake !== undefined && object.minStake !== null) {
            message.minStake = Number(object.minStake);
        }
        else {
            message.minStake = 0;
        }
        if (object.coinsPerCU !== undefined && object.coinsPerCU !== null) {
            message.coinsPerCU = Number(object.coinsPerCU);
        }
        else {
            message.coinsPerCU = 0;
        }
        if (object.unstakeHoldBlocks !== undefined &&
            object.unstakeHoldBlocks !== null) {
            message.unstakeHoldBlocks = Number(object.unstakeHoldBlocks);
        }
        else {
            message.unstakeHoldBlocks = 0;
        }
        if (object.fraudStakeSlashingFactor !== undefined &&
            object.fraudStakeSlashingFactor !== null) {
            message.fraudStakeSlashingFactor = Number(object.fraudStakeSlashingFactor);
        }
        else {
            message.fraudStakeSlashingFactor = 0;
        }
        if (object.fraudSlashingAmount !== undefined &&
            object.fraudSlashingAmount !== null) {
            message.fraudSlashingAmount = Number(object.fraudSlashingAmount);
        }
        else {
            message.fraudSlashingAmount = 0;
        }
        if (object.servicersToPairCount !== undefined &&
            object.servicersToPairCount !== null) {
            message.servicersToPairCount = Number(object.servicersToPairCount);
        }
        else {
            message.servicersToPairCount = 0;
        }
        if (object.sessionBlocks !== undefined && object.sessionBlocks !== null) {
            message.sessionBlocks = Number(object.sessionBlocks);
        }
        else {
            message.sessionBlocks = 0;
        }
        if (object.sessionsToSave !== undefined && object.sessionsToSave !== null) {
            message.sessionsToSave = Number(object.sessionsToSave);
        }
        else {
            message.sessionsToSave = 0;
        }
        if (object.sessionBlocksOverlap !== undefined &&
            object.sessionBlocksOverlap !== null) {
            message.sessionBlocksOverlap = Number(object.sessionBlocksOverlap);
        }
        else {
            message.sessionBlocksOverlap = 0;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.minStake !== undefined && (obj.minStake = message.minStake);
        message.coinsPerCU !== undefined && (obj.coinsPerCU = message.coinsPerCU);
        message.unstakeHoldBlocks !== undefined &&
            (obj.unstakeHoldBlocks = message.unstakeHoldBlocks);
        message.fraudStakeSlashingFactor !== undefined &&
            (obj.fraudStakeSlashingFactor = message.fraudStakeSlashingFactor);
        message.fraudSlashingAmount !== undefined &&
            (obj.fraudSlashingAmount = message.fraudSlashingAmount);
        message.servicersToPairCount !== undefined &&
            (obj.servicersToPairCount = message.servicersToPairCount);
        message.sessionBlocks !== undefined &&
            (obj.sessionBlocks = message.sessionBlocks);
        message.sessionsToSave !== undefined &&
            (obj.sessionsToSave = message.sessionsToSave);
        message.sessionBlocksOverlap !== undefined &&
            (obj.sessionBlocksOverlap = message.sessionBlocksOverlap);
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseParams };
        if (object.minStake !== undefined && object.minStake !== null) {
            message.minStake = object.minStake;
        }
        else {
            message.minStake = 0;
        }
        if (object.coinsPerCU !== undefined && object.coinsPerCU !== null) {
            message.coinsPerCU = object.coinsPerCU;
        }
        else {
            message.coinsPerCU = 0;
        }
        if (object.unstakeHoldBlocks !== undefined &&
            object.unstakeHoldBlocks !== null) {
            message.unstakeHoldBlocks = object.unstakeHoldBlocks;
        }
        else {
            message.unstakeHoldBlocks = 0;
        }
        if (object.fraudStakeSlashingFactor !== undefined &&
            object.fraudStakeSlashingFactor !== null) {
            message.fraudStakeSlashingFactor = object.fraudStakeSlashingFactor;
        }
        else {
            message.fraudStakeSlashingFactor = 0;
        }
        if (object.fraudSlashingAmount !== undefined &&
            object.fraudSlashingAmount !== null) {
            message.fraudSlashingAmount = object.fraudSlashingAmount;
        }
        else {
            message.fraudSlashingAmount = 0;
        }
        if (object.servicersToPairCount !== undefined &&
            object.servicersToPairCount !== null) {
            message.servicersToPairCount = object.servicersToPairCount;
        }
        else {
            message.servicersToPairCount = 0;
        }
        if (object.sessionBlocks !== undefined && object.sessionBlocks !== null) {
            message.sessionBlocks = object.sessionBlocks;
        }
        else {
            message.sessionBlocks = 0;
        }
        if (object.sessionsToSave !== undefined && object.sessionsToSave !== null) {
            message.sessionsToSave = object.sessionsToSave;
        }
        else {
            message.sessionsToSave = 0;
        }
        if (object.sessionBlocksOverlap !== undefined &&
            object.sessionBlocksOverlap !== null) {
            message.sessionBlocksOverlap = object.sessionBlocksOverlap;
        }
        else {
            message.sessionBlocksOverlap = 0;
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

/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";
import { Params } from "../servicer/params";
import { StakeMap } from "../servicer/stake_map";
import { SpecStakeStorage } from "../servicer/spec_stake_storage";
import { BlockDeadlineForCallback } from "../servicer/block_deadline_for_callback";
import { UnstakingServicersAllSpecs } from "../servicer/unstaking_servicers_all_specs";
import { CurrentSessionStart } from "../servicer/current_session_start";
import { PreviousSessionBlocks } from "../servicer/previous_session_blocks";
import { SessionStorageForSpec } from "../servicer/session_storage_for_spec";
export const protobufPackage = "lavanet.lava.servicer";
const baseGenesisState = { unstakingServicersAllSpecsCount: 0 };
export const GenesisState = {
    encode(message, writer = Writer.create()) {
        if (message.params !== undefined) {
            Params.encode(message.params, writer.uint32(10).fork()).ldelim();
        }
        for (const v of message.stakeMapList) {
            StakeMap.encode(v, writer.uint32(18).fork()).ldelim();
        }
        for (const v of message.specStakeStorageList) {
            SpecStakeStorage.encode(v, writer.uint32(26).fork()).ldelim();
        }
        if (message.blockDeadlineForCallback !== undefined) {
            BlockDeadlineForCallback.encode(message.blockDeadlineForCallback, writer.uint32(34).fork()).ldelim();
        }
        for (const v of message.unstakingServicersAllSpecsList) {
            UnstakingServicersAllSpecs.encode(v, writer.uint32(42).fork()).ldelim();
        }
        if (message.unstakingServicersAllSpecsCount !== 0) {
            writer.uint32(48).uint64(message.unstakingServicersAllSpecsCount);
        }
        if (message.currentSessionStart !== undefined) {
            CurrentSessionStart.encode(message.currentSessionStart, writer.uint32(58).fork()).ldelim();
        }
        if (message.previousSessionBlocks !== undefined) {
            PreviousSessionBlocks.encode(message.previousSessionBlocks, writer.uint32(66).fork()).ldelim();
        }
        for (const v of message.sessionStorageForSpecList) {
            SessionStorageForSpec.encode(v, writer.uint32(74).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseGenesisState };
        message.stakeMapList = [];
        message.specStakeStorageList = [];
        message.unstakingServicersAllSpecsList = [];
        message.sessionStorageForSpecList = [];
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.params = Params.decode(reader, reader.uint32());
                    break;
                case 2:
                    message.stakeMapList.push(StakeMap.decode(reader, reader.uint32()));
                    break;
                case 3:
                    message.specStakeStorageList.push(SpecStakeStorage.decode(reader, reader.uint32()));
                    break;
                case 4:
                    message.blockDeadlineForCallback = BlockDeadlineForCallback.decode(reader, reader.uint32());
                    break;
                case 5:
                    message.unstakingServicersAllSpecsList.push(UnstakingServicersAllSpecs.decode(reader, reader.uint32()));
                    break;
                case 6:
                    message.unstakingServicersAllSpecsCount = longToNumber(reader.uint64());
                    break;
                case 7:
                    message.currentSessionStart = CurrentSessionStart.decode(reader, reader.uint32());
                    break;
                case 8:
                    message.previousSessionBlocks = PreviousSessionBlocks.decode(reader, reader.uint32());
                    break;
                case 9:
                    message.sessionStorageForSpecList.push(SessionStorageForSpec.decode(reader, reader.uint32()));
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = { ...baseGenesisState };
        message.stakeMapList = [];
        message.specStakeStorageList = [];
        message.unstakingServicersAllSpecsList = [];
        message.sessionStorageForSpecList = [];
        if (object.params !== undefined && object.params !== null) {
            message.params = Params.fromJSON(object.params);
        }
        else {
            message.params = undefined;
        }
        if (object.stakeMapList !== undefined && object.stakeMapList !== null) {
            for (const e of object.stakeMapList) {
                message.stakeMapList.push(StakeMap.fromJSON(e));
            }
        }
        if (object.specStakeStorageList !== undefined &&
            object.specStakeStorageList !== null) {
            for (const e of object.specStakeStorageList) {
                message.specStakeStorageList.push(SpecStakeStorage.fromJSON(e));
            }
        }
        if (object.blockDeadlineForCallback !== undefined &&
            object.blockDeadlineForCallback !== null) {
            message.blockDeadlineForCallback = BlockDeadlineForCallback.fromJSON(object.blockDeadlineForCallback);
        }
        else {
            message.blockDeadlineForCallback = undefined;
        }
        if (object.unstakingServicersAllSpecsList !== undefined &&
            object.unstakingServicersAllSpecsList !== null) {
            for (const e of object.unstakingServicersAllSpecsList) {
                message.unstakingServicersAllSpecsList.push(UnstakingServicersAllSpecs.fromJSON(e));
            }
        }
        if (object.unstakingServicersAllSpecsCount !== undefined &&
            object.unstakingServicersAllSpecsCount !== null) {
            message.unstakingServicersAllSpecsCount = Number(object.unstakingServicersAllSpecsCount);
        }
        else {
            message.unstakingServicersAllSpecsCount = 0;
        }
        if (object.currentSessionStart !== undefined &&
            object.currentSessionStart !== null) {
            message.currentSessionStart = CurrentSessionStart.fromJSON(object.currentSessionStart);
        }
        else {
            message.currentSessionStart = undefined;
        }
        if (object.previousSessionBlocks !== undefined &&
            object.previousSessionBlocks !== null) {
            message.previousSessionBlocks = PreviousSessionBlocks.fromJSON(object.previousSessionBlocks);
        }
        else {
            message.previousSessionBlocks = undefined;
        }
        if (object.sessionStorageForSpecList !== undefined &&
            object.sessionStorageForSpecList !== null) {
            for (const e of object.sessionStorageForSpecList) {
                message.sessionStorageForSpecList.push(SessionStorageForSpec.fromJSON(e));
            }
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.params !== undefined &&
            (obj.params = message.params ? Params.toJSON(message.params) : undefined);
        if (message.stakeMapList) {
            obj.stakeMapList = message.stakeMapList.map((e) => e ? StakeMap.toJSON(e) : undefined);
        }
        else {
            obj.stakeMapList = [];
        }
        if (message.specStakeStorageList) {
            obj.specStakeStorageList = message.specStakeStorageList.map((e) => e ? SpecStakeStorage.toJSON(e) : undefined);
        }
        else {
            obj.specStakeStorageList = [];
        }
        message.blockDeadlineForCallback !== undefined &&
            (obj.blockDeadlineForCallback = message.blockDeadlineForCallback
                ? BlockDeadlineForCallback.toJSON(message.blockDeadlineForCallback)
                : undefined);
        if (message.unstakingServicersAllSpecsList) {
            obj.unstakingServicersAllSpecsList = message.unstakingServicersAllSpecsList.map((e) => (e ? UnstakingServicersAllSpecs.toJSON(e) : undefined));
        }
        else {
            obj.unstakingServicersAllSpecsList = [];
        }
        message.unstakingServicersAllSpecsCount !== undefined &&
            (obj.unstakingServicersAllSpecsCount =
                message.unstakingServicersAllSpecsCount);
        message.currentSessionStart !== undefined &&
            (obj.currentSessionStart = message.currentSessionStart
                ? CurrentSessionStart.toJSON(message.currentSessionStart)
                : undefined);
        message.previousSessionBlocks !== undefined &&
            (obj.previousSessionBlocks = message.previousSessionBlocks
                ? PreviousSessionBlocks.toJSON(message.previousSessionBlocks)
                : undefined);
        if (message.sessionStorageForSpecList) {
            obj.sessionStorageForSpecList = message.sessionStorageForSpecList.map((e) => (e ? SessionStorageForSpec.toJSON(e) : undefined));
        }
        else {
            obj.sessionStorageForSpecList = [];
        }
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseGenesisState };
        message.stakeMapList = [];
        message.specStakeStorageList = [];
        message.unstakingServicersAllSpecsList = [];
        message.sessionStorageForSpecList = [];
        if (object.params !== undefined && object.params !== null) {
            message.params = Params.fromPartial(object.params);
        }
        else {
            message.params = undefined;
        }
        if (object.stakeMapList !== undefined && object.stakeMapList !== null) {
            for (const e of object.stakeMapList) {
                message.stakeMapList.push(StakeMap.fromPartial(e));
            }
        }
        if (object.specStakeStorageList !== undefined &&
            object.specStakeStorageList !== null) {
            for (const e of object.specStakeStorageList) {
                message.specStakeStorageList.push(SpecStakeStorage.fromPartial(e));
            }
        }
        if (object.blockDeadlineForCallback !== undefined &&
            object.blockDeadlineForCallback !== null) {
            message.blockDeadlineForCallback = BlockDeadlineForCallback.fromPartial(object.blockDeadlineForCallback);
        }
        else {
            message.blockDeadlineForCallback = undefined;
        }
        if (object.unstakingServicersAllSpecsList !== undefined &&
            object.unstakingServicersAllSpecsList !== null) {
            for (const e of object.unstakingServicersAllSpecsList) {
                message.unstakingServicersAllSpecsList.push(UnstakingServicersAllSpecs.fromPartial(e));
            }
        }
        if (object.unstakingServicersAllSpecsCount !== undefined &&
            object.unstakingServicersAllSpecsCount !== null) {
            message.unstakingServicersAllSpecsCount =
                object.unstakingServicersAllSpecsCount;
        }
        else {
            message.unstakingServicersAllSpecsCount = 0;
        }
        if (object.currentSessionStart !== undefined &&
            object.currentSessionStart !== null) {
            message.currentSessionStart = CurrentSessionStart.fromPartial(object.currentSessionStart);
        }
        else {
            message.currentSessionStart = undefined;
        }
        if (object.previousSessionBlocks !== undefined &&
            object.previousSessionBlocks !== null) {
            message.previousSessionBlocks = PreviousSessionBlocks.fromPartial(object.previousSessionBlocks);
        }
        else {
            message.previousSessionBlocks = undefined;
        }
        if (object.sessionStorageForSpecList !== undefined &&
            object.sessionStorageForSpecList !== null) {
            for (const e of object.sessionStorageForSpecList) {
                message.sessionStorageForSpecList.push(SessionStorageForSpec.fromPartial(e));
            }
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

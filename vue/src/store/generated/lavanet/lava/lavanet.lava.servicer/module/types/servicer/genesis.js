/* eslint-disable */
import { Params } from "../servicer/params";
import { StakeMap } from "../servicer/stake_map";
import { SpecStakeStorage } from "../servicer/spec_stake_storage";
import { Writer, Reader } from "protobufjs/minimal";
export const protobufPackage = "lavanet.lava.servicer";
const baseGenesisState = {};
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
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseGenesisState };
        message.stakeMapList = [];
        message.specStakeStorageList = [];
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
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseGenesisState };
        message.stakeMapList = [];
        message.specStakeStorageList = [];
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
        return message;
    },
};

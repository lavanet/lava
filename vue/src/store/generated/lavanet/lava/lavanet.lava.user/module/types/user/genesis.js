/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";
import { Params } from "../user/params";
import { UserStake } from "../user/user_stake";
import { SpecStakeStorage } from "../user/spec_stake_storage";
import { BlockDeadlineForCallback } from "../user/block_deadline_for_callback";
import { UnstakingUsersAllSpecs } from "../user/unstaking_users_all_specs";
export const protobufPackage = "lavanet.lava.user";
const baseGenesisState = { unstakingUsersAllSpecsCount: 0 };
export const GenesisState = {
    encode(message, writer = Writer.create()) {
        if (message.params !== undefined) {
            Params.encode(message.params, writer.uint32(10).fork()).ldelim();
        }
        for (const v of message.userStakeList) {
            UserStake.encode(v, writer.uint32(18).fork()).ldelim();
        }
        for (const v of message.specStakeStorageList) {
            SpecStakeStorage.encode(v, writer.uint32(26).fork()).ldelim();
        }
        if (message.blockDeadlineForCallback !== undefined) {
            BlockDeadlineForCallback.encode(message.blockDeadlineForCallback, writer.uint32(34).fork()).ldelim();
        }
        for (const v of message.unstakingUsersAllSpecsList) {
            UnstakingUsersAllSpecs.encode(v, writer.uint32(42).fork()).ldelim();
        }
        if (message.unstakingUsersAllSpecsCount !== 0) {
            writer.uint32(48).uint64(message.unstakingUsersAllSpecsCount);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseGenesisState };
        message.userStakeList = [];
        message.specStakeStorageList = [];
        message.unstakingUsersAllSpecsList = [];
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.params = Params.decode(reader, reader.uint32());
                    break;
                case 2:
                    message.userStakeList.push(UserStake.decode(reader, reader.uint32()));
                    break;
                case 3:
                    message.specStakeStorageList.push(SpecStakeStorage.decode(reader, reader.uint32()));
                    break;
                case 4:
                    message.blockDeadlineForCallback = BlockDeadlineForCallback.decode(reader, reader.uint32());
                    break;
                case 5:
                    message.unstakingUsersAllSpecsList.push(UnstakingUsersAllSpecs.decode(reader, reader.uint32()));
                    break;
                case 6:
                    message.unstakingUsersAllSpecsCount = longToNumber(reader.uint64());
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
        message.userStakeList = [];
        message.specStakeStorageList = [];
        message.unstakingUsersAllSpecsList = [];
        if (object.params !== undefined && object.params !== null) {
            message.params = Params.fromJSON(object.params);
        }
        else {
            message.params = undefined;
        }
        if (object.userStakeList !== undefined && object.userStakeList !== null) {
            for (const e of object.userStakeList) {
                message.userStakeList.push(UserStake.fromJSON(e));
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
        if (object.unstakingUsersAllSpecsList !== undefined &&
            object.unstakingUsersAllSpecsList !== null) {
            for (const e of object.unstakingUsersAllSpecsList) {
                message.unstakingUsersAllSpecsList.push(UnstakingUsersAllSpecs.fromJSON(e));
            }
        }
        if (object.unstakingUsersAllSpecsCount !== undefined &&
            object.unstakingUsersAllSpecsCount !== null) {
            message.unstakingUsersAllSpecsCount = Number(object.unstakingUsersAllSpecsCount);
        }
        else {
            message.unstakingUsersAllSpecsCount = 0;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.params !== undefined &&
            (obj.params = message.params ? Params.toJSON(message.params) : undefined);
        if (message.userStakeList) {
            obj.userStakeList = message.userStakeList.map((e) => e ? UserStake.toJSON(e) : undefined);
        }
        else {
            obj.userStakeList = [];
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
        if (message.unstakingUsersAllSpecsList) {
            obj.unstakingUsersAllSpecsList = message.unstakingUsersAllSpecsList.map((e) => (e ? UnstakingUsersAllSpecs.toJSON(e) : undefined));
        }
        else {
            obj.unstakingUsersAllSpecsList = [];
        }
        message.unstakingUsersAllSpecsCount !== undefined &&
            (obj.unstakingUsersAllSpecsCount = message.unstakingUsersAllSpecsCount);
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseGenesisState };
        message.userStakeList = [];
        message.specStakeStorageList = [];
        message.unstakingUsersAllSpecsList = [];
        if (object.params !== undefined && object.params !== null) {
            message.params = Params.fromPartial(object.params);
        }
        else {
            message.params = undefined;
        }
        if (object.userStakeList !== undefined && object.userStakeList !== null) {
            for (const e of object.userStakeList) {
                message.userStakeList.push(UserStake.fromPartial(e));
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
        if (object.unstakingUsersAllSpecsList !== undefined &&
            object.unstakingUsersAllSpecsList !== null) {
            for (const e of object.unstakingUsersAllSpecsList) {
                message.unstakingUsersAllSpecsList.push(UnstakingUsersAllSpecs.fromPartial(e));
            }
        }
        if (object.unstakingUsersAllSpecsCount !== undefined &&
            object.unstakingUsersAllSpecsCount !== null) {
            message.unstakingUsersAllSpecsCount = object.unstakingUsersAllSpecsCount;
        }
        else {
            message.unstakingUsersAllSpecsCount = 0;
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

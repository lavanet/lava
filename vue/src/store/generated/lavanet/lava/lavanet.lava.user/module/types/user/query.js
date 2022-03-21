/* eslint-disable */
import { Reader, util, configure, Writer } from "protobufjs/minimal";
import * as Long from "long";
import { Params } from "../user/params";
import { UserStake } from "../user/user_stake";
import { PageRequest, PageResponse, } from "../cosmos/base/query/v1beta1/pagination";
import { SpecStakeStorage } from "../user/spec_stake_storage";
import { BlockDeadlineForCallback } from "../user/block_deadline_for_callback";
import { UnstakingUsersAllSpecs } from "../user/unstaking_users_all_specs";
import { StakeStorage } from "../user/stake_storage";
export const protobufPackage = "lavanet.lava.user";
const baseQueryParamsRequest = {};
export const QueryParamsRequest = {
    encode(_, writer = Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseQueryParamsRequest };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(_) {
        const message = { ...baseQueryParamsRequest };
        return message;
    },
    toJSON(_) {
        const obj = {};
        return obj;
    },
    fromPartial(_) {
        const message = { ...baseQueryParamsRequest };
        return message;
    },
};
const baseQueryParamsResponse = {};
export const QueryParamsResponse = {
    encode(message, writer = Writer.create()) {
        if (message.params !== undefined) {
            Params.encode(message.params, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseQueryParamsResponse };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.params = Params.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = { ...baseQueryParamsResponse };
        if (object.params !== undefined && object.params !== null) {
            message.params = Params.fromJSON(object.params);
        }
        else {
            message.params = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.params !== undefined &&
            (obj.params = message.params ? Params.toJSON(message.params) : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseQueryParamsResponse };
        if (object.params !== undefined && object.params !== null) {
            message.params = Params.fromPartial(object.params);
        }
        else {
            message.params = undefined;
        }
        return message;
    },
};
const baseQueryGetUserStakeRequest = { index: "" };
export const QueryGetUserStakeRequest = {
    encode(message, writer = Writer.create()) {
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetUserStakeRequest,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.index = reader.string();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryGetUserStakeRequest,
        };
        if (object.index !== undefined && object.index !== null) {
            message.index = String(object.index);
        }
        else {
            message.index = "";
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = message.index);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryGetUserStakeRequest,
        };
        if (object.index !== undefined && object.index !== null) {
            message.index = object.index;
        }
        else {
            message.index = "";
        }
        return message;
    },
};
const baseQueryGetUserStakeResponse = {};
export const QueryGetUserStakeResponse = {
    encode(message, writer = Writer.create()) {
        if (message.userStake !== undefined) {
            UserStake.encode(message.userStake, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetUserStakeResponse,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.userStake = UserStake.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryGetUserStakeResponse,
        };
        if (object.userStake !== undefined && object.userStake !== null) {
            message.userStake = UserStake.fromJSON(object.userStake);
        }
        else {
            message.userStake = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.userStake !== undefined &&
            (obj.userStake = message.userStake
                ? UserStake.toJSON(message.userStake)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryGetUserStakeResponse,
        };
        if (object.userStake !== undefined && object.userStake !== null) {
            message.userStake = UserStake.fromPartial(object.userStake);
        }
        else {
            message.userStake = undefined;
        }
        return message;
    },
};
const baseQueryAllUserStakeRequest = {};
export const QueryAllUserStakeRequest = {
    encode(message, writer = Writer.create()) {
        if (message.pagination !== undefined) {
            PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryAllUserStakeRequest,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.pagination = PageRequest.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryAllUserStakeRequest,
        };
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageRequest.fromJSON(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? PageRequest.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryAllUserStakeRequest,
        };
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageRequest.fromPartial(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
};
const baseQueryAllUserStakeResponse = {};
export const QueryAllUserStakeResponse = {
    encode(message, writer = Writer.create()) {
        for (const v of message.userStake) {
            UserStake.encode(v, writer.uint32(10).fork()).ldelim();
        }
        if (message.pagination !== undefined) {
            PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryAllUserStakeResponse,
        };
        message.userStake = [];
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.userStake.push(UserStake.decode(reader, reader.uint32()));
                    break;
                case 2:
                    message.pagination = PageResponse.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryAllUserStakeResponse,
        };
        message.userStake = [];
        if (object.userStake !== undefined && object.userStake !== null) {
            for (const e of object.userStake) {
                message.userStake.push(UserStake.fromJSON(e));
            }
        }
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageResponse.fromJSON(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        if (message.userStake) {
            obj.userStake = message.userStake.map((e) => e ? UserStake.toJSON(e) : undefined);
        }
        else {
            obj.userStake = [];
        }
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? PageResponse.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryAllUserStakeResponse,
        };
        message.userStake = [];
        if (object.userStake !== undefined && object.userStake !== null) {
            for (const e of object.userStake) {
                message.userStake.push(UserStake.fromPartial(e));
            }
        }
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageResponse.fromPartial(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
};
const baseQueryGetSpecStakeStorageRequest = { index: "" };
export const QueryGetSpecStakeStorageRequest = {
    encode(message, writer = Writer.create()) {
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetSpecStakeStorageRequest,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.index = reader.string();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryGetSpecStakeStorageRequest,
        };
        if (object.index !== undefined && object.index !== null) {
            message.index = String(object.index);
        }
        else {
            message.index = "";
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = message.index);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryGetSpecStakeStorageRequest,
        };
        if (object.index !== undefined && object.index !== null) {
            message.index = object.index;
        }
        else {
            message.index = "";
        }
        return message;
    },
};
const baseQueryGetSpecStakeStorageResponse = {};
export const QueryGetSpecStakeStorageResponse = {
    encode(message, writer = Writer.create()) {
        if (message.specStakeStorage !== undefined) {
            SpecStakeStorage.encode(message.specStakeStorage, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetSpecStakeStorageResponse,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.specStakeStorage = SpecStakeStorage.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryGetSpecStakeStorageResponse,
        };
        if (object.specStakeStorage !== undefined &&
            object.specStakeStorage !== null) {
            message.specStakeStorage = SpecStakeStorage.fromJSON(object.specStakeStorage);
        }
        else {
            message.specStakeStorage = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.specStakeStorage !== undefined &&
            (obj.specStakeStorage = message.specStakeStorage
                ? SpecStakeStorage.toJSON(message.specStakeStorage)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryGetSpecStakeStorageResponse,
        };
        if (object.specStakeStorage !== undefined &&
            object.specStakeStorage !== null) {
            message.specStakeStorage = SpecStakeStorage.fromPartial(object.specStakeStorage);
        }
        else {
            message.specStakeStorage = undefined;
        }
        return message;
    },
};
const baseQueryAllSpecStakeStorageRequest = {};
export const QueryAllSpecStakeStorageRequest = {
    encode(message, writer = Writer.create()) {
        if (message.pagination !== undefined) {
            PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryAllSpecStakeStorageRequest,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.pagination = PageRequest.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryAllSpecStakeStorageRequest,
        };
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageRequest.fromJSON(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? PageRequest.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryAllSpecStakeStorageRequest,
        };
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageRequest.fromPartial(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
};
const baseQueryAllSpecStakeStorageResponse = {};
export const QueryAllSpecStakeStorageResponse = {
    encode(message, writer = Writer.create()) {
        for (const v of message.specStakeStorage) {
            SpecStakeStorage.encode(v, writer.uint32(10).fork()).ldelim();
        }
        if (message.pagination !== undefined) {
            PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryAllSpecStakeStorageResponse,
        };
        message.specStakeStorage = [];
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.specStakeStorage.push(SpecStakeStorage.decode(reader, reader.uint32()));
                    break;
                case 2:
                    message.pagination = PageResponse.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryAllSpecStakeStorageResponse,
        };
        message.specStakeStorage = [];
        if (object.specStakeStorage !== undefined &&
            object.specStakeStorage !== null) {
            for (const e of object.specStakeStorage) {
                message.specStakeStorage.push(SpecStakeStorage.fromJSON(e));
            }
        }
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageResponse.fromJSON(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        if (message.specStakeStorage) {
            obj.specStakeStorage = message.specStakeStorage.map((e) => e ? SpecStakeStorage.toJSON(e) : undefined);
        }
        else {
            obj.specStakeStorage = [];
        }
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? PageResponse.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryAllSpecStakeStorageResponse,
        };
        message.specStakeStorage = [];
        if (object.specStakeStorage !== undefined &&
            object.specStakeStorage !== null) {
            for (const e of object.specStakeStorage) {
                message.specStakeStorage.push(SpecStakeStorage.fromPartial(e));
            }
        }
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageResponse.fromPartial(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
};
const baseQueryGetBlockDeadlineForCallbackRequest = {};
export const QueryGetBlockDeadlineForCallbackRequest = {
    encode(_, writer = Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetBlockDeadlineForCallbackRequest,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(_) {
        const message = {
            ...baseQueryGetBlockDeadlineForCallbackRequest,
        };
        return message;
    },
    toJSON(_) {
        const obj = {};
        return obj;
    },
    fromPartial(_) {
        const message = {
            ...baseQueryGetBlockDeadlineForCallbackRequest,
        };
        return message;
    },
};
const baseQueryGetBlockDeadlineForCallbackResponse = {};
export const QueryGetBlockDeadlineForCallbackResponse = {
    encode(message, writer = Writer.create()) {
        if (message.BlockDeadlineForCallback !== undefined) {
            BlockDeadlineForCallback.encode(message.BlockDeadlineForCallback, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetBlockDeadlineForCallbackResponse,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.BlockDeadlineForCallback = BlockDeadlineForCallback.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryGetBlockDeadlineForCallbackResponse,
        };
        if (object.BlockDeadlineForCallback !== undefined &&
            object.BlockDeadlineForCallback !== null) {
            message.BlockDeadlineForCallback = BlockDeadlineForCallback.fromJSON(object.BlockDeadlineForCallback);
        }
        else {
            message.BlockDeadlineForCallback = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.BlockDeadlineForCallback !== undefined &&
            (obj.BlockDeadlineForCallback = message.BlockDeadlineForCallback
                ? BlockDeadlineForCallback.toJSON(message.BlockDeadlineForCallback)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryGetBlockDeadlineForCallbackResponse,
        };
        if (object.BlockDeadlineForCallback !== undefined &&
            object.BlockDeadlineForCallback !== null) {
            message.BlockDeadlineForCallback = BlockDeadlineForCallback.fromPartial(object.BlockDeadlineForCallback);
        }
        else {
            message.BlockDeadlineForCallback = undefined;
        }
        return message;
    },
};
const baseQueryGetUnstakingUsersAllSpecsRequest = { id: 0 };
export const QueryGetUnstakingUsersAllSpecsRequest = {
    encode(message, writer = Writer.create()) {
        if (message.id !== 0) {
            writer.uint32(8).uint64(message.id);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetUnstakingUsersAllSpecsRequest,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.id = longToNumber(reader.uint64());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryGetUnstakingUsersAllSpecsRequest,
        };
        if (object.id !== undefined && object.id !== null) {
            message.id = Number(object.id);
        }
        else {
            message.id = 0;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.id !== undefined && (obj.id = message.id);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryGetUnstakingUsersAllSpecsRequest,
        };
        if (object.id !== undefined && object.id !== null) {
            message.id = object.id;
        }
        else {
            message.id = 0;
        }
        return message;
    },
};
const baseQueryGetUnstakingUsersAllSpecsResponse = {};
export const QueryGetUnstakingUsersAllSpecsResponse = {
    encode(message, writer = Writer.create()) {
        if (message.UnstakingUsersAllSpecs !== undefined) {
            UnstakingUsersAllSpecs.encode(message.UnstakingUsersAllSpecs, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetUnstakingUsersAllSpecsResponse,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.UnstakingUsersAllSpecs = UnstakingUsersAllSpecs.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryGetUnstakingUsersAllSpecsResponse,
        };
        if (object.UnstakingUsersAllSpecs !== undefined &&
            object.UnstakingUsersAllSpecs !== null) {
            message.UnstakingUsersAllSpecs = UnstakingUsersAllSpecs.fromJSON(object.UnstakingUsersAllSpecs);
        }
        else {
            message.UnstakingUsersAllSpecs = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.UnstakingUsersAllSpecs !== undefined &&
            (obj.UnstakingUsersAllSpecs = message.UnstakingUsersAllSpecs
                ? UnstakingUsersAllSpecs.toJSON(message.UnstakingUsersAllSpecs)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryGetUnstakingUsersAllSpecsResponse,
        };
        if (object.UnstakingUsersAllSpecs !== undefined &&
            object.UnstakingUsersAllSpecs !== null) {
            message.UnstakingUsersAllSpecs = UnstakingUsersAllSpecs.fromPartial(object.UnstakingUsersAllSpecs);
        }
        else {
            message.UnstakingUsersAllSpecs = undefined;
        }
        return message;
    },
};
const baseQueryAllUnstakingUsersAllSpecsRequest = {};
export const QueryAllUnstakingUsersAllSpecsRequest = {
    encode(message, writer = Writer.create()) {
        if (message.pagination !== undefined) {
            PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryAllUnstakingUsersAllSpecsRequest,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.pagination = PageRequest.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryAllUnstakingUsersAllSpecsRequest,
        };
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageRequest.fromJSON(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? PageRequest.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryAllUnstakingUsersAllSpecsRequest,
        };
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageRequest.fromPartial(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
};
const baseQueryAllUnstakingUsersAllSpecsResponse = {};
export const QueryAllUnstakingUsersAllSpecsResponse = {
    encode(message, writer = Writer.create()) {
        for (const v of message.UnstakingUsersAllSpecs) {
            UnstakingUsersAllSpecs.encode(v, writer.uint32(10).fork()).ldelim();
        }
        if (message.pagination !== undefined) {
            PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryAllUnstakingUsersAllSpecsResponse,
        };
        message.UnstakingUsersAllSpecs = [];
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.UnstakingUsersAllSpecs.push(UnstakingUsersAllSpecs.decode(reader, reader.uint32()));
                    break;
                case 2:
                    message.pagination = PageResponse.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryAllUnstakingUsersAllSpecsResponse,
        };
        message.UnstakingUsersAllSpecs = [];
        if (object.UnstakingUsersAllSpecs !== undefined &&
            object.UnstakingUsersAllSpecs !== null) {
            for (const e of object.UnstakingUsersAllSpecs) {
                message.UnstakingUsersAllSpecs.push(UnstakingUsersAllSpecs.fromJSON(e));
            }
        }
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageResponse.fromJSON(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        if (message.UnstakingUsersAllSpecs) {
            obj.UnstakingUsersAllSpecs = message.UnstakingUsersAllSpecs.map((e) => e ? UnstakingUsersAllSpecs.toJSON(e) : undefined);
        }
        else {
            obj.UnstakingUsersAllSpecs = [];
        }
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? PageResponse.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryAllUnstakingUsersAllSpecsResponse,
        };
        message.UnstakingUsersAllSpecs = [];
        if (object.UnstakingUsersAllSpecs !== undefined &&
            object.UnstakingUsersAllSpecs !== null) {
            for (const e of object.UnstakingUsersAllSpecs) {
                message.UnstakingUsersAllSpecs.push(UnstakingUsersAllSpecs.fromPartial(e));
            }
        }
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageResponse.fromPartial(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
};
const baseQueryStakedUsersRequest = { specName: "", output: "" };
export const QueryStakedUsersRequest = {
    encode(message, writer = Writer.create()) {
        if (message.specName !== "") {
            writer.uint32(10).string(message.specName);
        }
        if (message.output !== "") {
            writer.uint32(18).string(message.output);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryStakedUsersRequest,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.specName = reader.string();
                    break;
                case 2:
                    message.output = reader.string();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryStakedUsersRequest,
        };
        if (object.specName !== undefined && object.specName !== null) {
            message.specName = String(object.specName);
        }
        else {
            message.specName = "";
        }
        if (object.output !== undefined && object.output !== null) {
            message.output = String(object.output);
        }
        else {
            message.output = "";
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.specName !== undefined && (obj.specName = message.specName);
        message.output !== undefined && (obj.output = message.output);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryStakedUsersRequest,
        };
        if (object.specName !== undefined && object.specName !== null) {
            message.specName = object.specName;
        }
        else {
            message.specName = "";
        }
        if (object.output !== undefined && object.output !== null) {
            message.output = object.output;
        }
        else {
            message.output = "";
        }
        return message;
    },
};
const baseQueryStakedUsersResponse = {};
export const QueryStakedUsersResponse = {
    encode(message, writer = Writer.create()) {
        if (message.stakeStorage !== undefined) {
            StakeStorage.encode(message.stakeStorage, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryStakedUsersResponse,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.stakeStorage = StakeStorage.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryStakedUsersResponse,
        };
        if (object.stakeStorage !== undefined && object.stakeStorage !== null) {
            message.stakeStorage = StakeStorage.fromJSON(object.stakeStorage);
        }
        else {
            message.stakeStorage = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.stakeStorage !== undefined &&
            (obj.stakeStorage = message.stakeStorage
                ? StakeStorage.toJSON(message.stakeStorage)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryStakedUsersResponse,
        };
        if (object.stakeStorage !== undefined && object.stakeStorage !== null) {
            message.stakeStorage = StakeStorage.fromPartial(object.stakeStorage);
        }
        else {
            message.stakeStorage = undefined;
        }
        return message;
    },
};
export class QueryClientImpl {
    constructor(rpc) {
        this.rpc = rpc;
    }
    Params(request) {
        const data = QueryParamsRequest.encode(request).finish();
        const promise = this.rpc.request("lavanet.lava.user.Query", "Params", data);
        return promise.then((data) => QueryParamsResponse.decode(new Reader(data)));
    }
    UserStake(request) {
        const data = QueryGetUserStakeRequest.encode(request).finish();
        const promise = this.rpc.request("lavanet.lava.user.Query", "UserStake", data);
        return promise.then((data) => QueryGetUserStakeResponse.decode(new Reader(data)));
    }
    UserStakeAll(request) {
        const data = QueryAllUserStakeRequest.encode(request).finish();
        const promise = this.rpc.request("lavanet.lava.user.Query", "UserStakeAll", data);
        return promise.then((data) => QueryAllUserStakeResponse.decode(new Reader(data)));
    }
    SpecStakeStorage(request) {
        const data = QueryGetSpecStakeStorageRequest.encode(request).finish();
        const promise = this.rpc.request("lavanet.lava.user.Query", "SpecStakeStorage", data);
        return promise.then((data) => QueryGetSpecStakeStorageResponse.decode(new Reader(data)));
    }
    SpecStakeStorageAll(request) {
        const data = QueryAllSpecStakeStorageRequest.encode(request).finish();
        const promise = this.rpc.request("lavanet.lava.user.Query", "SpecStakeStorageAll", data);
        return promise.then((data) => QueryAllSpecStakeStorageResponse.decode(new Reader(data)));
    }
    BlockDeadlineForCallback(request) {
        const data = QueryGetBlockDeadlineForCallbackRequest.encode(request).finish();
        const promise = this.rpc.request("lavanet.lava.user.Query", "BlockDeadlineForCallback", data);
        return promise.then((data) => QueryGetBlockDeadlineForCallbackResponse.decode(new Reader(data)));
    }
    UnstakingUsersAllSpecs(request) {
        const data = QueryGetUnstakingUsersAllSpecsRequest.encode(request).finish();
        const promise = this.rpc.request("lavanet.lava.user.Query", "UnstakingUsersAllSpecs", data);
        return promise.then((data) => QueryGetUnstakingUsersAllSpecsResponse.decode(new Reader(data)));
    }
    UnstakingUsersAllSpecsAll(request) {
        const data = QueryAllUnstakingUsersAllSpecsRequest.encode(request).finish();
        const promise = this.rpc.request("lavanet.lava.user.Query", "UnstakingUsersAllSpecsAll", data);
        return promise.then((data) => QueryAllUnstakingUsersAllSpecsResponse.decode(new Reader(data)));
    }
    StakedUsers(request) {
        const data = QueryStakedUsersRequest.encode(request).finish();
        const promise = this.rpc.request("lavanet.lava.user.Query", "StakedUsers", data);
        return promise.then((data) => QueryStakedUsersResponse.decode(new Reader(data)));
    }
}
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

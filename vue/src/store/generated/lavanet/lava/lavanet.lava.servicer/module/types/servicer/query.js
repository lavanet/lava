/* eslint-disable */
import { Reader, Writer } from "protobufjs/minimal";
import { Params } from "../servicer/params";
import { StakeMap } from "../servicer/stake_map";
import { PageRequest, PageResponse, } from "../cosmos/base/query/v1beta1/pagination";
import { SpecStakeStorage } from "../servicer/spec_stake_storage";
import { StakeStorage } from "../servicer/stake_storage";
export const protobufPackage = "lavanet.lava.servicer";
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
const baseQueryGetStakeMapRequest = { index: "" };
export const QueryGetStakeMapRequest = {
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
            ...baseQueryGetStakeMapRequest,
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
            ...baseQueryGetStakeMapRequest,
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
            ...baseQueryGetStakeMapRequest,
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
const baseQueryGetStakeMapResponse = {};
export const QueryGetStakeMapResponse = {
    encode(message, writer = Writer.create()) {
        if (message.stakeMap !== undefined) {
            StakeMap.encode(message.stakeMap, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetStakeMapResponse,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.stakeMap = StakeMap.decode(reader, reader.uint32());
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
            ...baseQueryGetStakeMapResponse,
        };
        if (object.stakeMap !== undefined && object.stakeMap !== null) {
            message.stakeMap = StakeMap.fromJSON(object.stakeMap);
        }
        else {
            message.stakeMap = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.stakeMap !== undefined &&
            (obj.stakeMap = message.stakeMap
                ? StakeMap.toJSON(message.stakeMap)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryGetStakeMapResponse,
        };
        if (object.stakeMap !== undefined && object.stakeMap !== null) {
            message.stakeMap = StakeMap.fromPartial(object.stakeMap);
        }
        else {
            message.stakeMap = undefined;
        }
        return message;
    },
};
const baseQueryAllStakeMapRequest = {};
export const QueryAllStakeMapRequest = {
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
            ...baseQueryAllStakeMapRequest,
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
            ...baseQueryAllStakeMapRequest,
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
            ...baseQueryAllStakeMapRequest,
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
const baseQueryAllStakeMapResponse = {};
export const QueryAllStakeMapResponse = {
    encode(message, writer = Writer.create()) {
        for (const v of message.stakeMap) {
            StakeMap.encode(v, writer.uint32(10).fork()).ldelim();
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
            ...baseQueryAllStakeMapResponse,
        };
        message.stakeMap = [];
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.stakeMap.push(StakeMap.decode(reader, reader.uint32()));
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
            ...baseQueryAllStakeMapResponse,
        };
        message.stakeMap = [];
        if (object.stakeMap !== undefined && object.stakeMap !== null) {
            for (const e of object.stakeMap) {
                message.stakeMap.push(StakeMap.fromJSON(e));
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
        if (message.stakeMap) {
            obj.stakeMap = message.stakeMap.map((e) => e ? StakeMap.toJSON(e) : undefined);
        }
        else {
            obj.stakeMap = [];
        }
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? PageResponse.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryAllStakeMapResponse,
        };
        message.stakeMap = [];
        if (object.stakeMap !== undefined && object.stakeMap !== null) {
            for (const e of object.stakeMap) {
                message.stakeMap.push(StakeMap.fromPartial(e));
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
const baseQueryStakedServicersRequest = { specName: "" };
export const QueryStakedServicersRequest = {
    encode(message, writer = Writer.create()) {
        if (message.specName !== "") {
            writer.uint32(10).string(message.specName);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryStakedServicersRequest,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.specName = reader.string();
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
            ...baseQueryStakedServicersRequest,
        };
        if (object.specName !== undefined && object.specName !== null) {
            message.specName = String(object.specName);
        }
        else {
            message.specName = "";
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.specName !== undefined && (obj.specName = message.specName);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryStakedServicersRequest,
        };
        if (object.specName !== undefined && object.specName !== null) {
            message.specName = object.specName;
        }
        else {
            message.specName = "";
        }
        return message;
    },
};
const baseQueryStakedServicersResponse = {};
export const QueryStakedServicersResponse = {
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
            ...baseQueryStakedServicersResponse,
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
            ...baseQueryStakedServicersResponse,
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
            ...baseQueryStakedServicersResponse,
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
        const promise = this.rpc.request("lavanet.lava.servicer.Query", "Params", data);
        return promise.then((data) => QueryParamsResponse.decode(new Reader(data)));
    }
    StakeMap(request) {
        const data = QueryGetStakeMapRequest.encode(request).finish();
        const promise = this.rpc.request("lavanet.lava.servicer.Query", "StakeMap", data);
        return promise.then((data) => QueryGetStakeMapResponse.decode(new Reader(data)));
    }
    StakeMapAll(request) {
        const data = QueryAllStakeMapRequest.encode(request).finish();
        const promise = this.rpc.request("lavanet.lava.servicer.Query", "StakeMapAll", data);
        return promise.then((data) => QueryAllStakeMapResponse.decode(new Reader(data)));
    }
    SpecStakeStorage(request) {
        const data = QueryGetSpecStakeStorageRequest.encode(request).finish();
        const promise = this.rpc.request("lavanet.lava.servicer.Query", "SpecStakeStorage", data);
        return promise.then((data) => QueryGetSpecStakeStorageResponse.decode(new Reader(data)));
    }
    SpecStakeStorageAll(request) {
        const data = QueryAllSpecStakeStorageRequest.encode(request).finish();
        const promise = this.rpc.request("lavanet.lava.servicer.Query", "SpecStakeStorageAll", data);
        return promise.then((data) => QueryAllSpecStakeStorageResponse.decode(new Reader(data)));
    }
    StakedServicers(request) {
        const data = QueryStakedServicersRequest.encode(request).finish();
        const promise = this.rpc.request("lavanet.lava.servicer.Query", "StakedServicers", data);
        return promise.then((data) => QueryStakedServicersResponse.decode(new Reader(data)));
    }
}

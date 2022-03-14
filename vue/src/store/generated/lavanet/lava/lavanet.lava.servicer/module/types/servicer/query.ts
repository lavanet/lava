/* eslint-disable */
import { Reader, Writer } from "protobufjs/minimal";
import { Params } from "../servicer/params";
import { StakeMap } from "../servicer/stake_map";
import {
  PageRequest,
  PageResponse,
} from "../cosmos/base/query/v1beta1/pagination";
import { SpecStakeStorage } from "../servicer/spec_stake_storage";
import { StakeStorage } from "../servicer/stake_storage";

export const protobufPackage = "lavanet.lava.servicer";

/** QueryParamsRequest is request type for the Query/Params RPC method. */
export interface QueryParamsRequest {}

/** QueryParamsResponse is response type for the Query/Params RPC method. */
export interface QueryParamsResponse {
  /** params holds all the parameters of this module. */
  params: Params | undefined;
}

export interface QueryGetStakeMapRequest {
  index: string;
}

export interface QueryGetStakeMapResponse {
  stakeMap: StakeMap | undefined;
}

export interface QueryAllStakeMapRequest {
  pagination: PageRequest | undefined;
}

export interface QueryAllStakeMapResponse {
  stakeMap: StakeMap[];
  pagination: PageResponse | undefined;
}

export interface QueryGetSpecStakeStorageRequest {
  index: string;
}

export interface QueryGetSpecStakeStorageResponse {
  specStakeStorage: SpecStakeStorage | undefined;
}

export interface QueryAllSpecStakeStorageRequest {
  pagination: PageRequest | undefined;
}

export interface QueryAllSpecStakeStorageResponse {
  specStakeStorage: SpecStakeStorage[];
  pagination: PageResponse | undefined;
}

export interface QueryStakedServicersRequest {
  specName: string;
}

export interface QueryStakedServicersResponse {
  stakeStorage: StakeStorage | undefined;
}

const baseQueryParamsRequest: object = {};

export const QueryParamsRequest = {
  encode(_: QueryParamsRequest, writer: Writer = Writer.create()): Writer {
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryParamsRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseQueryParamsRequest } as QueryParamsRequest;
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

  fromJSON(_: any): QueryParamsRequest {
    const message = { ...baseQueryParamsRequest } as QueryParamsRequest;
    return message;
  },

  toJSON(_: QueryParamsRequest): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(_: DeepPartial<QueryParamsRequest>): QueryParamsRequest {
    const message = { ...baseQueryParamsRequest } as QueryParamsRequest;
    return message;
  },
};

const baseQueryParamsResponse: object = {};

export const QueryParamsResponse = {
  encode(
    message: QueryParamsResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryParamsResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseQueryParamsResponse } as QueryParamsResponse;
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

  fromJSON(object: any): QueryParamsResponse {
    const message = { ...baseQueryParamsResponse } as QueryParamsResponse;
    if (object.params !== undefined && object.params !== null) {
      message.params = Params.fromJSON(object.params);
    } else {
      message.params = undefined;
    }
    return message;
  },

  toJSON(message: QueryParamsResponse): unknown {
    const obj: any = {};
    message.params !== undefined &&
      (obj.params = message.params ? Params.toJSON(message.params) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<QueryParamsResponse>): QueryParamsResponse {
    const message = { ...baseQueryParamsResponse } as QueryParamsResponse;
    if (object.params !== undefined && object.params !== null) {
      message.params = Params.fromPartial(object.params);
    } else {
      message.params = undefined;
    }
    return message;
  },
};

const baseQueryGetStakeMapRequest: object = { index: "" };

export const QueryGetStakeMapRequest = {
  encode(
    message: QueryGetStakeMapRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryGetStakeMapRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetStakeMapRequest,
    } as QueryGetStakeMapRequest;
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

  fromJSON(object: any): QueryGetStakeMapRequest {
    const message = {
      ...baseQueryGetStakeMapRequest,
    } as QueryGetStakeMapRequest;
    if (object.index !== undefined && object.index !== null) {
      message.index = String(object.index);
    } else {
      message.index = "";
    }
    return message;
  },

  toJSON(message: QueryGetStakeMapRequest): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetStakeMapRequest>
  ): QueryGetStakeMapRequest {
    const message = {
      ...baseQueryGetStakeMapRequest,
    } as QueryGetStakeMapRequest;
    if (object.index !== undefined && object.index !== null) {
      message.index = object.index;
    } else {
      message.index = "";
    }
    return message;
  },
};

const baseQueryGetStakeMapResponse: object = {};

export const QueryGetStakeMapResponse = {
  encode(
    message: QueryGetStakeMapResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.stakeMap !== undefined) {
      StakeMap.encode(message.stakeMap, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetStakeMapResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetStakeMapResponse,
    } as QueryGetStakeMapResponse;
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

  fromJSON(object: any): QueryGetStakeMapResponse {
    const message = {
      ...baseQueryGetStakeMapResponse,
    } as QueryGetStakeMapResponse;
    if (object.stakeMap !== undefined && object.stakeMap !== null) {
      message.stakeMap = StakeMap.fromJSON(object.stakeMap);
    } else {
      message.stakeMap = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetStakeMapResponse): unknown {
    const obj: any = {};
    message.stakeMap !== undefined &&
      (obj.stakeMap = message.stakeMap
        ? StakeMap.toJSON(message.stakeMap)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetStakeMapResponse>
  ): QueryGetStakeMapResponse {
    const message = {
      ...baseQueryGetStakeMapResponse,
    } as QueryGetStakeMapResponse;
    if (object.stakeMap !== undefined && object.stakeMap !== null) {
      message.stakeMap = StakeMap.fromPartial(object.stakeMap);
    } else {
      message.stakeMap = undefined;
    }
    return message;
  },
};

const baseQueryAllStakeMapRequest: object = {};

export const QueryAllStakeMapRequest = {
  encode(
    message: QueryAllStakeMapRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryAllStakeMapRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllStakeMapRequest,
    } as QueryAllStakeMapRequest;
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

  fromJSON(object: any): QueryAllStakeMapRequest {
    const message = {
      ...baseQueryAllStakeMapRequest,
    } as QueryAllStakeMapRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllStakeMapRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllStakeMapRequest>
  ): QueryAllStakeMapRequest {
    const message = {
      ...baseQueryAllStakeMapRequest,
    } as QueryAllStakeMapRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryAllStakeMapResponse: object = {};

export const QueryAllStakeMapResponse = {
  encode(
    message: QueryAllStakeMapResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.stakeMap) {
      StakeMap.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(
        message.pagination,
        writer.uint32(18).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryAllStakeMapResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllStakeMapResponse,
    } as QueryAllStakeMapResponse;
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

  fromJSON(object: any): QueryAllStakeMapResponse {
    const message = {
      ...baseQueryAllStakeMapResponse,
    } as QueryAllStakeMapResponse;
    message.stakeMap = [];
    if (object.stakeMap !== undefined && object.stakeMap !== null) {
      for (const e of object.stakeMap) {
        message.stakeMap.push(StakeMap.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllStakeMapResponse): unknown {
    const obj: any = {};
    if (message.stakeMap) {
      obj.stakeMap = message.stakeMap.map((e) =>
        e ? StakeMap.toJSON(e) : undefined
      );
    } else {
      obj.stakeMap = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllStakeMapResponse>
  ): QueryAllStakeMapResponse {
    const message = {
      ...baseQueryAllStakeMapResponse,
    } as QueryAllStakeMapResponse;
    message.stakeMap = [];
    if (object.stakeMap !== undefined && object.stakeMap !== null) {
      for (const e of object.stakeMap) {
        message.stakeMap.push(StakeMap.fromPartial(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryGetSpecStakeStorageRequest: object = { index: "" };

export const QueryGetSpecStakeStorageRequest = {
  encode(
    message: QueryGetSpecStakeStorageRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetSpecStakeStorageRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetSpecStakeStorageRequest,
    } as QueryGetSpecStakeStorageRequest;
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

  fromJSON(object: any): QueryGetSpecStakeStorageRequest {
    const message = {
      ...baseQueryGetSpecStakeStorageRequest,
    } as QueryGetSpecStakeStorageRequest;
    if (object.index !== undefined && object.index !== null) {
      message.index = String(object.index);
    } else {
      message.index = "";
    }
    return message;
  },

  toJSON(message: QueryGetSpecStakeStorageRequest): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetSpecStakeStorageRequest>
  ): QueryGetSpecStakeStorageRequest {
    const message = {
      ...baseQueryGetSpecStakeStorageRequest,
    } as QueryGetSpecStakeStorageRequest;
    if (object.index !== undefined && object.index !== null) {
      message.index = object.index;
    } else {
      message.index = "";
    }
    return message;
  },
};

const baseQueryGetSpecStakeStorageResponse: object = {};

export const QueryGetSpecStakeStorageResponse = {
  encode(
    message: QueryGetSpecStakeStorageResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.specStakeStorage !== undefined) {
      SpecStakeStorage.encode(
        message.specStakeStorage,
        writer.uint32(10).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetSpecStakeStorageResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetSpecStakeStorageResponse,
    } as QueryGetSpecStakeStorageResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.specStakeStorage = SpecStakeStorage.decode(
            reader,
            reader.uint32()
          );
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetSpecStakeStorageResponse {
    const message = {
      ...baseQueryGetSpecStakeStorageResponse,
    } as QueryGetSpecStakeStorageResponse;
    if (
      object.specStakeStorage !== undefined &&
      object.specStakeStorage !== null
    ) {
      message.specStakeStorage = SpecStakeStorage.fromJSON(
        object.specStakeStorage
      );
    } else {
      message.specStakeStorage = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetSpecStakeStorageResponse): unknown {
    const obj: any = {};
    message.specStakeStorage !== undefined &&
      (obj.specStakeStorage = message.specStakeStorage
        ? SpecStakeStorage.toJSON(message.specStakeStorage)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetSpecStakeStorageResponse>
  ): QueryGetSpecStakeStorageResponse {
    const message = {
      ...baseQueryGetSpecStakeStorageResponse,
    } as QueryGetSpecStakeStorageResponse;
    if (
      object.specStakeStorage !== undefined &&
      object.specStakeStorage !== null
    ) {
      message.specStakeStorage = SpecStakeStorage.fromPartial(
        object.specStakeStorage
      );
    } else {
      message.specStakeStorage = undefined;
    }
    return message;
  },
};

const baseQueryAllSpecStakeStorageRequest: object = {};

export const QueryAllSpecStakeStorageRequest = {
  encode(
    message: QueryAllSpecStakeStorageRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryAllSpecStakeStorageRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllSpecStakeStorageRequest,
    } as QueryAllSpecStakeStorageRequest;
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

  fromJSON(object: any): QueryAllSpecStakeStorageRequest {
    const message = {
      ...baseQueryAllSpecStakeStorageRequest,
    } as QueryAllSpecStakeStorageRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllSpecStakeStorageRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllSpecStakeStorageRequest>
  ): QueryAllSpecStakeStorageRequest {
    const message = {
      ...baseQueryAllSpecStakeStorageRequest,
    } as QueryAllSpecStakeStorageRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryAllSpecStakeStorageResponse: object = {};

export const QueryAllSpecStakeStorageResponse = {
  encode(
    message: QueryAllSpecStakeStorageResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.specStakeStorage) {
      SpecStakeStorage.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(
        message.pagination,
        writer.uint32(18).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryAllSpecStakeStorageResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllSpecStakeStorageResponse,
    } as QueryAllSpecStakeStorageResponse;
    message.specStakeStorage = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.specStakeStorage.push(
            SpecStakeStorage.decode(reader, reader.uint32())
          );
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

  fromJSON(object: any): QueryAllSpecStakeStorageResponse {
    const message = {
      ...baseQueryAllSpecStakeStorageResponse,
    } as QueryAllSpecStakeStorageResponse;
    message.specStakeStorage = [];
    if (
      object.specStakeStorage !== undefined &&
      object.specStakeStorage !== null
    ) {
      for (const e of object.specStakeStorage) {
        message.specStakeStorage.push(SpecStakeStorage.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllSpecStakeStorageResponse): unknown {
    const obj: any = {};
    if (message.specStakeStorage) {
      obj.specStakeStorage = message.specStakeStorage.map((e) =>
        e ? SpecStakeStorage.toJSON(e) : undefined
      );
    } else {
      obj.specStakeStorage = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllSpecStakeStorageResponse>
  ): QueryAllSpecStakeStorageResponse {
    const message = {
      ...baseQueryAllSpecStakeStorageResponse,
    } as QueryAllSpecStakeStorageResponse;
    message.specStakeStorage = [];
    if (
      object.specStakeStorage !== undefined &&
      object.specStakeStorage !== null
    ) {
      for (const e of object.specStakeStorage) {
        message.specStakeStorage.push(SpecStakeStorage.fromPartial(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryStakedServicersRequest: object = { specName: "" };

export const QueryStakedServicersRequest = {
  encode(
    message: QueryStakedServicersRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.specName !== "") {
      writer.uint32(10).string(message.specName);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryStakedServicersRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryStakedServicersRequest,
    } as QueryStakedServicersRequest;
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

  fromJSON(object: any): QueryStakedServicersRequest {
    const message = {
      ...baseQueryStakedServicersRequest,
    } as QueryStakedServicersRequest;
    if (object.specName !== undefined && object.specName !== null) {
      message.specName = String(object.specName);
    } else {
      message.specName = "";
    }
    return message;
  },

  toJSON(message: QueryStakedServicersRequest): unknown {
    const obj: any = {};
    message.specName !== undefined && (obj.specName = message.specName);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryStakedServicersRequest>
  ): QueryStakedServicersRequest {
    const message = {
      ...baseQueryStakedServicersRequest,
    } as QueryStakedServicersRequest;
    if (object.specName !== undefined && object.specName !== null) {
      message.specName = object.specName;
    } else {
      message.specName = "";
    }
    return message;
  },
};

const baseQueryStakedServicersResponse: object = {};

export const QueryStakedServicersResponse = {
  encode(
    message: QueryStakedServicersResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.stakeStorage !== undefined) {
      StakeStorage.encode(
        message.stakeStorage,
        writer.uint32(10).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryStakedServicersResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryStakedServicersResponse,
    } as QueryStakedServicersResponse;
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

  fromJSON(object: any): QueryStakedServicersResponse {
    const message = {
      ...baseQueryStakedServicersResponse,
    } as QueryStakedServicersResponse;
    if (object.stakeStorage !== undefined && object.stakeStorage !== null) {
      message.stakeStorage = StakeStorage.fromJSON(object.stakeStorage);
    } else {
      message.stakeStorage = undefined;
    }
    return message;
  },

  toJSON(message: QueryStakedServicersResponse): unknown {
    const obj: any = {};
    message.stakeStorage !== undefined &&
      (obj.stakeStorage = message.stakeStorage
        ? StakeStorage.toJSON(message.stakeStorage)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryStakedServicersResponse>
  ): QueryStakedServicersResponse {
    const message = {
      ...baseQueryStakedServicersResponse,
    } as QueryStakedServicersResponse;
    if (object.stakeStorage !== undefined && object.stakeStorage !== null) {
      message.stakeStorage = StakeStorage.fromPartial(object.stakeStorage);
    } else {
      message.stakeStorage = undefined;
    }
    return message;
  },
};

/** Query defines the gRPC querier service. */
export interface Query {
  /** Parameters queries the parameters of the module. */
  Params(request: QueryParamsRequest): Promise<QueryParamsResponse>;
  /** Queries a StakeMap by index. */
  StakeMap(request: QueryGetStakeMapRequest): Promise<QueryGetStakeMapResponse>;
  /** Queries a list of StakeMap items. */
  StakeMapAll(
    request: QueryAllStakeMapRequest
  ): Promise<QueryAllStakeMapResponse>;
  /** Queries a SpecStakeStorage by index. */
  SpecStakeStorage(
    request: QueryGetSpecStakeStorageRequest
  ): Promise<QueryGetSpecStakeStorageResponse>;
  /** Queries a list of SpecStakeStorage items. */
  SpecStakeStorageAll(
    request: QueryAllSpecStakeStorageRequest
  ): Promise<QueryAllSpecStakeStorageResponse>;
  /** Queries a list of StakedServicers items. */
  StakedServicers(
    request: QueryStakedServicersRequest
  ): Promise<QueryStakedServicersResponse>;
}

export class QueryClientImpl implements Query {
  private readonly rpc: Rpc;
  constructor(rpc: Rpc) {
    this.rpc = rpc;
  }
  Params(request: QueryParamsRequest): Promise<QueryParamsResponse> {
    const data = QueryParamsRequest.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.servicer.Query",
      "Params",
      data
    );
    return promise.then((data) => QueryParamsResponse.decode(new Reader(data)));
  }

  StakeMap(
    request: QueryGetStakeMapRequest
  ): Promise<QueryGetStakeMapResponse> {
    const data = QueryGetStakeMapRequest.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.servicer.Query",
      "StakeMap",
      data
    );
    return promise.then((data) =>
      QueryGetStakeMapResponse.decode(new Reader(data))
    );
  }

  StakeMapAll(
    request: QueryAllStakeMapRequest
  ): Promise<QueryAllStakeMapResponse> {
    const data = QueryAllStakeMapRequest.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.servicer.Query",
      "StakeMapAll",
      data
    );
    return promise.then((data) =>
      QueryAllStakeMapResponse.decode(new Reader(data))
    );
  }

  SpecStakeStorage(
    request: QueryGetSpecStakeStorageRequest
  ): Promise<QueryGetSpecStakeStorageResponse> {
    const data = QueryGetSpecStakeStorageRequest.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.servicer.Query",
      "SpecStakeStorage",
      data
    );
    return promise.then((data) =>
      QueryGetSpecStakeStorageResponse.decode(new Reader(data))
    );
  }

  SpecStakeStorageAll(
    request: QueryAllSpecStakeStorageRequest
  ): Promise<QueryAllSpecStakeStorageResponse> {
    const data = QueryAllSpecStakeStorageRequest.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.servicer.Query",
      "SpecStakeStorageAll",
      data
    );
    return promise.then((data) =>
      QueryAllSpecStakeStorageResponse.decode(new Reader(data))
    );
  }

  StakedServicers(
    request: QueryStakedServicersRequest
  ): Promise<QueryStakedServicersResponse> {
    const data = QueryStakedServicersRequest.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.servicer.Query",
      "StakedServicers",
      data
    );
    return promise.then((data) =>
      QueryStakedServicersResponse.decode(new Reader(data))
    );
  }
}

interface Rpc {
  request(
    service: string,
    method: string,
    data: Uint8Array
  ): Promise<Uint8Array>;
}

type Builtin = Date | Function | Uint8Array | string | number | undefined;
export type DeepPartial<T> = T extends Builtin
  ? T
  : T extends Array<infer U>
  ? Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U>
  ? ReadonlyArray<DeepPartial<U>>
  : T extends {}
  ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

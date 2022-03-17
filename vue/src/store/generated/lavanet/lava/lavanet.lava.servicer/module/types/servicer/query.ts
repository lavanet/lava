/* eslint-disable */
import { Reader, util, configure, Writer } from "protobufjs/minimal";
import * as Long from "long";
import { Params } from "../servicer/params";
import { StakeMap } from "../servicer/stake_map";
import {
  PageRequest,
  PageResponse,
} from "../cosmos/base/query/v1beta1/pagination";
import { SpecStakeStorage } from "../servicer/spec_stake_storage";
import { StakeStorage } from "../servicer/stake_storage";
import { BlockDeadlineForCallback } from "../servicer/block_deadline_for_callback";
import { UnstakingServicersAllSpecs } from "../servicer/unstaking_servicers_all_specs";

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
  output: string;
}

export interface QueryGetBlockDeadlineForCallbackRequest {}

export interface QueryGetBlockDeadlineForCallbackResponse {
  BlockDeadlineForCallback: BlockDeadlineForCallback | undefined;
}

export interface QueryGetUnstakingServicersAllSpecsRequest {
  id: number;
}

export interface QueryGetUnstakingServicersAllSpecsResponse {
  UnstakingServicersAllSpecs: UnstakingServicersAllSpecs | undefined;
}

export interface QueryAllUnstakingServicersAllSpecsRequest {
  pagination: PageRequest | undefined;
}

export interface QueryAllUnstakingServicersAllSpecsResponse {
  UnstakingServicersAllSpecs: UnstakingServicersAllSpecs[];
  pagination: PageResponse | undefined;
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

const baseQueryStakedServicersResponse: object = { output: "" };

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
    if (message.output !== "") {
      writer.uint32(18).string(message.output);
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

  fromJSON(object: any): QueryStakedServicersResponse {
    const message = {
      ...baseQueryStakedServicersResponse,
    } as QueryStakedServicersResponse;
    if (object.stakeStorage !== undefined && object.stakeStorage !== null) {
      message.stakeStorage = StakeStorage.fromJSON(object.stakeStorage);
    } else {
      message.stakeStorage = undefined;
    }
    if (object.output !== undefined && object.output !== null) {
      message.output = String(object.output);
    } else {
      message.output = "";
    }
    return message;
  },

  toJSON(message: QueryStakedServicersResponse): unknown {
    const obj: any = {};
    message.stakeStorage !== undefined &&
      (obj.stakeStorage = message.stakeStorage
        ? StakeStorage.toJSON(message.stakeStorage)
        : undefined);
    message.output !== undefined && (obj.output = message.output);
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
    if (object.output !== undefined && object.output !== null) {
      message.output = object.output;
    } else {
      message.output = "";
    }
    return message;
  },
};

const baseQueryGetBlockDeadlineForCallbackRequest: object = {};

export const QueryGetBlockDeadlineForCallbackRequest = {
  encode(
    _: QueryGetBlockDeadlineForCallbackRequest,
    writer: Writer = Writer.create()
  ): Writer {
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetBlockDeadlineForCallbackRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetBlockDeadlineForCallbackRequest,
    } as QueryGetBlockDeadlineForCallbackRequest;
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

  fromJSON(_: any): QueryGetBlockDeadlineForCallbackRequest {
    const message = {
      ...baseQueryGetBlockDeadlineForCallbackRequest,
    } as QueryGetBlockDeadlineForCallbackRequest;
    return message;
  },

  toJSON(_: QueryGetBlockDeadlineForCallbackRequest): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(
    _: DeepPartial<QueryGetBlockDeadlineForCallbackRequest>
  ): QueryGetBlockDeadlineForCallbackRequest {
    const message = {
      ...baseQueryGetBlockDeadlineForCallbackRequest,
    } as QueryGetBlockDeadlineForCallbackRequest;
    return message;
  },
};

const baseQueryGetBlockDeadlineForCallbackResponse: object = {};

export const QueryGetBlockDeadlineForCallbackResponse = {
  encode(
    message: QueryGetBlockDeadlineForCallbackResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.BlockDeadlineForCallback !== undefined) {
      BlockDeadlineForCallback.encode(
        message.BlockDeadlineForCallback,
        writer.uint32(10).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetBlockDeadlineForCallbackResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetBlockDeadlineForCallbackResponse,
    } as QueryGetBlockDeadlineForCallbackResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.BlockDeadlineForCallback = BlockDeadlineForCallback.decode(
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

  fromJSON(object: any): QueryGetBlockDeadlineForCallbackResponse {
    const message = {
      ...baseQueryGetBlockDeadlineForCallbackResponse,
    } as QueryGetBlockDeadlineForCallbackResponse;
    if (
      object.BlockDeadlineForCallback !== undefined &&
      object.BlockDeadlineForCallback !== null
    ) {
      message.BlockDeadlineForCallback = BlockDeadlineForCallback.fromJSON(
        object.BlockDeadlineForCallback
      );
    } else {
      message.BlockDeadlineForCallback = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetBlockDeadlineForCallbackResponse): unknown {
    const obj: any = {};
    message.BlockDeadlineForCallback !== undefined &&
      (obj.BlockDeadlineForCallback = message.BlockDeadlineForCallback
        ? BlockDeadlineForCallback.toJSON(message.BlockDeadlineForCallback)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetBlockDeadlineForCallbackResponse>
  ): QueryGetBlockDeadlineForCallbackResponse {
    const message = {
      ...baseQueryGetBlockDeadlineForCallbackResponse,
    } as QueryGetBlockDeadlineForCallbackResponse;
    if (
      object.BlockDeadlineForCallback !== undefined &&
      object.BlockDeadlineForCallback !== null
    ) {
      message.BlockDeadlineForCallback = BlockDeadlineForCallback.fromPartial(
        object.BlockDeadlineForCallback
      );
    } else {
      message.BlockDeadlineForCallback = undefined;
    }
    return message;
  },
};

const baseQueryGetUnstakingServicersAllSpecsRequest: object = { id: 0 };

export const QueryGetUnstakingServicersAllSpecsRequest = {
  encode(
    message: QueryGetUnstakingServicersAllSpecsRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.id !== 0) {
      writer.uint32(8).uint64(message.id);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetUnstakingServicersAllSpecsRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetUnstakingServicersAllSpecsRequest,
    } as QueryGetUnstakingServicersAllSpecsRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = longToNumber(reader.uint64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetUnstakingServicersAllSpecsRequest {
    const message = {
      ...baseQueryGetUnstakingServicersAllSpecsRequest,
    } as QueryGetUnstakingServicersAllSpecsRequest;
    if (object.id !== undefined && object.id !== null) {
      message.id = Number(object.id);
    } else {
      message.id = 0;
    }
    return message;
  },

  toJSON(message: QueryGetUnstakingServicersAllSpecsRequest): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetUnstakingServicersAllSpecsRequest>
  ): QueryGetUnstakingServicersAllSpecsRequest {
    const message = {
      ...baseQueryGetUnstakingServicersAllSpecsRequest,
    } as QueryGetUnstakingServicersAllSpecsRequest;
    if (object.id !== undefined && object.id !== null) {
      message.id = object.id;
    } else {
      message.id = 0;
    }
    return message;
  },
};

const baseQueryGetUnstakingServicersAllSpecsResponse: object = {};

export const QueryGetUnstakingServicersAllSpecsResponse = {
  encode(
    message: QueryGetUnstakingServicersAllSpecsResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.UnstakingServicersAllSpecs !== undefined) {
      UnstakingServicersAllSpecs.encode(
        message.UnstakingServicersAllSpecs,
        writer.uint32(10).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetUnstakingServicersAllSpecsResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetUnstakingServicersAllSpecsResponse,
    } as QueryGetUnstakingServicersAllSpecsResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.UnstakingServicersAllSpecs = UnstakingServicersAllSpecs.decode(
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

  fromJSON(object: any): QueryGetUnstakingServicersAllSpecsResponse {
    const message = {
      ...baseQueryGetUnstakingServicersAllSpecsResponse,
    } as QueryGetUnstakingServicersAllSpecsResponse;
    if (
      object.UnstakingServicersAllSpecs !== undefined &&
      object.UnstakingServicersAllSpecs !== null
    ) {
      message.UnstakingServicersAllSpecs = UnstakingServicersAllSpecs.fromJSON(
        object.UnstakingServicersAllSpecs
      );
    } else {
      message.UnstakingServicersAllSpecs = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetUnstakingServicersAllSpecsResponse): unknown {
    const obj: any = {};
    message.UnstakingServicersAllSpecs !== undefined &&
      (obj.UnstakingServicersAllSpecs = message.UnstakingServicersAllSpecs
        ? UnstakingServicersAllSpecs.toJSON(message.UnstakingServicersAllSpecs)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetUnstakingServicersAllSpecsResponse>
  ): QueryGetUnstakingServicersAllSpecsResponse {
    const message = {
      ...baseQueryGetUnstakingServicersAllSpecsResponse,
    } as QueryGetUnstakingServicersAllSpecsResponse;
    if (
      object.UnstakingServicersAllSpecs !== undefined &&
      object.UnstakingServicersAllSpecs !== null
    ) {
      message.UnstakingServicersAllSpecs = UnstakingServicersAllSpecs.fromPartial(
        object.UnstakingServicersAllSpecs
      );
    } else {
      message.UnstakingServicersAllSpecs = undefined;
    }
    return message;
  },
};

const baseQueryAllUnstakingServicersAllSpecsRequest: object = {};

export const QueryAllUnstakingServicersAllSpecsRequest = {
  encode(
    message: QueryAllUnstakingServicersAllSpecsRequest,
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
  ): QueryAllUnstakingServicersAllSpecsRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllUnstakingServicersAllSpecsRequest,
    } as QueryAllUnstakingServicersAllSpecsRequest;
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

  fromJSON(object: any): QueryAllUnstakingServicersAllSpecsRequest {
    const message = {
      ...baseQueryAllUnstakingServicersAllSpecsRequest,
    } as QueryAllUnstakingServicersAllSpecsRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllUnstakingServicersAllSpecsRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllUnstakingServicersAllSpecsRequest>
  ): QueryAllUnstakingServicersAllSpecsRequest {
    const message = {
      ...baseQueryAllUnstakingServicersAllSpecsRequest,
    } as QueryAllUnstakingServicersAllSpecsRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryAllUnstakingServicersAllSpecsResponse: object = {};

export const QueryAllUnstakingServicersAllSpecsResponse = {
  encode(
    message: QueryAllUnstakingServicersAllSpecsResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.UnstakingServicersAllSpecs) {
      UnstakingServicersAllSpecs.encode(v!, writer.uint32(10).fork()).ldelim();
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
  ): QueryAllUnstakingServicersAllSpecsResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllUnstakingServicersAllSpecsResponse,
    } as QueryAllUnstakingServicersAllSpecsResponse;
    message.UnstakingServicersAllSpecs = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.UnstakingServicersAllSpecs.push(
            UnstakingServicersAllSpecs.decode(reader, reader.uint32())
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

  fromJSON(object: any): QueryAllUnstakingServicersAllSpecsResponse {
    const message = {
      ...baseQueryAllUnstakingServicersAllSpecsResponse,
    } as QueryAllUnstakingServicersAllSpecsResponse;
    message.UnstakingServicersAllSpecs = [];
    if (
      object.UnstakingServicersAllSpecs !== undefined &&
      object.UnstakingServicersAllSpecs !== null
    ) {
      for (const e of object.UnstakingServicersAllSpecs) {
        message.UnstakingServicersAllSpecs.push(
          UnstakingServicersAllSpecs.fromJSON(e)
        );
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllUnstakingServicersAllSpecsResponse): unknown {
    const obj: any = {};
    if (message.UnstakingServicersAllSpecs) {
      obj.UnstakingServicersAllSpecs = message.UnstakingServicersAllSpecs.map(
        (e) => (e ? UnstakingServicersAllSpecs.toJSON(e) : undefined)
      );
    } else {
      obj.UnstakingServicersAllSpecs = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllUnstakingServicersAllSpecsResponse>
  ): QueryAllUnstakingServicersAllSpecsResponse {
    const message = {
      ...baseQueryAllUnstakingServicersAllSpecsResponse,
    } as QueryAllUnstakingServicersAllSpecsResponse;
    message.UnstakingServicersAllSpecs = [];
    if (
      object.UnstakingServicersAllSpecs !== undefined &&
      object.UnstakingServicersAllSpecs !== null
    ) {
      for (const e of object.UnstakingServicersAllSpecs) {
        message.UnstakingServicersAllSpecs.push(
          UnstakingServicersAllSpecs.fromPartial(e)
        );
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
  /** Queries a BlockDeadlineForCallback by index. */
  BlockDeadlineForCallback(
    request: QueryGetBlockDeadlineForCallbackRequest
  ): Promise<QueryGetBlockDeadlineForCallbackResponse>;
  /** Queries a UnstakingServicersAllSpecs by id. */
  UnstakingServicersAllSpecs(
    request: QueryGetUnstakingServicersAllSpecsRequest
  ): Promise<QueryGetUnstakingServicersAllSpecsResponse>;
  /** Queries a list of UnstakingServicersAllSpecs items. */
  UnstakingServicersAllSpecsAll(
    request: QueryAllUnstakingServicersAllSpecsRequest
  ): Promise<QueryAllUnstakingServicersAllSpecsResponse>;
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

  BlockDeadlineForCallback(
    request: QueryGetBlockDeadlineForCallbackRequest
  ): Promise<QueryGetBlockDeadlineForCallbackResponse> {
    const data = QueryGetBlockDeadlineForCallbackRequest.encode(
      request
    ).finish();
    const promise = this.rpc.request(
      "lavanet.lava.servicer.Query",
      "BlockDeadlineForCallback",
      data
    );
    return promise.then((data) =>
      QueryGetBlockDeadlineForCallbackResponse.decode(new Reader(data))
    );
  }

  UnstakingServicersAllSpecs(
    request: QueryGetUnstakingServicersAllSpecsRequest
  ): Promise<QueryGetUnstakingServicersAllSpecsResponse> {
    const data = QueryGetUnstakingServicersAllSpecsRequest.encode(
      request
    ).finish();
    const promise = this.rpc.request(
      "lavanet.lava.servicer.Query",
      "UnstakingServicersAllSpecs",
      data
    );
    return promise.then((data) =>
      QueryGetUnstakingServicersAllSpecsResponse.decode(new Reader(data))
    );
  }

  UnstakingServicersAllSpecsAll(
    request: QueryAllUnstakingServicersAllSpecsRequest
  ): Promise<QueryAllUnstakingServicersAllSpecsResponse> {
    const data = QueryAllUnstakingServicersAllSpecsRequest.encode(
      request
    ).finish();
    const promise = this.rpc.request(
      "lavanet.lava.servicer.Query",
      "UnstakingServicersAllSpecsAll",
      data
    );
    return promise.then((data) =>
      QueryAllUnstakingServicersAllSpecsResponse.decode(new Reader(data))
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

declare var self: any | undefined;
declare var window: any | undefined;
var globalThis: any = (() => {
  if (typeof globalThis !== "undefined") return globalThis;
  if (typeof self !== "undefined") return self;
  if (typeof window !== "undefined") return window;
  if (typeof global !== "undefined") return global;
  throw "Unable to locate global object";
})();

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

function longToNumber(long: Long): number {
  if (long.gt(Number.MAX_SAFE_INTEGER)) {
    throw new globalThis.Error("Value is larger than Number.MAX_SAFE_INTEGER");
  }
  return long.toNumber();
}

if (util.Long !== Long) {
  util.Long = Long as any;
  configure();
}

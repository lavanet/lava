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
import { CurrentSessionStart } from "../servicer/current_session_start";
import { PreviousSessionBlocks } from "../servicer/previous_session_blocks";
import { SessionStorageForSpec } from "../servicer/session_storage_for_spec";
import { EarliestSessionStart } from "../servicer/earliest_session_start";

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

export interface QueryGetPairingRequest {
  specName: string;
  userAddr: string;
}

export interface QueryGetPairingResponse {
  servicers: StakeStorage | undefined;
}

export interface QueryGetCurrentSessionStartRequest {}

export interface QueryGetCurrentSessionStartResponse {
  CurrentSessionStart: CurrentSessionStart | undefined;
}

export interface QueryGetPreviousSessionBlocksRequest {}

export interface QueryGetPreviousSessionBlocksResponse {
  PreviousSessionBlocks: PreviousSessionBlocks | undefined;
}

export interface QueryGetSessionStorageForSpecRequest {
  index: string;
}

export interface QueryGetSessionStorageForSpecResponse {
  sessionStorageForSpec: SessionStorageForSpec | undefined;
}

export interface QueryAllSessionStorageForSpecRequest {
  pagination: PageRequest | undefined;
}

export interface QueryAllSessionStorageForSpecResponse {
  sessionStorageForSpec: SessionStorageForSpec[];
  pagination: PageResponse | undefined;
}

export interface QuerySessionStorageForAllSpecsRequest {
  blockNum: number;
}

export interface QuerySessionStorageForAllSpecsResponse {
  servicers: StakeStorage | undefined;
}

export interface QueryAllSessionStoragesForSpecRequest {
  specName: string;
}

export interface QueryAllSessionStoragesForSpecResponse {
  storages: SessionStorageForSpec[];
}

export interface QueryGetEarliestSessionStartRequest {}

export interface QueryGetEarliestSessionStartResponse {
  EarliestSessionStart: EarliestSessionStart | undefined;
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

const baseQueryGetPairingRequest: object = { specName: "", userAddr: "" };

export const QueryGetPairingRequest = {
  encode(
    message: QueryGetPairingRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.specName !== "") {
      writer.uint32(10).string(message.specName);
    }
    if (message.userAddr !== "") {
      writer.uint32(18).string(message.userAddr);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryGetPairingRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseQueryGetPairingRequest } as QueryGetPairingRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.specName = reader.string();
          break;
        case 2:
          message.userAddr = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetPairingRequest {
    const message = { ...baseQueryGetPairingRequest } as QueryGetPairingRequest;
    if (object.specName !== undefined && object.specName !== null) {
      message.specName = String(object.specName);
    } else {
      message.specName = "";
    }
    if (object.userAddr !== undefined && object.userAddr !== null) {
      message.userAddr = String(object.userAddr);
    } else {
      message.userAddr = "";
    }
    return message;
  },

  toJSON(message: QueryGetPairingRequest): unknown {
    const obj: any = {};
    message.specName !== undefined && (obj.specName = message.specName);
    message.userAddr !== undefined && (obj.userAddr = message.userAddr);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetPairingRequest>
  ): QueryGetPairingRequest {
    const message = { ...baseQueryGetPairingRequest } as QueryGetPairingRequest;
    if (object.specName !== undefined && object.specName !== null) {
      message.specName = object.specName;
    } else {
      message.specName = "";
    }
    if (object.userAddr !== undefined && object.userAddr !== null) {
      message.userAddr = object.userAddr;
    } else {
      message.userAddr = "";
    }
    return message;
  },
};

const baseQueryGetPairingResponse: object = {};

export const QueryGetPairingResponse = {
  encode(
    message: QueryGetPairingResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.servicers !== undefined) {
      StakeStorage.encode(message.servicers, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryGetPairingResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetPairingResponse,
    } as QueryGetPairingResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.servicers = StakeStorage.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetPairingResponse {
    const message = {
      ...baseQueryGetPairingResponse,
    } as QueryGetPairingResponse;
    if (object.servicers !== undefined && object.servicers !== null) {
      message.servicers = StakeStorage.fromJSON(object.servicers);
    } else {
      message.servicers = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetPairingResponse): unknown {
    const obj: any = {};
    message.servicers !== undefined &&
      (obj.servicers = message.servicers
        ? StakeStorage.toJSON(message.servicers)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetPairingResponse>
  ): QueryGetPairingResponse {
    const message = {
      ...baseQueryGetPairingResponse,
    } as QueryGetPairingResponse;
    if (object.servicers !== undefined && object.servicers !== null) {
      message.servicers = StakeStorage.fromPartial(object.servicers);
    } else {
      message.servicers = undefined;
    }
    return message;
  },
};

const baseQueryGetCurrentSessionStartRequest: object = {};

export const QueryGetCurrentSessionStartRequest = {
  encode(
    _: QueryGetCurrentSessionStartRequest,
    writer: Writer = Writer.create()
  ): Writer {
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetCurrentSessionStartRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetCurrentSessionStartRequest,
    } as QueryGetCurrentSessionStartRequest;
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

  fromJSON(_: any): QueryGetCurrentSessionStartRequest {
    const message = {
      ...baseQueryGetCurrentSessionStartRequest,
    } as QueryGetCurrentSessionStartRequest;
    return message;
  },

  toJSON(_: QueryGetCurrentSessionStartRequest): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(
    _: DeepPartial<QueryGetCurrentSessionStartRequest>
  ): QueryGetCurrentSessionStartRequest {
    const message = {
      ...baseQueryGetCurrentSessionStartRequest,
    } as QueryGetCurrentSessionStartRequest;
    return message;
  },
};

const baseQueryGetCurrentSessionStartResponse: object = {};

export const QueryGetCurrentSessionStartResponse = {
  encode(
    message: QueryGetCurrentSessionStartResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.CurrentSessionStart !== undefined) {
      CurrentSessionStart.encode(
        message.CurrentSessionStart,
        writer.uint32(10).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetCurrentSessionStartResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetCurrentSessionStartResponse,
    } as QueryGetCurrentSessionStartResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.CurrentSessionStart = CurrentSessionStart.decode(
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

  fromJSON(object: any): QueryGetCurrentSessionStartResponse {
    const message = {
      ...baseQueryGetCurrentSessionStartResponse,
    } as QueryGetCurrentSessionStartResponse;
    if (
      object.CurrentSessionStart !== undefined &&
      object.CurrentSessionStart !== null
    ) {
      message.CurrentSessionStart = CurrentSessionStart.fromJSON(
        object.CurrentSessionStart
      );
    } else {
      message.CurrentSessionStart = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetCurrentSessionStartResponse): unknown {
    const obj: any = {};
    message.CurrentSessionStart !== undefined &&
      (obj.CurrentSessionStart = message.CurrentSessionStart
        ? CurrentSessionStart.toJSON(message.CurrentSessionStart)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetCurrentSessionStartResponse>
  ): QueryGetCurrentSessionStartResponse {
    const message = {
      ...baseQueryGetCurrentSessionStartResponse,
    } as QueryGetCurrentSessionStartResponse;
    if (
      object.CurrentSessionStart !== undefined &&
      object.CurrentSessionStart !== null
    ) {
      message.CurrentSessionStart = CurrentSessionStart.fromPartial(
        object.CurrentSessionStart
      );
    } else {
      message.CurrentSessionStart = undefined;
    }
    return message;
  },
};

const baseQueryGetPreviousSessionBlocksRequest: object = {};

export const QueryGetPreviousSessionBlocksRequest = {
  encode(
    _: QueryGetPreviousSessionBlocksRequest,
    writer: Writer = Writer.create()
  ): Writer {
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetPreviousSessionBlocksRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetPreviousSessionBlocksRequest,
    } as QueryGetPreviousSessionBlocksRequest;
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

  fromJSON(_: any): QueryGetPreviousSessionBlocksRequest {
    const message = {
      ...baseQueryGetPreviousSessionBlocksRequest,
    } as QueryGetPreviousSessionBlocksRequest;
    return message;
  },

  toJSON(_: QueryGetPreviousSessionBlocksRequest): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(
    _: DeepPartial<QueryGetPreviousSessionBlocksRequest>
  ): QueryGetPreviousSessionBlocksRequest {
    const message = {
      ...baseQueryGetPreviousSessionBlocksRequest,
    } as QueryGetPreviousSessionBlocksRequest;
    return message;
  },
};

const baseQueryGetPreviousSessionBlocksResponse: object = {};

export const QueryGetPreviousSessionBlocksResponse = {
  encode(
    message: QueryGetPreviousSessionBlocksResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.PreviousSessionBlocks !== undefined) {
      PreviousSessionBlocks.encode(
        message.PreviousSessionBlocks,
        writer.uint32(10).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetPreviousSessionBlocksResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetPreviousSessionBlocksResponse,
    } as QueryGetPreviousSessionBlocksResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.PreviousSessionBlocks = PreviousSessionBlocks.decode(
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

  fromJSON(object: any): QueryGetPreviousSessionBlocksResponse {
    const message = {
      ...baseQueryGetPreviousSessionBlocksResponse,
    } as QueryGetPreviousSessionBlocksResponse;
    if (
      object.PreviousSessionBlocks !== undefined &&
      object.PreviousSessionBlocks !== null
    ) {
      message.PreviousSessionBlocks = PreviousSessionBlocks.fromJSON(
        object.PreviousSessionBlocks
      );
    } else {
      message.PreviousSessionBlocks = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetPreviousSessionBlocksResponse): unknown {
    const obj: any = {};
    message.PreviousSessionBlocks !== undefined &&
      (obj.PreviousSessionBlocks = message.PreviousSessionBlocks
        ? PreviousSessionBlocks.toJSON(message.PreviousSessionBlocks)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetPreviousSessionBlocksResponse>
  ): QueryGetPreviousSessionBlocksResponse {
    const message = {
      ...baseQueryGetPreviousSessionBlocksResponse,
    } as QueryGetPreviousSessionBlocksResponse;
    if (
      object.PreviousSessionBlocks !== undefined &&
      object.PreviousSessionBlocks !== null
    ) {
      message.PreviousSessionBlocks = PreviousSessionBlocks.fromPartial(
        object.PreviousSessionBlocks
      );
    } else {
      message.PreviousSessionBlocks = undefined;
    }
    return message;
  },
};

const baseQueryGetSessionStorageForSpecRequest: object = { index: "" };

export const QueryGetSessionStorageForSpecRequest = {
  encode(
    message: QueryGetSessionStorageForSpecRequest,
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
  ): QueryGetSessionStorageForSpecRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetSessionStorageForSpecRequest,
    } as QueryGetSessionStorageForSpecRequest;
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

  fromJSON(object: any): QueryGetSessionStorageForSpecRequest {
    const message = {
      ...baseQueryGetSessionStorageForSpecRequest,
    } as QueryGetSessionStorageForSpecRequest;
    if (object.index !== undefined && object.index !== null) {
      message.index = String(object.index);
    } else {
      message.index = "";
    }
    return message;
  },

  toJSON(message: QueryGetSessionStorageForSpecRequest): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetSessionStorageForSpecRequest>
  ): QueryGetSessionStorageForSpecRequest {
    const message = {
      ...baseQueryGetSessionStorageForSpecRequest,
    } as QueryGetSessionStorageForSpecRequest;
    if (object.index !== undefined && object.index !== null) {
      message.index = object.index;
    } else {
      message.index = "";
    }
    return message;
  },
};

const baseQueryGetSessionStorageForSpecResponse: object = {};

export const QueryGetSessionStorageForSpecResponse = {
  encode(
    message: QueryGetSessionStorageForSpecResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.sessionStorageForSpec !== undefined) {
      SessionStorageForSpec.encode(
        message.sessionStorageForSpec,
        writer.uint32(10).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetSessionStorageForSpecResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetSessionStorageForSpecResponse,
    } as QueryGetSessionStorageForSpecResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sessionStorageForSpec = SessionStorageForSpec.decode(
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

  fromJSON(object: any): QueryGetSessionStorageForSpecResponse {
    const message = {
      ...baseQueryGetSessionStorageForSpecResponse,
    } as QueryGetSessionStorageForSpecResponse;
    if (
      object.sessionStorageForSpec !== undefined &&
      object.sessionStorageForSpec !== null
    ) {
      message.sessionStorageForSpec = SessionStorageForSpec.fromJSON(
        object.sessionStorageForSpec
      );
    } else {
      message.sessionStorageForSpec = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetSessionStorageForSpecResponse): unknown {
    const obj: any = {};
    message.sessionStorageForSpec !== undefined &&
      (obj.sessionStorageForSpec = message.sessionStorageForSpec
        ? SessionStorageForSpec.toJSON(message.sessionStorageForSpec)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetSessionStorageForSpecResponse>
  ): QueryGetSessionStorageForSpecResponse {
    const message = {
      ...baseQueryGetSessionStorageForSpecResponse,
    } as QueryGetSessionStorageForSpecResponse;
    if (
      object.sessionStorageForSpec !== undefined &&
      object.sessionStorageForSpec !== null
    ) {
      message.sessionStorageForSpec = SessionStorageForSpec.fromPartial(
        object.sessionStorageForSpec
      );
    } else {
      message.sessionStorageForSpec = undefined;
    }
    return message;
  },
};

const baseQueryAllSessionStorageForSpecRequest: object = {};

export const QueryAllSessionStorageForSpecRequest = {
  encode(
    message: QueryAllSessionStorageForSpecRequest,
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
  ): QueryAllSessionStorageForSpecRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllSessionStorageForSpecRequest,
    } as QueryAllSessionStorageForSpecRequest;
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

  fromJSON(object: any): QueryAllSessionStorageForSpecRequest {
    const message = {
      ...baseQueryAllSessionStorageForSpecRequest,
    } as QueryAllSessionStorageForSpecRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllSessionStorageForSpecRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllSessionStorageForSpecRequest>
  ): QueryAllSessionStorageForSpecRequest {
    const message = {
      ...baseQueryAllSessionStorageForSpecRequest,
    } as QueryAllSessionStorageForSpecRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryAllSessionStorageForSpecResponse: object = {};

export const QueryAllSessionStorageForSpecResponse = {
  encode(
    message: QueryAllSessionStorageForSpecResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.sessionStorageForSpec) {
      SessionStorageForSpec.encode(v!, writer.uint32(10).fork()).ldelim();
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
  ): QueryAllSessionStorageForSpecResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllSessionStorageForSpecResponse,
    } as QueryAllSessionStorageForSpecResponse;
    message.sessionStorageForSpec = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sessionStorageForSpec.push(
            SessionStorageForSpec.decode(reader, reader.uint32())
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

  fromJSON(object: any): QueryAllSessionStorageForSpecResponse {
    const message = {
      ...baseQueryAllSessionStorageForSpecResponse,
    } as QueryAllSessionStorageForSpecResponse;
    message.sessionStorageForSpec = [];
    if (
      object.sessionStorageForSpec !== undefined &&
      object.sessionStorageForSpec !== null
    ) {
      for (const e of object.sessionStorageForSpec) {
        message.sessionStorageForSpec.push(SessionStorageForSpec.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllSessionStorageForSpecResponse): unknown {
    const obj: any = {};
    if (message.sessionStorageForSpec) {
      obj.sessionStorageForSpec = message.sessionStorageForSpec.map((e) =>
        e ? SessionStorageForSpec.toJSON(e) : undefined
      );
    } else {
      obj.sessionStorageForSpec = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllSessionStorageForSpecResponse>
  ): QueryAllSessionStorageForSpecResponse {
    const message = {
      ...baseQueryAllSessionStorageForSpecResponse,
    } as QueryAllSessionStorageForSpecResponse;
    message.sessionStorageForSpec = [];
    if (
      object.sessionStorageForSpec !== undefined &&
      object.sessionStorageForSpec !== null
    ) {
      for (const e of object.sessionStorageForSpec) {
        message.sessionStorageForSpec.push(
          SessionStorageForSpec.fromPartial(e)
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

const baseQuerySessionStorageForAllSpecsRequest: object = { blockNum: 0 };

export const QuerySessionStorageForAllSpecsRequest = {
  encode(
    message: QuerySessionStorageForAllSpecsRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.blockNum !== 0) {
      writer.uint32(8).uint64(message.blockNum);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QuerySessionStorageForAllSpecsRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQuerySessionStorageForAllSpecsRequest,
    } as QuerySessionStorageForAllSpecsRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.blockNum = longToNumber(reader.uint64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QuerySessionStorageForAllSpecsRequest {
    const message = {
      ...baseQuerySessionStorageForAllSpecsRequest,
    } as QuerySessionStorageForAllSpecsRequest;
    if (object.blockNum !== undefined && object.blockNum !== null) {
      message.blockNum = Number(object.blockNum);
    } else {
      message.blockNum = 0;
    }
    return message;
  },

  toJSON(message: QuerySessionStorageForAllSpecsRequest): unknown {
    const obj: any = {};
    message.blockNum !== undefined && (obj.blockNum = message.blockNum);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QuerySessionStorageForAllSpecsRequest>
  ): QuerySessionStorageForAllSpecsRequest {
    const message = {
      ...baseQuerySessionStorageForAllSpecsRequest,
    } as QuerySessionStorageForAllSpecsRequest;
    if (object.blockNum !== undefined && object.blockNum !== null) {
      message.blockNum = object.blockNum;
    } else {
      message.blockNum = 0;
    }
    return message;
  },
};

const baseQuerySessionStorageForAllSpecsResponse: object = {};

export const QuerySessionStorageForAllSpecsResponse = {
  encode(
    message: QuerySessionStorageForAllSpecsResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.servicers !== undefined) {
      StakeStorage.encode(message.servicers, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QuerySessionStorageForAllSpecsResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQuerySessionStorageForAllSpecsResponse,
    } as QuerySessionStorageForAllSpecsResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.servicers = StakeStorage.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QuerySessionStorageForAllSpecsResponse {
    const message = {
      ...baseQuerySessionStorageForAllSpecsResponse,
    } as QuerySessionStorageForAllSpecsResponse;
    if (object.servicers !== undefined && object.servicers !== null) {
      message.servicers = StakeStorage.fromJSON(object.servicers);
    } else {
      message.servicers = undefined;
    }
    return message;
  },

  toJSON(message: QuerySessionStorageForAllSpecsResponse): unknown {
    const obj: any = {};
    message.servicers !== undefined &&
      (obj.servicers = message.servicers
        ? StakeStorage.toJSON(message.servicers)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QuerySessionStorageForAllSpecsResponse>
  ): QuerySessionStorageForAllSpecsResponse {
    const message = {
      ...baseQuerySessionStorageForAllSpecsResponse,
    } as QuerySessionStorageForAllSpecsResponse;
    if (object.servicers !== undefined && object.servicers !== null) {
      message.servicers = StakeStorage.fromPartial(object.servicers);
    } else {
      message.servicers = undefined;
    }
    return message;
  },
};

const baseQueryAllSessionStoragesForSpecRequest: object = { specName: "" };

export const QueryAllSessionStoragesForSpecRequest = {
  encode(
    message: QueryAllSessionStoragesForSpecRequest,
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
  ): QueryAllSessionStoragesForSpecRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllSessionStoragesForSpecRequest,
    } as QueryAllSessionStoragesForSpecRequest;
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

  fromJSON(object: any): QueryAllSessionStoragesForSpecRequest {
    const message = {
      ...baseQueryAllSessionStoragesForSpecRequest,
    } as QueryAllSessionStoragesForSpecRequest;
    if (object.specName !== undefined && object.specName !== null) {
      message.specName = String(object.specName);
    } else {
      message.specName = "";
    }
    return message;
  },

  toJSON(message: QueryAllSessionStoragesForSpecRequest): unknown {
    const obj: any = {};
    message.specName !== undefined && (obj.specName = message.specName);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllSessionStoragesForSpecRequest>
  ): QueryAllSessionStoragesForSpecRequest {
    const message = {
      ...baseQueryAllSessionStoragesForSpecRequest,
    } as QueryAllSessionStoragesForSpecRequest;
    if (object.specName !== undefined && object.specName !== null) {
      message.specName = object.specName;
    } else {
      message.specName = "";
    }
    return message;
  },
};

const baseQueryAllSessionStoragesForSpecResponse: object = {};

export const QueryAllSessionStoragesForSpecResponse = {
  encode(
    message: QueryAllSessionStoragesForSpecResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.storages) {
      SessionStorageForSpec.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryAllSessionStoragesForSpecResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllSessionStoragesForSpecResponse,
    } as QueryAllSessionStoragesForSpecResponse;
    message.storages = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.storages.push(
            SessionStorageForSpec.decode(reader, reader.uint32())
          );
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryAllSessionStoragesForSpecResponse {
    const message = {
      ...baseQueryAllSessionStoragesForSpecResponse,
    } as QueryAllSessionStoragesForSpecResponse;
    message.storages = [];
    if (object.storages !== undefined && object.storages !== null) {
      for (const e of object.storages) {
        message.storages.push(SessionStorageForSpec.fromJSON(e));
      }
    }
    return message;
  },

  toJSON(message: QueryAllSessionStoragesForSpecResponse): unknown {
    const obj: any = {};
    if (message.storages) {
      obj.storages = message.storages.map((e) =>
        e ? SessionStorageForSpec.toJSON(e) : undefined
      );
    } else {
      obj.storages = [];
    }
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllSessionStoragesForSpecResponse>
  ): QueryAllSessionStoragesForSpecResponse {
    const message = {
      ...baseQueryAllSessionStoragesForSpecResponse,
    } as QueryAllSessionStoragesForSpecResponse;
    message.storages = [];
    if (object.storages !== undefined && object.storages !== null) {
      for (const e of object.storages) {
        message.storages.push(SessionStorageForSpec.fromPartial(e));
      }
    }
    return message;
  },
};

const baseQueryGetEarliestSessionStartRequest: object = {};

export const QueryGetEarliestSessionStartRequest = {
  encode(
    _: QueryGetEarliestSessionStartRequest,
    writer: Writer = Writer.create()
  ): Writer {
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetEarliestSessionStartRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetEarliestSessionStartRequest,
    } as QueryGetEarliestSessionStartRequest;
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

  fromJSON(_: any): QueryGetEarliestSessionStartRequest {
    const message = {
      ...baseQueryGetEarliestSessionStartRequest,
    } as QueryGetEarliestSessionStartRequest;
    return message;
  },

  toJSON(_: QueryGetEarliestSessionStartRequest): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(
    _: DeepPartial<QueryGetEarliestSessionStartRequest>
  ): QueryGetEarliestSessionStartRequest {
    const message = {
      ...baseQueryGetEarliestSessionStartRequest,
    } as QueryGetEarliestSessionStartRequest;
    return message;
  },
};

const baseQueryGetEarliestSessionStartResponse: object = {};

export const QueryGetEarliestSessionStartResponse = {
  encode(
    message: QueryGetEarliestSessionStartResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.EarliestSessionStart !== undefined) {
      EarliestSessionStart.encode(
        message.EarliestSessionStart,
        writer.uint32(10).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetEarliestSessionStartResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetEarliestSessionStartResponse,
    } as QueryGetEarliestSessionStartResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.EarliestSessionStart = EarliestSessionStart.decode(
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

  fromJSON(object: any): QueryGetEarliestSessionStartResponse {
    const message = {
      ...baseQueryGetEarliestSessionStartResponse,
    } as QueryGetEarliestSessionStartResponse;
    if (
      object.EarliestSessionStart !== undefined &&
      object.EarliestSessionStart !== null
    ) {
      message.EarliestSessionStart = EarliestSessionStart.fromJSON(
        object.EarliestSessionStart
      );
    } else {
      message.EarliestSessionStart = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetEarliestSessionStartResponse): unknown {
    const obj: any = {};
    message.EarliestSessionStart !== undefined &&
      (obj.EarliestSessionStart = message.EarliestSessionStart
        ? EarliestSessionStart.toJSON(message.EarliestSessionStart)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetEarliestSessionStartResponse>
  ): QueryGetEarliestSessionStartResponse {
    const message = {
      ...baseQueryGetEarliestSessionStartResponse,
    } as QueryGetEarliestSessionStartResponse;
    if (
      object.EarliestSessionStart !== undefined &&
      object.EarliestSessionStart !== null
    ) {
      message.EarliestSessionStart = EarliestSessionStart.fromPartial(
        object.EarliestSessionStart
      );
    } else {
      message.EarliestSessionStart = undefined;
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
  /** Queries a list of GetPairing items. */
  GetPairing(request: QueryGetPairingRequest): Promise<QueryGetPairingResponse>;
  /** Queries a CurrentSessionStart by index. */
  CurrentSessionStart(
    request: QueryGetCurrentSessionStartRequest
  ): Promise<QueryGetCurrentSessionStartResponse>;
  /** Queries a PreviousSessionBlocks by index. */
  PreviousSessionBlocks(
    request: QueryGetPreviousSessionBlocksRequest
  ): Promise<QueryGetPreviousSessionBlocksResponse>;
  /** Queries a SessionStorageForSpec by index. */
  SessionStorageForSpec(
    request: QueryGetSessionStorageForSpecRequest
  ): Promise<QueryGetSessionStorageForSpecResponse>;
  /** Queries a list of SessionStorageForSpec items. */
  SessionStorageForSpecAll(
    request: QueryAllSessionStorageForSpecRequest
  ): Promise<QueryAllSessionStorageForSpecResponse>;
  /** Queries a list of SessionStorageForAllSpecs items. */
  SessionStorageForAllSpecs(
    request: QuerySessionStorageForAllSpecsRequest
  ): Promise<QuerySessionStorageForAllSpecsResponse>;
  /** Queries a list of AllSessionStoragesForSpec items. */
  AllSessionStoragesForSpec(
    request: QueryAllSessionStoragesForSpecRequest
  ): Promise<QueryAllSessionStoragesForSpecResponse>;
  /** Queries a EarliestSessionStart by index. */
  EarliestSessionStart(
    request: QueryGetEarliestSessionStartRequest
  ): Promise<QueryGetEarliestSessionStartResponse>;
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

  GetPairing(
    request: QueryGetPairingRequest
  ): Promise<QueryGetPairingResponse> {
    const data = QueryGetPairingRequest.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.servicer.Query",
      "GetPairing",
      data
    );
    return promise.then((data) =>
      QueryGetPairingResponse.decode(new Reader(data))
    );
  }

  CurrentSessionStart(
    request: QueryGetCurrentSessionStartRequest
  ): Promise<QueryGetCurrentSessionStartResponse> {
    const data = QueryGetCurrentSessionStartRequest.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.servicer.Query",
      "CurrentSessionStart",
      data
    );
    return promise.then((data) =>
      QueryGetCurrentSessionStartResponse.decode(new Reader(data))
    );
  }

  PreviousSessionBlocks(
    request: QueryGetPreviousSessionBlocksRequest
  ): Promise<QueryGetPreviousSessionBlocksResponse> {
    const data = QueryGetPreviousSessionBlocksRequest.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.servicer.Query",
      "PreviousSessionBlocks",
      data
    );
    return promise.then((data) =>
      QueryGetPreviousSessionBlocksResponse.decode(new Reader(data))
    );
  }

  SessionStorageForSpec(
    request: QueryGetSessionStorageForSpecRequest
  ): Promise<QueryGetSessionStorageForSpecResponse> {
    const data = QueryGetSessionStorageForSpecRequest.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.servicer.Query",
      "SessionStorageForSpec",
      data
    );
    return promise.then((data) =>
      QueryGetSessionStorageForSpecResponse.decode(new Reader(data))
    );
  }

  SessionStorageForSpecAll(
    request: QueryAllSessionStorageForSpecRequest
  ): Promise<QueryAllSessionStorageForSpecResponse> {
    const data = QueryAllSessionStorageForSpecRequest.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.servicer.Query",
      "SessionStorageForSpecAll",
      data
    );
    return promise.then((data) =>
      QueryAllSessionStorageForSpecResponse.decode(new Reader(data))
    );
  }

  SessionStorageForAllSpecs(
    request: QuerySessionStorageForAllSpecsRequest
  ): Promise<QuerySessionStorageForAllSpecsResponse> {
    const data = QuerySessionStorageForAllSpecsRequest.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.servicer.Query",
      "SessionStorageForAllSpecs",
      data
    );
    return promise.then((data) =>
      QuerySessionStorageForAllSpecsResponse.decode(new Reader(data))
    );
  }

  AllSessionStoragesForSpec(
    request: QueryAllSessionStoragesForSpecRequest
  ): Promise<QueryAllSessionStoragesForSpecResponse> {
    const data = QueryAllSessionStoragesForSpecRequest.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.servicer.Query",
      "AllSessionStoragesForSpec",
      data
    );
    return promise.then((data) =>
      QueryAllSessionStoragesForSpecResponse.decode(new Reader(data))
    );
  }

  EarliestSessionStart(
    request: QueryGetEarliestSessionStartRequest
  ): Promise<QueryGetEarliestSessionStartResponse> {
    const data = QueryGetEarliestSessionStartRequest.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.servicer.Query",
      "EarliestSessionStart",
      data
    );
    return promise.then((data) =>
      QueryGetEarliestSessionStartResponse.decode(new Reader(data))
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

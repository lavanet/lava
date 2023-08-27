/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { PageRequest, PageResponse } from "../../../cosmos/base/query/v1beta1/pagination";
import { EpochDetails } from "./epoch_details";
import { FixatedParams } from "./fixated_params";
import { Params } from "./params";
import { StakeStorage } from "./stake_storage";

export const protobufPackage = "lavanet.lava.epochstorage";

/** QueryParamsRequest is request type for the Query/Params RPC method. */
export interface QueryParamsRequest {
}

/** QueryParamsResponse is response type for the Query/Params RPC method. */
export interface QueryParamsResponse {
  /** params holds all the parameters of this module. */
  params?: Params;
}

export interface QueryGetStakeStorageRequest {
  index: string;
}

export interface QueryGetStakeStorageResponse {
  stakeStorage?: StakeStorage;
}

export interface QueryAllStakeStorageRequest {
  pagination?: PageRequest;
}

export interface QueryAllStakeStorageResponse {
  stakeStorage: StakeStorage[];
  pagination?: PageResponse;
}

export interface QueryGetEpochDetailsRequest {
}

export interface QueryGetEpochDetailsResponse {
  EpochDetails?: EpochDetails;
}

export interface QueryGetFixatedParamsRequest {
  index: string;
}

export interface QueryGetFixatedParamsResponse {
  fixatedParams?: FixatedParams;
}

export interface QueryAllFixatedParamsRequest {
  pagination?: PageRequest;
}

export interface QueryAllFixatedParamsResponse {
  fixatedParams: FixatedParams[];
  pagination?: PageResponse;
}

function createBaseQueryParamsRequest(): QueryParamsRequest {
  return {};
}

export const QueryParamsRequest = {
  encode(_: QueryParamsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryParamsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryParamsRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(_: any): QueryParamsRequest {
    return {};
  },

  toJSON(_: QueryParamsRequest): unknown {
    const obj: any = {};
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryParamsRequest>, I>>(base?: I): QueryParamsRequest {
    return QueryParamsRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryParamsRequest>, I>>(_: I): QueryParamsRequest {
    const message = createBaseQueryParamsRequest();
    return message;
  },
};

function createBaseQueryParamsResponse(): QueryParamsResponse {
  return { params: undefined };
}

export const QueryParamsResponse = {
  encode(message: QueryParamsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryParamsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryParamsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.params = Params.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryParamsResponse {
    return { params: isSet(object.params) ? Params.fromJSON(object.params) : undefined };
  },

  toJSON(message: QueryParamsResponse): unknown {
    const obj: any = {};
    message.params !== undefined && (obj.params = message.params ? Params.toJSON(message.params) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryParamsResponse>, I>>(base?: I): QueryParamsResponse {
    return QueryParamsResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryParamsResponse>, I>>(object: I): QueryParamsResponse {
    const message = createBaseQueryParamsResponse();
    message.params = (object.params !== undefined && object.params !== null)
      ? Params.fromPartial(object.params)
      : undefined;
    return message;
  },
};

function createBaseQueryGetStakeStorageRequest(): QueryGetStakeStorageRequest {
  return { index: "" };
}

export const QueryGetStakeStorageRequest = {
  encode(message: QueryGetStakeStorageRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetStakeStorageRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetStakeStorageRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.index = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryGetStakeStorageRequest {
    return { index: isSet(object.index) ? String(object.index) : "" };
  },

  toJSON(message: QueryGetStakeStorageRequest): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryGetStakeStorageRequest>, I>>(base?: I): QueryGetStakeStorageRequest {
    return QueryGetStakeStorageRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetStakeStorageRequest>, I>>(object: I): QueryGetStakeStorageRequest {
    const message = createBaseQueryGetStakeStorageRequest();
    message.index = object.index ?? "";
    return message;
  },
};

function createBaseQueryGetStakeStorageResponse(): QueryGetStakeStorageResponse {
  return { stakeStorage: undefined };
}

export const QueryGetStakeStorageResponse = {
  encode(message: QueryGetStakeStorageResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.stakeStorage !== undefined) {
      StakeStorage.encode(message.stakeStorage, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetStakeStorageResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetStakeStorageResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.stakeStorage = StakeStorage.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryGetStakeStorageResponse {
    return { stakeStorage: isSet(object.stakeStorage) ? StakeStorage.fromJSON(object.stakeStorage) : undefined };
  },

  toJSON(message: QueryGetStakeStorageResponse): unknown {
    const obj: any = {};
    message.stakeStorage !== undefined &&
      (obj.stakeStorage = message.stakeStorage ? StakeStorage.toJSON(message.stakeStorage) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryGetStakeStorageResponse>, I>>(base?: I): QueryGetStakeStorageResponse {
    return QueryGetStakeStorageResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetStakeStorageResponse>, I>>(object: I): QueryGetStakeStorageResponse {
    const message = createBaseQueryGetStakeStorageResponse();
    message.stakeStorage = (object.stakeStorage !== undefined && object.stakeStorage !== null)
      ? StakeStorage.fromPartial(object.stakeStorage)
      : undefined;
    return message;
  },
};

function createBaseQueryAllStakeStorageRequest(): QueryAllStakeStorageRequest {
  return { pagination: undefined };
}

export const QueryAllStakeStorageRequest = {
  encode(message: QueryAllStakeStorageRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllStakeStorageRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAllStakeStorageRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.pagination = PageRequest.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryAllStakeStorageRequest {
    return { pagination: isSet(object.pagination) ? PageRequest.fromJSON(object.pagination) : undefined };
  },

  toJSON(message: QueryAllStakeStorageRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination ? PageRequest.toJSON(message.pagination) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryAllStakeStorageRequest>, I>>(base?: I): QueryAllStakeStorageRequest {
    return QueryAllStakeStorageRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryAllStakeStorageRequest>, I>>(object: I): QueryAllStakeStorageRequest {
    const message = createBaseQueryAllStakeStorageRequest();
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageRequest.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseQueryAllStakeStorageResponse(): QueryAllStakeStorageResponse {
  return { stakeStorage: [], pagination: undefined };
}

export const QueryAllStakeStorageResponse = {
  encode(message: QueryAllStakeStorageResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.stakeStorage) {
      StakeStorage.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllStakeStorageResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAllStakeStorageResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.stakeStorage.push(StakeStorage.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.pagination = PageResponse.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryAllStakeStorageResponse {
    return {
      stakeStorage: Array.isArray(object?.stakeStorage)
        ? object.stakeStorage.map((e: any) => StakeStorage.fromJSON(e))
        : [],
      pagination: isSet(object.pagination) ? PageResponse.fromJSON(object.pagination) : undefined,
    };
  },

  toJSON(message: QueryAllStakeStorageResponse): unknown {
    const obj: any = {};
    if (message.stakeStorage) {
      obj.stakeStorage = message.stakeStorage.map((e) => e ? StakeStorage.toJSON(e) : undefined);
    } else {
      obj.stakeStorage = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination ? PageResponse.toJSON(message.pagination) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryAllStakeStorageResponse>, I>>(base?: I): QueryAllStakeStorageResponse {
    return QueryAllStakeStorageResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryAllStakeStorageResponse>, I>>(object: I): QueryAllStakeStorageResponse {
    const message = createBaseQueryAllStakeStorageResponse();
    message.stakeStorage = object.stakeStorage?.map((e) => StakeStorage.fromPartial(e)) || [];
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageResponse.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseQueryGetEpochDetailsRequest(): QueryGetEpochDetailsRequest {
  return {};
}

export const QueryGetEpochDetailsRequest = {
  encode(_: QueryGetEpochDetailsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetEpochDetailsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetEpochDetailsRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(_: any): QueryGetEpochDetailsRequest {
    return {};
  },

  toJSON(_: QueryGetEpochDetailsRequest): unknown {
    const obj: any = {};
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryGetEpochDetailsRequest>, I>>(base?: I): QueryGetEpochDetailsRequest {
    return QueryGetEpochDetailsRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetEpochDetailsRequest>, I>>(_: I): QueryGetEpochDetailsRequest {
    const message = createBaseQueryGetEpochDetailsRequest();
    return message;
  },
};

function createBaseQueryGetEpochDetailsResponse(): QueryGetEpochDetailsResponse {
  return { EpochDetails: undefined };
}

export const QueryGetEpochDetailsResponse = {
  encode(message: QueryGetEpochDetailsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.EpochDetails !== undefined) {
      EpochDetails.encode(message.EpochDetails, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetEpochDetailsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetEpochDetailsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.EpochDetails = EpochDetails.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryGetEpochDetailsResponse {
    return { EpochDetails: isSet(object.EpochDetails) ? EpochDetails.fromJSON(object.EpochDetails) : undefined };
  },

  toJSON(message: QueryGetEpochDetailsResponse): unknown {
    const obj: any = {};
    message.EpochDetails !== undefined &&
      (obj.EpochDetails = message.EpochDetails ? EpochDetails.toJSON(message.EpochDetails) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryGetEpochDetailsResponse>, I>>(base?: I): QueryGetEpochDetailsResponse {
    return QueryGetEpochDetailsResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetEpochDetailsResponse>, I>>(object: I): QueryGetEpochDetailsResponse {
    const message = createBaseQueryGetEpochDetailsResponse();
    message.EpochDetails = (object.EpochDetails !== undefined && object.EpochDetails !== null)
      ? EpochDetails.fromPartial(object.EpochDetails)
      : undefined;
    return message;
  },
};

function createBaseQueryGetFixatedParamsRequest(): QueryGetFixatedParamsRequest {
  return { index: "" };
}

export const QueryGetFixatedParamsRequest = {
  encode(message: QueryGetFixatedParamsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetFixatedParamsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetFixatedParamsRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.index = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryGetFixatedParamsRequest {
    return { index: isSet(object.index) ? String(object.index) : "" };
  },

  toJSON(message: QueryGetFixatedParamsRequest): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryGetFixatedParamsRequest>, I>>(base?: I): QueryGetFixatedParamsRequest {
    return QueryGetFixatedParamsRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetFixatedParamsRequest>, I>>(object: I): QueryGetFixatedParamsRequest {
    const message = createBaseQueryGetFixatedParamsRequest();
    message.index = object.index ?? "";
    return message;
  },
};

function createBaseQueryGetFixatedParamsResponse(): QueryGetFixatedParamsResponse {
  return { fixatedParams: undefined };
}

export const QueryGetFixatedParamsResponse = {
  encode(message: QueryGetFixatedParamsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.fixatedParams !== undefined) {
      FixatedParams.encode(message.fixatedParams, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetFixatedParamsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetFixatedParamsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.fixatedParams = FixatedParams.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryGetFixatedParamsResponse {
    return { fixatedParams: isSet(object.fixatedParams) ? FixatedParams.fromJSON(object.fixatedParams) : undefined };
  },

  toJSON(message: QueryGetFixatedParamsResponse): unknown {
    const obj: any = {};
    message.fixatedParams !== undefined &&
      (obj.fixatedParams = message.fixatedParams ? FixatedParams.toJSON(message.fixatedParams) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryGetFixatedParamsResponse>, I>>(base?: I): QueryGetFixatedParamsResponse {
    return QueryGetFixatedParamsResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetFixatedParamsResponse>, I>>(
    object: I,
  ): QueryGetFixatedParamsResponse {
    const message = createBaseQueryGetFixatedParamsResponse();
    message.fixatedParams = (object.fixatedParams !== undefined && object.fixatedParams !== null)
      ? FixatedParams.fromPartial(object.fixatedParams)
      : undefined;
    return message;
  },
};

function createBaseQueryAllFixatedParamsRequest(): QueryAllFixatedParamsRequest {
  return { pagination: undefined };
}

export const QueryAllFixatedParamsRequest = {
  encode(message: QueryAllFixatedParamsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllFixatedParamsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAllFixatedParamsRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.pagination = PageRequest.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryAllFixatedParamsRequest {
    return { pagination: isSet(object.pagination) ? PageRequest.fromJSON(object.pagination) : undefined };
  },

  toJSON(message: QueryAllFixatedParamsRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination ? PageRequest.toJSON(message.pagination) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryAllFixatedParamsRequest>, I>>(base?: I): QueryAllFixatedParamsRequest {
    return QueryAllFixatedParamsRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryAllFixatedParamsRequest>, I>>(object: I): QueryAllFixatedParamsRequest {
    const message = createBaseQueryAllFixatedParamsRequest();
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageRequest.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseQueryAllFixatedParamsResponse(): QueryAllFixatedParamsResponse {
  return { fixatedParams: [], pagination: undefined };
}

export const QueryAllFixatedParamsResponse = {
  encode(message: QueryAllFixatedParamsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.fixatedParams) {
      FixatedParams.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllFixatedParamsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAllFixatedParamsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.fixatedParams.push(FixatedParams.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.pagination = PageResponse.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryAllFixatedParamsResponse {
    return {
      fixatedParams: Array.isArray(object?.fixatedParams)
        ? object.fixatedParams.map((e: any) => FixatedParams.fromJSON(e))
        : [],
      pagination: isSet(object.pagination) ? PageResponse.fromJSON(object.pagination) : undefined,
    };
  },

  toJSON(message: QueryAllFixatedParamsResponse): unknown {
    const obj: any = {};
    if (message.fixatedParams) {
      obj.fixatedParams = message.fixatedParams.map((e) => e ? FixatedParams.toJSON(e) : undefined);
    } else {
      obj.fixatedParams = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination ? PageResponse.toJSON(message.pagination) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryAllFixatedParamsResponse>, I>>(base?: I): QueryAllFixatedParamsResponse {
    return QueryAllFixatedParamsResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryAllFixatedParamsResponse>, I>>(
    object: I,
  ): QueryAllFixatedParamsResponse {
    const message = createBaseQueryAllFixatedParamsResponse();
    message.fixatedParams = object.fixatedParams?.map((e) => FixatedParams.fromPartial(e)) || [];
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageResponse.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

/** Query defines the gRPC querier service. */
export interface Query {
  /** Parameters queries the parameters of the module. */
  Params(request: QueryParamsRequest): Promise<QueryParamsResponse>;
  /** Queries a StakeStorage by index. */
  StakeStorage(request: QueryGetStakeStorageRequest): Promise<QueryGetStakeStorageResponse>;
  /** Queries a list of StakeStorage items. */
  StakeStorageAll(request: QueryAllStakeStorageRequest): Promise<QueryAllStakeStorageResponse>;
  /** Queries a EpochDetails by index. */
  EpochDetails(request: QueryGetEpochDetailsRequest): Promise<QueryGetEpochDetailsResponse>;
  /** Queries a FixatedParams by index. */
  FixatedParams(request: QueryGetFixatedParamsRequest): Promise<QueryGetFixatedParamsResponse>;
  /** Queries a list of FixatedParams items. */
  FixatedParamsAll(request: QueryAllFixatedParamsRequest): Promise<QueryAllFixatedParamsResponse>;
}

export class QueryClientImpl implements Query {
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || "lavanet.lava.epochstorage.Query";
    this.rpc = rpc;
    this.Params = this.Params.bind(this);
    this.StakeStorage = this.StakeStorage.bind(this);
    this.StakeStorageAll = this.StakeStorageAll.bind(this);
    this.EpochDetails = this.EpochDetails.bind(this);
    this.FixatedParams = this.FixatedParams.bind(this);
    this.FixatedParamsAll = this.FixatedParamsAll.bind(this);
  }
  Params(request: QueryParamsRequest): Promise<QueryParamsResponse> {
    const data = QueryParamsRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "Params", data);
    return promise.then((data) => QueryParamsResponse.decode(_m0.Reader.create(data)));
  }

  StakeStorage(request: QueryGetStakeStorageRequest): Promise<QueryGetStakeStorageResponse> {
    const data = QueryGetStakeStorageRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "StakeStorage", data);
    return promise.then((data) => QueryGetStakeStorageResponse.decode(_m0.Reader.create(data)));
  }

  StakeStorageAll(request: QueryAllStakeStorageRequest): Promise<QueryAllStakeStorageResponse> {
    const data = QueryAllStakeStorageRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "StakeStorageAll", data);
    return promise.then((data) => QueryAllStakeStorageResponse.decode(_m0.Reader.create(data)));
  }

  EpochDetails(request: QueryGetEpochDetailsRequest): Promise<QueryGetEpochDetailsResponse> {
    const data = QueryGetEpochDetailsRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "EpochDetails", data);
    return promise.then((data) => QueryGetEpochDetailsResponse.decode(_m0.Reader.create(data)));
  }

  FixatedParams(request: QueryGetFixatedParamsRequest): Promise<QueryGetFixatedParamsResponse> {
    const data = QueryGetFixatedParamsRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "FixatedParams", data);
    return promise.then((data) => QueryGetFixatedParamsResponse.decode(_m0.Reader.create(data)));
  }

  FixatedParamsAll(request: QueryAllFixatedParamsRequest): Promise<QueryAllFixatedParamsResponse> {
    const data = QueryAllFixatedParamsRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "FixatedParamsAll", data);
    return promise.then((data) => QueryAllFixatedParamsResponse.decode(_m0.Reader.create(data)));
  }
}

interface Rpc {
  request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Long ? string | number | Long : T extends Array<infer U> ? Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

type KeysOfUnion<T> = T extends T ? keyof T : never;
export type Exact<P, I extends P> = P extends Builtin ? P
  : P & { [K in keyof P]: Exact<P[K], I[K]> } & { [K in Exclude<keyof I, KeysOfUnion<P>>]: never };

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}

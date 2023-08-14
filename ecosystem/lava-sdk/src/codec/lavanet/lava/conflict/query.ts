/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { PageRequest, PageResponse } from "../../../cosmos/base/query/v1beta1/pagination";
import { ConflictVote } from "./conflict_vote";
import { Params } from "./params";

export const protobufPackage = "lavanet.lava.conflict";

/** QueryParamsRequest is request type for the Query/Params RPC method. */
export interface QueryParamsRequest {
}

/** QueryParamsResponse is response type for the Query/Params RPC method. */
export interface QueryParamsResponse {
  /** params holds all the parameters of this module. */
  params?: Params;
}

export interface QueryGetConflictVoteRequest {
  index: string;
}

export interface QueryGetConflictVoteResponse {
  conflictVote?: ConflictVote;
}

export interface QueryAllConflictVoteRequest {
  pagination?: PageRequest;
}

export interface QueryAllConflictVoteResponse {
  conflictVote: ConflictVote[];
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

function createBaseQueryGetConflictVoteRequest(): QueryGetConflictVoteRequest {
  return { index: "" };
}

export const QueryGetConflictVoteRequest = {
  encode(message: QueryGetConflictVoteRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetConflictVoteRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetConflictVoteRequest();
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

  fromJSON(object: any): QueryGetConflictVoteRequest {
    return { index: isSet(object.index) ? String(object.index) : "" };
  },

  toJSON(message: QueryGetConflictVoteRequest): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryGetConflictVoteRequest>, I>>(base?: I): QueryGetConflictVoteRequest {
    return QueryGetConflictVoteRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetConflictVoteRequest>, I>>(object: I): QueryGetConflictVoteRequest {
    const message = createBaseQueryGetConflictVoteRequest();
    message.index = object.index ?? "";
    return message;
  },
};

function createBaseQueryGetConflictVoteResponse(): QueryGetConflictVoteResponse {
  return { conflictVote: undefined };
}

export const QueryGetConflictVoteResponse = {
  encode(message: QueryGetConflictVoteResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.conflictVote !== undefined) {
      ConflictVote.encode(message.conflictVote, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetConflictVoteResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetConflictVoteResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.conflictVote = ConflictVote.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryGetConflictVoteResponse {
    return { conflictVote: isSet(object.conflictVote) ? ConflictVote.fromJSON(object.conflictVote) : undefined };
  },

  toJSON(message: QueryGetConflictVoteResponse): unknown {
    const obj: any = {};
    message.conflictVote !== undefined &&
      (obj.conflictVote = message.conflictVote ? ConflictVote.toJSON(message.conflictVote) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryGetConflictVoteResponse>, I>>(base?: I): QueryGetConflictVoteResponse {
    return QueryGetConflictVoteResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetConflictVoteResponse>, I>>(object: I): QueryGetConflictVoteResponse {
    const message = createBaseQueryGetConflictVoteResponse();
    message.conflictVote = (object.conflictVote !== undefined && object.conflictVote !== null)
      ? ConflictVote.fromPartial(object.conflictVote)
      : undefined;
    return message;
  },
};

function createBaseQueryAllConflictVoteRequest(): QueryAllConflictVoteRequest {
  return { pagination: undefined };
}

export const QueryAllConflictVoteRequest = {
  encode(message: QueryAllConflictVoteRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllConflictVoteRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAllConflictVoteRequest();
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

  fromJSON(object: any): QueryAllConflictVoteRequest {
    return { pagination: isSet(object.pagination) ? PageRequest.fromJSON(object.pagination) : undefined };
  },

  toJSON(message: QueryAllConflictVoteRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination ? PageRequest.toJSON(message.pagination) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryAllConflictVoteRequest>, I>>(base?: I): QueryAllConflictVoteRequest {
    return QueryAllConflictVoteRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryAllConflictVoteRequest>, I>>(object: I): QueryAllConflictVoteRequest {
    const message = createBaseQueryAllConflictVoteRequest();
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageRequest.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseQueryAllConflictVoteResponse(): QueryAllConflictVoteResponse {
  return { conflictVote: [], pagination: undefined };
}

export const QueryAllConflictVoteResponse = {
  encode(message: QueryAllConflictVoteResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.conflictVote) {
      ConflictVote.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllConflictVoteResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAllConflictVoteResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.conflictVote.push(ConflictVote.decode(reader, reader.uint32()));
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

  fromJSON(object: any): QueryAllConflictVoteResponse {
    return {
      conflictVote: Array.isArray(object?.conflictVote)
        ? object.conflictVote.map((e: any) => ConflictVote.fromJSON(e))
        : [],
      pagination: isSet(object.pagination) ? PageResponse.fromJSON(object.pagination) : undefined,
    };
  },

  toJSON(message: QueryAllConflictVoteResponse): unknown {
    const obj: any = {};
    if (message.conflictVote) {
      obj.conflictVote = message.conflictVote.map((e) => e ? ConflictVote.toJSON(e) : undefined);
    } else {
      obj.conflictVote = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination ? PageResponse.toJSON(message.pagination) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryAllConflictVoteResponse>, I>>(base?: I): QueryAllConflictVoteResponse {
    return QueryAllConflictVoteResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryAllConflictVoteResponse>, I>>(object: I): QueryAllConflictVoteResponse {
    const message = createBaseQueryAllConflictVoteResponse();
    message.conflictVote = object.conflictVote?.map((e) => ConflictVote.fromPartial(e)) || [];
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
  /** Queries a ConflictVote by index. */
  ConflictVote(request: QueryGetConflictVoteRequest): Promise<QueryGetConflictVoteResponse>;
  /** Queries a list of ConflictVote items. */
  ConflictVoteAll(request: QueryAllConflictVoteRequest): Promise<QueryAllConflictVoteResponse>;
}

export class QueryClientImpl implements Query {
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || "lavanet.lava.conflict.Query";
    this.rpc = rpc;
    this.Params = this.Params.bind(this);
    this.ConflictVote = this.ConflictVote.bind(this);
    this.ConflictVoteAll = this.ConflictVoteAll.bind(this);
  }
  Params(request: QueryParamsRequest): Promise<QueryParamsResponse> {
    const data = QueryParamsRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "Params", data);
    return promise.then((data) => QueryParamsResponse.decode(_m0.Reader.create(data)));
  }

  ConflictVote(request: QueryGetConflictVoteRequest): Promise<QueryGetConflictVoteResponse> {
    const data = QueryGetConflictVoteRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "ConflictVote", data);
    return promise.then((data) => QueryGetConflictVoteResponse.decode(_m0.Reader.create(data)));
  }

  ConflictVoteAll(request: QueryAllConflictVoteRequest): Promise<QueryAllConflictVoteResponse> {
    const data = QueryAllConflictVoteRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "ConflictVoteAll", data);
    return promise.then((data) => QueryAllConflictVoteResponse.decode(_m0.Reader.create(data)));
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

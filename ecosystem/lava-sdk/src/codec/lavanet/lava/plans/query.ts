/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Coin } from "../../../cosmos/base/v1beta1/coin";
import { Params } from "./params";
import { Plan } from "./plan";

export const protobufPackage = "lavanet.lava.plans";

/** QueryParamsRequest is request type for the Query/Params RPC method. */
export interface QueryParamsRequest {
}

/** QueryParamsResponse is response type for the Query/Params RPC method. */
export interface QueryParamsResponse {
  /** params holds all the parameters of this module. */
  params?: Params;
}

export interface QueryListRequest {
}

export interface QueryListResponse {
  plansInfo: ListInfoStruct[];
}

export interface ListInfoStruct {
  index: string;
  description: string;
  price?: Coin;
}

export interface QueryInfoRequest {
  planIndex: string;
}

export interface QueryInfoResponse {
  planInfo?: Plan;
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

function createBaseQueryListRequest(): QueryListRequest {
  return {};
}

export const QueryListRequest = {
  encode(_: QueryListRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryListRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryListRequest();
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

  fromJSON(_: any): QueryListRequest {
    return {};
  },

  toJSON(_: QueryListRequest): unknown {
    const obj: any = {};
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryListRequest>, I>>(base?: I): QueryListRequest {
    return QueryListRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryListRequest>, I>>(_: I): QueryListRequest {
    const message = createBaseQueryListRequest();
    return message;
  },
};

function createBaseQueryListResponse(): QueryListResponse {
  return { plansInfo: [] };
}

export const QueryListResponse = {
  encode(message: QueryListResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.plansInfo) {
      ListInfoStruct.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryListResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryListResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.plansInfo.push(ListInfoStruct.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryListResponse {
    return {
      plansInfo: Array.isArray(object?.plansInfo) ? object.plansInfo.map((e: any) => ListInfoStruct.fromJSON(e)) : [],
    };
  },

  toJSON(message: QueryListResponse): unknown {
    const obj: any = {};
    if (message.plansInfo) {
      obj.plansInfo = message.plansInfo.map((e) => e ? ListInfoStruct.toJSON(e) : undefined);
    } else {
      obj.plansInfo = [];
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryListResponse>, I>>(base?: I): QueryListResponse {
    return QueryListResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryListResponse>, I>>(object: I): QueryListResponse {
    const message = createBaseQueryListResponse();
    message.plansInfo = object.plansInfo?.map((e) => ListInfoStruct.fromPartial(e)) || [];
    return message;
  },
};

function createBaseListInfoStruct(): ListInfoStruct {
  return { index: "", description: "", price: undefined };
}

export const ListInfoStruct = {
  encode(message: ListInfoStruct, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    if (message.description !== "") {
      writer.uint32(18).string(message.description);
    }
    if (message.price !== undefined) {
      Coin.encode(message.price, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListInfoStruct {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListInfoStruct();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.index = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.description = reader.string();
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.price = Coin.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListInfoStruct {
    return {
      index: isSet(object.index) ? String(object.index) : "",
      description: isSet(object.description) ? String(object.description) : "",
      price: isSet(object.price) ? Coin.fromJSON(object.price) : undefined,
    };
  },

  toJSON(message: ListInfoStruct): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    message.description !== undefined && (obj.description = message.description);
    message.price !== undefined && (obj.price = message.price ? Coin.toJSON(message.price) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<ListInfoStruct>, I>>(base?: I): ListInfoStruct {
    return ListInfoStruct.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<ListInfoStruct>, I>>(object: I): ListInfoStruct {
    const message = createBaseListInfoStruct();
    message.index = object.index ?? "";
    message.description = object.description ?? "";
    message.price = (object.price !== undefined && object.price !== null) ? Coin.fromPartial(object.price) : undefined;
    return message;
  },
};

function createBaseQueryInfoRequest(): QueryInfoRequest {
  return { planIndex: "" };
}

export const QueryInfoRequest = {
  encode(message: QueryInfoRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.planIndex !== "") {
      writer.uint32(10).string(message.planIndex);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryInfoRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryInfoRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.planIndex = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryInfoRequest {
    return { planIndex: isSet(object.planIndex) ? String(object.planIndex) : "" };
  },

  toJSON(message: QueryInfoRequest): unknown {
    const obj: any = {};
    message.planIndex !== undefined && (obj.planIndex = message.planIndex);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryInfoRequest>, I>>(base?: I): QueryInfoRequest {
    return QueryInfoRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryInfoRequest>, I>>(object: I): QueryInfoRequest {
    const message = createBaseQueryInfoRequest();
    message.planIndex = object.planIndex ?? "";
    return message;
  },
};

function createBaseQueryInfoResponse(): QueryInfoResponse {
  return { planInfo: undefined };
}

export const QueryInfoResponse = {
  encode(message: QueryInfoResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.planInfo !== undefined) {
      Plan.encode(message.planInfo, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryInfoResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryInfoResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.planInfo = Plan.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryInfoResponse {
    return { planInfo: isSet(object.planInfo) ? Plan.fromJSON(object.planInfo) : undefined };
  },

  toJSON(message: QueryInfoResponse): unknown {
    const obj: any = {};
    message.planInfo !== undefined && (obj.planInfo = message.planInfo ? Plan.toJSON(message.planInfo) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryInfoResponse>, I>>(base?: I): QueryInfoResponse {
    return QueryInfoResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryInfoResponse>, I>>(object: I): QueryInfoResponse {
    const message = createBaseQueryInfoResponse();
    message.planInfo = (object.planInfo !== undefined && object.planInfo !== null)
      ? Plan.fromPartial(object.planInfo)
      : undefined;
    return message;
  },
};

/** Query defines the gRPC querier service. */
export interface Query {
  /** Parameters queries the parameters of the module. */
  Params(request: QueryParamsRequest): Promise<QueryParamsResponse>;
  /** Queries a list of List items. */
  List(request: QueryListRequest): Promise<QueryListResponse>;
  /** Queries an Info item. */
  Info(request: QueryInfoRequest): Promise<QueryInfoResponse>;
}

export class QueryClientImpl implements Query {
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || "lavanet.lava.plans.Query";
    this.rpc = rpc;
    this.Params = this.Params.bind(this);
    this.List = this.List.bind(this);
    this.Info = this.Info.bind(this);
  }
  Params(request: QueryParamsRequest): Promise<QueryParamsResponse> {
    const data = QueryParamsRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "Params", data);
    return promise.then((data) => QueryParamsResponse.decode(_m0.Reader.create(data)));
  }

  List(request: QueryListRequest): Promise<QueryListResponse> {
    const data = QueryListRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "List", data);
    return promise.then((data) => QueryListResponse.decode(_m0.Reader.create(data)));
  }

  Info(request: QueryInfoRequest): Promise<QueryInfoResponse> {
    const data = QueryInfoRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "Info", data);
    return promise.then((data) => QueryInfoResponse.decode(_m0.Reader.create(data)));
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

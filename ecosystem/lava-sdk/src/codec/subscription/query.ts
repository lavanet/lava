/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Params } from "./params";
import { Subscription } from "./subscription";

export const protobufPackage = "lavanet.lava.subscription";

/** QueryParamsRequest is request type for the Query/Params RPC method. */
export interface QueryParamsRequest {
}

/** QueryParamsResponse is response type for the Query/Params RPC method. */
export interface QueryParamsResponse {
  /** params holds all the parameters of this module. */
  params?: Params;
}

export interface QueryCurrentRequest {
  consumer: string;
}

export interface QueryCurrentResponse {
  sub?: Subscription;
}

export interface QueryListProjectsRequest {
  subscription: string;
}

export interface QueryListProjectsResponse {
  projects: string[];
}

export interface QueryListRequest {
}

export interface QueryListResponse {
  subsInfo: ListInfoStruct[];
}

export interface ListInfoStruct {
  /** beneficiary consumer */
  consumer: string;
  /** plan assosiated with the subscription */
  plan: string;
  /** total duration in months (purchase/renewal) */
  durationTotal: Long;
  /** remaining duration in months */
  durationLeft: Long;
  /** upcoming expiry (of current month) in unix time */
  monthExpiry: Long;
  /** total CU allowance per month */
  monthCuTotal: Long;
  /** remaining CU allowance this month */
  monthCuLeft: Long;
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

function createBaseQueryCurrentRequest(): QueryCurrentRequest {
  return { consumer: "" };
}

export const QueryCurrentRequest = {
  encode(message: QueryCurrentRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.consumer !== "") {
      writer.uint32(10).string(message.consumer);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryCurrentRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryCurrentRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.consumer = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryCurrentRequest {
    return { consumer: isSet(object.consumer) ? String(object.consumer) : "" };
  },

  toJSON(message: QueryCurrentRequest): unknown {
    const obj: any = {};
    message.consumer !== undefined && (obj.consumer = message.consumer);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryCurrentRequest>, I>>(base?: I): QueryCurrentRequest {
    return QueryCurrentRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryCurrentRequest>, I>>(object: I): QueryCurrentRequest {
    const message = createBaseQueryCurrentRequest();
    message.consumer = object.consumer ?? "";
    return message;
  },
};

function createBaseQueryCurrentResponse(): QueryCurrentResponse {
  return { sub: undefined };
}

export const QueryCurrentResponse = {
  encode(message: QueryCurrentResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.sub !== undefined) {
      Subscription.encode(message.sub, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryCurrentResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryCurrentResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.sub = Subscription.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryCurrentResponse {
    return { sub: isSet(object.sub) ? Subscription.fromJSON(object.sub) : undefined };
  },

  toJSON(message: QueryCurrentResponse): unknown {
    const obj: any = {};
    message.sub !== undefined && (obj.sub = message.sub ? Subscription.toJSON(message.sub) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryCurrentResponse>, I>>(base?: I): QueryCurrentResponse {
    return QueryCurrentResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryCurrentResponse>, I>>(object: I): QueryCurrentResponse {
    const message = createBaseQueryCurrentResponse();
    message.sub = (object.sub !== undefined && object.sub !== null) ? Subscription.fromPartial(object.sub) : undefined;
    return message;
  },
};

function createBaseQueryListProjectsRequest(): QueryListProjectsRequest {
  return { subscription: "" };
}

export const QueryListProjectsRequest = {
  encode(message: QueryListProjectsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.subscription !== "") {
      writer.uint32(10).string(message.subscription);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryListProjectsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryListProjectsRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.subscription = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryListProjectsRequest {
    return { subscription: isSet(object.subscription) ? String(object.subscription) : "" };
  },

  toJSON(message: QueryListProjectsRequest): unknown {
    const obj: any = {};
    message.subscription !== undefined && (obj.subscription = message.subscription);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryListProjectsRequest>, I>>(base?: I): QueryListProjectsRequest {
    return QueryListProjectsRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryListProjectsRequest>, I>>(object: I): QueryListProjectsRequest {
    const message = createBaseQueryListProjectsRequest();
    message.subscription = object.subscription ?? "";
    return message;
  },
};

function createBaseQueryListProjectsResponse(): QueryListProjectsResponse {
  return { projects: [] };
}

export const QueryListProjectsResponse = {
  encode(message: QueryListProjectsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.projects) {
      writer.uint32(10).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryListProjectsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryListProjectsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.projects.push(reader.string());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryListProjectsResponse {
    return { projects: Array.isArray(object?.projects) ? object.projects.map((e: any) => String(e)) : [] };
  },

  toJSON(message: QueryListProjectsResponse): unknown {
    const obj: any = {};
    if (message.projects) {
      obj.projects = message.projects.map((e) => e);
    } else {
      obj.projects = [];
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryListProjectsResponse>, I>>(base?: I): QueryListProjectsResponse {
    return QueryListProjectsResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryListProjectsResponse>, I>>(object: I): QueryListProjectsResponse {
    const message = createBaseQueryListProjectsResponse();
    message.projects = object.projects?.map((e) => e) || [];
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
  return { subsInfo: [] };
}

export const QueryListResponse = {
  encode(message: QueryListResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.subsInfo) {
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

          message.subsInfo.push(ListInfoStruct.decode(reader, reader.uint32()));
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
      subsInfo: Array.isArray(object?.subsInfo) ? object.subsInfo.map((e: any) => ListInfoStruct.fromJSON(e)) : [],
    };
  },

  toJSON(message: QueryListResponse): unknown {
    const obj: any = {};
    if (message.subsInfo) {
      obj.subsInfo = message.subsInfo.map((e) => e ? ListInfoStruct.toJSON(e) : undefined);
    } else {
      obj.subsInfo = [];
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryListResponse>, I>>(base?: I): QueryListResponse {
    return QueryListResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryListResponse>, I>>(object: I): QueryListResponse {
    const message = createBaseQueryListResponse();
    message.subsInfo = object.subsInfo?.map((e) => ListInfoStruct.fromPartial(e)) || [];
    return message;
  },
};

function createBaseListInfoStruct(): ListInfoStruct {
  return {
    consumer: "",
    plan: "",
    durationTotal: Long.UZERO,
    durationLeft: Long.UZERO,
    monthExpiry: Long.UZERO,
    monthCuTotal: Long.UZERO,
    monthCuLeft: Long.UZERO,
  };
}

export const ListInfoStruct = {
  encode(message: ListInfoStruct, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.consumer !== "") {
      writer.uint32(10).string(message.consumer);
    }
    if (message.plan !== "") {
      writer.uint32(18).string(message.plan);
    }
    if (!message.durationTotal.isZero()) {
      writer.uint32(24).uint64(message.durationTotal);
    }
    if (!message.durationLeft.isZero()) {
      writer.uint32(32).uint64(message.durationLeft);
    }
    if (!message.monthExpiry.isZero()) {
      writer.uint32(40).uint64(message.monthExpiry);
    }
    if (!message.monthCuTotal.isZero()) {
      writer.uint32(48).uint64(message.monthCuTotal);
    }
    if (!message.monthCuLeft.isZero()) {
      writer.uint32(56).uint64(message.monthCuLeft);
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

          message.consumer = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.plan = reader.string();
          continue;
        case 3:
          if (tag != 24) {
            break;
          }

          message.durationTotal = reader.uint64() as Long;
          continue;
        case 4:
          if (tag != 32) {
            break;
          }

          message.durationLeft = reader.uint64() as Long;
          continue;
        case 5:
          if (tag != 40) {
            break;
          }

          message.monthExpiry = reader.uint64() as Long;
          continue;
        case 6:
          if (tag != 48) {
            break;
          }

          message.monthCuTotal = reader.uint64() as Long;
          continue;
        case 7:
          if (tag != 56) {
            break;
          }

          message.monthCuLeft = reader.uint64() as Long;
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
      consumer: isSet(object.consumer) ? String(object.consumer) : "",
      plan: isSet(object.plan) ? String(object.plan) : "",
      durationTotal: isSet(object.durationTotal) ? Long.fromValue(object.durationTotal) : Long.UZERO,
      durationLeft: isSet(object.durationLeft) ? Long.fromValue(object.durationLeft) : Long.UZERO,
      monthExpiry: isSet(object.monthExpiry) ? Long.fromValue(object.monthExpiry) : Long.UZERO,
      monthCuTotal: isSet(object.monthCuTotal) ? Long.fromValue(object.monthCuTotal) : Long.UZERO,
      monthCuLeft: isSet(object.monthCuLeft) ? Long.fromValue(object.monthCuLeft) : Long.UZERO,
    };
  },

  toJSON(message: ListInfoStruct): unknown {
    const obj: any = {};
    message.consumer !== undefined && (obj.consumer = message.consumer);
    message.plan !== undefined && (obj.plan = message.plan);
    message.durationTotal !== undefined && (obj.durationTotal = (message.durationTotal || Long.UZERO).toString());
    message.durationLeft !== undefined && (obj.durationLeft = (message.durationLeft || Long.UZERO).toString());
    message.monthExpiry !== undefined && (obj.monthExpiry = (message.monthExpiry || Long.UZERO).toString());
    message.monthCuTotal !== undefined && (obj.monthCuTotal = (message.monthCuTotal || Long.UZERO).toString());
    message.monthCuLeft !== undefined && (obj.monthCuLeft = (message.monthCuLeft || Long.UZERO).toString());
    return obj;
  },

  create<I extends Exact<DeepPartial<ListInfoStruct>, I>>(base?: I): ListInfoStruct {
    return ListInfoStruct.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<ListInfoStruct>, I>>(object: I): ListInfoStruct {
    const message = createBaseListInfoStruct();
    message.consumer = object.consumer ?? "";
    message.plan = object.plan ?? "";
    message.durationTotal = (object.durationTotal !== undefined && object.durationTotal !== null)
      ? Long.fromValue(object.durationTotal)
      : Long.UZERO;
    message.durationLeft = (object.durationLeft !== undefined && object.durationLeft !== null)
      ? Long.fromValue(object.durationLeft)
      : Long.UZERO;
    message.monthExpiry = (object.monthExpiry !== undefined && object.monthExpiry !== null)
      ? Long.fromValue(object.monthExpiry)
      : Long.UZERO;
    message.monthCuTotal = (object.monthCuTotal !== undefined && object.monthCuTotal !== null)
      ? Long.fromValue(object.monthCuTotal)
      : Long.UZERO;
    message.monthCuLeft = (object.monthCuLeft !== undefined && object.monthCuLeft !== null)
      ? Long.fromValue(object.monthCuLeft)
      : Long.UZERO;
    return message;
  },
};

/** Query defines the gRPC querier service. */
export interface Query {
  /** Parameters queries the parameters of the module. */
  Params(request: QueryParamsRequest): Promise<QueryParamsResponse>;
  /** Queries a list of Current items. */
  Current(request: QueryCurrentRequest): Promise<QueryCurrentResponse>;
  /** Queries a list of ListProjects items. */
  ListProjects(request: QueryListProjectsRequest): Promise<QueryListProjectsResponse>;
  /** Queries a list of List items. */
  List(request: QueryListRequest): Promise<QueryListResponse>;
}

export class QueryClientImpl implements Query {
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || "lavanet.lava.subscription.Query";
    this.rpc = rpc;
    this.Params = this.Params.bind(this);
    this.Current = this.Current.bind(this);
    this.ListProjects = this.ListProjects.bind(this);
    this.List = this.List.bind(this);
  }
  Params(request: QueryParamsRequest): Promise<QueryParamsResponse> {
    const data = QueryParamsRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "Params", data);
    return promise.then((data) => QueryParamsResponse.decode(_m0.Reader.create(data)));
  }

  Current(request: QueryCurrentRequest): Promise<QueryCurrentResponse> {
    const data = QueryCurrentRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "Current", data);
    return promise.then((data) => QueryCurrentResponse.decode(_m0.Reader.create(data)));
  }

  ListProjects(request: QueryListProjectsRequest): Promise<QueryListProjectsResponse> {
    const data = QueryListProjectsRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "ListProjects", data);
    return promise.then((data) => QueryListProjectsResponse.decode(_m0.Reader.create(data)));
  }

  List(request: QueryListRequest): Promise<QueryListResponse> {
    const data = QueryListRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "List", data);
    return promise.then((data) => QueryListResponse.decode(_m0.Reader.create(data)));
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

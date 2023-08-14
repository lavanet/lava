/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { PageRequest, PageResponse } from "../../../cosmos/base/query/v1beta1/pagination";
import { Params } from "./params";
import { Spec } from "./spec";

export const protobufPackage = "lavanet.lava.spec";

/** QueryParamsRequest is request type for the Query/Params RPC method. */
export interface QueryParamsRequest {
}

/** QueryParamsResponse is response type for the Query/Params RPC method. */
export interface QueryParamsResponse {
  /** params holds all the parameters of this module. */
  params?: Params;
}

export interface QueryGetSpecRequest {
  ChainID: string;
}

export interface QueryGetSpecResponse {
  Spec?: Spec;
}

export interface QueryAllSpecRequest {
  pagination?: PageRequest;
}

export interface QueryAllSpecResponse {
  Spec: Spec[];
  pagination?: PageResponse;
}

export interface QueryShowAllChainsRequest {
}

export interface QueryShowAllChainsResponse {
  chainInfoList: ShowAllChainsInfoStruct[];
}

export interface ShowAllChainsInfoStruct {
  chainName: string;
  chainID: string;
  enabledApiInterfaces: string[];
  apiCount: Long;
}

export interface QueryShowChainInfoRequest {
  chainName: string;
}

export interface ApiList {
  interface: string;
  supportedApis: string[];
  addon: string;
}

export interface QueryShowChainInfoResponse {
  chainID: string;
  interfaces: string[];
  supportedApisInterfaceList: ApiList[];
  optionalInterfaces: string[];
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

function createBaseQueryGetSpecRequest(): QueryGetSpecRequest {
  return { ChainID: "" };
}

export const QueryGetSpecRequest = {
  encode(message: QueryGetSpecRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.ChainID !== "") {
      writer.uint32(10).string(message.ChainID);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetSpecRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetSpecRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.ChainID = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryGetSpecRequest {
    return { ChainID: isSet(object.ChainID) ? String(object.ChainID) : "" };
  },

  toJSON(message: QueryGetSpecRequest): unknown {
    const obj: any = {};
    message.ChainID !== undefined && (obj.ChainID = message.ChainID);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryGetSpecRequest>, I>>(base?: I): QueryGetSpecRequest {
    return QueryGetSpecRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetSpecRequest>, I>>(object: I): QueryGetSpecRequest {
    const message = createBaseQueryGetSpecRequest();
    message.ChainID = object.ChainID ?? "";
    return message;
  },
};

function createBaseQueryGetSpecResponse(): QueryGetSpecResponse {
  return { Spec: undefined };
}

export const QueryGetSpecResponse = {
  encode(message: QueryGetSpecResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.Spec !== undefined) {
      Spec.encode(message.Spec, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetSpecResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetSpecResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.Spec = Spec.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryGetSpecResponse {
    return { Spec: isSet(object.Spec) ? Spec.fromJSON(object.Spec) : undefined };
  },

  toJSON(message: QueryGetSpecResponse): unknown {
    const obj: any = {};
    message.Spec !== undefined && (obj.Spec = message.Spec ? Spec.toJSON(message.Spec) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryGetSpecResponse>, I>>(base?: I): QueryGetSpecResponse {
    return QueryGetSpecResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetSpecResponse>, I>>(object: I): QueryGetSpecResponse {
    const message = createBaseQueryGetSpecResponse();
    message.Spec = (object.Spec !== undefined && object.Spec !== null) ? Spec.fromPartial(object.Spec) : undefined;
    return message;
  },
};

function createBaseQueryAllSpecRequest(): QueryAllSpecRequest {
  return { pagination: undefined };
}

export const QueryAllSpecRequest = {
  encode(message: QueryAllSpecRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllSpecRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAllSpecRequest();
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

  fromJSON(object: any): QueryAllSpecRequest {
    return { pagination: isSet(object.pagination) ? PageRequest.fromJSON(object.pagination) : undefined };
  },

  toJSON(message: QueryAllSpecRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination ? PageRequest.toJSON(message.pagination) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryAllSpecRequest>, I>>(base?: I): QueryAllSpecRequest {
    return QueryAllSpecRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryAllSpecRequest>, I>>(object: I): QueryAllSpecRequest {
    const message = createBaseQueryAllSpecRequest();
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageRequest.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseQueryAllSpecResponse(): QueryAllSpecResponse {
  return { Spec: [], pagination: undefined };
}

export const QueryAllSpecResponse = {
  encode(message: QueryAllSpecResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.Spec) {
      Spec.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllSpecResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAllSpecResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.Spec.push(Spec.decode(reader, reader.uint32()));
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

  fromJSON(object: any): QueryAllSpecResponse {
    return {
      Spec: Array.isArray(object?.Spec) ? object.Spec.map((e: any) => Spec.fromJSON(e)) : [],
      pagination: isSet(object.pagination) ? PageResponse.fromJSON(object.pagination) : undefined,
    };
  },

  toJSON(message: QueryAllSpecResponse): unknown {
    const obj: any = {};
    if (message.Spec) {
      obj.Spec = message.Spec.map((e) => e ? Spec.toJSON(e) : undefined);
    } else {
      obj.Spec = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination ? PageResponse.toJSON(message.pagination) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryAllSpecResponse>, I>>(base?: I): QueryAllSpecResponse {
    return QueryAllSpecResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryAllSpecResponse>, I>>(object: I): QueryAllSpecResponse {
    const message = createBaseQueryAllSpecResponse();
    message.Spec = object.Spec?.map((e) => Spec.fromPartial(e)) || [];
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageResponse.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseQueryShowAllChainsRequest(): QueryShowAllChainsRequest {
  return {};
}

export const QueryShowAllChainsRequest = {
  encode(_: QueryShowAllChainsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryShowAllChainsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryShowAllChainsRequest();
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

  fromJSON(_: any): QueryShowAllChainsRequest {
    return {};
  },

  toJSON(_: QueryShowAllChainsRequest): unknown {
    const obj: any = {};
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryShowAllChainsRequest>, I>>(base?: I): QueryShowAllChainsRequest {
    return QueryShowAllChainsRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryShowAllChainsRequest>, I>>(_: I): QueryShowAllChainsRequest {
    const message = createBaseQueryShowAllChainsRequest();
    return message;
  },
};

function createBaseQueryShowAllChainsResponse(): QueryShowAllChainsResponse {
  return { chainInfoList: [] };
}

export const QueryShowAllChainsResponse = {
  encode(message: QueryShowAllChainsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.chainInfoList) {
      ShowAllChainsInfoStruct.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryShowAllChainsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryShowAllChainsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 2:
          if (tag != 18) {
            break;
          }

          message.chainInfoList.push(ShowAllChainsInfoStruct.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryShowAllChainsResponse {
    return {
      chainInfoList: Array.isArray(object?.chainInfoList)
        ? object.chainInfoList.map((e: any) => ShowAllChainsInfoStruct.fromJSON(e))
        : [],
    };
  },

  toJSON(message: QueryShowAllChainsResponse): unknown {
    const obj: any = {};
    if (message.chainInfoList) {
      obj.chainInfoList = message.chainInfoList.map((e) => e ? ShowAllChainsInfoStruct.toJSON(e) : undefined);
    } else {
      obj.chainInfoList = [];
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryShowAllChainsResponse>, I>>(base?: I): QueryShowAllChainsResponse {
    return QueryShowAllChainsResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryShowAllChainsResponse>, I>>(object: I): QueryShowAllChainsResponse {
    const message = createBaseQueryShowAllChainsResponse();
    message.chainInfoList = object.chainInfoList?.map((e) => ShowAllChainsInfoStruct.fromPartial(e)) || [];
    return message;
  },
};

function createBaseShowAllChainsInfoStruct(): ShowAllChainsInfoStruct {
  return { chainName: "", chainID: "", enabledApiInterfaces: [], apiCount: Long.UZERO };
}

export const ShowAllChainsInfoStruct = {
  encode(message: ShowAllChainsInfoStruct, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.chainName !== "") {
      writer.uint32(10).string(message.chainName);
    }
    if (message.chainID !== "") {
      writer.uint32(18).string(message.chainID);
    }
    for (const v of message.enabledApiInterfaces) {
      writer.uint32(26).string(v!);
    }
    if (!message.apiCount.isZero()) {
      writer.uint32(32).uint64(message.apiCount);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ShowAllChainsInfoStruct {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseShowAllChainsInfoStruct();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.chainName = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.chainID = reader.string();
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.enabledApiInterfaces.push(reader.string());
          continue;
        case 4:
          if (tag != 32) {
            break;
          }

          message.apiCount = reader.uint64() as Long;
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ShowAllChainsInfoStruct {
    return {
      chainName: isSet(object.chainName) ? String(object.chainName) : "",
      chainID: isSet(object.chainID) ? String(object.chainID) : "",
      enabledApiInterfaces: Array.isArray(object?.enabledApiInterfaces)
        ? object.enabledApiInterfaces.map((e: any) => String(e))
        : [],
      apiCount: isSet(object.apiCount) ? Long.fromValue(object.apiCount) : Long.UZERO,
    };
  },

  toJSON(message: ShowAllChainsInfoStruct): unknown {
    const obj: any = {};
    message.chainName !== undefined && (obj.chainName = message.chainName);
    message.chainID !== undefined && (obj.chainID = message.chainID);
    if (message.enabledApiInterfaces) {
      obj.enabledApiInterfaces = message.enabledApiInterfaces.map((e) => e);
    } else {
      obj.enabledApiInterfaces = [];
    }
    message.apiCount !== undefined && (obj.apiCount = (message.apiCount || Long.UZERO).toString());
    return obj;
  },

  create<I extends Exact<DeepPartial<ShowAllChainsInfoStruct>, I>>(base?: I): ShowAllChainsInfoStruct {
    return ShowAllChainsInfoStruct.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<ShowAllChainsInfoStruct>, I>>(object: I): ShowAllChainsInfoStruct {
    const message = createBaseShowAllChainsInfoStruct();
    message.chainName = object.chainName ?? "";
    message.chainID = object.chainID ?? "";
    message.enabledApiInterfaces = object.enabledApiInterfaces?.map((e) => e) || [];
    message.apiCount = (object.apiCount !== undefined && object.apiCount !== null)
      ? Long.fromValue(object.apiCount)
      : Long.UZERO;
    return message;
  },
};

function createBaseQueryShowChainInfoRequest(): QueryShowChainInfoRequest {
  return { chainName: "" };
}

export const QueryShowChainInfoRequest = {
  encode(message: QueryShowChainInfoRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.chainName !== "") {
      writer.uint32(10).string(message.chainName);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryShowChainInfoRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryShowChainInfoRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.chainName = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryShowChainInfoRequest {
    return { chainName: isSet(object.chainName) ? String(object.chainName) : "" };
  },

  toJSON(message: QueryShowChainInfoRequest): unknown {
    const obj: any = {};
    message.chainName !== undefined && (obj.chainName = message.chainName);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryShowChainInfoRequest>, I>>(base?: I): QueryShowChainInfoRequest {
    return QueryShowChainInfoRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryShowChainInfoRequest>, I>>(object: I): QueryShowChainInfoRequest {
    const message = createBaseQueryShowChainInfoRequest();
    message.chainName = object.chainName ?? "";
    return message;
  },
};

function createBaseApiList(): ApiList {
  return { interface: "", supportedApis: [], addon: "" };
}

export const ApiList = {
  encode(message: ApiList, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.interface !== "") {
      writer.uint32(34).string(message.interface);
    }
    for (const v of message.supportedApis) {
      writer.uint32(42).string(v!);
    }
    if (message.addon !== "") {
      writer.uint32(50).string(message.addon);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ApiList {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApiList();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 4:
          if (tag != 34) {
            break;
          }

          message.interface = reader.string();
          continue;
        case 5:
          if (tag != 42) {
            break;
          }

          message.supportedApis.push(reader.string());
          continue;
        case 6:
          if (tag != 50) {
            break;
          }

          message.addon = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ApiList {
    return {
      interface: isSet(object.interface) ? String(object.interface) : "",
      supportedApis: Array.isArray(object?.supportedApis) ? object.supportedApis.map((e: any) => String(e)) : [],
      addon: isSet(object.addon) ? String(object.addon) : "",
    };
  },

  toJSON(message: ApiList): unknown {
    const obj: any = {};
    message.interface !== undefined && (obj.interface = message.interface);
    if (message.supportedApis) {
      obj.supportedApis = message.supportedApis.map((e) => e);
    } else {
      obj.supportedApis = [];
    }
    message.addon !== undefined && (obj.addon = message.addon);
    return obj;
  },

  create<I extends Exact<DeepPartial<ApiList>, I>>(base?: I): ApiList {
    return ApiList.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<ApiList>, I>>(object: I): ApiList {
    const message = createBaseApiList();
    message.interface = object.interface ?? "";
    message.supportedApis = object.supportedApis?.map((e) => e) || [];
    message.addon = object.addon ?? "";
    return message;
  },
};

function createBaseQueryShowChainInfoResponse(): QueryShowChainInfoResponse {
  return { chainID: "", interfaces: [], supportedApisInterfaceList: [], optionalInterfaces: [] };
}

export const QueryShowChainInfoResponse = {
  encode(message: QueryShowChainInfoResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.chainID !== "") {
      writer.uint32(10).string(message.chainID);
    }
    for (const v of message.interfaces) {
      writer.uint32(18).string(v!);
    }
    for (const v of message.supportedApisInterfaceList) {
      ApiList.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.optionalInterfaces) {
      writer.uint32(34).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryShowChainInfoResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryShowChainInfoResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.chainID = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.interfaces.push(reader.string());
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.supportedApisInterfaceList.push(ApiList.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.optionalInterfaces.push(reader.string());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryShowChainInfoResponse {
    return {
      chainID: isSet(object.chainID) ? String(object.chainID) : "",
      interfaces: Array.isArray(object?.interfaces) ? object.interfaces.map((e: any) => String(e)) : [],
      supportedApisInterfaceList: Array.isArray(object?.supportedApisInterfaceList)
        ? object.supportedApisInterfaceList.map((e: any) => ApiList.fromJSON(e))
        : [],
      optionalInterfaces: Array.isArray(object?.optionalInterfaces)
        ? object.optionalInterfaces.map((e: any) => String(e))
        : [],
    };
  },

  toJSON(message: QueryShowChainInfoResponse): unknown {
    const obj: any = {};
    message.chainID !== undefined && (obj.chainID = message.chainID);
    if (message.interfaces) {
      obj.interfaces = message.interfaces.map((e) => e);
    } else {
      obj.interfaces = [];
    }
    if (message.supportedApisInterfaceList) {
      obj.supportedApisInterfaceList = message.supportedApisInterfaceList.map((e) => e ? ApiList.toJSON(e) : undefined);
    } else {
      obj.supportedApisInterfaceList = [];
    }
    if (message.optionalInterfaces) {
      obj.optionalInterfaces = message.optionalInterfaces.map((e) => e);
    } else {
      obj.optionalInterfaces = [];
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryShowChainInfoResponse>, I>>(base?: I): QueryShowChainInfoResponse {
    return QueryShowChainInfoResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryShowChainInfoResponse>, I>>(object: I): QueryShowChainInfoResponse {
    const message = createBaseQueryShowChainInfoResponse();
    message.chainID = object.chainID ?? "";
    message.interfaces = object.interfaces?.map((e) => e) || [];
    message.supportedApisInterfaceList = object.supportedApisInterfaceList?.map((e) => ApiList.fromPartial(e)) || [];
    message.optionalInterfaces = object.optionalInterfaces?.map((e) => e) || [];
    return message;
  },
};

/** Query defines the gRPC querier service. */
export interface Query {
  /** Parameters queries the parameters of the module. */
  Params(request: QueryParamsRequest): Promise<QueryParamsResponse>;
  /** Queries a Spec by id. */
  Spec(request: QueryGetSpecRequest): Promise<QueryGetSpecResponse>;
  /** Queries a list of Spec items. */
  SpecAll(request: QueryAllSpecRequest): Promise<QueryAllSpecResponse>;
  /** Queries a Spec by id (raw form). */
  SpecRaw(request: QueryGetSpecRequest): Promise<QueryGetSpecResponse>;
  /** Queries a list of Spec items (raw form). */
  SpecAllRaw(request: QueryAllSpecRequest): Promise<QueryAllSpecResponse>;
  /** Queries a list of ShowAllChains items. */
  ShowAllChains(request: QueryShowAllChainsRequest): Promise<QueryShowAllChainsResponse>;
  /** Queries a list of ShowChainInfo items. */
  ShowChainInfo(request: QueryShowChainInfoRequest): Promise<QueryShowChainInfoResponse>;
}

export class QueryClientImpl implements Query {
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || "lavanet.lava.spec.Query";
    this.rpc = rpc;
    this.Params = this.Params.bind(this);
    this.Spec = this.Spec.bind(this);
    this.SpecAll = this.SpecAll.bind(this);
    this.SpecRaw = this.SpecRaw.bind(this);
    this.SpecAllRaw = this.SpecAllRaw.bind(this);
    this.ShowAllChains = this.ShowAllChains.bind(this);
    this.ShowChainInfo = this.ShowChainInfo.bind(this);
  }
  Params(request: QueryParamsRequest): Promise<QueryParamsResponse> {
    const data = QueryParamsRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "Params", data);
    return promise.then((data) => QueryParamsResponse.decode(_m0.Reader.create(data)));
  }

  Spec(request: QueryGetSpecRequest): Promise<QueryGetSpecResponse> {
    const data = QueryGetSpecRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "Spec", data);
    return promise.then((data) => QueryGetSpecResponse.decode(_m0.Reader.create(data)));
  }

  SpecAll(request: QueryAllSpecRequest): Promise<QueryAllSpecResponse> {
    const data = QueryAllSpecRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "SpecAll", data);
    return promise.then((data) => QueryAllSpecResponse.decode(_m0.Reader.create(data)));
  }

  SpecRaw(request: QueryGetSpecRequest): Promise<QueryGetSpecResponse> {
    const data = QueryGetSpecRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "SpecRaw", data);
    return promise.then((data) => QueryGetSpecResponse.decode(_m0.Reader.create(data)));
  }

  SpecAllRaw(request: QueryAllSpecRequest): Promise<QueryAllSpecResponse> {
    const data = QueryAllSpecRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "SpecAllRaw", data);
    return promise.then((data) => QueryAllSpecResponse.decode(_m0.Reader.create(data)));
  }

  ShowAllChains(request: QueryShowAllChainsRequest): Promise<QueryShowAllChainsResponse> {
    const data = QueryShowAllChainsRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "ShowAllChains", data);
    return promise.then((data) => QueryShowAllChainsResponse.decode(_m0.Reader.create(data)));
  }

  ShowChainInfo(request: QueryShowChainInfoRequest): Promise<QueryShowChainInfoResponse> {
    const data = QueryShowChainInfoRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "ShowChainInfo", data);
    return promise.then((data) => QueryShowChainInfoResponse.decode(_m0.Reader.create(data)));
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

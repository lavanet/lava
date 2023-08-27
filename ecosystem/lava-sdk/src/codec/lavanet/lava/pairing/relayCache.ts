/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Empty } from "../../../google/protobuf/empty";
import { RelayReply, RelayRequest } from "./relay";

export const protobufPackage = "lavanet.lava.pairing";

export interface CacheUsage {
  CacheHits: Long;
  CacheMisses: Long;
}

export interface RelayCacheGet {
  request?: RelayRequest;
  apiInterface: string;
  blockHash: Uint8Array;
  /** Used to differentiate between different chains so each has its own bucket */
  chainID: string;
  finalized: boolean;
}

export interface RelayCacheSet {
  request?: RelayRequest;
  apiInterface: string;
  blockHash: Uint8Array;
  /** Used to differentiate between different chains so each has its own bucket */
  chainID: string;
  /** bucketID is used to make sure a big user doesnt flood the cache, on providers this will be consumer address, on portal it will be dappID */
  bucketID: string;
  response?: RelayReply;
  finalized: boolean;
}

function createBaseCacheUsage(): CacheUsage {
  return { CacheHits: Long.UZERO, CacheMisses: Long.UZERO };
}

export const CacheUsage = {
  encode(message: CacheUsage, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (!message.CacheHits.isZero()) {
      writer.uint32(8).uint64(message.CacheHits);
    }
    if (!message.CacheMisses.isZero()) {
      writer.uint32(16).uint64(message.CacheMisses);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CacheUsage {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCacheUsage();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 8) {
            break;
          }

          message.CacheHits = reader.uint64() as Long;
          continue;
        case 2:
          if (tag != 16) {
            break;
          }

          message.CacheMisses = reader.uint64() as Long;
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CacheUsage {
    return {
      CacheHits: isSet(object.CacheHits) ? Long.fromValue(object.CacheHits) : Long.UZERO,
      CacheMisses: isSet(object.CacheMisses) ? Long.fromValue(object.CacheMisses) : Long.UZERO,
    };
  },

  toJSON(message: CacheUsage): unknown {
    const obj: any = {};
    message.CacheHits !== undefined && (obj.CacheHits = (message.CacheHits || Long.UZERO).toString());
    message.CacheMisses !== undefined && (obj.CacheMisses = (message.CacheMisses || Long.UZERO).toString());
    return obj;
  },

  create<I extends Exact<DeepPartial<CacheUsage>, I>>(base?: I): CacheUsage {
    return CacheUsage.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<CacheUsage>, I>>(object: I): CacheUsage {
    const message = createBaseCacheUsage();
    message.CacheHits = (object.CacheHits !== undefined && object.CacheHits !== null)
      ? Long.fromValue(object.CacheHits)
      : Long.UZERO;
    message.CacheMisses = (object.CacheMisses !== undefined && object.CacheMisses !== null)
      ? Long.fromValue(object.CacheMisses)
      : Long.UZERO;
    return message;
  },
};

function createBaseRelayCacheGet(): RelayCacheGet {
  return { request: undefined, apiInterface: "", blockHash: new Uint8Array(), chainID: "", finalized: false };
}

export const RelayCacheGet = {
  encode(message: RelayCacheGet, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.request !== undefined) {
      RelayRequest.encode(message.request, writer.uint32(10).fork()).ldelim();
    }
    if (message.apiInterface !== "") {
      writer.uint32(18).string(message.apiInterface);
    }
    if (message.blockHash.length !== 0) {
      writer.uint32(26).bytes(message.blockHash);
    }
    if (message.chainID !== "") {
      writer.uint32(34).string(message.chainID);
    }
    if (message.finalized === true) {
      writer.uint32(40).bool(message.finalized);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RelayCacheGet {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRelayCacheGet();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.request = RelayRequest.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.apiInterface = reader.string();
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.blockHash = reader.bytes();
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.chainID = reader.string();
          continue;
        case 5:
          if (tag != 40) {
            break;
          }

          message.finalized = reader.bool();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): RelayCacheGet {
    return {
      request: isSet(object.request) ? RelayRequest.fromJSON(object.request) : undefined,
      apiInterface: isSet(object.apiInterface) ? String(object.apiInterface) : "",
      blockHash: isSet(object.blockHash) ? bytesFromBase64(object.blockHash) : new Uint8Array(),
      chainID: isSet(object.chainID) ? String(object.chainID) : "",
      finalized: isSet(object.finalized) ? Boolean(object.finalized) : false,
    };
  },

  toJSON(message: RelayCacheGet): unknown {
    const obj: any = {};
    message.request !== undefined && (obj.request = message.request ? RelayRequest.toJSON(message.request) : undefined);
    message.apiInterface !== undefined && (obj.apiInterface = message.apiInterface);
    message.blockHash !== undefined &&
      (obj.blockHash = base64FromBytes(message.blockHash !== undefined ? message.blockHash : new Uint8Array()));
    message.chainID !== undefined && (obj.chainID = message.chainID);
    message.finalized !== undefined && (obj.finalized = message.finalized);
    return obj;
  },

  create<I extends Exact<DeepPartial<RelayCacheGet>, I>>(base?: I): RelayCacheGet {
    return RelayCacheGet.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<RelayCacheGet>, I>>(object: I): RelayCacheGet {
    const message = createBaseRelayCacheGet();
    message.request = (object.request !== undefined && object.request !== null)
      ? RelayRequest.fromPartial(object.request)
      : undefined;
    message.apiInterface = object.apiInterface ?? "";
    message.blockHash = object.blockHash ?? new Uint8Array();
    message.chainID = object.chainID ?? "";
    message.finalized = object.finalized ?? false;
    return message;
  },
};

function createBaseRelayCacheSet(): RelayCacheSet {
  return {
    request: undefined,
    apiInterface: "",
    blockHash: new Uint8Array(),
    chainID: "",
    bucketID: "",
    response: undefined,
    finalized: false,
  };
}

export const RelayCacheSet = {
  encode(message: RelayCacheSet, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.request !== undefined) {
      RelayRequest.encode(message.request, writer.uint32(10).fork()).ldelim();
    }
    if (message.apiInterface !== "") {
      writer.uint32(18).string(message.apiInterface);
    }
    if (message.blockHash.length !== 0) {
      writer.uint32(26).bytes(message.blockHash);
    }
    if (message.chainID !== "") {
      writer.uint32(34).string(message.chainID);
    }
    if (message.bucketID !== "") {
      writer.uint32(42).string(message.bucketID);
    }
    if (message.response !== undefined) {
      RelayReply.encode(message.response, writer.uint32(50).fork()).ldelim();
    }
    if (message.finalized === true) {
      writer.uint32(56).bool(message.finalized);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RelayCacheSet {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRelayCacheSet();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.request = RelayRequest.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.apiInterface = reader.string();
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.blockHash = reader.bytes();
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.chainID = reader.string();
          continue;
        case 5:
          if (tag != 42) {
            break;
          }

          message.bucketID = reader.string();
          continue;
        case 6:
          if (tag != 50) {
            break;
          }

          message.response = RelayReply.decode(reader, reader.uint32());
          continue;
        case 7:
          if (tag != 56) {
            break;
          }

          message.finalized = reader.bool();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): RelayCacheSet {
    return {
      request: isSet(object.request) ? RelayRequest.fromJSON(object.request) : undefined,
      apiInterface: isSet(object.apiInterface) ? String(object.apiInterface) : "",
      blockHash: isSet(object.blockHash) ? bytesFromBase64(object.blockHash) : new Uint8Array(),
      chainID: isSet(object.chainID) ? String(object.chainID) : "",
      bucketID: isSet(object.bucketID) ? String(object.bucketID) : "",
      response: isSet(object.response) ? RelayReply.fromJSON(object.response) : undefined,
      finalized: isSet(object.finalized) ? Boolean(object.finalized) : false,
    };
  },

  toJSON(message: RelayCacheSet): unknown {
    const obj: any = {};
    message.request !== undefined && (obj.request = message.request ? RelayRequest.toJSON(message.request) : undefined);
    message.apiInterface !== undefined && (obj.apiInterface = message.apiInterface);
    message.blockHash !== undefined &&
      (obj.blockHash = base64FromBytes(message.blockHash !== undefined ? message.blockHash : new Uint8Array()));
    message.chainID !== undefined && (obj.chainID = message.chainID);
    message.bucketID !== undefined && (obj.bucketID = message.bucketID);
    message.response !== undefined &&
      (obj.response = message.response ? RelayReply.toJSON(message.response) : undefined);
    message.finalized !== undefined && (obj.finalized = message.finalized);
    return obj;
  },

  create<I extends Exact<DeepPartial<RelayCacheSet>, I>>(base?: I): RelayCacheSet {
    return RelayCacheSet.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<RelayCacheSet>, I>>(object: I): RelayCacheSet {
    const message = createBaseRelayCacheSet();
    message.request = (object.request !== undefined && object.request !== null)
      ? RelayRequest.fromPartial(object.request)
      : undefined;
    message.apiInterface = object.apiInterface ?? "";
    message.blockHash = object.blockHash ?? new Uint8Array();
    message.chainID = object.chainID ?? "";
    message.bucketID = object.bucketID ?? "";
    message.response = (object.response !== undefined && object.response !== null)
      ? RelayReply.fromPartial(object.response)
      : undefined;
    message.finalized = object.finalized ?? false;
    return message;
  },
};

export interface RelayerCache {
  GetRelay(request: RelayCacheGet): Promise<RelayReply>;
  SetRelay(request: RelayCacheSet): Promise<Empty>;
  Health(request: Empty): Promise<CacheUsage>;
}

export class RelayerCacheClientImpl implements RelayerCache {
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || "lavanet.lava.pairing.RelayerCache";
    this.rpc = rpc;
    this.GetRelay = this.GetRelay.bind(this);
    this.SetRelay = this.SetRelay.bind(this);
    this.Health = this.Health.bind(this);
  }
  GetRelay(request: RelayCacheGet): Promise<RelayReply> {
    const data = RelayCacheGet.encode(request).finish();
    const promise = this.rpc.request(this.service, "GetRelay", data);
    return promise.then((data) => RelayReply.decode(_m0.Reader.create(data)));
  }

  SetRelay(request: RelayCacheSet): Promise<Empty> {
    const data = RelayCacheSet.encode(request).finish();
    const promise = this.rpc.request(this.service, "SetRelay", data);
    return promise.then((data) => Empty.decode(_m0.Reader.create(data)));
  }

  Health(request: Empty): Promise<CacheUsage> {
    const data = Empty.encode(request).finish();
    const promise = this.rpc.request(this.service, "Health", data);
    return promise.then((data) => CacheUsage.decode(_m0.Reader.create(data)));
  }
}

interface Rpc {
  request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}

declare var self: any | undefined;
declare var window: any | undefined;
declare var global: any | undefined;
var tsProtoGlobalThis: any = (() => {
  if (typeof globalThis !== "undefined") {
    return globalThis;
  }
  if (typeof self !== "undefined") {
    return self;
  }
  if (typeof window !== "undefined") {
    return window;
  }
  if (typeof global !== "undefined") {
    return global;
  }
  throw "Unable to locate global object";
})();

function bytesFromBase64(b64: string): Uint8Array {
  if (tsProtoGlobalThis.Buffer) {
    return Uint8Array.from(tsProtoGlobalThis.Buffer.from(b64, "base64"));
  } else {
    const bin = tsProtoGlobalThis.atob(b64);
    const arr = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; ++i) {
      arr[i] = bin.charCodeAt(i);
    }
    return arr;
  }
}

function base64FromBytes(arr: Uint8Array): string {
  if (tsProtoGlobalThis.Buffer) {
    return tsProtoGlobalThis.Buffer.from(arr).toString("base64");
  } else {
    const bin: string[] = [];
    arr.forEach((byte) => {
      bin.push(String.fromCharCode(byte));
    });
    return tsProtoGlobalThis.btoa(bin.join(""));
  }
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

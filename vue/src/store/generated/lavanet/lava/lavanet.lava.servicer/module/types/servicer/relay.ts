/* eslint-disable */
import { Reader, util, configure, Writer } from "protobufjs/minimal";
import * as Long from "long";

export const protobufPackage = "lavanet.lava.servicer";

export interface RelayRequest {
  spec_id: number;
  api_id: number;
  session_id: number;
  /** total compute unit used including this relay */
  cu_sum: number;
  data: Uint8Array;
  sig: Uint8Array;
  servicer: string;
  block_height: number;
}

export interface RelayReply {
  data: Uint8Array;
  sig: Uint8Array;
}

const baseRelayRequest: object = {
  spec_id: 0,
  api_id: 0,
  session_id: 0,
  cu_sum: 0,
  servicer: "",
  block_height: 0,
};

export const RelayRequest = {
  encode(message: RelayRequest, writer: Writer = Writer.create()): Writer {
    if (message.spec_id !== 0) {
      writer.uint32(8).uint32(message.spec_id);
    }
    if (message.api_id !== 0) {
      writer.uint32(16).uint32(message.api_id);
    }
    if (message.session_id !== 0) {
      writer.uint32(24).uint64(message.session_id);
    }
    if (message.cu_sum !== 0) {
      writer.uint32(32).uint64(message.cu_sum);
    }
    if (message.data.length !== 0) {
      writer.uint32(42).bytes(message.data);
    }
    if (message.sig.length !== 0) {
      writer.uint32(50).bytes(message.sig);
    }
    if (message.servicer !== "") {
      writer.uint32(58).string(message.servicer);
    }
    if (message.block_height !== 0) {
      writer.uint32(64).int64(message.block_height);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): RelayRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseRelayRequest } as RelayRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.spec_id = reader.uint32();
          break;
        case 2:
          message.api_id = reader.uint32();
          break;
        case 3:
          message.session_id = longToNumber(reader.uint64() as Long);
          break;
        case 4:
          message.cu_sum = longToNumber(reader.uint64() as Long);
          break;
        case 5:
          message.data = reader.bytes();
          break;
        case 6:
          message.sig = reader.bytes();
          break;
        case 7:
          message.servicer = reader.string();
          break;
        case 8:
          message.block_height = longToNumber(reader.int64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): RelayRequest {
    const message = { ...baseRelayRequest } as RelayRequest;
    if (object.spec_id !== undefined && object.spec_id !== null) {
      message.spec_id = Number(object.spec_id);
    } else {
      message.spec_id = 0;
    }
    if (object.api_id !== undefined && object.api_id !== null) {
      message.api_id = Number(object.api_id);
    } else {
      message.api_id = 0;
    }
    if (object.session_id !== undefined && object.session_id !== null) {
      message.session_id = Number(object.session_id);
    } else {
      message.session_id = 0;
    }
    if (object.cu_sum !== undefined && object.cu_sum !== null) {
      message.cu_sum = Number(object.cu_sum);
    } else {
      message.cu_sum = 0;
    }
    if (object.data !== undefined && object.data !== null) {
      message.data = bytesFromBase64(object.data);
    }
    if (object.sig !== undefined && object.sig !== null) {
      message.sig = bytesFromBase64(object.sig);
    }
    if (object.servicer !== undefined && object.servicer !== null) {
      message.servicer = String(object.servicer);
    } else {
      message.servicer = "";
    }
    if (object.block_height !== undefined && object.block_height !== null) {
      message.block_height = Number(object.block_height);
    } else {
      message.block_height = 0;
    }
    return message;
  },

  toJSON(message: RelayRequest): unknown {
    const obj: any = {};
    message.spec_id !== undefined && (obj.spec_id = message.spec_id);
    message.api_id !== undefined && (obj.api_id = message.api_id);
    message.session_id !== undefined && (obj.session_id = message.session_id);
    message.cu_sum !== undefined && (obj.cu_sum = message.cu_sum);
    message.data !== undefined &&
      (obj.data = base64FromBytes(
        message.data !== undefined ? message.data : new Uint8Array()
      ));
    message.sig !== undefined &&
      (obj.sig = base64FromBytes(
        message.sig !== undefined ? message.sig : new Uint8Array()
      ));
    message.servicer !== undefined && (obj.servicer = message.servicer);
    message.block_height !== undefined &&
      (obj.block_height = message.block_height);
    return obj;
  },

  fromPartial(object: DeepPartial<RelayRequest>): RelayRequest {
    const message = { ...baseRelayRequest } as RelayRequest;
    if (object.spec_id !== undefined && object.spec_id !== null) {
      message.spec_id = object.spec_id;
    } else {
      message.spec_id = 0;
    }
    if (object.api_id !== undefined && object.api_id !== null) {
      message.api_id = object.api_id;
    } else {
      message.api_id = 0;
    }
    if (object.session_id !== undefined && object.session_id !== null) {
      message.session_id = object.session_id;
    } else {
      message.session_id = 0;
    }
    if (object.cu_sum !== undefined && object.cu_sum !== null) {
      message.cu_sum = object.cu_sum;
    } else {
      message.cu_sum = 0;
    }
    if (object.data !== undefined && object.data !== null) {
      message.data = object.data;
    } else {
      message.data = new Uint8Array();
    }
    if (object.sig !== undefined && object.sig !== null) {
      message.sig = object.sig;
    } else {
      message.sig = new Uint8Array();
    }
    if (object.servicer !== undefined && object.servicer !== null) {
      message.servicer = object.servicer;
    } else {
      message.servicer = "";
    }
    if (object.block_height !== undefined && object.block_height !== null) {
      message.block_height = object.block_height;
    } else {
      message.block_height = 0;
    }
    return message;
  },
};

const baseRelayReply: object = {};

export const RelayReply = {
  encode(message: RelayReply, writer: Writer = Writer.create()): Writer {
    if (message.data.length !== 0) {
      writer.uint32(10).bytes(message.data);
    }
    if (message.sig.length !== 0) {
      writer.uint32(18).bytes(message.sig);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): RelayReply {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseRelayReply } as RelayReply;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.data = reader.bytes();
          break;
        case 2:
          message.sig = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): RelayReply {
    const message = { ...baseRelayReply } as RelayReply;
    if (object.data !== undefined && object.data !== null) {
      message.data = bytesFromBase64(object.data);
    }
    if (object.sig !== undefined && object.sig !== null) {
      message.sig = bytesFromBase64(object.sig);
    }
    return message;
  },

  toJSON(message: RelayReply): unknown {
    const obj: any = {};
    message.data !== undefined &&
      (obj.data = base64FromBytes(
        message.data !== undefined ? message.data : new Uint8Array()
      ));
    message.sig !== undefined &&
      (obj.sig = base64FromBytes(
        message.sig !== undefined ? message.sig : new Uint8Array()
      ));
    return obj;
  },

  fromPartial(object: DeepPartial<RelayReply>): RelayReply {
    const message = { ...baseRelayReply } as RelayReply;
    if (object.data !== undefined && object.data !== null) {
      message.data = object.data;
    } else {
      message.data = new Uint8Array();
    }
    if (object.sig !== undefined && object.sig !== null) {
      message.sig = object.sig;
    } else {
      message.sig = new Uint8Array();
    }
    return message;
  },
};

export interface Relayer {
  Relay(request: RelayRequest): Promise<RelayReply>;
}

export class RelayerClientImpl implements Relayer {
  private readonly rpc: Rpc;
  constructor(rpc: Rpc) {
    this.rpc = rpc;
  }
  Relay(request: RelayRequest): Promise<RelayReply> {
    const data = RelayRequest.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.servicer.Relayer",
      "Relay",
      data
    );
    return promise.then((data) => RelayReply.decode(new Reader(data)));
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

const atob: (b64: string) => string =
  globalThis.atob ||
  ((b64) => globalThis.Buffer.from(b64, "base64").toString("binary"));
function bytesFromBase64(b64: string): Uint8Array {
  const bin = atob(b64);
  const arr = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; ++i) {
    arr[i] = bin.charCodeAt(i);
  }
  return arr;
}

const btoa: (bin: string) => string =
  globalThis.btoa ||
  ((bin) => globalThis.Buffer.from(bin, "binary").toString("base64"));
function base64FromBytes(arr: Uint8Array): string {
  const bin: string[] = [];
  for (let i = 0; i < arr.byteLength; ++i) {
    bin.push(String.fromCharCode(arr[i]));
  }
  return btoa(bin.join(""));
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

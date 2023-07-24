/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.epochstorage";

export interface FixatedParams {
  index: string;
  parameter: Uint8Array;
  fixationBlock: Long;
}

function createBaseFixatedParams(): FixatedParams {
  return { index: "", parameter: new Uint8Array(), fixationBlock: Long.UZERO };
}

export const FixatedParams = {
  encode(message: FixatedParams, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    if (message.parameter.length !== 0) {
      writer.uint32(18).bytes(message.parameter);
    }
    if (!message.fixationBlock.isZero()) {
      writer.uint32(24).uint64(message.fixationBlock);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): FixatedParams {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFixatedParams();
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

          message.parameter = reader.bytes();
          continue;
        case 3:
          if (tag != 24) {
            break;
          }

          message.fixationBlock = reader.uint64() as Long;
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): FixatedParams {
    return {
      index: isSet(object.index) ? String(object.index) : "",
      parameter: isSet(object.parameter) ? bytesFromBase64(object.parameter) : new Uint8Array(),
      fixationBlock: isSet(object.fixationBlock) ? Long.fromValue(object.fixationBlock) : Long.UZERO,
    };
  },

  toJSON(message: FixatedParams): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    message.parameter !== undefined &&
      (obj.parameter = base64FromBytes(message.parameter !== undefined ? message.parameter : new Uint8Array()));
    message.fixationBlock !== undefined && (obj.fixationBlock = (message.fixationBlock || Long.UZERO).toString());
    return obj;
  },

  create<I extends Exact<DeepPartial<FixatedParams>, I>>(base?: I): FixatedParams {
    return FixatedParams.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<FixatedParams>, I>>(object: I): FixatedParams {
    const message = createBaseFixatedParams();
    message.index = object.index ?? "";
    message.parameter = object.parameter ?? new Uint8Array();
    message.fixationBlock = (object.fixationBlock !== undefined && object.fixationBlock !== null)
      ? Long.fromValue(object.fixationBlock)
      : Long.UZERO;
    return message;
  },
};

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

/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.servicer";

export interface UniquePaymentStorageUserServicer {
  index: string;
  block: number;
}

const baseUniquePaymentStorageUserServicer: object = { index: "", block: 0 };

export const UniquePaymentStorageUserServicer = {
  encode(
    message: UniquePaymentStorageUserServicer,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    if (message.block !== 0) {
      writer.uint32(16).uint64(message.block);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): UniquePaymentStorageUserServicer {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseUniquePaymentStorageUserServicer,
    } as UniquePaymentStorageUserServicer;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.index = reader.string();
          break;
        case 2:
          message.block = longToNumber(reader.uint64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): UniquePaymentStorageUserServicer {
    const message = {
      ...baseUniquePaymentStorageUserServicer,
    } as UniquePaymentStorageUserServicer;
    if (object.index !== undefined && object.index !== null) {
      message.index = String(object.index);
    } else {
      message.index = "";
    }
    if (object.block !== undefined && object.block !== null) {
      message.block = Number(object.block);
    } else {
      message.block = 0;
    }
    return message;
  },

  toJSON(message: UniquePaymentStorageUserServicer): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    message.block !== undefined && (obj.block = message.block);
    return obj;
  },

  fromPartial(
    object: DeepPartial<UniquePaymentStorageUserServicer>
  ): UniquePaymentStorageUserServicer {
    const message = {
      ...baseUniquePaymentStorageUserServicer,
    } as UniquePaymentStorageUserServicer;
    if (object.index !== undefined && object.index !== null) {
      message.index = object.index;
    } else {
      message.index = "";
    }
    if (object.block !== undefined && object.block !== null) {
      message.block = object.block;
    } else {
      message.block = 0;
    }
    return message;
  },
};

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

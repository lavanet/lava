/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.servicer";

export interface PreviousSessionBlocks {
  blocksNum: number;
}

const basePreviousSessionBlocks: object = { blocksNum: 0 };

export const PreviousSessionBlocks = {
  encode(
    message: PreviousSessionBlocks,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.blocksNum !== 0) {
      writer.uint32(8).uint64(message.blocksNum);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): PreviousSessionBlocks {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...basePreviousSessionBlocks } as PreviousSessionBlocks;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.blocksNum = longToNumber(reader.uint64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PreviousSessionBlocks {
    const message = { ...basePreviousSessionBlocks } as PreviousSessionBlocks;
    if (object.blocksNum !== undefined && object.blocksNum !== null) {
      message.blocksNum = Number(object.blocksNum);
    } else {
      message.blocksNum = 0;
    }
    return message;
  },

  toJSON(message: PreviousSessionBlocks): unknown {
    const obj: any = {};
    message.blocksNum !== undefined && (obj.blocksNum = message.blocksNum);
    return obj;
  },

  fromPartial(
    object: DeepPartial<PreviousSessionBlocks>
  ): PreviousSessionBlocks {
    const message = { ...basePreviousSessionBlocks } as PreviousSessionBlocks;
    if (object.blocksNum !== undefined && object.blocksNum !== null) {
      message.blocksNum = object.blocksNum;
    } else {
      message.blocksNum = 0;
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

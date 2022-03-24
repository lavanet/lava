/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";
import { Params } from "../spec/params";
import { Spec } from "../spec/spec";

export const protobufPackage = "lavanet.lava.spec";

/** GenesisState defines the spec module's genesis state. */
export interface GenesisState {
  params: Params | undefined;
  specList: Spec[];
  /** this line is used by starport scaffolding # genesis/proto/state */
  specCount: number;
}

const baseGenesisState: object = { specCount: 0 };

export const GenesisState = {
  encode(message: GenesisState, writer: Writer = Writer.create()): Writer {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.specList) {
      Spec.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    if (message.specCount !== 0) {
      writer.uint32(24).uint64(message.specCount);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): GenesisState {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseGenesisState } as GenesisState;
    message.specList = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.params = Params.decode(reader, reader.uint32());
          break;
        case 2:
          message.specList.push(Spec.decode(reader, reader.uint32()));
          break;
        case 3:
          message.specCount = longToNumber(reader.uint64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): GenesisState {
    const message = { ...baseGenesisState } as GenesisState;
    message.specList = [];
    if (object.params !== undefined && object.params !== null) {
      message.params = Params.fromJSON(object.params);
    } else {
      message.params = undefined;
    }
    if (object.specList !== undefined && object.specList !== null) {
      for (const e of object.specList) {
        message.specList.push(Spec.fromJSON(e));
      }
    }
    if (object.specCount !== undefined && object.specCount !== null) {
      message.specCount = Number(object.specCount);
    } else {
      message.specCount = 0;
    }
    return message;
  },

  toJSON(message: GenesisState): unknown {
    const obj: any = {};
    message.params !== undefined &&
      (obj.params = message.params ? Params.toJSON(message.params) : undefined);
    if (message.specList) {
      obj.specList = message.specList.map((e) =>
        e ? Spec.toJSON(e) : undefined
      );
    } else {
      obj.specList = [];
    }
    message.specCount !== undefined && (obj.specCount = message.specCount);
    return obj;
  },

  fromPartial(object: DeepPartial<GenesisState>): GenesisState {
    const message = { ...baseGenesisState } as GenesisState;
    message.specList = [];
    if (object.params !== undefined && object.params !== null) {
      message.params = Params.fromPartial(object.params);
    } else {
      message.params = undefined;
    }
    if (object.specList !== undefined && object.specList !== null) {
      for (const e of object.specList) {
        message.specList.push(Spec.fromPartial(e));
      }
    }
    if (object.specCount !== undefined && object.specCount !== null) {
      message.specCount = object.specCount;
    } else {
      message.specCount = 0;
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

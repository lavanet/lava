/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { StakeEntry } from "./stake_entry";

export const protobufPackage = "lavanet.lava.epochstorage";

export interface StakeStorage {
  index: string;
  stakeEntries: StakeEntry[];
  epochBlockHash: Uint8Array;
}

function createBaseStakeStorage(): StakeStorage {
  return { index: "", stakeEntries: [], epochBlockHash: new Uint8Array() };
}

export const StakeStorage = {
  encode(message: StakeStorage, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    for (const v of message.stakeEntries) {
      StakeEntry.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    if (message.epochBlockHash.length !== 0) {
      writer.uint32(26).bytes(message.epochBlockHash);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): StakeStorage {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseStakeStorage();
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

          message.stakeEntries.push(StakeEntry.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.epochBlockHash = reader.bytes();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): StakeStorage {
    return {
      index: isSet(object.index) ? String(object.index) : "",
      stakeEntries: Array.isArray(object?.stakeEntries)
        ? object.stakeEntries.map((e: any) => StakeEntry.fromJSON(e))
        : [],
      epochBlockHash: isSet(object.epochBlockHash) ? bytesFromBase64(object.epochBlockHash) : new Uint8Array(),
    };
  },

  toJSON(message: StakeStorage): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    if (message.stakeEntries) {
      obj.stakeEntries = message.stakeEntries.map((e) => e ? StakeEntry.toJSON(e) : undefined);
    } else {
      obj.stakeEntries = [];
    }
    message.epochBlockHash !== undefined &&
      (obj.epochBlockHash = base64FromBytes(
        message.epochBlockHash !== undefined ? message.epochBlockHash : new Uint8Array(),
      ));
    return obj;
  },

  create<I extends Exact<DeepPartial<StakeStorage>, I>>(base?: I): StakeStorage {
    return StakeStorage.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<StakeStorage>, I>>(object: I): StakeStorage {
    const message = createBaseStakeStorage();
    message.index = object.index ?? "";
    message.stakeEntries = object.stakeEntries?.map((e) => StakeEntry.fromPartial(e)) || [];
    message.epochBlockHash = object.epochBlockHash ?? new Uint8Array();
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

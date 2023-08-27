/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.common";

export interface Entry {
  /** unique entry index (i.e. list key) */
  index: string;
  /** block when the entry was created */
  block: Long;
  /** block when the entry becomes stale */
  staleAt: Long;
  /** reference count */
  refcount: Long;
  /** the data saved in the entry */
  data: Uint8Array;
  /** block when the entry becomes deleted */
  deleteAt: Long;
  /** is this entry the latest entry now? */
  isLatest: boolean;
}

export interface RawMessage {
  key: Uint8Array;
  value: Uint8Array;
}

function createBaseEntry(): Entry {
  return {
    index: "",
    block: Long.UZERO,
    staleAt: Long.UZERO,
    refcount: Long.UZERO,
    data: new Uint8Array(),
    deleteAt: Long.UZERO,
    isLatest: false,
  };
}

export const Entry = {
  encode(message: Entry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    if (!message.block.isZero()) {
      writer.uint32(16).uint64(message.block);
    }
    if (!message.staleAt.isZero()) {
      writer.uint32(24).uint64(message.staleAt);
    }
    if (!message.refcount.isZero()) {
      writer.uint32(32).uint64(message.refcount);
    }
    if (message.data.length !== 0) {
      writer.uint32(42).bytes(message.data);
    }
    if (!message.deleteAt.isZero()) {
      writer.uint32(48).uint64(message.deleteAt);
    }
    if (message.isLatest === true) {
      writer.uint32(56).bool(message.isLatest);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Entry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEntry();
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
          if (tag != 16) {
            break;
          }

          message.block = reader.uint64() as Long;
          continue;
        case 3:
          if (tag != 24) {
            break;
          }

          message.staleAt = reader.uint64() as Long;
          continue;
        case 4:
          if (tag != 32) {
            break;
          }

          message.refcount = reader.uint64() as Long;
          continue;
        case 5:
          if (tag != 42) {
            break;
          }

          message.data = reader.bytes();
          continue;
        case 6:
          if (tag != 48) {
            break;
          }

          message.deleteAt = reader.uint64() as Long;
          continue;
        case 7:
          if (tag != 56) {
            break;
          }

          message.isLatest = reader.bool();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Entry {
    return {
      index: isSet(object.index) ? String(object.index) : "",
      block: isSet(object.block) ? Long.fromValue(object.block) : Long.UZERO,
      staleAt: isSet(object.staleAt) ? Long.fromValue(object.staleAt) : Long.UZERO,
      refcount: isSet(object.refcount) ? Long.fromValue(object.refcount) : Long.UZERO,
      data: isSet(object.data) ? bytesFromBase64(object.data) : new Uint8Array(),
      deleteAt: isSet(object.deleteAt) ? Long.fromValue(object.deleteAt) : Long.UZERO,
      isLatest: isSet(object.isLatest) ? Boolean(object.isLatest) : false,
    };
  },

  toJSON(message: Entry): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    message.block !== undefined && (obj.block = (message.block || Long.UZERO).toString());
    message.staleAt !== undefined && (obj.staleAt = (message.staleAt || Long.UZERO).toString());
    message.refcount !== undefined && (obj.refcount = (message.refcount || Long.UZERO).toString());
    message.data !== undefined &&
      (obj.data = base64FromBytes(message.data !== undefined ? message.data : new Uint8Array()));
    message.deleteAt !== undefined && (obj.deleteAt = (message.deleteAt || Long.UZERO).toString());
    message.isLatest !== undefined && (obj.isLatest = message.isLatest);
    return obj;
  },

  create<I extends Exact<DeepPartial<Entry>, I>>(base?: I): Entry {
    return Entry.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<Entry>, I>>(object: I): Entry {
    const message = createBaseEntry();
    message.index = object.index ?? "";
    message.block = (object.block !== undefined && object.block !== null) ? Long.fromValue(object.block) : Long.UZERO;
    message.staleAt = (object.staleAt !== undefined && object.staleAt !== null)
      ? Long.fromValue(object.staleAt)
      : Long.UZERO;
    message.refcount = (object.refcount !== undefined && object.refcount !== null)
      ? Long.fromValue(object.refcount)
      : Long.UZERO;
    message.data = object.data ?? new Uint8Array();
    message.deleteAt = (object.deleteAt !== undefined && object.deleteAt !== null)
      ? Long.fromValue(object.deleteAt)
      : Long.UZERO;
    message.isLatest = object.isLatest ?? false;
    return message;
  },
};

function createBaseRawMessage(): RawMessage {
  return { key: new Uint8Array(), value: new Uint8Array() };
}

export const RawMessage = {
  encode(message: RawMessage, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key.length !== 0) {
      writer.uint32(10).bytes(message.key);
    }
    if (message.value.length !== 0) {
      writer.uint32(18).bytes(message.value);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RawMessage {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRawMessage();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.key = reader.bytes();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.value = reader.bytes();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): RawMessage {
    return {
      key: isSet(object.key) ? bytesFromBase64(object.key) : new Uint8Array(),
      value: isSet(object.value) ? bytesFromBase64(object.value) : new Uint8Array(),
    };
  },

  toJSON(message: RawMessage): unknown {
    const obj: any = {};
    message.key !== undefined &&
      (obj.key = base64FromBytes(message.key !== undefined ? message.key : new Uint8Array()));
    message.value !== undefined &&
      (obj.value = base64FromBytes(message.value !== undefined ? message.value : new Uint8Array()));
    return obj;
  },

  create<I extends Exact<DeepPartial<RawMessage>, I>>(base?: I): RawMessage {
    return RawMessage.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<RawMessage>, I>>(object: I): RawMessage {
    const message = createBaseRawMessage();
    message.key = object.key ?? new Uint8Array();
    message.value = object.value ?? new Uint8Array();
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

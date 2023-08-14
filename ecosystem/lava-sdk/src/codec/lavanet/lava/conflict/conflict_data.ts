/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { RelayReply, RelayRequest } from "../pairing/relay";

export const protobufPackage = "lavanet.lava.conflict";

export interface ResponseConflict {
  conflictRelayData0?: ConflictRelayData;
  conflictRelayData1?: ConflictRelayData;
}

export interface ConflictRelayData {
  request?: RelayRequest;
  reply?: ReplyMetadata;
}

export interface ReplyMetadata {
  hashAllDataHash: Uint8Array;
  sig: Uint8Array;
  latestBlock: Long;
  finalizedBlocksHashes: Uint8Array;
  sigBlocks: Uint8Array;
}

export interface FinalizationConflict {
  relayReply0?: RelayReply;
  relayReply1?: RelayReply;
}

function createBaseResponseConflict(): ResponseConflict {
  return { conflictRelayData0: undefined, conflictRelayData1: undefined };
}

export const ResponseConflict = {
  encode(message: ResponseConflict, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.conflictRelayData0 !== undefined) {
      ConflictRelayData.encode(message.conflictRelayData0, writer.uint32(10).fork()).ldelim();
    }
    if (message.conflictRelayData1 !== undefined) {
      ConflictRelayData.encode(message.conflictRelayData1, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ResponseConflict {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseResponseConflict();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.conflictRelayData0 = ConflictRelayData.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.conflictRelayData1 = ConflictRelayData.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ResponseConflict {
    return {
      conflictRelayData0: isSet(object.conflictRelayData0)
        ? ConflictRelayData.fromJSON(object.conflictRelayData0)
        : undefined,
      conflictRelayData1: isSet(object.conflictRelayData1)
        ? ConflictRelayData.fromJSON(object.conflictRelayData1)
        : undefined,
    };
  },

  toJSON(message: ResponseConflict): unknown {
    const obj: any = {};
    message.conflictRelayData0 !== undefined && (obj.conflictRelayData0 = message.conflictRelayData0
      ? ConflictRelayData.toJSON(message.conflictRelayData0)
      : undefined);
    message.conflictRelayData1 !== undefined && (obj.conflictRelayData1 = message.conflictRelayData1
      ? ConflictRelayData.toJSON(message.conflictRelayData1)
      : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<ResponseConflict>, I>>(base?: I): ResponseConflict {
    return ResponseConflict.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<ResponseConflict>, I>>(object: I): ResponseConflict {
    const message = createBaseResponseConflict();
    message.conflictRelayData0 = (object.conflictRelayData0 !== undefined && object.conflictRelayData0 !== null)
      ? ConflictRelayData.fromPartial(object.conflictRelayData0)
      : undefined;
    message.conflictRelayData1 = (object.conflictRelayData1 !== undefined && object.conflictRelayData1 !== null)
      ? ConflictRelayData.fromPartial(object.conflictRelayData1)
      : undefined;
    return message;
  },
};

function createBaseConflictRelayData(): ConflictRelayData {
  return { request: undefined, reply: undefined };
}

export const ConflictRelayData = {
  encode(message: ConflictRelayData, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.request !== undefined) {
      RelayRequest.encode(message.request, writer.uint32(10).fork()).ldelim();
    }
    if (message.reply !== undefined) {
      ReplyMetadata.encode(message.reply, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ConflictRelayData {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseConflictRelayData();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.request = RelayRequest.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.reply = ReplyMetadata.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ConflictRelayData {
    return {
      request: isSet(object.request) ? RelayRequest.fromJSON(object.request) : undefined,
      reply: isSet(object.reply) ? ReplyMetadata.fromJSON(object.reply) : undefined,
    };
  },

  toJSON(message: ConflictRelayData): unknown {
    const obj: any = {};
    message.request !== undefined && (obj.request = message.request ? RelayRequest.toJSON(message.request) : undefined);
    message.reply !== undefined && (obj.reply = message.reply ? ReplyMetadata.toJSON(message.reply) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<ConflictRelayData>, I>>(base?: I): ConflictRelayData {
    return ConflictRelayData.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<ConflictRelayData>, I>>(object: I): ConflictRelayData {
    const message = createBaseConflictRelayData();
    message.request = (object.request !== undefined && object.request !== null)
      ? RelayRequest.fromPartial(object.request)
      : undefined;
    message.reply = (object.reply !== undefined && object.reply !== null)
      ? ReplyMetadata.fromPartial(object.reply)
      : undefined;
    return message;
  },
};

function createBaseReplyMetadata(): ReplyMetadata {
  return {
    hashAllDataHash: new Uint8Array(),
    sig: new Uint8Array(),
    latestBlock: Long.ZERO,
    finalizedBlocksHashes: new Uint8Array(),
    sigBlocks: new Uint8Array(),
  };
}

export const ReplyMetadata = {
  encode(message: ReplyMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.hashAllDataHash.length !== 0) {
      writer.uint32(10).bytes(message.hashAllDataHash);
    }
    if (message.sig.length !== 0) {
      writer.uint32(18).bytes(message.sig);
    }
    if (!message.latestBlock.isZero()) {
      writer.uint32(24).int64(message.latestBlock);
    }
    if (message.finalizedBlocksHashes.length !== 0) {
      writer.uint32(34).bytes(message.finalizedBlocksHashes);
    }
    if (message.sigBlocks.length !== 0) {
      writer.uint32(42).bytes(message.sigBlocks);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ReplyMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseReplyMetadata();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.hashAllDataHash = reader.bytes();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.sig = reader.bytes();
          continue;
        case 3:
          if (tag != 24) {
            break;
          }

          message.latestBlock = reader.int64() as Long;
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.finalizedBlocksHashes = reader.bytes();
          continue;
        case 5:
          if (tag != 42) {
            break;
          }

          message.sigBlocks = reader.bytes();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ReplyMetadata {
    return {
      hashAllDataHash: isSet(object.hashAllDataHash) ? bytesFromBase64(object.hashAllDataHash) : new Uint8Array(),
      sig: isSet(object.sig) ? bytesFromBase64(object.sig) : new Uint8Array(),
      latestBlock: isSet(object.latestBlock) ? Long.fromValue(object.latestBlock) : Long.ZERO,
      finalizedBlocksHashes: isSet(object.finalizedBlocksHashes)
        ? bytesFromBase64(object.finalizedBlocksHashes)
        : new Uint8Array(),
      sigBlocks: isSet(object.sigBlocks) ? bytesFromBase64(object.sigBlocks) : new Uint8Array(),
    };
  },

  toJSON(message: ReplyMetadata): unknown {
    const obj: any = {};
    message.hashAllDataHash !== undefined &&
      (obj.hashAllDataHash = base64FromBytes(
        message.hashAllDataHash !== undefined ? message.hashAllDataHash : new Uint8Array(),
      ));
    message.sig !== undefined &&
      (obj.sig = base64FromBytes(message.sig !== undefined ? message.sig : new Uint8Array()));
    message.latestBlock !== undefined && (obj.latestBlock = (message.latestBlock || Long.ZERO).toString());
    message.finalizedBlocksHashes !== undefined &&
      (obj.finalizedBlocksHashes = base64FromBytes(
        message.finalizedBlocksHashes !== undefined ? message.finalizedBlocksHashes : new Uint8Array(),
      ));
    message.sigBlocks !== undefined &&
      (obj.sigBlocks = base64FromBytes(message.sigBlocks !== undefined ? message.sigBlocks : new Uint8Array()));
    return obj;
  },

  create<I extends Exact<DeepPartial<ReplyMetadata>, I>>(base?: I): ReplyMetadata {
    return ReplyMetadata.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<ReplyMetadata>, I>>(object: I): ReplyMetadata {
    const message = createBaseReplyMetadata();
    message.hashAllDataHash = object.hashAllDataHash ?? new Uint8Array();
    message.sig = object.sig ?? new Uint8Array();
    message.latestBlock = (object.latestBlock !== undefined && object.latestBlock !== null)
      ? Long.fromValue(object.latestBlock)
      : Long.ZERO;
    message.finalizedBlocksHashes = object.finalizedBlocksHashes ?? new Uint8Array();
    message.sigBlocks = object.sigBlocks ?? new Uint8Array();
    return message;
  },
};

function createBaseFinalizationConflict(): FinalizationConflict {
  return { relayReply0: undefined, relayReply1: undefined };
}

export const FinalizationConflict = {
  encode(message: FinalizationConflict, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.relayReply0 !== undefined) {
      RelayReply.encode(message.relayReply0, writer.uint32(10).fork()).ldelim();
    }
    if (message.relayReply1 !== undefined) {
      RelayReply.encode(message.relayReply1, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): FinalizationConflict {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFinalizationConflict();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.relayReply0 = RelayReply.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.relayReply1 = RelayReply.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): FinalizationConflict {
    return {
      relayReply0: isSet(object.relayReply0) ? RelayReply.fromJSON(object.relayReply0) : undefined,
      relayReply1: isSet(object.relayReply1) ? RelayReply.fromJSON(object.relayReply1) : undefined,
    };
  },

  toJSON(message: FinalizationConflict): unknown {
    const obj: any = {};
    message.relayReply0 !== undefined &&
      (obj.relayReply0 = message.relayReply0 ? RelayReply.toJSON(message.relayReply0) : undefined);
    message.relayReply1 !== undefined &&
      (obj.relayReply1 = message.relayReply1 ? RelayReply.toJSON(message.relayReply1) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<FinalizationConflict>, I>>(base?: I): FinalizationConflict {
    return FinalizationConflict.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<FinalizationConflict>, I>>(object: I): FinalizationConflict {
    const message = createBaseFinalizationConflict();
    message.relayReply0 = (object.relayReply0 !== undefined && object.relayReply0 !== null)
      ? RelayReply.fromPartial(object.relayReply0)
      : undefined;
    message.relayReply1 = (object.relayReply1 !== undefined && object.relayReply1 !== null)
      ? RelayReply.fromPartial(object.relayReply1)
      : undefined;
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

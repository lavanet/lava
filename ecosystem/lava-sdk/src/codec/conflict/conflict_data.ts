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
  reply?: RelayReply;
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
          if (tag !== 10) {
            break;
          }

          message.conflictRelayData0 = ConflictRelayData.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.conflictRelayData1 = ConflictRelayData.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
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
      RelayReply.encode(message.reply, writer.uint32(18).fork()).ldelim();
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
          if (tag !== 10) {
            break;
          }

          message.request = RelayRequest.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.reply = RelayReply.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ConflictRelayData {
    return {
      request: isSet(object.request) ? RelayRequest.fromJSON(object.request) : undefined,
      reply: isSet(object.reply) ? RelayReply.fromJSON(object.reply) : undefined,
    };
  },

  toJSON(message: ConflictRelayData): unknown {
    const obj: any = {};
    message.request !== undefined && (obj.request = message.request ? RelayRequest.toJSON(message.request) : undefined);
    message.reply !== undefined && (obj.reply = message.reply ? RelayReply.toJSON(message.reply) : undefined);
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
      ? RelayReply.fromPartial(object.reply)
      : undefined;
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
          if (tag !== 10) {
            break;
          }

          message.relayReply0 = RelayReply.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.relayReply1 = RelayReply.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
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

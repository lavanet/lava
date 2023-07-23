/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { FinalizationConflict, ResponseConflict } from "./conflict_data";

export const protobufPackage = "lavanet.lava.conflict";

/** TODO:: change coin type to another proto (define proto in another file int this directory) */
export interface MsgDetection {
  creator: string;
  finalizationConflict?: FinalizationConflict;
  responseConflict?: ResponseConflict;
  sameProviderConflict?: FinalizationConflict;
}

export interface MsgDetectionResponse {
}

export interface MsgConflictVoteCommit {
  creator: string;
  voteID: string;
  hash: Uint8Array;
}

export interface MsgConflictVoteCommitResponse {
}

export interface MsgConflictVoteReveal {
  creator: string;
  voteID: string;
  nonce: Long;
  hash: Uint8Array;
}

export interface MsgConflictVoteRevealResponse {
}

function createBaseMsgDetection(): MsgDetection {
  return { creator: "", finalizationConflict: undefined, responseConflict: undefined, sameProviderConflict: undefined };
}

export const MsgDetection = {
  encode(message: MsgDetection, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.finalizationConflict !== undefined) {
      FinalizationConflict.encode(message.finalizationConflict, writer.uint32(18).fork()).ldelim();
    }
    if (message.responseConflict !== undefined) {
      ResponseConflict.encode(message.responseConflict, writer.uint32(26).fork()).ldelim();
    }
    if (message.sameProviderConflict !== undefined) {
      FinalizationConflict.encode(message.sameProviderConflict, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgDetection {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgDetection();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.creator = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.finalizationConflict = FinalizationConflict.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.responseConflict = ResponseConflict.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.sameProviderConflict = FinalizationConflict.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MsgDetection {
    return {
      creator: isSet(object.creator) ? String(object.creator) : "",
      finalizationConflict: isSet(object.finalizationConflict)
        ? FinalizationConflict.fromJSON(object.finalizationConflict)
        : undefined,
      responseConflict: isSet(object.responseConflict) ? ResponseConflict.fromJSON(object.responseConflict) : undefined,
      sameProviderConflict: isSet(object.sameProviderConflict)
        ? FinalizationConflict.fromJSON(object.sameProviderConflict)
        : undefined,
    };
  },

  toJSON(message: MsgDetection): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.finalizationConflict !== undefined && (obj.finalizationConflict = message.finalizationConflict
      ? FinalizationConflict.toJSON(message.finalizationConflict)
      : undefined);
    message.responseConflict !== undefined &&
      (obj.responseConflict = message.responseConflict ? ResponseConflict.toJSON(message.responseConflict) : undefined);
    message.sameProviderConflict !== undefined && (obj.sameProviderConflict = message.sameProviderConflict
      ? FinalizationConflict.toJSON(message.sameProviderConflict)
      : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgDetection>, I>>(base?: I): MsgDetection {
    return MsgDetection.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgDetection>, I>>(object: I): MsgDetection {
    const message = createBaseMsgDetection();
    message.creator = object.creator ?? "";
    message.finalizationConflict = (object.finalizationConflict !== undefined && object.finalizationConflict !== null)
      ? FinalizationConflict.fromPartial(object.finalizationConflict)
      : undefined;
    message.responseConflict = (object.responseConflict !== undefined && object.responseConflict !== null)
      ? ResponseConflict.fromPartial(object.responseConflict)
      : undefined;
    message.sameProviderConflict = (object.sameProviderConflict !== undefined && object.sameProviderConflict !== null)
      ? FinalizationConflict.fromPartial(object.sameProviderConflict)
      : undefined;
    return message;
  },
};

function createBaseMsgDetectionResponse(): MsgDetectionResponse {
  return {};
}

export const MsgDetectionResponse = {
  encode(_: MsgDetectionResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgDetectionResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgDetectionResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(_: any): MsgDetectionResponse {
    return {};
  },

  toJSON(_: MsgDetectionResponse): unknown {
    const obj: any = {};
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgDetectionResponse>, I>>(base?: I): MsgDetectionResponse {
    return MsgDetectionResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgDetectionResponse>, I>>(_: I): MsgDetectionResponse {
    const message = createBaseMsgDetectionResponse();
    return message;
  },
};

function createBaseMsgConflictVoteCommit(): MsgConflictVoteCommit {
  return { creator: "", voteID: "", hash: new Uint8Array() };
}

export const MsgConflictVoteCommit = {
  encode(message: MsgConflictVoteCommit, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.voteID !== "") {
      writer.uint32(18).string(message.voteID);
    }
    if (message.hash.length !== 0) {
      writer.uint32(26).bytes(message.hash);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgConflictVoteCommit {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgConflictVoteCommit();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.creator = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.voteID = reader.string();
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.hash = reader.bytes();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MsgConflictVoteCommit {
    return {
      creator: isSet(object.creator) ? String(object.creator) : "",
      voteID: isSet(object.voteID) ? String(object.voteID) : "",
      hash: isSet(object.hash) ? bytesFromBase64(object.hash) : new Uint8Array(),
    };
  },

  toJSON(message: MsgConflictVoteCommit): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.voteID !== undefined && (obj.voteID = message.voteID);
    message.hash !== undefined &&
      (obj.hash = base64FromBytes(message.hash !== undefined ? message.hash : new Uint8Array()));
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgConflictVoteCommit>, I>>(base?: I): MsgConflictVoteCommit {
    return MsgConflictVoteCommit.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgConflictVoteCommit>, I>>(object: I): MsgConflictVoteCommit {
    const message = createBaseMsgConflictVoteCommit();
    message.creator = object.creator ?? "";
    message.voteID = object.voteID ?? "";
    message.hash = object.hash ?? new Uint8Array();
    return message;
  },
};

function createBaseMsgConflictVoteCommitResponse(): MsgConflictVoteCommitResponse {
  return {};
}

export const MsgConflictVoteCommitResponse = {
  encode(_: MsgConflictVoteCommitResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgConflictVoteCommitResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgConflictVoteCommitResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(_: any): MsgConflictVoteCommitResponse {
    return {};
  },

  toJSON(_: MsgConflictVoteCommitResponse): unknown {
    const obj: any = {};
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgConflictVoteCommitResponse>, I>>(base?: I): MsgConflictVoteCommitResponse {
    return MsgConflictVoteCommitResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgConflictVoteCommitResponse>, I>>(_: I): MsgConflictVoteCommitResponse {
    const message = createBaseMsgConflictVoteCommitResponse();
    return message;
  },
};

function createBaseMsgConflictVoteReveal(): MsgConflictVoteReveal {
  return { creator: "", voteID: "", nonce: Long.ZERO, hash: new Uint8Array() };
}

export const MsgConflictVoteReveal = {
  encode(message: MsgConflictVoteReveal, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.voteID !== "") {
      writer.uint32(18).string(message.voteID);
    }
    if (!message.nonce.isZero()) {
      writer.uint32(24).int64(message.nonce);
    }
    if (message.hash.length !== 0) {
      writer.uint32(34).bytes(message.hash);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgConflictVoteReveal {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgConflictVoteReveal();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.creator = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.voteID = reader.string();
          continue;
        case 3:
          if (tag != 24) {
            break;
          }

          message.nonce = reader.int64() as Long;
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.hash = reader.bytes();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MsgConflictVoteReveal {
    return {
      creator: isSet(object.creator) ? String(object.creator) : "",
      voteID: isSet(object.voteID) ? String(object.voteID) : "",
      nonce: isSet(object.nonce) ? Long.fromValue(object.nonce) : Long.ZERO,
      hash: isSet(object.hash) ? bytesFromBase64(object.hash) : new Uint8Array(),
    };
  },

  toJSON(message: MsgConflictVoteReveal): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.voteID !== undefined && (obj.voteID = message.voteID);
    message.nonce !== undefined && (obj.nonce = (message.nonce || Long.ZERO).toString());
    message.hash !== undefined &&
      (obj.hash = base64FromBytes(message.hash !== undefined ? message.hash : new Uint8Array()));
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgConflictVoteReveal>, I>>(base?: I): MsgConflictVoteReveal {
    return MsgConflictVoteReveal.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgConflictVoteReveal>, I>>(object: I): MsgConflictVoteReveal {
    const message = createBaseMsgConflictVoteReveal();
    message.creator = object.creator ?? "";
    message.voteID = object.voteID ?? "";
    message.nonce = (object.nonce !== undefined && object.nonce !== null) ? Long.fromValue(object.nonce) : Long.ZERO;
    message.hash = object.hash ?? new Uint8Array();
    return message;
  },
};

function createBaseMsgConflictVoteRevealResponse(): MsgConflictVoteRevealResponse {
  return {};
}

export const MsgConflictVoteRevealResponse = {
  encode(_: MsgConflictVoteRevealResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgConflictVoteRevealResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgConflictVoteRevealResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(_: any): MsgConflictVoteRevealResponse {
    return {};
  },

  toJSON(_: MsgConflictVoteRevealResponse): unknown {
    const obj: any = {};
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgConflictVoteRevealResponse>, I>>(base?: I): MsgConflictVoteRevealResponse {
    return MsgConflictVoteRevealResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgConflictVoteRevealResponse>, I>>(_: I): MsgConflictVoteRevealResponse {
    const message = createBaseMsgConflictVoteRevealResponse();
    return message;
  },
};

/** Msg defines the Msg service. */
export interface Msg {
  Detection(request: MsgDetection): Promise<MsgDetectionResponse>;
  ConflictVoteCommit(request: MsgConflictVoteCommit): Promise<MsgConflictVoteCommitResponse>;
  /** this line is used by starport scaffolding # proto/tx/rpc */
  ConflictVoteReveal(request: MsgConflictVoteReveal): Promise<MsgConflictVoteRevealResponse>;
}

export class MsgClientImpl implements Msg {
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || "lavanet.lava.conflict.Msg";
    this.rpc = rpc;
    this.Detection = this.Detection.bind(this);
    this.ConflictVoteCommit = this.ConflictVoteCommit.bind(this);
    this.ConflictVoteReveal = this.ConflictVoteReveal.bind(this);
  }
  Detection(request: MsgDetection): Promise<MsgDetectionResponse> {
    const data = MsgDetection.encode(request).finish();
    const promise = this.rpc.request(this.service, "Detection", data);
    return promise.then((data) => MsgDetectionResponse.decode(_m0.Reader.create(data)));
  }

  ConflictVoteCommit(request: MsgConflictVoteCommit): Promise<MsgConflictVoteCommitResponse> {
    const data = MsgConflictVoteCommit.encode(request).finish();
    const promise = this.rpc.request(this.service, "ConflictVoteCommit", data);
    return promise.then((data) => MsgConflictVoteCommitResponse.decode(_m0.Reader.create(data)));
  }

  ConflictVoteReveal(request: MsgConflictVoteReveal): Promise<MsgConflictVoteRevealResponse> {
    const data = MsgConflictVoteReveal.encode(request).finish();
    const promise = this.rpc.request(this.service, "ConflictVoteReveal", data);
    return promise.then((data) => MsgConflictVoteRevealResponse.decode(_m0.Reader.create(data)));
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

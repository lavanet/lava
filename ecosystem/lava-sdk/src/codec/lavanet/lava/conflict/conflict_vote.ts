/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.conflict";

export interface Provider {
  account: string;
  response: Uint8Array;
}

export interface Vote {
  address: string;
  Hash: Uint8Array;
  Result: Long;
}

export interface ConflictVote {
  index: string;
  clientAddress: string;
  voteDeadline: Long;
  voteStartBlock: Long;
  voteState: Long;
  chainID: string;
  apiUrl: string;
  requestData: Uint8Array;
  requestBlock: Long;
  firstProvider?: Provider;
  secondProvider?: Provider;
  votes: Vote[];
}

function createBaseProvider(): Provider {
  return { account: "", response: new Uint8Array() };
}

export const Provider = {
  encode(message: Provider, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.account !== "") {
      writer.uint32(10).string(message.account);
    }
    if (message.response.length !== 0) {
      writer.uint32(18).bytes(message.response);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Provider {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseProvider();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.account = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.response = reader.bytes();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Provider {
    return {
      account: isSet(object.account) ? String(object.account) : "",
      response: isSet(object.response) ? bytesFromBase64(object.response) : new Uint8Array(),
    };
  },

  toJSON(message: Provider): unknown {
    const obj: any = {};
    message.account !== undefined && (obj.account = message.account);
    message.response !== undefined &&
      (obj.response = base64FromBytes(message.response !== undefined ? message.response : new Uint8Array()));
    return obj;
  },

  create<I extends Exact<DeepPartial<Provider>, I>>(base?: I): Provider {
    return Provider.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<Provider>, I>>(object: I): Provider {
    const message = createBaseProvider();
    message.account = object.account ?? "";
    message.response = object.response ?? new Uint8Array();
    return message;
  },
};

function createBaseVote(): Vote {
  return { address: "", Hash: new Uint8Array(), Result: Long.ZERO };
}

export const Vote = {
  encode(message: Vote, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.address !== "") {
      writer.uint32(10).string(message.address);
    }
    if (message.Hash.length !== 0) {
      writer.uint32(18).bytes(message.Hash);
    }
    if (!message.Result.isZero()) {
      writer.uint32(24).int64(message.Result);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Vote {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseVote();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.address = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.Hash = reader.bytes();
          continue;
        case 3:
          if (tag != 24) {
            break;
          }

          message.Result = reader.int64() as Long;
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Vote {
    return {
      address: isSet(object.address) ? String(object.address) : "",
      Hash: isSet(object.Hash) ? bytesFromBase64(object.Hash) : new Uint8Array(),
      Result: isSet(object.Result) ? Long.fromValue(object.Result) : Long.ZERO,
    };
  },

  toJSON(message: Vote): unknown {
    const obj: any = {};
    message.address !== undefined && (obj.address = message.address);
    message.Hash !== undefined &&
      (obj.Hash = base64FromBytes(message.Hash !== undefined ? message.Hash : new Uint8Array()));
    message.Result !== undefined && (obj.Result = (message.Result || Long.ZERO).toString());
    return obj;
  },

  create<I extends Exact<DeepPartial<Vote>, I>>(base?: I): Vote {
    return Vote.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<Vote>, I>>(object: I): Vote {
    const message = createBaseVote();
    message.address = object.address ?? "";
    message.Hash = object.Hash ?? new Uint8Array();
    message.Result = (object.Result !== undefined && object.Result !== null)
      ? Long.fromValue(object.Result)
      : Long.ZERO;
    return message;
  },
};

function createBaseConflictVote(): ConflictVote {
  return {
    index: "",
    clientAddress: "",
    voteDeadline: Long.UZERO,
    voteStartBlock: Long.UZERO,
    voteState: Long.ZERO,
    chainID: "",
    apiUrl: "",
    requestData: new Uint8Array(),
    requestBlock: Long.UZERO,
    firstProvider: undefined,
    secondProvider: undefined,
    votes: [],
  };
}

export const ConflictVote = {
  encode(message: ConflictVote, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    if (message.clientAddress !== "") {
      writer.uint32(18).string(message.clientAddress);
    }
    if (!message.voteDeadline.isZero()) {
      writer.uint32(24).uint64(message.voteDeadline);
    }
    if (!message.voteStartBlock.isZero()) {
      writer.uint32(32).uint64(message.voteStartBlock);
    }
    if (!message.voteState.isZero()) {
      writer.uint32(40).int64(message.voteState);
    }
    if (message.chainID !== "") {
      writer.uint32(50).string(message.chainID);
    }
    if (message.apiUrl !== "") {
      writer.uint32(58).string(message.apiUrl);
    }
    if (message.requestData.length !== 0) {
      writer.uint32(66).bytes(message.requestData);
    }
    if (!message.requestBlock.isZero()) {
      writer.uint32(72).uint64(message.requestBlock);
    }
    if (message.firstProvider !== undefined) {
      Provider.encode(message.firstProvider, writer.uint32(82).fork()).ldelim();
    }
    if (message.secondProvider !== undefined) {
      Provider.encode(message.secondProvider, writer.uint32(90).fork()).ldelim();
    }
    for (const v of message.votes) {
      Vote.encode(v!, writer.uint32(98).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ConflictVote {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseConflictVote();
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

          message.clientAddress = reader.string();
          continue;
        case 3:
          if (tag != 24) {
            break;
          }

          message.voteDeadline = reader.uint64() as Long;
          continue;
        case 4:
          if (tag != 32) {
            break;
          }

          message.voteStartBlock = reader.uint64() as Long;
          continue;
        case 5:
          if (tag != 40) {
            break;
          }

          message.voteState = reader.int64() as Long;
          continue;
        case 6:
          if (tag != 50) {
            break;
          }

          message.chainID = reader.string();
          continue;
        case 7:
          if (tag != 58) {
            break;
          }

          message.apiUrl = reader.string();
          continue;
        case 8:
          if (tag != 66) {
            break;
          }

          message.requestData = reader.bytes();
          continue;
        case 9:
          if (tag != 72) {
            break;
          }

          message.requestBlock = reader.uint64() as Long;
          continue;
        case 10:
          if (tag != 82) {
            break;
          }

          message.firstProvider = Provider.decode(reader, reader.uint32());
          continue;
        case 11:
          if (tag != 90) {
            break;
          }

          message.secondProvider = Provider.decode(reader, reader.uint32());
          continue;
        case 12:
          if (tag != 98) {
            break;
          }

          message.votes.push(Vote.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ConflictVote {
    return {
      index: isSet(object.index) ? String(object.index) : "",
      clientAddress: isSet(object.clientAddress) ? String(object.clientAddress) : "",
      voteDeadline: isSet(object.voteDeadline) ? Long.fromValue(object.voteDeadline) : Long.UZERO,
      voteStartBlock: isSet(object.voteStartBlock) ? Long.fromValue(object.voteStartBlock) : Long.UZERO,
      voteState: isSet(object.voteState) ? Long.fromValue(object.voteState) : Long.ZERO,
      chainID: isSet(object.chainID) ? String(object.chainID) : "",
      apiUrl: isSet(object.apiUrl) ? String(object.apiUrl) : "",
      requestData: isSet(object.requestData) ? bytesFromBase64(object.requestData) : new Uint8Array(),
      requestBlock: isSet(object.requestBlock) ? Long.fromValue(object.requestBlock) : Long.UZERO,
      firstProvider: isSet(object.firstProvider) ? Provider.fromJSON(object.firstProvider) : undefined,
      secondProvider: isSet(object.secondProvider) ? Provider.fromJSON(object.secondProvider) : undefined,
      votes: Array.isArray(object?.votes) ? object.votes.map((e: any) => Vote.fromJSON(e)) : [],
    };
  },

  toJSON(message: ConflictVote): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    message.clientAddress !== undefined && (obj.clientAddress = message.clientAddress);
    message.voteDeadline !== undefined && (obj.voteDeadline = (message.voteDeadline || Long.UZERO).toString());
    message.voteStartBlock !== undefined && (obj.voteStartBlock = (message.voteStartBlock || Long.UZERO).toString());
    message.voteState !== undefined && (obj.voteState = (message.voteState || Long.ZERO).toString());
    message.chainID !== undefined && (obj.chainID = message.chainID);
    message.apiUrl !== undefined && (obj.apiUrl = message.apiUrl);
    message.requestData !== undefined &&
      (obj.requestData = base64FromBytes(message.requestData !== undefined ? message.requestData : new Uint8Array()));
    message.requestBlock !== undefined && (obj.requestBlock = (message.requestBlock || Long.UZERO).toString());
    message.firstProvider !== undefined &&
      (obj.firstProvider = message.firstProvider ? Provider.toJSON(message.firstProvider) : undefined);
    message.secondProvider !== undefined &&
      (obj.secondProvider = message.secondProvider ? Provider.toJSON(message.secondProvider) : undefined);
    if (message.votes) {
      obj.votes = message.votes.map((e) => e ? Vote.toJSON(e) : undefined);
    } else {
      obj.votes = [];
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<ConflictVote>, I>>(base?: I): ConflictVote {
    return ConflictVote.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<ConflictVote>, I>>(object: I): ConflictVote {
    const message = createBaseConflictVote();
    message.index = object.index ?? "";
    message.clientAddress = object.clientAddress ?? "";
    message.voteDeadline = (object.voteDeadline !== undefined && object.voteDeadline !== null)
      ? Long.fromValue(object.voteDeadline)
      : Long.UZERO;
    message.voteStartBlock = (object.voteStartBlock !== undefined && object.voteStartBlock !== null)
      ? Long.fromValue(object.voteStartBlock)
      : Long.UZERO;
    message.voteState = (object.voteState !== undefined && object.voteState !== null)
      ? Long.fromValue(object.voteState)
      : Long.ZERO;
    message.chainID = object.chainID ?? "";
    message.apiUrl = object.apiUrl ?? "";
    message.requestData = object.requestData ?? new Uint8Array();
    message.requestBlock = (object.requestBlock !== undefined && object.requestBlock !== null)
      ? Long.fromValue(object.requestBlock)
      : Long.UZERO;
    message.firstProvider = (object.firstProvider !== undefined && object.firstProvider !== null)
      ? Provider.fromPartial(object.firstProvider)
      : undefined;
    message.secondProvider = (object.secondProvider !== undefined && object.secondProvider !== null)
      ? Provider.fromPartial(object.secondProvider)
      : undefined;
    message.votes = object.votes?.map((e) => Vote.fromPartial(e)) || [];
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

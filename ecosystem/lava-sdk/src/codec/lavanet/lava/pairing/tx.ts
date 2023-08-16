/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Coin } from "../../../cosmos/base/v1beta1/coin";
import { Endpoint } from "../epochstorage/endpoint";
import { RelaySession } from "./relay";

export const protobufPackage = "lavanet.lava.pairing";

export interface MsgStakeProvider {
  creator: string;
  chainID: string;
  amount?: Coin;
  endpoints: Endpoint[];
  geolocation: Long;
  moniker: string;
}

export interface MsgStakeProviderResponse {
}

export interface MsgUnstakeProvider {
  creator: string;
  chainID: string;
}

export interface MsgUnstakeProviderResponse {
}

export interface MsgRelayPayment {
  creator: string;
  relays: RelaySession[];
  descriptionString: string;
}

export interface MsgRelayPaymentResponse {
}

export interface MsgFreezeProvider {
  creator: string;
  chainIds: string[];
  reason: string;
}

export interface MsgFreezeProviderResponse {
}

export interface MsgUnfreezeProvider {
  creator: string;
  chainIds: string[];
}

export interface MsgUnfreezeProviderResponse {
}

function createBaseMsgStakeProvider(): MsgStakeProvider {
  return { creator: "", chainID: "", amount: undefined, endpoints: [], geolocation: Long.UZERO, moniker: "" };
}

export const MsgStakeProvider = {
  encode(message: MsgStakeProvider, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.chainID !== "") {
      writer.uint32(18).string(message.chainID);
    }
    if (message.amount !== undefined) {
      Coin.encode(message.amount, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.endpoints) {
      Endpoint.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    if (!message.geolocation.isZero()) {
      writer.uint32(40).uint64(message.geolocation);
    }
    if (message.moniker !== "") {
      writer.uint32(50).string(message.moniker);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgStakeProvider {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgStakeProvider();
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

          message.chainID = reader.string();
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.amount = Coin.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.endpoints.push(Endpoint.decode(reader, reader.uint32()));
          continue;
        case 5:
          if (tag != 40) {
            break;
          }

          message.geolocation = reader.uint64() as Long;
          continue;
        case 6:
          if (tag != 50) {
            break;
          }

          message.moniker = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MsgStakeProvider {
    return {
      creator: isSet(object.creator) ? String(object.creator) : "",
      chainID: isSet(object.chainID) ? String(object.chainID) : "",
      amount: isSet(object.amount) ? Coin.fromJSON(object.amount) : undefined,
      endpoints: Array.isArray(object?.endpoints) ? object.endpoints.map((e: any) => Endpoint.fromJSON(e)) : [],
      geolocation: isSet(object.geolocation) ? Long.fromValue(object.geolocation) : Long.UZERO,
      moniker: isSet(object.moniker) ? String(object.moniker) : "",
    };
  },

  toJSON(message: MsgStakeProvider): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.chainID !== undefined && (obj.chainID = message.chainID);
    message.amount !== undefined && (obj.amount = message.amount ? Coin.toJSON(message.amount) : undefined);
    if (message.endpoints) {
      obj.endpoints = message.endpoints.map((e) => e ? Endpoint.toJSON(e) : undefined);
    } else {
      obj.endpoints = [];
    }
    message.geolocation !== undefined && (obj.geolocation = (message.geolocation || Long.UZERO).toString());
    message.moniker !== undefined && (obj.moniker = message.moniker);
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgStakeProvider>, I>>(base?: I): MsgStakeProvider {
    return MsgStakeProvider.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgStakeProvider>, I>>(object: I): MsgStakeProvider {
    const message = createBaseMsgStakeProvider();
    message.creator = object.creator ?? "";
    message.chainID = object.chainID ?? "";
    message.amount = (object.amount !== undefined && object.amount !== null)
      ? Coin.fromPartial(object.amount)
      : undefined;
    message.endpoints = object.endpoints?.map((e) => Endpoint.fromPartial(e)) || [];
    message.geolocation = (object.geolocation !== undefined && object.geolocation !== null)
      ? Long.fromValue(object.geolocation)
      : Long.UZERO;
    message.moniker = object.moniker ?? "";
    return message;
  },
};

function createBaseMsgStakeProviderResponse(): MsgStakeProviderResponse {
  return {};
}

export const MsgStakeProviderResponse = {
  encode(_: MsgStakeProviderResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgStakeProviderResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgStakeProviderResponse();
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

  fromJSON(_: any): MsgStakeProviderResponse {
    return {};
  },

  toJSON(_: MsgStakeProviderResponse): unknown {
    const obj: any = {};
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgStakeProviderResponse>, I>>(base?: I): MsgStakeProviderResponse {
    return MsgStakeProviderResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgStakeProviderResponse>, I>>(_: I): MsgStakeProviderResponse {
    const message = createBaseMsgStakeProviderResponse();
    return message;
  },
};

function createBaseMsgUnstakeProvider(): MsgUnstakeProvider {
  return { creator: "", chainID: "" };
}

export const MsgUnstakeProvider = {
  encode(message: MsgUnstakeProvider, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.chainID !== "") {
      writer.uint32(18).string(message.chainID);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgUnstakeProvider {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUnstakeProvider();
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

          message.chainID = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MsgUnstakeProvider {
    return {
      creator: isSet(object.creator) ? String(object.creator) : "",
      chainID: isSet(object.chainID) ? String(object.chainID) : "",
    };
  },

  toJSON(message: MsgUnstakeProvider): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.chainID !== undefined && (obj.chainID = message.chainID);
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgUnstakeProvider>, I>>(base?: I): MsgUnstakeProvider {
    return MsgUnstakeProvider.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgUnstakeProvider>, I>>(object: I): MsgUnstakeProvider {
    const message = createBaseMsgUnstakeProvider();
    message.creator = object.creator ?? "";
    message.chainID = object.chainID ?? "";
    return message;
  },
};

function createBaseMsgUnstakeProviderResponse(): MsgUnstakeProviderResponse {
  return {};
}

export const MsgUnstakeProviderResponse = {
  encode(_: MsgUnstakeProviderResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgUnstakeProviderResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUnstakeProviderResponse();
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

  fromJSON(_: any): MsgUnstakeProviderResponse {
    return {};
  },

  toJSON(_: MsgUnstakeProviderResponse): unknown {
    const obj: any = {};
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgUnstakeProviderResponse>, I>>(base?: I): MsgUnstakeProviderResponse {
    return MsgUnstakeProviderResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgUnstakeProviderResponse>, I>>(_: I): MsgUnstakeProviderResponse {
    const message = createBaseMsgUnstakeProviderResponse();
    return message;
  },
};

function createBaseMsgRelayPayment(): MsgRelayPayment {
  return { creator: "", relays: [], descriptionString: "" };
}

export const MsgRelayPayment = {
  encode(message: MsgRelayPayment, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    for (const v of message.relays) {
      RelaySession.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    if (message.descriptionString !== "") {
      writer.uint32(34).string(message.descriptionString);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgRelayPayment {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRelayPayment();
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

          message.relays.push(RelaySession.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.descriptionString = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MsgRelayPayment {
    return {
      creator: isSet(object.creator) ? String(object.creator) : "",
      relays: Array.isArray(object?.relays) ? object.relays.map((e: any) => RelaySession.fromJSON(e)) : [],
      descriptionString: isSet(object.descriptionString) ? String(object.descriptionString) : "",
    };
  },

  toJSON(message: MsgRelayPayment): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    if (message.relays) {
      obj.relays = message.relays.map((e) => e ? RelaySession.toJSON(e) : undefined);
    } else {
      obj.relays = [];
    }
    message.descriptionString !== undefined && (obj.descriptionString = message.descriptionString);
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgRelayPayment>, I>>(base?: I): MsgRelayPayment {
    return MsgRelayPayment.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgRelayPayment>, I>>(object: I): MsgRelayPayment {
    const message = createBaseMsgRelayPayment();
    message.creator = object.creator ?? "";
    message.relays = object.relays?.map((e) => RelaySession.fromPartial(e)) || [];
    message.descriptionString = object.descriptionString ?? "";
    return message;
  },
};

function createBaseMsgRelayPaymentResponse(): MsgRelayPaymentResponse {
  return {};
}

export const MsgRelayPaymentResponse = {
  encode(_: MsgRelayPaymentResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgRelayPaymentResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRelayPaymentResponse();
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

  fromJSON(_: any): MsgRelayPaymentResponse {
    return {};
  },

  toJSON(_: MsgRelayPaymentResponse): unknown {
    const obj: any = {};
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgRelayPaymentResponse>, I>>(base?: I): MsgRelayPaymentResponse {
    return MsgRelayPaymentResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgRelayPaymentResponse>, I>>(_: I): MsgRelayPaymentResponse {
    const message = createBaseMsgRelayPaymentResponse();
    return message;
  },
};

function createBaseMsgFreezeProvider(): MsgFreezeProvider {
  return { creator: "", chainIds: [], reason: "" };
}

export const MsgFreezeProvider = {
  encode(message: MsgFreezeProvider, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    for (const v of message.chainIds) {
      writer.uint32(18).string(v!);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgFreezeProvider {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgFreezeProvider();
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

          message.chainIds.push(reader.string());
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.reason = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MsgFreezeProvider {
    return {
      creator: isSet(object.creator) ? String(object.creator) : "",
      chainIds: Array.isArray(object?.chainIds) ? object.chainIds.map((e: any) => String(e)) : [],
      reason: isSet(object.reason) ? String(object.reason) : "",
    };
  },

  toJSON(message: MsgFreezeProvider): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    if (message.chainIds) {
      obj.chainIds = message.chainIds.map((e) => e);
    } else {
      obj.chainIds = [];
    }
    message.reason !== undefined && (obj.reason = message.reason);
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgFreezeProvider>, I>>(base?: I): MsgFreezeProvider {
    return MsgFreezeProvider.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgFreezeProvider>, I>>(object: I): MsgFreezeProvider {
    const message = createBaseMsgFreezeProvider();
    message.creator = object.creator ?? "";
    message.chainIds = object.chainIds?.map((e) => e) || [];
    message.reason = object.reason ?? "";
    return message;
  },
};

function createBaseMsgFreezeProviderResponse(): MsgFreezeProviderResponse {
  return {};
}

export const MsgFreezeProviderResponse = {
  encode(_: MsgFreezeProviderResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgFreezeProviderResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgFreezeProviderResponse();
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

  fromJSON(_: any): MsgFreezeProviderResponse {
    return {};
  },

  toJSON(_: MsgFreezeProviderResponse): unknown {
    const obj: any = {};
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgFreezeProviderResponse>, I>>(base?: I): MsgFreezeProviderResponse {
    return MsgFreezeProviderResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgFreezeProviderResponse>, I>>(_: I): MsgFreezeProviderResponse {
    const message = createBaseMsgFreezeProviderResponse();
    return message;
  },
};

function createBaseMsgUnfreezeProvider(): MsgUnfreezeProvider {
  return { creator: "", chainIds: [] };
}

export const MsgUnfreezeProvider = {
  encode(message: MsgUnfreezeProvider, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    for (const v of message.chainIds) {
      writer.uint32(18).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgUnfreezeProvider {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUnfreezeProvider();
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

          message.chainIds.push(reader.string());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MsgUnfreezeProvider {
    return {
      creator: isSet(object.creator) ? String(object.creator) : "",
      chainIds: Array.isArray(object?.chainIds) ? object.chainIds.map((e: any) => String(e)) : [],
    };
  },

  toJSON(message: MsgUnfreezeProvider): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    if (message.chainIds) {
      obj.chainIds = message.chainIds.map((e) => e);
    } else {
      obj.chainIds = [];
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgUnfreezeProvider>, I>>(base?: I): MsgUnfreezeProvider {
    return MsgUnfreezeProvider.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgUnfreezeProvider>, I>>(object: I): MsgUnfreezeProvider {
    const message = createBaseMsgUnfreezeProvider();
    message.creator = object.creator ?? "";
    message.chainIds = object.chainIds?.map((e) => e) || [];
    return message;
  },
};

function createBaseMsgUnfreezeProviderResponse(): MsgUnfreezeProviderResponse {
  return {};
}

export const MsgUnfreezeProviderResponse = {
  encode(_: MsgUnfreezeProviderResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgUnfreezeProviderResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUnfreezeProviderResponse();
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

  fromJSON(_: any): MsgUnfreezeProviderResponse {
    return {};
  },

  toJSON(_: MsgUnfreezeProviderResponse): unknown {
    const obj: any = {};
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgUnfreezeProviderResponse>, I>>(base?: I): MsgUnfreezeProviderResponse {
    return MsgUnfreezeProviderResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgUnfreezeProviderResponse>, I>>(_: I): MsgUnfreezeProviderResponse {
    const message = createBaseMsgUnfreezeProviderResponse();
    return message;
  },
};

/** Msg defines the Msg service. */
export interface Msg {
  StakeProvider(request: MsgStakeProvider): Promise<MsgStakeProviderResponse>;
  UnstakeProvider(request: MsgUnstakeProvider): Promise<MsgUnstakeProviderResponse>;
  RelayPayment(request: MsgRelayPayment): Promise<MsgRelayPaymentResponse>;
  FreezeProvider(request: MsgFreezeProvider): Promise<MsgFreezeProviderResponse>;
  /** this line is used by starport scaffolding # proto/tx/rpc */
  UnfreezeProvider(request: MsgUnfreezeProvider): Promise<MsgUnfreezeProviderResponse>;
}

export class MsgClientImpl implements Msg {
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || "lavanet.lava.pairing.Msg";
    this.rpc = rpc;
    this.StakeProvider = this.StakeProvider.bind(this);
    this.UnstakeProvider = this.UnstakeProvider.bind(this);
    this.RelayPayment = this.RelayPayment.bind(this);
    this.FreezeProvider = this.FreezeProvider.bind(this);
    this.UnfreezeProvider = this.UnfreezeProvider.bind(this);
  }
  StakeProvider(request: MsgStakeProvider): Promise<MsgStakeProviderResponse> {
    const data = MsgStakeProvider.encode(request).finish();
    const promise = this.rpc.request(this.service, "StakeProvider", data);
    return promise.then((data) => MsgStakeProviderResponse.decode(_m0.Reader.create(data)));
  }

  UnstakeProvider(request: MsgUnstakeProvider): Promise<MsgUnstakeProviderResponse> {
    const data = MsgUnstakeProvider.encode(request).finish();
    const promise = this.rpc.request(this.service, "UnstakeProvider", data);
    return promise.then((data) => MsgUnstakeProviderResponse.decode(_m0.Reader.create(data)));
  }

  RelayPayment(request: MsgRelayPayment): Promise<MsgRelayPaymentResponse> {
    const data = MsgRelayPayment.encode(request).finish();
    const promise = this.rpc.request(this.service, "RelayPayment", data);
    return promise.then((data) => MsgRelayPaymentResponse.decode(_m0.Reader.create(data)));
  }

  FreezeProvider(request: MsgFreezeProvider): Promise<MsgFreezeProviderResponse> {
    const data = MsgFreezeProvider.encode(request).finish();
    const promise = this.rpc.request(this.service, "FreezeProvider", data);
    return promise.then((data) => MsgFreezeProviderResponse.decode(_m0.Reader.create(data)));
  }

  UnfreezeProvider(request: MsgUnfreezeProvider): Promise<MsgUnfreezeProviderResponse> {
    const data = MsgUnfreezeProvider.encode(request).finish();
    const promise = this.rpc.request(this.service, "UnfreezeProvider", data);
    return promise.then((data) => MsgUnfreezeProviderResponse.decode(_m0.Reader.create(data)));
  }
}

interface Rpc {
  request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
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

/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { RawMessage } from "../common/fixationEntry";
import { EpochPayments } from "./epoch_payments";
import { Params } from "./params";
import { ProviderPaymentStorage } from "./provider_payment_storage";
import { UniquePaymentStorageClientProvider } from "./unique_payment_storage_client_provider";

export const protobufPackage = "lavanet.lava.pairing";

export interface BadgeUsedCu {
  badgeUsedCuKey: Uint8Array;
  usedCu: Long;
}

/** GenesisState defines the pairing module's genesis state. */
export interface GenesisState {
  params?: Params;
  uniquePaymentStorageClientProviderList: UniquePaymentStorageClientProvider[];
  providerPaymentStorageList: ProviderPaymentStorage[];
  epochPaymentsList: EpochPayments[];
  badgeUsedCuList: BadgeUsedCu[];
  /** this line is used by starport scaffolding # genesis/proto/state */
  badgesTS: RawMessage[];
}

function createBaseBadgeUsedCu(): BadgeUsedCu {
  return { badgeUsedCuKey: new Uint8Array(), usedCu: Long.UZERO };
}

export const BadgeUsedCu = {
  encode(message: BadgeUsedCu, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.badgeUsedCuKey.length !== 0) {
      writer.uint32(10).bytes(message.badgeUsedCuKey);
    }
    if (!message.usedCu.isZero()) {
      writer.uint32(16).uint64(message.usedCu);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BadgeUsedCu {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBadgeUsedCu();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.badgeUsedCuKey = reader.bytes();
          continue;
        case 2:
          if (tag != 16) {
            break;
          }

          message.usedCu = reader.uint64() as Long;
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BadgeUsedCu {
    return {
      badgeUsedCuKey: isSet(object.badgeUsedCuKey) ? bytesFromBase64(object.badgeUsedCuKey) : new Uint8Array(),
      usedCu: isSet(object.usedCu) ? Long.fromValue(object.usedCu) : Long.UZERO,
    };
  },

  toJSON(message: BadgeUsedCu): unknown {
    const obj: any = {};
    message.badgeUsedCuKey !== undefined &&
      (obj.badgeUsedCuKey = base64FromBytes(
        message.badgeUsedCuKey !== undefined ? message.badgeUsedCuKey : new Uint8Array(),
      ));
    message.usedCu !== undefined && (obj.usedCu = (message.usedCu || Long.UZERO).toString());
    return obj;
  },

  create<I extends Exact<DeepPartial<BadgeUsedCu>, I>>(base?: I): BadgeUsedCu {
    return BadgeUsedCu.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<BadgeUsedCu>, I>>(object: I): BadgeUsedCu {
    const message = createBaseBadgeUsedCu();
    message.badgeUsedCuKey = object.badgeUsedCuKey ?? new Uint8Array();
    message.usedCu = (object.usedCu !== undefined && object.usedCu !== null)
      ? Long.fromValue(object.usedCu)
      : Long.UZERO;
    return message;
  },
};

function createBaseGenesisState(): GenesisState {
  return {
    params: undefined,
    uniquePaymentStorageClientProviderList: [],
    providerPaymentStorageList: [],
    epochPaymentsList: [],
    badgeUsedCuList: [],
    badgesTS: [],
  };
}

export const GenesisState = {
  encode(message: GenesisState, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.uniquePaymentStorageClientProviderList) {
      UniquePaymentStorageClientProvider.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.providerPaymentStorageList) {
      ProviderPaymentStorage.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.epochPaymentsList) {
      EpochPayments.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.badgeUsedCuList) {
      BadgeUsedCu.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    for (const v of message.badgesTS) {
      RawMessage.encode(v!, writer.uint32(50).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GenesisState {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGenesisState();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.params = Params.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.uniquePaymentStorageClientProviderList.push(
            UniquePaymentStorageClientProvider.decode(reader, reader.uint32()),
          );
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.providerPaymentStorageList.push(ProviderPaymentStorage.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.epochPaymentsList.push(EpochPayments.decode(reader, reader.uint32()));
          continue;
        case 5:
          if (tag != 42) {
            break;
          }

          message.badgeUsedCuList.push(BadgeUsedCu.decode(reader, reader.uint32()));
          continue;
        case 6:
          if (tag != 50) {
            break;
          }

          message.badgesTS.push(RawMessage.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GenesisState {
    return {
      params: isSet(object.params) ? Params.fromJSON(object.params) : undefined,
      uniquePaymentStorageClientProviderList: Array.isArray(object?.uniquePaymentStorageClientProviderList)
        ? object.uniquePaymentStorageClientProviderList.map((e: any) => UniquePaymentStorageClientProvider.fromJSON(e))
        : [],
      providerPaymentStorageList: Array.isArray(object?.providerPaymentStorageList)
        ? object.providerPaymentStorageList.map((e: any) => ProviderPaymentStorage.fromJSON(e))
        : [],
      epochPaymentsList: Array.isArray(object?.epochPaymentsList)
        ? object.epochPaymentsList.map((e: any) => EpochPayments.fromJSON(e))
        : [],
      badgeUsedCuList: Array.isArray(object?.badgeUsedCuList)
        ? object.badgeUsedCuList.map((e: any) => BadgeUsedCu.fromJSON(e))
        : [],
      badgesTS: Array.isArray(object?.badgesTS) ? object.badgesTS.map((e: any) => RawMessage.fromJSON(e)) : [],
    };
  },

  toJSON(message: GenesisState): unknown {
    const obj: any = {};
    message.params !== undefined && (obj.params = message.params ? Params.toJSON(message.params) : undefined);
    if (message.uniquePaymentStorageClientProviderList) {
      obj.uniquePaymentStorageClientProviderList = message.uniquePaymentStorageClientProviderList.map((e) =>
        e ? UniquePaymentStorageClientProvider.toJSON(e) : undefined
      );
    } else {
      obj.uniquePaymentStorageClientProviderList = [];
    }
    if (message.providerPaymentStorageList) {
      obj.providerPaymentStorageList = message.providerPaymentStorageList.map((e) =>
        e ? ProviderPaymentStorage.toJSON(e) : undefined
      );
    } else {
      obj.providerPaymentStorageList = [];
    }
    if (message.epochPaymentsList) {
      obj.epochPaymentsList = message.epochPaymentsList.map((e) => e ? EpochPayments.toJSON(e) : undefined);
    } else {
      obj.epochPaymentsList = [];
    }
    if (message.badgeUsedCuList) {
      obj.badgeUsedCuList = message.badgeUsedCuList.map((e) => e ? BadgeUsedCu.toJSON(e) : undefined);
    } else {
      obj.badgeUsedCuList = [];
    }
    if (message.badgesTS) {
      obj.badgesTS = message.badgesTS.map((e) => e ? RawMessage.toJSON(e) : undefined);
    } else {
      obj.badgesTS = [];
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<GenesisState>, I>>(base?: I): GenesisState {
    return GenesisState.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<GenesisState>, I>>(object: I): GenesisState {
    const message = createBaseGenesisState();
    message.params = (object.params !== undefined && object.params !== null)
      ? Params.fromPartial(object.params)
      : undefined;
    message.uniquePaymentStorageClientProviderList =
      object.uniquePaymentStorageClientProviderList?.map((e) => UniquePaymentStorageClientProvider.fromPartial(e)) ||
      [];
    message.providerPaymentStorageList =
      object.providerPaymentStorageList?.map((e) => ProviderPaymentStorage.fromPartial(e)) || [];
    message.epochPaymentsList = object.epochPaymentsList?.map((e) => EpochPayments.fromPartial(e)) || [];
    message.badgeUsedCuList = object.badgeUsedCuList?.map((e) => BadgeUsedCu.fromPartial(e)) || [];
    message.badgesTS = object.badgesTS?.map((e) => RawMessage.fromPartial(e)) || [];
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

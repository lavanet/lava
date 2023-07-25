/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.pairing";

export interface ProviderPaymentStorage {
  index: string;
  epoch: Long;
  uniquePaymentStorageClientProviderKeys: string[];
  /** total CU that were supposed to be served by the provider but didn't because he was unavailable (so consumers complained about him) */
  complainersTotalCu: Long;
}

function createBaseProviderPaymentStorage(): ProviderPaymentStorage {
  return { index: "", epoch: Long.UZERO, uniquePaymentStorageClientProviderKeys: [], complainersTotalCu: Long.UZERO };
}

export const ProviderPaymentStorage = {
  encode(message: ProviderPaymentStorage, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    if (!message.epoch.isZero()) {
      writer.uint32(24).uint64(message.epoch);
    }
    for (const v of message.uniquePaymentStorageClientProviderKeys) {
      writer.uint32(42).string(v!);
    }
    if (!message.complainersTotalCu.isZero()) {
      writer.uint32(48).uint64(message.complainersTotalCu);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ProviderPaymentStorage {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseProviderPaymentStorage();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.index = reader.string();
          continue;
        case 3:
          if (tag != 24) {
            break;
          }

          message.epoch = reader.uint64() as Long;
          continue;
        case 5:
          if (tag != 42) {
            break;
          }

          message.uniquePaymentStorageClientProviderKeys.push(reader.string());
          continue;
        case 6:
          if (tag != 48) {
            break;
          }

          message.complainersTotalCu = reader.uint64() as Long;
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ProviderPaymentStorage {
    return {
      index: isSet(object.index) ? String(object.index) : "",
      epoch: isSet(object.epoch) ? Long.fromValue(object.epoch) : Long.UZERO,
      uniquePaymentStorageClientProviderKeys: Array.isArray(object?.uniquePaymentStorageClientProviderKeys)
        ? object.uniquePaymentStorageClientProviderKeys.map((e: any) => String(e))
        : [],
      complainersTotalCu: isSet(object.complainersTotalCu) ? Long.fromValue(object.complainersTotalCu) : Long.UZERO,
    };
  },

  toJSON(message: ProviderPaymentStorage): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    message.epoch !== undefined && (obj.epoch = (message.epoch || Long.UZERO).toString());
    if (message.uniquePaymentStorageClientProviderKeys) {
      obj.uniquePaymentStorageClientProviderKeys = message.uniquePaymentStorageClientProviderKeys.map((e) => e);
    } else {
      obj.uniquePaymentStorageClientProviderKeys = [];
    }
    message.complainersTotalCu !== undefined &&
      (obj.complainersTotalCu = (message.complainersTotalCu || Long.UZERO).toString());
    return obj;
  },

  create<I extends Exact<DeepPartial<ProviderPaymentStorage>, I>>(base?: I): ProviderPaymentStorage {
    return ProviderPaymentStorage.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<ProviderPaymentStorage>, I>>(object: I): ProviderPaymentStorage {
    const message = createBaseProviderPaymentStorage();
    message.index = object.index ?? "";
    message.epoch = (object.epoch !== undefined && object.epoch !== null) ? Long.fromValue(object.epoch) : Long.UZERO;
    message.uniquePaymentStorageClientProviderKeys = object.uniquePaymentStorageClientProviderKeys?.map((e) => e) || [];
    message.complainersTotalCu = (object.complainersTotalCu !== undefined && object.complainersTotalCu !== null)
      ? Long.fromValue(object.complainersTotalCu)
      : Long.UZERO;
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

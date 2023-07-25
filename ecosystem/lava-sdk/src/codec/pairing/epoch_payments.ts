/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.pairing";

export interface EpochPayments {
  index: string;
  providerPaymentStorageKeys: string[];
}

function createBaseEpochPayments(): EpochPayments {
  return { index: "", providerPaymentStorageKeys: [] };
}

export const EpochPayments = {
  encode(message: EpochPayments, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    for (const v of message.providerPaymentStorageKeys) {
      writer.uint32(26).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): EpochPayments {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEpochPayments();
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
          if (tag != 26) {
            break;
          }

          message.providerPaymentStorageKeys.push(reader.string());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): EpochPayments {
    return {
      index: isSet(object.index) ? String(object.index) : "",
      providerPaymentStorageKeys: Array.isArray(object?.providerPaymentStorageKeys)
        ? object.providerPaymentStorageKeys.map((e: any) => String(e))
        : [],
    };
  },

  toJSON(message: EpochPayments): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    if (message.providerPaymentStorageKeys) {
      obj.providerPaymentStorageKeys = message.providerPaymentStorageKeys.map((e) => e);
    } else {
      obj.providerPaymentStorageKeys = [];
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<EpochPayments>, I>>(base?: I): EpochPayments {
    return EpochPayments.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<EpochPayments>, I>>(object: I): EpochPayments {
    const message = createBaseEpochPayments();
    message.index = object.index ?? "";
    message.providerPaymentStorageKeys = object.providerPaymentStorageKeys?.map((e) => e) || [];
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

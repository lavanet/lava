/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.pairing";

export interface UniquePaymentStorageClientProvider {
  index: string;
  block: Long;
  usedCU: Long;
}

function createBaseUniquePaymentStorageClientProvider(): UniquePaymentStorageClientProvider {
  return { index: "", block: Long.UZERO, usedCU: Long.UZERO };
}

export const UniquePaymentStorageClientProvider = {
  encode(message: UniquePaymentStorageClientProvider, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    if (!message.block.isZero()) {
      writer.uint32(16).uint64(message.block);
    }
    if (!message.usedCU.isZero()) {
      writer.uint32(24).uint64(message.usedCU);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UniquePaymentStorageClientProvider {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUniquePaymentStorageClientProvider();
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

          message.usedCU = reader.uint64() as Long;
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): UniquePaymentStorageClientProvider {
    return {
      index: isSet(object.index) ? String(object.index) : "",
      block: isSet(object.block) ? Long.fromValue(object.block) : Long.UZERO,
      usedCU: isSet(object.usedCU) ? Long.fromValue(object.usedCU) : Long.UZERO,
    };
  },

  toJSON(message: UniquePaymentStorageClientProvider): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    message.block !== undefined && (obj.block = (message.block || Long.UZERO).toString());
    message.usedCU !== undefined && (obj.usedCU = (message.usedCU || Long.UZERO).toString());
    return obj;
  },

  create<I extends Exact<DeepPartial<UniquePaymentStorageClientProvider>, I>>(
    base?: I,
  ): UniquePaymentStorageClientProvider {
    return UniquePaymentStorageClientProvider.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<UniquePaymentStorageClientProvider>, I>>(
    object: I,
  ): UniquePaymentStorageClientProvider {
    const message = createBaseUniquePaymentStorageClientProvider();
    message.index = object.index ?? "";
    message.block = (object.block !== undefined && object.block !== null) ? Long.fromValue(object.block) : Long.UZERO;
    message.usedCU = (object.usedCU !== undefined && object.usedCU !== null)
      ? Long.fromValue(object.usedCU)
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

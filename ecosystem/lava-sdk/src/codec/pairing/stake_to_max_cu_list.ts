/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Coin } from "../cosmos/base/v1beta1/coin";

export const protobufPackage = "lavanet.lava.pairing";

export interface StakeToMaxCUList {
  List: StakeToMaxCU[];
}

export interface StakeToMaxCU {
  StakeThreshold?: Coin;
  MaxComputeUnits: Long;
}

function createBaseStakeToMaxCUList(): StakeToMaxCUList {
  return { List: [] };
}

export const StakeToMaxCUList = {
  encode(message: StakeToMaxCUList, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.List) {
      StakeToMaxCU.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): StakeToMaxCUList {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseStakeToMaxCUList();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.List.push(StakeToMaxCU.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): StakeToMaxCUList {
    return { List: Array.isArray(object?.List) ? object.List.map((e: any) => StakeToMaxCU.fromJSON(e)) : [] };
  },

  toJSON(message: StakeToMaxCUList): unknown {
    const obj: any = {};
    if (message.List) {
      obj.List = message.List.map((e) => e ? StakeToMaxCU.toJSON(e) : undefined);
    } else {
      obj.List = [];
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<StakeToMaxCUList>, I>>(base?: I): StakeToMaxCUList {
    return StakeToMaxCUList.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<StakeToMaxCUList>, I>>(object: I): StakeToMaxCUList {
    const message = createBaseStakeToMaxCUList();
    message.List = object.List?.map((e) => StakeToMaxCU.fromPartial(e)) || [];
    return message;
  },
};

function createBaseStakeToMaxCU(): StakeToMaxCU {
  return { StakeThreshold: undefined, MaxComputeUnits: Long.UZERO };
}

export const StakeToMaxCU = {
  encode(message: StakeToMaxCU, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.StakeThreshold !== undefined) {
      Coin.encode(message.StakeThreshold, writer.uint32(10).fork()).ldelim();
    }
    if (!message.MaxComputeUnits.isZero()) {
      writer.uint32(16).uint64(message.MaxComputeUnits);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): StakeToMaxCU {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseStakeToMaxCU();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.StakeThreshold = Coin.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.MaxComputeUnits = reader.uint64() as Long;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): StakeToMaxCU {
    return {
      StakeThreshold: isSet(object.StakeThreshold) ? Coin.fromJSON(object.StakeThreshold) : undefined,
      MaxComputeUnits: isSet(object.MaxComputeUnits) ? Long.fromValue(object.MaxComputeUnits) : Long.UZERO,
    };
  },

  toJSON(message: StakeToMaxCU): unknown {
    const obj: any = {};
    message.StakeThreshold !== undefined &&
      (obj.StakeThreshold = message.StakeThreshold ? Coin.toJSON(message.StakeThreshold) : undefined);
    message.MaxComputeUnits !== undefined && (obj.MaxComputeUnits = (message.MaxComputeUnits || Long.UZERO).toString());
    return obj;
  },

  create<I extends Exact<DeepPartial<StakeToMaxCU>, I>>(base?: I): StakeToMaxCU {
    return StakeToMaxCU.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<StakeToMaxCU>, I>>(object: I): StakeToMaxCU {
    const message = createBaseStakeToMaxCU();
    message.StakeThreshold = (object.StakeThreshold !== undefined && object.StakeThreshold !== null)
      ? Coin.fromPartial(object.StakeThreshold)
      : undefined;
    message.MaxComputeUnits = (object.MaxComputeUnits !== undefined && object.MaxComputeUnits !== null)
      ? Long.fromValue(object.MaxComputeUnits)
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

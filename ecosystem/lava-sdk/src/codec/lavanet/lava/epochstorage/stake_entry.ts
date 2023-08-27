/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Coin } from "../../../cosmos/base/v1beta1/coin";
import { Endpoint } from "./endpoint";

export const protobufPackage = "lavanet.lava.epochstorage";

export interface StakeEntry {
  stake?: Coin;
  address: string;
  stakeAppliedBlock: Long;
  endpoints: Endpoint[];
  geolocation: Long;
  chain: string;
  moniker: string;
}

function createBaseStakeEntry(): StakeEntry {
  return {
    stake: undefined,
    address: "",
    stakeAppliedBlock: Long.UZERO,
    endpoints: [],
    geolocation: Long.UZERO,
    chain: "",
    moniker: "",
  };
}

export const StakeEntry = {
  encode(message: StakeEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.stake !== undefined) {
      Coin.encode(message.stake, writer.uint32(10).fork()).ldelim();
    }
    if (message.address !== "") {
      writer.uint32(18).string(message.address);
    }
    if (!message.stakeAppliedBlock.isZero()) {
      writer.uint32(24).uint64(message.stakeAppliedBlock);
    }
    for (const v of message.endpoints) {
      Endpoint.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    if (!message.geolocation.isZero()) {
      writer.uint32(40).uint64(message.geolocation);
    }
    if (message.chain !== "") {
      writer.uint32(50).string(message.chain);
    }
    if (message.moniker !== "") {
      writer.uint32(66).string(message.moniker);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): StakeEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseStakeEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.stake = Coin.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.address = reader.string();
          continue;
        case 3:
          if (tag != 24) {
            break;
          }

          message.stakeAppliedBlock = reader.uint64() as Long;
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

          message.chain = reader.string();
          continue;
        case 8:
          if (tag != 66) {
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

  fromJSON(object: any): StakeEntry {
    return {
      stake: isSet(object.stake) ? Coin.fromJSON(object.stake) : undefined,
      address: isSet(object.address) ? String(object.address) : "",
      stakeAppliedBlock: isSet(object.stakeAppliedBlock) ? Long.fromValue(object.stakeAppliedBlock) : Long.UZERO,
      endpoints: Array.isArray(object?.endpoints) ? object.endpoints.map((e: any) => Endpoint.fromJSON(e)) : [],
      geolocation: isSet(object.geolocation) ? Long.fromValue(object.geolocation) : Long.UZERO,
      chain: isSet(object.chain) ? String(object.chain) : "",
      moniker: isSet(object.moniker) ? String(object.moniker) : "",
    };
  },

  toJSON(message: StakeEntry): unknown {
    const obj: any = {};
    message.stake !== undefined && (obj.stake = message.stake ? Coin.toJSON(message.stake) : undefined);
    message.address !== undefined && (obj.address = message.address);
    message.stakeAppliedBlock !== undefined &&
      (obj.stakeAppliedBlock = (message.stakeAppliedBlock || Long.UZERO).toString());
    if (message.endpoints) {
      obj.endpoints = message.endpoints.map((e) => e ? Endpoint.toJSON(e) : undefined);
    } else {
      obj.endpoints = [];
    }
    message.geolocation !== undefined && (obj.geolocation = (message.geolocation || Long.UZERO).toString());
    message.chain !== undefined && (obj.chain = message.chain);
    message.moniker !== undefined && (obj.moniker = message.moniker);
    return obj;
  },

  create<I extends Exact<DeepPartial<StakeEntry>, I>>(base?: I): StakeEntry {
    return StakeEntry.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<StakeEntry>, I>>(object: I): StakeEntry {
    const message = createBaseStakeEntry();
    message.stake = (object.stake !== undefined && object.stake !== null) ? Coin.fromPartial(object.stake) : undefined;
    message.address = object.address ?? "";
    message.stakeAppliedBlock = (object.stakeAppliedBlock !== undefined && object.stakeAppliedBlock !== null)
      ? Long.fromValue(object.stakeAppliedBlock)
      : Long.UZERO;
    message.endpoints = object.endpoints?.map((e) => Endpoint.fromPartial(e)) || [];
    message.geolocation = (object.geolocation !== undefined && object.geolocation !== null)
      ? Long.fromValue(object.geolocation)
      : Long.UZERO;
    message.chain = object.chain ?? "";
    message.moniker = object.moniker ?? "";
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

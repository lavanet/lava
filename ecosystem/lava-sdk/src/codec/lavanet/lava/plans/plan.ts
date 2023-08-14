/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Coin } from "../../../cosmos/base/v1beta1/coin";
import { Policy } from "./policy";

export const protobufPackage = "lavanet.lava.plans";

/**
 * The geolocation values are encoded as bits in a bitmask, with two special values:
 * GLS is set to 0 so it will be restrictive with the AND operator.
 * GL is set to -1 so it will be permissive with the AND operator.
 */
export enum Geolocation {
  /** GLS - Global-strict */
  GLS = 0,
  /** USC - US-Center */
  USC = 1,
  EU = 2,
  /** USE - US-East */
  USE = 4,
  /** USW - US-West */
  USW = 8,
  AF = 16,
  AS = 32,
  /** AU - (includes NZ) */
  AU = 64,
  /** GL - Global */
  GL = 65535,
  UNRECOGNIZED = -1,
}

export function geolocationFromJSON(object: any): Geolocation {
  switch (object) {
    case 0:
    case "GLS":
      return Geolocation.GLS;
    case 1:
    case "USC":
      return Geolocation.USC;
    case 2:
    case "EU":
      return Geolocation.EU;
    case 4:
    case "USE":
      return Geolocation.USE;
    case 8:
    case "USW":
      return Geolocation.USW;
    case 16:
    case "AF":
      return Geolocation.AF;
    case 32:
    case "AS":
      return Geolocation.AS;
    case 64:
    case "AU":
      return Geolocation.AU;
    case 65535:
    case "GL":
      return Geolocation.GL;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Geolocation.UNRECOGNIZED;
  }
}

export function geolocationToJSON(object: Geolocation): string {
  switch (object) {
    case Geolocation.GLS:
      return "GLS";
    case Geolocation.USC:
      return "USC";
    case Geolocation.EU:
      return "EU";
    case Geolocation.USE:
      return "USE";
    case Geolocation.USW:
      return "USW";
    case Geolocation.AF:
      return "AF";
    case Geolocation.AS:
      return "AS";
    case Geolocation.AU:
      return "AU";
    case Geolocation.GL:
      return "GL";
    case Geolocation.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface Plan {
  index: string;
  /** the epoch that this plan was created */
  block: Long;
  /** plan price (in ulava) */
  price?: Coin;
  /** allow CU overuse flag */
  allowOveruse: boolean;
  /** price of CU overuse */
  overuseRate: Long;
  /** plan description (for humans) */
  description: string;
  /** plan type */
  type: string;
  /** discount for buying the plan for a year */
  annualDiscountPercentage: Long;
  planPolicy?: Policy;
}

function createBasePlan(): Plan {
  return {
    index: "",
    block: Long.UZERO,
    price: undefined,
    allowOveruse: false,
    overuseRate: Long.UZERO,
    description: "",
    type: "",
    annualDiscountPercentage: Long.UZERO,
    planPolicy: undefined,
  };
}

export const Plan = {
  encode(message: Plan, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    if (!message.block.isZero()) {
      writer.uint32(24).uint64(message.block);
    }
    if (message.price !== undefined) {
      Coin.encode(message.price, writer.uint32(34).fork()).ldelim();
    }
    if (message.allowOveruse === true) {
      writer.uint32(64).bool(message.allowOveruse);
    }
    if (!message.overuseRate.isZero()) {
      writer.uint32(72).uint64(message.overuseRate);
    }
    if (message.description !== "") {
      writer.uint32(90).string(message.description);
    }
    if (message.type !== "") {
      writer.uint32(98).string(message.type);
    }
    if (!message.annualDiscountPercentage.isZero()) {
      writer.uint32(104).uint64(message.annualDiscountPercentage);
    }
    if (message.planPolicy !== undefined) {
      Policy.encode(message.planPolicy, writer.uint32(114).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Plan {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlan();
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

          message.block = reader.uint64() as Long;
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.price = Coin.decode(reader, reader.uint32());
          continue;
        case 8:
          if (tag != 64) {
            break;
          }

          message.allowOveruse = reader.bool();
          continue;
        case 9:
          if (tag != 72) {
            break;
          }

          message.overuseRate = reader.uint64() as Long;
          continue;
        case 11:
          if (tag != 90) {
            break;
          }

          message.description = reader.string();
          continue;
        case 12:
          if (tag != 98) {
            break;
          }

          message.type = reader.string();
          continue;
        case 13:
          if (tag != 104) {
            break;
          }

          message.annualDiscountPercentage = reader.uint64() as Long;
          continue;
        case 14:
          if (tag != 114) {
            break;
          }

          message.planPolicy = Policy.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Plan {
    return {
      index: isSet(object.index) ? String(object.index) : "",
      block: isSet(object.block) ? Long.fromValue(object.block) : Long.UZERO,
      price: isSet(object.price) ? Coin.fromJSON(object.price) : undefined,
      allowOveruse: isSet(object.allowOveruse) ? Boolean(object.allowOveruse) : false,
      overuseRate: isSet(object.overuseRate) ? Long.fromValue(object.overuseRate) : Long.UZERO,
      description: isSet(object.description) ? String(object.description) : "",
      type: isSet(object.type) ? String(object.type) : "",
      annualDiscountPercentage: isSet(object.annualDiscountPercentage)
        ? Long.fromValue(object.annualDiscountPercentage)
        : Long.UZERO,
      planPolicy: isSet(object.planPolicy) ? Policy.fromJSON(object.planPolicy) : undefined,
    };
  },

  toJSON(message: Plan): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    message.block !== undefined && (obj.block = (message.block || Long.UZERO).toString());
    message.price !== undefined && (obj.price = message.price ? Coin.toJSON(message.price) : undefined);
    message.allowOveruse !== undefined && (obj.allowOveruse = message.allowOveruse);
    message.overuseRate !== undefined && (obj.overuseRate = (message.overuseRate || Long.UZERO).toString());
    message.description !== undefined && (obj.description = message.description);
    message.type !== undefined && (obj.type = message.type);
    message.annualDiscountPercentage !== undefined &&
      (obj.annualDiscountPercentage = (message.annualDiscountPercentage || Long.UZERO).toString());
    message.planPolicy !== undefined &&
      (obj.planPolicy = message.planPolicy ? Policy.toJSON(message.planPolicy) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<Plan>, I>>(base?: I): Plan {
    return Plan.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<Plan>, I>>(object: I): Plan {
    const message = createBasePlan();
    message.index = object.index ?? "";
    message.block = (object.block !== undefined && object.block !== null) ? Long.fromValue(object.block) : Long.UZERO;
    message.price = (object.price !== undefined && object.price !== null) ? Coin.fromPartial(object.price) : undefined;
    message.allowOveruse = object.allowOveruse ?? false;
    message.overuseRate = (object.overuseRate !== undefined && object.overuseRate !== null)
      ? Long.fromValue(object.overuseRate)
      : Long.UZERO;
    message.description = object.description ?? "";
    message.type = object.type ?? "";
    message.annualDiscountPercentage =
      (object.annualDiscountPercentage !== undefined && object.annualDiscountPercentage !== null)
        ? Long.fromValue(object.annualDiscountPercentage)
        : Long.UZERO;
    message.planPolicy = (object.planPolicy !== undefined && object.planPolicy !== null)
      ? Policy.fromPartial(object.planPolicy)
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

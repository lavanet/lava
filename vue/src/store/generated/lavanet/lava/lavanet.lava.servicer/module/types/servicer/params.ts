/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.servicer";

/** Params defines the parameters for the module. */
export interface Params {
  minStake: number;
  coinsPerCU: number;
  unstakeHoldBlocks: number;
  fraudStakeSlashingFactor: number;
  fraudSlashingAmount: number;
}

const baseParams: object = {
  minStake: 0,
  coinsPerCU: 0,
  unstakeHoldBlocks: 0,
  fraudStakeSlashingFactor: 0,
  fraudSlashingAmount: 0,
};

export const Params = {
  encode(message: Params, writer: Writer = Writer.create()): Writer {
    if (message.minStake !== 0) {
      writer.uint32(8).uint64(message.minStake);
    }
    if (message.coinsPerCU !== 0) {
      writer.uint32(16).uint64(message.coinsPerCU);
    }
    if (message.unstakeHoldBlocks !== 0) {
      writer.uint32(24).uint64(message.unstakeHoldBlocks);
    }
    if (message.fraudStakeSlashingFactor !== 0) {
      writer.uint32(32).uint64(message.fraudStakeSlashingFactor);
    }
    if (message.fraudSlashingAmount !== 0) {
      writer.uint32(40).uint64(message.fraudSlashingAmount);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): Params {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseParams } as Params;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.minStake = longToNumber(reader.uint64() as Long);
          break;
        case 2:
          message.coinsPerCU = longToNumber(reader.uint64() as Long);
          break;
        case 3:
          message.unstakeHoldBlocks = longToNumber(reader.uint64() as Long);
          break;
        case 4:
          message.fraudStakeSlashingFactor = longToNumber(
            reader.uint64() as Long
          );
          break;
        case 5:
          message.fraudSlashingAmount = longToNumber(reader.uint64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Params {
    const message = { ...baseParams } as Params;
    if (object.minStake !== undefined && object.minStake !== null) {
      message.minStake = Number(object.minStake);
    } else {
      message.minStake = 0;
    }
    if (object.coinsPerCU !== undefined && object.coinsPerCU !== null) {
      message.coinsPerCU = Number(object.coinsPerCU);
    } else {
      message.coinsPerCU = 0;
    }
    if (
      object.unstakeHoldBlocks !== undefined &&
      object.unstakeHoldBlocks !== null
    ) {
      message.unstakeHoldBlocks = Number(object.unstakeHoldBlocks);
    } else {
      message.unstakeHoldBlocks = 0;
    }
    if (
      object.fraudStakeSlashingFactor !== undefined &&
      object.fraudStakeSlashingFactor !== null
    ) {
      message.fraudStakeSlashingFactor = Number(
        object.fraudStakeSlashingFactor
      );
    } else {
      message.fraudStakeSlashingFactor = 0;
    }
    if (
      object.fraudSlashingAmount !== undefined &&
      object.fraudSlashingAmount !== null
    ) {
      message.fraudSlashingAmount = Number(object.fraudSlashingAmount);
    } else {
      message.fraudSlashingAmount = 0;
    }
    return message;
  },

  toJSON(message: Params): unknown {
    const obj: any = {};
    message.minStake !== undefined && (obj.minStake = message.minStake);
    message.coinsPerCU !== undefined && (obj.coinsPerCU = message.coinsPerCU);
    message.unstakeHoldBlocks !== undefined &&
      (obj.unstakeHoldBlocks = message.unstakeHoldBlocks);
    message.fraudStakeSlashingFactor !== undefined &&
      (obj.fraudStakeSlashingFactor = message.fraudStakeSlashingFactor);
    message.fraudSlashingAmount !== undefined &&
      (obj.fraudSlashingAmount = message.fraudSlashingAmount);
    return obj;
  },

  fromPartial(object: DeepPartial<Params>): Params {
    const message = { ...baseParams } as Params;
    if (object.minStake !== undefined && object.minStake !== null) {
      message.minStake = object.minStake;
    } else {
      message.minStake = 0;
    }
    if (object.coinsPerCU !== undefined && object.coinsPerCU !== null) {
      message.coinsPerCU = object.coinsPerCU;
    } else {
      message.coinsPerCU = 0;
    }
    if (
      object.unstakeHoldBlocks !== undefined &&
      object.unstakeHoldBlocks !== null
    ) {
      message.unstakeHoldBlocks = object.unstakeHoldBlocks;
    } else {
      message.unstakeHoldBlocks = 0;
    }
    if (
      object.fraudStakeSlashingFactor !== undefined &&
      object.fraudStakeSlashingFactor !== null
    ) {
      message.fraudStakeSlashingFactor = object.fraudStakeSlashingFactor;
    } else {
      message.fraudStakeSlashingFactor = 0;
    }
    if (
      object.fraudSlashingAmount !== undefined &&
      object.fraudSlashingAmount !== null
    ) {
      message.fraudSlashingAmount = object.fraudSlashingAmount;
    } else {
      message.fraudSlashingAmount = 0;
    }
    return message;
  },
};

declare var self: any | undefined;
declare var window: any | undefined;
var globalThis: any = (() => {
  if (typeof globalThis !== "undefined") return globalThis;
  if (typeof self !== "undefined") return self;
  if (typeof window !== "undefined") return window;
  if (typeof global !== "undefined") return global;
  throw "Unable to locate global object";
})();

type Builtin = Date | Function | Uint8Array | string | number | undefined;
export type DeepPartial<T> = T extends Builtin
  ? T
  : T extends Array<infer U>
  ? Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U>
  ? ReadonlyArray<DeepPartial<U>>
  : T extends {}
  ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function longToNumber(long: Long): number {
  if (long.gt(Number.MAX_SAFE_INTEGER)) {
    throw new globalThis.Error("Value is larger than Number.MAX_SAFE_INTEGER");
  }
  return long.toNumber();
}

if (util.Long !== Long) {
  util.Long = Long as any;
  configure();
}

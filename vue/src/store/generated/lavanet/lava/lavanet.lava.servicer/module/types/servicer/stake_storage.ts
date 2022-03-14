/* eslint-disable */
import { StakeMap } from "../servicer/stake_map";
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.servicer";

export interface StakeStorage {
  staked: StakeMap | undefined;
  unstaking: StakeMap | undefined;
}

const baseStakeStorage: object = {};

export const StakeStorage = {
  encode(message: StakeStorage, writer: Writer = Writer.create()): Writer {
    if (message.staked !== undefined) {
      StakeMap.encode(message.staked, writer.uint32(10).fork()).ldelim();
    }
    if (message.unstaking !== undefined) {
      StakeMap.encode(message.unstaking, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): StakeStorage {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseStakeStorage } as StakeStorage;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.staked = StakeMap.decode(reader, reader.uint32());
          break;
        case 2:
          message.unstaking = StakeMap.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): StakeStorage {
    const message = { ...baseStakeStorage } as StakeStorage;
    if (object.staked !== undefined && object.staked !== null) {
      message.staked = StakeMap.fromJSON(object.staked);
    } else {
      message.staked = undefined;
    }
    if (object.unstaking !== undefined && object.unstaking !== null) {
      message.unstaking = StakeMap.fromJSON(object.unstaking);
    } else {
      message.unstaking = undefined;
    }
    return message;
  },

  toJSON(message: StakeStorage): unknown {
    const obj: any = {};
    message.staked !== undefined &&
      (obj.staked = message.staked
        ? StakeMap.toJSON(message.staked)
        : undefined);
    message.unstaking !== undefined &&
      (obj.unstaking = message.unstaking
        ? StakeMap.toJSON(message.unstaking)
        : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<StakeStorage>): StakeStorage {
    const message = { ...baseStakeStorage } as StakeStorage;
    if (object.staked !== undefined && object.staked !== null) {
      message.staked = StakeMap.fromPartial(object.staked);
    } else {
      message.staked = undefined;
    }
    if (object.unstaking !== undefined && object.unstaking !== null) {
      message.unstaking = StakeMap.fromPartial(object.unstaking);
    } else {
      message.unstaking = undefined;
    }
    return message;
  },
};

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

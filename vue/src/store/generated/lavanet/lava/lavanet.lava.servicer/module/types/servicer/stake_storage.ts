/* eslint-disable */
import { StakeMap } from "../servicer/stake_map";
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.servicer";

export interface StakeStorage {
  staked: StakeMap[];
  unstaking: StakeMap[];
}

const baseStakeStorage: object = {};

export const StakeStorage = {
  encode(message: StakeStorage, writer: Writer = Writer.create()): Writer {
    for (const v of message.staked) {
      StakeMap.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.unstaking) {
      StakeMap.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): StakeStorage {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseStakeStorage } as StakeStorage;
    message.staked = [];
    message.unstaking = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.staked.push(StakeMap.decode(reader, reader.uint32()));
          break;
        case 2:
          message.unstaking.push(StakeMap.decode(reader, reader.uint32()));
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
    message.staked = [];
    message.unstaking = [];
    if (object.staked !== undefined && object.staked !== null) {
      for (const e of object.staked) {
        message.staked.push(StakeMap.fromJSON(e));
      }
    }
    if (object.unstaking !== undefined && object.unstaking !== null) {
      for (const e of object.unstaking) {
        message.unstaking.push(StakeMap.fromJSON(e));
      }
    }
    return message;
  },

  toJSON(message: StakeStorage): unknown {
    const obj: any = {};
    if (message.staked) {
      obj.staked = message.staked.map((e) =>
        e ? StakeMap.toJSON(e) : undefined
      );
    } else {
      obj.staked = [];
    }
    if (message.unstaking) {
      obj.unstaking = message.unstaking.map((e) =>
        e ? StakeMap.toJSON(e) : undefined
      );
    } else {
      obj.unstaking = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<StakeStorage>): StakeStorage {
    const message = { ...baseStakeStorage } as StakeStorage;
    message.staked = [];
    message.unstaking = [];
    if (object.staked !== undefined && object.staked !== null) {
      for (const e of object.staked) {
        message.staked.push(StakeMap.fromPartial(e));
      }
    }
    if (object.unstaking !== undefined && object.unstaking !== null) {
      for (const e of object.unstaking) {
        message.unstaking.push(StakeMap.fromPartial(e));
      }
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

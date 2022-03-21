/* eslint-disable */
import { UserStake } from "../user/user_stake";
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.user";

export interface StakeStorage {
  stakedUsers: UserStake | undefined;
}

const baseStakeStorage: object = {};

export const StakeStorage = {
  encode(message: StakeStorage, writer: Writer = Writer.create()): Writer {
    if (message.stakedUsers !== undefined) {
      UserStake.encode(message.stakedUsers, writer.uint32(10).fork()).ldelim();
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
          message.stakedUsers = UserStake.decode(reader, reader.uint32());
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
    if (object.stakedUsers !== undefined && object.stakedUsers !== null) {
      message.stakedUsers = UserStake.fromJSON(object.stakedUsers);
    } else {
      message.stakedUsers = undefined;
    }
    return message;
  },

  toJSON(message: StakeStorage): unknown {
    const obj: any = {};
    message.stakedUsers !== undefined &&
      (obj.stakedUsers = message.stakedUsers
        ? UserStake.toJSON(message.stakedUsers)
        : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<StakeStorage>): StakeStorage {
    const message = { ...baseStakeStorage } as StakeStorage;
    if (object.stakedUsers !== undefined && object.stakedUsers !== null) {
      message.stakedUsers = UserStake.fromPartial(object.stakedUsers);
    } else {
      message.stakedUsers = undefined;
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

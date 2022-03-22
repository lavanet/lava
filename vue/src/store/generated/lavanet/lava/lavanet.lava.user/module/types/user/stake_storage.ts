/* eslint-disable */
import { UserStake } from "../user/user_stake";
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.user";

export interface StakeStorage {
  stakedUsers: UserStake[];
}

const baseStakeStorage: object = {};

export const StakeStorage = {
  encode(message: StakeStorage, writer: Writer = Writer.create()): Writer {
    for (const v of message.stakedUsers) {
      UserStake.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): StakeStorage {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseStakeStorage } as StakeStorage;
    message.stakedUsers = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.stakedUsers.push(UserStake.decode(reader, reader.uint32()));
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
    message.stakedUsers = [];
    if (object.stakedUsers !== undefined && object.stakedUsers !== null) {
      for (const e of object.stakedUsers) {
        message.stakedUsers.push(UserStake.fromJSON(e));
      }
    }
    return message;
  },

  toJSON(message: StakeStorage): unknown {
    const obj: any = {};
    if (message.stakedUsers) {
      obj.stakedUsers = message.stakedUsers.map((e) =>
        e ? UserStake.toJSON(e) : undefined
      );
    } else {
      obj.stakedUsers = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<StakeStorage>): StakeStorage {
    const message = { ...baseStakeStorage } as StakeStorage;
    message.stakedUsers = [];
    if (object.stakedUsers !== undefined && object.stakedUsers !== null) {
      for (const e of object.stakedUsers) {
        message.stakedUsers.push(UserStake.fromPartial(e));
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

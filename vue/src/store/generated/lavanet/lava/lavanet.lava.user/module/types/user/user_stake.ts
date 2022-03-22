/* eslint-disable */
import { Coin } from "../cosmos/base/v1beta1/coin";
import { BlockNum } from "../user/block_num";
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.user";

export interface UserStake {
  index: string;
  stake: Coin | undefined;
  deadline: BlockNum | undefined;
}

const baseUserStake: object = { index: "" };

export const UserStake = {
  encode(message: UserStake, writer: Writer = Writer.create()): Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    if (message.stake !== undefined) {
      Coin.encode(message.stake, writer.uint32(18).fork()).ldelim();
    }
    if (message.deadline !== undefined) {
      BlockNum.encode(message.deadline, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): UserStake {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseUserStake } as UserStake;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.index = reader.string();
          break;
        case 2:
          message.stake = Coin.decode(reader, reader.uint32());
          break;
        case 3:
          message.deadline = BlockNum.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): UserStake {
    const message = { ...baseUserStake } as UserStake;
    if (object.index !== undefined && object.index !== null) {
      message.index = String(object.index);
    } else {
      message.index = "";
    }
    if (object.stake !== undefined && object.stake !== null) {
      message.stake = Coin.fromJSON(object.stake);
    } else {
      message.stake = undefined;
    }
    if (object.deadline !== undefined && object.deadline !== null) {
      message.deadline = BlockNum.fromJSON(object.deadline);
    } else {
      message.deadline = undefined;
    }
    return message;
  },

  toJSON(message: UserStake): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    message.stake !== undefined &&
      (obj.stake = message.stake ? Coin.toJSON(message.stake) : undefined);
    message.deadline !== undefined &&
      (obj.deadline = message.deadline
        ? BlockNum.toJSON(message.deadline)
        : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<UserStake>): UserStake {
    const message = { ...baseUserStake } as UserStake;
    if (object.index !== undefined && object.index !== null) {
      message.index = object.index;
    } else {
      message.index = "";
    }
    if (object.stake !== undefined && object.stake !== null) {
      message.stake = Coin.fromPartial(object.stake);
    } else {
      message.stake = undefined;
    }
    if (object.deadline !== undefined && object.deadline !== null) {
      message.deadline = BlockNum.fromPartial(object.deadline);
    } else {
      message.deadline = undefined;
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

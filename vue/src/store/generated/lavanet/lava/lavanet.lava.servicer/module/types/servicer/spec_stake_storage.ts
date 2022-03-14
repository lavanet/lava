/* eslint-disable */
import { StakeStorage } from "../servicer/stake_storage";
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.servicer";

export interface SpecStakeStorage {
  index: string;
  stakeStorage: StakeStorage | undefined;
}

const baseSpecStakeStorage: object = { index: "" };

export const SpecStakeStorage = {
  encode(message: SpecStakeStorage, writer: Writer = Writer.create()): Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    if (message.stakeStorage !== undefined) {
      StakeStorage.encode(
        message.stakeStorage,
        writer.uint32(18).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): SpecStakeStorage {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseSpecStakeStorage } as SpecStakeStorage;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.index = reader.string();
          break;
        case 2:
          message.stakeStorage = StakeStorage.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): SpecStakeStorage {
    const message = { ...baseSpecStakeStorage } as SpecStakeStorage;
    if (object.index !== undefined && object.index !== null) {
      message.index = String(object.index);
    } else {
      message.index = "";
    }
    if (object.stakeStorage !== undefined && object.stakeStorage !== null) {
      message.stakeStorage = StakeStorage.fromJSON(object.stakeStorage);
    } else {
      message.stakeStorage = undefined;
    }
    return message;
  },

  toJSON(message: SpecStakeStorage): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    message.stakeStorage !== undefined &&
      (obj.stakeStorage = message.stakeStorage
        ? StakeStorage.toJSON(message.stakeStorage)
        : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<SpecStakeStorage>): SpecStakeStorage {
    const message = { ...baseSpecStakeStorage } as SpecStakeStorage;
    if (object.index !== undefined && object.index !== null) {
      message.index = object.index;
    } else {
      message.index = "";
    }
    if (object.stakeStorage !== undefined && object.stakeStorage !== null) {
      message.stakeStorage = StakeStorage.fromPartial(object.stakeStorage);
    } else {
      message.stakeStorage = undefined;
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

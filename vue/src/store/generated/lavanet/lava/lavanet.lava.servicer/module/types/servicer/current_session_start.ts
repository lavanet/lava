/* eslint-disable */
import { BlockNum } from "../servicer/block_num";
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.servicer";

export interface CurrentSessionStart {
  block: BlockNum | undefined;
}

const baseCurrentSessionStart: object = {};

export const CurrentSessionStart = {
  encode(
    message: CurrentSessionStart,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.block !== undefined) {
      BlockNum.encode(message.block, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): CurrentSessionStart {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseCurrentSessionStart } as CurrentSessionStart;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.block = BlockNum.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): CurrentSessionStart {
    const message = { ...baseCurrentSessionStart } as CurrentSessionStart;
    if (object.block !== undefined && object.block !== null) {
      message.block = BlockNum.fromJSON(object.block);
    } else {
      message.block = undefined;
    }
    return message;
  },

  toJSON(message: CurrentSessionStart): unknown {
    const obj: any = {};
    message.block !== undefined &&
      (obj.block = message.block ? BlockNum.toJSON(message.block) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<CurrentSessionStart>): CurrentSessionStart {
    const message = { ...baseCurrentSessionStart } as CurrentSessionStart;
    if (object.block !== undefined && object.block !== null) {
      message.block = BlockNum.fromPartial(object.block);
    } else {
      message.block = undefined;
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

/* eslint-disable */
import { BlockNum } from "../servicer/block_num";
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.servicer";

export interface BlockDeadlineForCallback {
  deadline: BlockNum | undefined;
}

const baseBlockDeadlineForCallback: object = {};

export const BlockDeadlineForCallback = {
  encode(
    message: BlockDeadlineForCallback,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.deadline !== undefined) {
      BlockNum.encode(message.deadline, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): BlockDeadlineForCallback {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseBlockDeadlineForCallback,
    } as BlockDeadlineForCallback;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.deadline = BlockNum.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): BlockDeadlineForCallback {
    const message = {
      ...baseBlockDeadlineForCallback,
    } as BlockDeadlineForCallback;
    if (object.deadline !== undefined && object.deadline !== null) {
      message.deadline = BlockNum.fromJSON(object.deadline);
    } else {
      message.deadline = undefined;
    }
    return message;
  },

  toJSON(message: BlockDeadlineForCallback): unknown {
    const obj: any = {};
    message.deadline !== undefined &&
      (obj.deadline = message.deadline
        ? BlockNum.toJSON(message.deadline)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<BlockDeadlineForCallback>
  ): BlockDeadlineForCallback {
    const message = {
      ...baseBlockDeadlineForCallback,
    } as BlockDeadlineForCallback;
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

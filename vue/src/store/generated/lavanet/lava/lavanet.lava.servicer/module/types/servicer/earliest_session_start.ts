/* eslint-disable */
import { BlockNum } from "../servicer/block_num";
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.servicer";

export interface EarliestSessionStart {
  block: BlockNum | undefined;
}

const baseEarliestSessionStart: object = {};

export const EarliestSessionStart = {
  encode(
    message: EarliestSessionStart,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.block !== undefined) {
      BlockNum.encode(message.block, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): EarliestSessionStart {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseEarliestSessionStart } as EarliestSessionStart;
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

  fromJSON(object: any): EarliestSessionStart {
    const message = { ...baseEarliestSessionStart } as EarliestSessionStart;
    if (object.block !== undefined && object.block !== null) {
      message.block = BlockNum.fromJSON(object.block);
    } else {
      message.block = undefined;
    }
    return message;
  },

  toJSON(message: EarliestSessionStart): unknown {
    const obj: any = {};
    message.block !== undefined &&
      (obj.block = message.block ? BlockNum.toJSON(message.block) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<EarliestSessionStart>): EarliestSessionStart {
    const message = { ...baseEarliestSessionStart } as EarliestSessionStart;
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

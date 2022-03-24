/* eslint-disable */
import { Spec } from "../spec/spec";
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.spec";

export interface SpecModifyProposal {
  title: string;
  description: string;
  specs: Spec[];
}

const baseSpecModifyProposal: object = { title: "", description: "" };

export const SpecModifyProposal = {
  encode(
    message: SpecModifyProposal,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.title !== "") {
      writer.uint32(10).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(18).string(message.description);
    }
    for (const v of message.specs) {
      Spec.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): SpecModifyProposal {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseSpecModifyProposal } as SpecModifyProposal;
    message.specs = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.title = reader.string();
          break;
        case 2:
          message.description = reader.string();
          break;
        case 3:
          message.specs.push(Spec.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): SpecModifyProposal {
    const message = { ...baseSpecModifyProposal } as SpecModifyProposal;
    message.specs = [];
    if (object.title !== undefined && object.title !== null) {
      message.title = String(object.title);
    } else {
      message.title = "";
    }
    if (object.description !== undefined && object.description !== null) {
      message.description = String(object.description);
    } else {
      message.description = "";
    }
    if (object.specs !== undefined && object.specs !== null) {
      for (const e of object.specs) {
        message.specs.push(Spec.fromJSON(e));
      }
    }
    return message;
  },

  toJSON(message: SpecModifyProposal): unknown {
    const obj: any = {};
    message.title !== undefined && (obj.title = message.title);
    message.description !== undefined &&
      (obj.description = message.description);
    if (message.specs) {
      obj.specs = message.specs.map((e) => (e ? Spec.toJSON(e) : undefined));
    } else {
      obj.specs = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<SpecModifyProposal>): SpecModifyProposal {
    const message = { ...baseSpecModifyProposal } as SpecModifyProposal;
    message.specs = [];
    if (object.title !== undefined && object.title !== null) {
      message.title = object.title;
    } else {
      message.title = "";
    }
    if (object.description !== undefined && object.description !== null) {
      message.description = object.description;
    } else {
      message.description = "";
    }
    if (object.specs !== undefined && object.specs !== null) {
      for (const e of object.specs) {
        message.specs.push(Spec.fromPartial(e));
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

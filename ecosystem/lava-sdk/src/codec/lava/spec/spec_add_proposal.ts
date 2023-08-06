/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Spec } from "./spec";

export const protobufPackage = "lavanet.lava.spec";

export interface SpecAddProposal {
  title: string;
  description: string;
  specs: Spec[];
}

function createBaseSpecAddProposal(): SpecAddProposal {
  return { title: "", description: "", specs: [] };
}

export const SpecAddProposal = {
  encode(message: SpecAddProposal, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): SpecAddProposal {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSpecAddProposal();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.title = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.description = reader.string();
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.specs.push(Spec.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SpecAddProposal {
    return {
      title: isSet(object.title) ? String(object.title) : "",
      description: isSet(object.description) ? String(object.description) : "",
      specs: Array.isArray(object?.specs) ? object.specs.map((e: any) => Spec.fromJSON(e)) : [],
    };
  },

  toJSON(message: SpecAddProposal): unknown {
    const obj: any = {};
    message.title !== undefined && (obj.title = message.title);
    message.description !== undefined && (obj.description = message.description);
    if (message.specs) {
      obj.specs = message.specs.map((e) => e ? Spec.toJSON(e) : undefined);
    } else {
      obj.specs = [];
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<SpecAddProposal>, I>>(base?: I): SpecAddProposal {
    return SpecAddProposal.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<SpecAddProposal>, I>>(object: I): SpecAddProposal {
    const message = createBaseSpecAddProposal();
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.specs = object.specs?.map((e) => Spec.fromPartial(e)) || [];
    return message;
  },
};

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Long ? string | number | Long : T extends Array<infer U> ? Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

type KeysOfUnion<T> = T extends T ? keyof T : never;
export type Exact<P, I extends P> = P extends Builtin ? P
  : P & { [K in keyof P]: Exact<P[K], I[K]> } & { [K in Exclude<keyof I, KeysOfUnion<P>>]: never };

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}

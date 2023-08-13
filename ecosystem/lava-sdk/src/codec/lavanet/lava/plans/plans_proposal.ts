/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Plan } from "./plan";

export const protobufPackage = "lavanet.lava.plans";

export interface PlansAddProposal {
  title: string;
  description: string;
  plans: Plan[];
}

export interface PlansDelProposal {
  title: string;
  description: string;
  plans: string[];
}

function createBasePlansAddProposal(): PlansAddProposal {
  return { title: "", description: "", plans: [] };
}

export const PlansAddProposal = {
  encode(message: PlansAddProposal, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.title !== "") {
      writer.uint32(10).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(18).string(message.description);
    }
    for (const v of message.plans) {
      Plan.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlansAddProposal {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlansAddProposal();
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

          message.plans.push(Plan.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlansAddProposal {
    return {
      title: isSet(object.title) ? String(object.title) : "",
      description: isSet(object.description) ? String(object.description) : "",
      plans: Array.isArray(object?.plans) ? object.plans.map((e: any) => Plan.fromJSON(e)) : [],
    };
  },

  toJSON(message: PlansAddProposal): unknown {
    const obj: any = {};
    message.title !== undefined && (obj.title = message.title);
    message.description !== undefined && (obj.description = message.description);
    if (message.plans) {
      obj.plans = message.plans.map((e) => e ? Plan.toJSON(e) : undefined);
    } else {
      obj.plans = [];
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<PlansAddProposal>, I>>(base?: I): PlansAddProposal {
    return PlansAddProposal.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<PlansAddProposal>, I>>(object: I): PlansAddProposal {
    const message = createBasePlansAddProposal();
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.plans = object.plans?.map((e) => Plan.fromPartial(e)) || [];
    return message;
  },
};

function createBasePlansDelProposal(): PlansDelProposal {
  return { title: "", description: "", plans: [] };
}

export const PlansDelProposal = {
  encode(message: PlansDelProposal, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.title !== "") {
      writer.uint32(10).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(18).string(message.description);
    }
    for (const v of message.plans) {
      writer.uint32(26).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlansDelProposal {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlansDelProposal();
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

          message.plans.push(reader.string());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlansDelProposal {
    return {
      title: isSet(object.title) ? String(object.title) : "",
      description: isSet(object.description) ? String(object.description) : "",
      plans: Array.isArray(object?.plans) ? object.plans.map((e: any) => String(e)) : [],
    };
  },

  toJSON(message: PlansDelProposal): unknown {
    const obj: any = {};
    message.title !== undefined && (obj.title = message.title);
    message.description !== undefined && (obj.description = message.description);
    if (message.plans) {
      obj.plans = message.plans.map((e) => e);
    } else {
      obj.plans = [];
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<PlansDelProposal>, I>>(base?: I): PlansDelProposal {
    return PlansDelProposal.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<PlansDelProposal>, I>>(object: I): PlansDelProposal {
    const message = createBasePlansDelProposal();
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.plans = object.plans?.map((e) => e) || [];
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

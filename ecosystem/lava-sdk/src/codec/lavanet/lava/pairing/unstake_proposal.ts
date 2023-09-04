/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.pairing";

export interface UnstakeProposal {
  title: string;
  description: string;
  providersInfo: ProviderUnstakeInfo[];
}

export interface ProviderUnstakeInfo {
  provider: string;
  chainId: string;
}

function createBaseUnstakeProposal(): UnstakeProposal {
  return { title: "", description: "", providersInfo: [] };
}

export const UnstakeProposal = {
  encode(message: UnstakeProposal, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.title !== "") {
      writer.uint32(10).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(18).string(message.description);
    }
    for (const v of message.providersInfo) {
      ProviderUnstakeInfo.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UnstakeProposal {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUnstakeProposal();
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

          message.providersInfo.push(ProviderUnstakeInfo.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): UnstakeProposal {
    return {
      title: isSet(object.title) ? String(object.title) : "",
      description: isSet(object.description) ? String(object.description) : "",
      providersInfo: Array.isArray(object?.providersInfo)
        ? object.providersInfo.map((e: any) => ProviderUnstakeInfo.fromJSON(e))
        : [],
    };
  },

  toJSON(message: UnstakeProposal): unknown {
    const obj: any = {};
    message.title !== undefined && (obj.title = message.title);
    message.description !== undefined && (obj.description = message.description);
    if (message.providersInfo) {
      obj.providersInfo = message.providersInfo.map((e) => e ? ProviderUnstakeInfo.toJSON(e) : undefined);
    } else {
      obj.providersInfo = [];
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<UnstakeProposal>, I>>(base?: I): UnstakeProposal {
    return UnstakeProposal.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<UnstakeProposal>, I>>(object: I): UnstakeProposal {
    const message = createBaseUnstakeProposal();
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.providersInfo = object.providersInfo?.map((e) => ProviderUnstakeInfo.fromPartial(e)) || [];
    return message;
  },
};

function createBaseProviderUnstakeInfo(): ProviderUnstakeInfo {
  return { provider: "", chainId: "" };
}

export const ProviderUnstakeInfo = {
  encode(message: ProviderUnstakeInfo, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.provider !== "") {
      writer.uint32(10).string(message.provider);
    }
    if (message.chainId !== "") {
      writer.uint32(18).string(message.chainId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ProviderUnstakeInfo {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseProviderUnstakeInfo();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.provider = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.chainId = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ProviderUnstakeInfo {
    return {
      provider: isSet(object.provider) ? String(object.provider) : "",
      chainId: isSet(object.chainId) ? String(object.chainId) : "",
    };
  },

  toJSON(message: ProviderUnstakeInfo): unknown {
    const obj: any = {};
    message.provider !== undefined && (obj.provider = message.provider);
    message.chainId !== undefined && (obj.chainId = message.chainId);
    return obj;
  },

  create<I extends Exact<DeepPartial<ProviderUnstakeInfo>, I>>(base?: I): ProviderUnstakeInfo {
    return ProviderUnstakeInfo.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<ProviderUnstakeInfo>, I>>(object: I): ProviderUnstakeInfo {
    const message = createBaseProviderUnstakeInfo();
    message.provider = object.provider ?? "";
    message.chainId = object.chainId ?? "";
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

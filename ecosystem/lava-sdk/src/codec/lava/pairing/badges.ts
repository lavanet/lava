/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { StakeEntry } from "../epochstorage/stake_entry";
import { Badge } from "./relay";

export const protobufPackage = "lavanet.lava.pairing";

export interface GenerateBadgeRequest {
  badgeAddress: string;
  projectId: string;
  specId: string;
}

export interface GenerateBadgeResponse {
  badge?: Badge;
  pairingList: StakeEntry[];
  badgeSignerAddress: string;
}

function createBaseGenerateBadgeRequest(): GenerateBadgeRequest {
  return { badgeAddress: "", projectId: "", specId: "" };
}

export const GenerateBadgeRequest = {
  encode(message: GenerateBadgeRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.badgeAddress !== "") {
      writer.uint32(10).string(message.badgeAddress);
    }
    if (message.projectId !== "") {
      writer.uint32(18).string(message.projectId);
    }
    if (message.specId !== "") {
      writer.uint32(26).string(message.specId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GenerateBadgeRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGenerateBadgeRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.badgeAddress = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.projectId = reader.string();
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.specId = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GenerateBadgeRequest {
    return {
      badgeAddress: isSet(object.badgeAddress) ? String(object.badgeAddress) : "",
      projectId: isSet(object.projectId) ? String(object.projectId) : "",
      specId: isSet(object.specId) ? String(object.specId) : "",
    };
  },

  toJSON(message: GenerateBadgeRequest): unknown {
    const obj: any = {};
    message.badgeAddress !== undefined && (obj.badgeAddress = message.badgeAddress);
    message.projectId !== undefined && (obj.projectId = message.projectId);
    message.specId !== undefined && (obj.specId = message.specId);
    return obj;
  },

  create<I extends Exact<DeepPartial<GenerateBadgeRequest>, I>>(base?: I): GenerateBadgeRequest {
    return GenerateBadgeRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<GenerateBadgeRequest>, I>>(object: I): GenerateBadgeRequest {
    const message = createBaseGenerateBadgeRequest();
    message.badgeAddress = object.badgeAddress ?? "";
    message.projectId = object.projectId ?? "";
    message.specId = object.specId ?? "";
    return message;
  },
};

function createBaseGenerateBadgeResponse(): GenerateBadgeResponse {
  return { badge: undefined, pairingList: [], badgeSignerAddress: "" };
}

export const GenerateBadgeResponse = {
  encode(message: GenerateBadgeResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.badge !== undefined) {
      Badge.encode(message.badge, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.pairingList) {
      StakeEntry.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    if (message.badgeSignerAddress !== "") {
      writer.uint32(26).string(message.badgeSignerAddress);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GenerateBadgeResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGenerateBadgeResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.badge = Badge.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.pairingList.push(StakeEntry.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.badgeSignerAddress = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GenerateBadgeResponse {
    return {
      badge: isSet(object.badge) ? Badge.fromJSON(object.badge) : undefined,
      pairingList: Array.isArray(object?.pairingList) ? object.pairingList.map((e: any) => StakeEntry.fromJSON(e)) : [],
      badgeSignerAddress: isSet(object.badgeSignerAddress) ? String(object.badgeSignerAddress) : "",
    };
  },

  toJSON(message: GenerateBadgeResponse): unknown {
    const obj: any = {};
    message.badge !== undefined && (obj.badge = message.badge ? Badge.toJSON(message.badge) : undefined);
    if (message.pairingList) {
      obj.pairingList = message.pairingList.map((e) => e ? StakeEntry.toJSON(e) : undefined);
    } else {
      obj.pairingList = [];
    }
    message.badgeSignerAddress !== undefined && (obj.badgeSignerAddress = message.badgeSignerAddress);
    return obj;
  },

  create<I extends Exact<DeepPartial<GenerateBadgeResponse>, I>>(base?: I): GenerateBadgeResponse {
    return GenerateBadgeResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<GenerateBadgeResponse>, I>>(object: I): GenerateBadgeResponse {
    const message = createBaseGenerateBadgeResponse();
    message.badge = (object.badge !== undefined && object.badge !== null) ? Badge.fromPartial(object.badge) : undefined;
    message.pairingList = object.pairingList?.map((e) => StakeEntry.fromPartial(e)) || [];
    message.badgeSignerAddress = object.badgeSignerAddress ?? "";
    return message;
  },
};

export interface BadgeGenerator {
  GenerateBadge(request: GenerateBadgeRequest): Promise<GenerateBadgeResponse>;
}

export class BadgeGeneratorClientImpl implements BadgeGenerator {
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || "lavanet.lava.pairing.BadgeGenerator";
    this.rpc = rpc;
    this.GenerateBadge = this.GenerateBadge.bind(this);
  }
  GenerateBadge(request: GenerateBadgeRequest): Promise<GenerateBadgeResponse> {
    const data = GenerateBadgeRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "GenerateBadge", data);
    return promise.then((data) => GenerateBadgeResponse.decode(_m0.Reader.create(data)));
  }
}

interface Rpc {
  request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}

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

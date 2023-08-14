/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Policy } from "../plans/policy";
import { ProjectKey } from "./project";

export const protobufPackage = "lavanet.lava.projects";

export interface MsgAddKeys {
  creator: string;
  project: string;
  projectKeys: ProjectKey[];
}

export interface MsgAddKeysResponse {
}

export interface MsgDelKeys {
  creator: string;
  project: string;
  projectKeys: ProjectKey[];
}

export interface MsgDelKeysResponse {
}

export interface MsgSetPolicy {
  creator: string;
  project: string;
  policy?: Policy;
}

export interface MsgSetPolicyResponse {
}

export interface MsgSetSubscriptionPolicy {
  creator: string;
  projects: string[];
  policy?: Policy;
}

export interface MsgSetSubscriptionPolicyResponse {
}

function createBaseMsgAddKeys(): MsgAddKeys {
  return { creator: "", project: "", projectKeys: [] };
}

export const MsgAddKeys = {
  encode(message: MsgAddKeys, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.project !== "") {
      writer.uint32(18).string(message.project);
    }
    for (const v of message.projectKeys) {
      ProjectKey.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgAddKeys {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAddKeys();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.creator = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.project = reader.string();
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.projectKeys.push(ProjectKey.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MsgAddKeys {
    return {
      creator: isSet(object.creator) ? String(object.creator) : "",
      project: isSet(object.project) ? String(object.project) : "",
      projectKeys: Array.isArray(object?.projectKeys) ? object.projectKeys.map((e: any) => ProjectKey.fromJSON(e)) : [],
    };
  },

  toJSON(message: MsgAddKeys): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.project !== undefined && (obj.project = message.project);
    if (message.projectKeys) {
      obj.projectKeys = message.projectKeys.map((e) => e ? ProjectKey.toJSON(e) : undefined);
    } else {
      obj.projectKeys = [];
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgAddKeys>, I>>(base?: I): MsgAddKeys {
    return MsgAddKeys.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgAddKeys>, I>>(object: I): MsgAddKeys {
    const message = createBaseMsgAddKeys();
    message.creator = object.creator ?? "";
    message.project = object.project ?? "";
    message.projectKeys = object.projectKeys?.map((e) => ProjectKey.fromPartial(e)) || [];
    return message;
  },
};

function createBaseMsgAddKeysResponse(): MsgAddKeysResponse {
  return {};
}

export const MsgAddKeysResponse = {
  encode(_: MsgAddKeysResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgAddKeysResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAddKeysResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(_: any): MsgAddKeysResponse {
    return {};
  },

  toJSON(_: MsgAddKeysResponse): unknown {
    const obj: any = {};
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgAddKeysResponse>, I>>(base?: I): MsgAddKeysResponse {
    return MsgAddKeysResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgAddKeysResponse>, I>>(_: I): MsgAddKeysResponse {
    const message = createBaseMsgAddKeysResponse();
    return message;
  },
};

function createBaseMsgDelKeys(): MsgDelKeys {
  return { creator: "", project: "", projectKeys: [] };
}

export const MsgDelKeys = {
  encode(message: MsgDelKeys, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.project !== "") {
      writer.uint32(18).string(message.project);
    }
    for (const v of message.projectKeys) {
      ProjectKey.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgDelKeys {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgDelKeys();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.creator = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.project = reader.string();
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.projectKeys.push(ProjectKey.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MsgDelKeys {
    return {
      creator: isSet(object.creator) ? String(object.creator) : "",
      project: isSet(object.project) ? String(object.project) : "",
      projectKeys: Array.isArray(object?.projectKeys) ? object.projectKeys.map((e: any) => ProjectKey.fromJSON(e)) : [],
    };
  },

  toJSON(message: MsgDelKeys): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.project !== undefined && (obj.project = message.project);
    if (message.projectKeys) {
      obj.projectKeys = message.projectKeys.map((e) => e ? ProjectKey.toJSON(e) : undefined);
    } else {
      obj.projectKeys = [];
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgDelKeys>, I>>(base?: I): MsgDelKeys {
    return MsgDelKeys.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgDelKeys>, I>>(object: I): MsgDelKeys {
    const message = createBaseMsgDelKeys();
    message.creator = object.creator ?? "";
    message.project = object.project ?? "";
    message.projectKeys = object.projectKeys?.map((e) => ProjectKey.fromPartial(e)) || [];
    return message;
  },
};

function createBaseMsgDelKeysResponse(): MsgDelKeysResponse {
  return {};
}

export const MsgDelKeysResponse = {
  encode(_: MsgDelKeysResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgDelKeysResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgDelKeysResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(_: any): MsgDelKeysResponse {
    return {};
  },

  toJSON(_: MsgDelKeysResponse): unknown {
    const obj: any = {};
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgDelKeysResponse>, I>>(base?: I): MsgDelKeysResponse {
    return MsgDelKeysResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgDelKeysResponse>, I>>(_: I): MsgDelKeysResponse {
    const message = createBaseMsgDelKeysResponse();
    return message;
  },
};

function createBaseMsgSetPolicy(): MsgSetPolicy {
  return { creator: "", project: "", policy: undefined };
}

export const MsgSetPolicy = {
  encode(message: MsgSetPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.project !== "") {
      writer.uint32(18).string(message.project);
    }
    if (message.policy !== undefined) {
      Policy.encode(message.policy, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgSetPolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSetPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.creator = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.project = reader.string();
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.policy = Policy.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MsgSetPolicy {
    return {
      creator: isSet(object.creator) ? String(object.creator) : "",
      project: isSet(object.project) ? String(object.project) : "",
      policy: isSet(object.policy) ? Policy.fromJSON(object.policy) : undefined,
    };
  },

  toJSON(message: MsgSetPolicy): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.project !== undefined && (obj.project = message.project);
    message.policy !== undefined && (obj.policy = message.policy ? Policy.toJSON(message.policy) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgSetPolicy>, I>>(base?: I): MsgSetPolicy {
    return MsgSetPolicy.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgSetPolicy>, I>>(object: I): MsgSetPolicy {
    const message = createBaseMsgSetPolicy();
    message.creator = object.creator ?? "";
    message.project = object.project ?? "";
    message.policy = (object.policy !== undefined && object.policy !== null)
      ? Policy.fromPartial(object.policy)
      : undefined;
    return message;
  },
};

function createBaseMsgSetPolicyResponse(): MsgSetPolicyResponse {
  return {};
}

export const MsgSetPolicyResponse = {
  encode(_: MsgSetPolicyResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgSetPolicyResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSetPolicyResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(_: any): MsgSetPolicyResponse {
    return {};
  },

  toJSON(_: MsgSetPolicyResponse): unknown {
    const obj: any = {};
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgSetPolicyResponse>, I>>(base?: I): MsgSetPolicyResponse {
    return MsgSetPolicyResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgSetPolicyResponse>, I>>(_: I): MsgSetPolicyResponse {
    const message = createBaseMsgSetPolicyResponse();
    return message;
  },
};

function createBaseMsgSetSubscriptionPolicy(): MsgSetSubscriptionPolicy {
  return { creator: "", projects: [], policy: undefined };
}

export const MsgSetSubscriptionPolicy = {
  encode(message: MsgSetSubscriptionPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    for (const v of message.projects) {
      writer.uint32(18).string(v!);
    }
    if (message.policy !== undefined) {
      Policy.encode(message.policy, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgSetSubscriptionPolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSetSubscriptionPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.creator = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.projects.push(reader.string());
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.policy = Policy.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MsgSetSubscriptionPolicy {
    return {
      creator: isSet(object.creator) ? String(object.creator) : "",
      projects: Array.isArray(object?.projects) ? object.projects.map((e: any) => String(e)) : [],
      policy: isSet(object.policy) ? Policy.fromJSON(object.policy) : undefined,
    };
  },

  toJSON(message: MsgSetSubscriptionPolicy): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    if (message.projects) {
      obj.projects = message.projects.map((e) => e);
    } else {
      obj.projects = [];
    }
    message.policy !== undefined && (obj.policy = message.policy ? Policy.toJSON(message.policy) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgSetSubscriptionPolicy>, I>>(base?: I): MsgSetSubscriptionPolicy {
    return MsgSetSubscriptionPolicy.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgSetSubscriptionPolicy>, I>>(object: I): MsgSetSubscriptionPolicy {
    const message = createBaseMsgSetSubscriptionPolicy();
    message.creator = object.creator ?? "";
    message.projects = object.projects?.map((e) => e) || [];
    message.policy = (object.policy !== undefined && object.policy !== null)
      ? Policy.fromPartial(object.policy)
      : undefined;
    return message;
  },
};

function createBaseMsgSetSubscriptionPolicyResponse(): MsgSetSubscriptionPolicyResponse {
  return {};
}

export const MsgSetSubscriptionPolicyResponse = {
  encode(_: MsgSetSubscriptionPolicyResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgSetSubscriptionPolicyResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSetSubscriptionPolicyResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(_: any): MsgSetSubscriptionPolicyResponse {
    return {};
  },

  toJSON(_: MsgSetSubscriptionPolicyResponse): unknown {
    const obj: any = {};
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgSetSubscriptionPolicyResponse>, I>>(
    base?: I,
  ): MsgSetSubscriptionPolicyResponse {
    return MsgSetSubscriptionPolicyResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgSetSubscriptionPolicyResponse>, I>>(
    _: I,
  ): MsgSetSubscriptionPolicyResponse {
    const message = createBaseMsgSetSubscriptionPolicyResponse();
    return message;
  },
};

/** Msg defines the Msg service. */
export interface Msg {
  AddKeys(request: MsgAddKeys): Promise<MsgAddKeysResponse>;
  DelKeys(request: MsgDelKeys): Promise<MsgDelKeysResponse>;
  SetPolicy(request: MsgSetPolicy): Promise<MsgSetPolicyResponse>;
  /** this line is used by starport scaffolding # proto/tx/rpc */
  SetSubscriptionPolicy(request: MsgSetSubscriptionPolicy): Promise<MsgSetSubscriptionPolicyResponse>;
}

export class MsgClientImpl implements Msg {
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || "lavanet.lava.projects.Msg";
    this.rpc = rpc;
    this.AddKeys = this.AddKeys.bind(this);
    this.DelKeys = this.DelKeys.bind(this);
    this.SetPolicy = this.SetPolicy.bind(this);
    this.SetSubscriptionPolicy = this.SetSubscriptionPolicy.bind(this);
  }
  AddKeys(request: MsgAddKeys): Promise<MsgAddKeysResponse> {
    const data = MsgAddKeys.encode(request).finish();
    const promise = this.rpc.request(this.service, "AddKeys", data);
    return promise.then((data) => MsgAddKeysResponse.decode(_m0.Reader.create(data)));
  }

  DelKeys(request: MsgDelKeys): Promise<MsgDelKeysResponse> {
    const data = MsgDelKeys.encode(request).finish();
    const promise = this.rpc.request(this.service, "DelKeys", data);
    return promise.then((data) => MsgDelKeysResponse.decode(_m0.Reader.create(data)));
  }

  SetPolicy(request: MsgSetPolicy): Promise<MsgSetPolicyResponse> {
    const data = MsgSetPolicy.encode(request).finish();
    const promise = this.rpc.request(this.service, "SetPolicy", data);
    return promise.then((data) => MsgSetPolicyResponse.decode(_m0.Reader.create(data)));
  }

  SetSubscriptionPolicy(request: MsgSetSubscriptionPolicy): Promise<MsgSetSubscriptionPolicyResponse> {
    const data = MsgSetSubscriptionPolicy.encode(request).finish();
    const promise = this.rpc.request(this.service, "SetSubscriptionPolicy", data);
    return promise.then((data) => MsgSetSubscriptionPolicyResponse.decode(_m0.Reader.create(data)));
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

/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { ProjectData } from "../projects/project";

export const protobufPackage = "lavanet.lava.subscription";

export interface MsgBuy {
  creator: string;
  consumer: string;
  index: string;
  /** in months */
  duration: Long;
}

export interface MsgBuyResponse {
}

export interface MsgAddProject {
  creator: string;
  projectData?: ProjectData;
}

export interface MsgAddProjectResponse {
}

export interface MsgDelProject {
  creator: string;
  name: string;
}

export interface MsgDelProjectResponse {
}

function createBaseMsgBuy(): MsgBuy {
  return { creator: "", consumer: "", index: "", duration: Long.UZERO };
}

export const MsgBuy = {
  encode(message: MsgBuy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.consumer !== "") {
      writer.uint32(18).string(message.consumer);
    }
    if (message.index !== "") {
      writer.uint32(26).string(message.index);
    }
    if (!message.duration.isZero()) {
      writer.uint32(32).uint64(message.duration);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgBuy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgBuy();
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

          message.consumer = reader.string();
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.index = reader.string();
          continue;
        case 4:
          if (tag != 32) {
            break;
          }

          message.duration = reader.uint64() as Long;
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MsgBuy {
    return {
      creator: isSet(object.creator) ? String(object.creator) : "",
      consumer: isSet(object.consumer) ? String(object.consumer) : "",
      index: isSet(object.index) ? String(object.index) : "",
      duration: isSet(object.duration) ? Long.fromValue(object.duration) : Long.UZERO,
    };
  },

  toJSON(message: MsgBuy): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.consumer !== undefined && (obj.consumer = message.consumer);
    message.index !== undefined && (obj.index = message.index);
    message.duration !== undefined && (obj.duration = (message.duration || Long.UZERO).toString());
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgBuy>, I>>(base?: I): MsgBuy {
    return MsgBuy.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgBuy>, I>>(object: I): MsgBuy {
    const message = createBaseMsgBuy();
    message.creator = object.creator ?? "";
    message.consumer = object.consumer ?? "";
    message.index = object.index ?? "";
    message.duration = (object.duration !== undefined && object.duration !== null)
      ? Long.fromValue(object.duration)
      : Long.UZERO;
    return message;
  },
};

function createBaseMsgBuyResponse(): MsgBuyResponse {
  return {};
}

export const MsgBuyResponse = {
  encode(_: MsgBuyResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgBuyResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgBuyResponse();
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

  fromJSON(_: any): MsgBuyResponse {
    return {};
  },

  toJSON(_: MsgBuyResponse): unknown {
    const obj: any = {};
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgBuyResponse>, I>>(base?: I): MsgBuyResponse {
    return MsgBuyResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgBuyResponse>, I>>(_: I): MsgBuyResponse {
    const message = createBaseMsgBuyResponse();
    return message;
  },
};

function createBaseMsgAddProject(): MsgAddProject {
  return { creator: "", projectData: undefined };
}

export const MsgAddProject = {
  encode(message: MsgAddProject, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.projectData !== undefined) {
      ProjectData.encode(message.projectData, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgAddProject {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAddProject();
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

          message.projectData = ProjectData.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MsgAddProject {
    return {
      creator: isSet(object.creator) ? String(object.creator) : "",
      projectData: isSet(object.projectData) ? ProjectData.fromJSON(object.projectData) : undefined,
    };
  },

  toJSON(message: MsgAddProject): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.projectData !== undefined &&
      (obj.projectData = message.projectData ? ProjectData.toJSON(message.projectData) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgAddProject>, I>>(base?: I): MsgAddProject {
    return MsgAddProject.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgAddProject>, I>>(object: I): MsgAddProject {
    const message = createBaseMsgAddProject();
    message.creator = object.creator ?? "";
    message.projectData = (object.projectData !== undefined && object.projectData !== null)
      ? ProjectData.fromPartial(object.projectData)
      : undefined;
    return message;
  },
};

function createBaseMsgAddProjectResponse(): MsgAddProjectResponse {
  return {};
}

export const MsgAddProjectResponse = {
  encode(_: MsgAddProjectResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgAddProjectResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAddProjectResponse();
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

  fromJSON(_: any): MsgAddProjectResponse {
    return {};
  },

  toJSON(_: MsgAddProjectResponse): unknown {
    const obj: any = {};
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgAddProjectResponse>, I>>(base?: I): MsgAddProjectResponse {
    return MsgAddProjectResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgAddProjectResponse>, I>>(_: I): MsgAddProjectResponse {
    const message = createBaseMsgAddProjectResponse();
    return message;
  },
};

function createBaseMsgDelProject(): MsgDelProject {
  return { creator: "", name: "" };
}

export const MsgDelProject = {
  encode(message: MsgDelProject, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgDelProject {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgDelProject();
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

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MsgDelProject {
    return {
      creator: isSet(object.creator) ? String(object.creator) : "",
      name: isSet(object.name) ? String(object.name) : "",
    };
  },

  toJSON(message: MsgDelProject): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgDelProject>, I>>(base?: I): MsgDelProject {
    return MsgDelProject.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgDelProject>, I>>(object: I): MsgDelProject {
    const message = createBaseMsgDelProject();
    message.creator = object.creator ?? "";
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseMsgDelProjectResponse(): MsgDelProjectResponse {
  return {};
}

export const MsgDelProjectResponse = {
  encode(_: MsgDelProjectResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgDelProjectResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgDelProjectResponse();
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

  fromJSON(_: any): MsgDelProjectResponse {
    return {};
  },

  toJSON(_: MsgDelProjectResponse): unknown {
    const obj: any = {};
    return obj;
  },

  create<I extends Exact<DeepPartial<MsgDelProjectResponse>, I>>(base?: I): MsgDelProjectResponse {
    return MsgDelProjectResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<MsgDelProjectResponse>, I>>(_: I): MsgDelProjectResponse {
    const message = createBaseMsgDelProjectResponse();
    return message;
  },
};

/** Msg defines the Msg service. */
export interface Msg {
  Buy(request: MsgBuy): Promise<MsgBuyResponse>;
  AddProject(request: MsgAddProject): Promise<MsgAddProjectResponse>;
  /** this line is used by starport scaffolding # proto/tx/rpc */
  DelProject(request: MsgDelProject): Promise<MsgDelProjectResponse>;
}

export class MsgClientImpl implements Msg {
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || "lavanet.lava.subscription.Msg";
    this.rpc = rpc;
    this.Buy = this.Buy.bind(this);
    this.AddProject = this.AddProject.bind(this);
    this.DelProject = this.DelProject.bind(this);
  }
  Buy(request: MsgBuy): Promise<MsgBuyResponse> {
    const data = MsgBuy.encode(request).finish();
    const promise = this.rpc.request(this.service, "Buy", data);
    return promise.then((data) => MsgBuyResponse.decode(_m0.Reader.create(data)));
  }

  AddProject(request: MsgAddProject): Promise<MsgAddProjectResponse> {
    const data = MsgAddProject.encode(request).finish();
    const promise = this.rpc.request(this.service, "AddProject", data);
    return promise.then((data) => MsgAddProjectResponse.decode(_m0.Reader.create(data)));
  }

  DelProject(request: MsgDelProject): Promise<MsgDelProjectResponse> {
    const data = MsgDelProject.encode(request).finish();
    const promise = this.rpc.request(this.service, "DelProject", data);
    return promise.then((data) => MsgDelProjectResponse.decode(_m0.Reader.create(data)));
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

/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Policy } from "../plans/policy";

export const protobufPackage = "lavanet.lava.projects";

export interface Project {
  /** unique id that will be the combination of subscription address and project name, cannot be changed once created */
  index: string;
  /** the subscription address that owns the project */
  subscription: string;
  /** enabled flag */
  enabled: boolean;
  /** list of the projects keys */
  projectKeys: ProjectKey[];
  adminPolicy?: Policy;
  usedCu: Long;
  subscriptionPolicy?: Policy;
  /** snapshot id to uniquely identify snapshots */
  snapshot: Long;
}

export interface ProjectKey {
  /** the address of the project key */
  key: string;
  kinds: number;
}

export enum ProjectKey_Type {
  NONE = 0,
  ADMIN = 1,
  DEVELOPER = 2,
  UNRECOGNIZED = -1,
}

export function projectKey_TypeFromJSON(object: any): ProjectKey_Type {
  switch (object) {
    case 0:
    case "NONE":
      return ProjectKey_Type.NONE;
    case 1:
    case "ADMIN":
      return ProjectKey_Type.ADMIN;
    case 2:
    case "DEVELOPER":
      return ProjectKey_Type.DEVELOPER;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ProjectKey_Type.UNRECOGNIZED;
  }
}

export function projectKey_TypeToJSON(object: ProjectKey_Type): string {
  switch (object) {
    case ProjectKey_Type.NONE:
      return "NONE";
    case ProjectKey_Type.ADMIN:
      return "ADMIN";
    case ProjectKey_Type.DEVELOPER:
      return "DEVELOPER";
    case ProjectKey_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface ProtoDeveloperData {
  projectID: string;
}

/** used as a container struct for the subscription module */
export interface ProjectData {
  name: string;
  enabled: boolean;
  projectKeys: ProjectKey[];
  policy?: Policy;
}

function createBaseProject(): Project {
  return {
    index: "",
    subscription: "",
    enabled: false,
    projectKeys: [],
    adminPolicy: undefined,
    usedCu: Long.UZERO,
    subscriptionPolicy: undefined,
    snapshot: Long.UZERO,
  };
}

export const Project = {
  encode(message: Project, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    if (message.subscription !== "") {
      writer.uint32(18).string(message.subscription);
    }
    if (message.enabled === true) {
      writer.uint32(32).bool(message.enabled);
    }
    for (const v of message.projectKeys) {
      ProjectKey.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    if (message.adminPolicy !== undefined) {
      Policy.encode(message.adminPolicy, writer.uint32(50).fork()).ldelim();
    }
    if (!message.usedCu.isZero()) {
      writer.uint32(56).uint64(message.usedCu);
    }
    if (message.subscriptionPolicy !== undefined) {
      Policy.encode(message.subscriptionPolicy, writer.uint32(66).fork()).ldelim();
    }
    if (!message.snapshot.isZero()) {
      writer.uint32(72).uint64(message.snapshot);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Project {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseProject();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.index = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.subscription = reader.string();
          continue;
        case 4:
          if (tag != 32) {
            break;
          }

          message.enabled = reader.bool();
          continue;
        case 5:
          if (tag != 42) {
            break;
          }

          message.projectKeys.push(ProjectKey.decode(reader, reader.uint32()));
          continue;
        case 6:
          if (tag != 50) {
            break;
          }

          message.adminPolicy = Policy.decode(reader, reader.uint32());
          continue;
        case 7:
          if (tag != 56) {
            break;
          }

          message.usedCu = reader.uint64() as Long;
          continue;
        case 8:
          if (tag != 66) {
            break;
          }

          message.subscriptionPolicy = Policy.decode(reader, reader.uint32());
          continue;
        case 9:
          if (tag != 72) {
            break;
          }

          message.snapshot = reader.uint64() as Long;
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Project {
    return {
      index: isSet(object.index) ? String(object.index) : "",
      subscription: isSet(object.subscription) ? String(object.subscription) : "",
      enabled: isSet(object.enabled) ? Boolean(object.enabled) : false,
      projectKeys: Array.isArray(object?.projectKeys) ? object.projectKeys.map((e: any) => ProjectKey.fromJSON(e)) : [],
      adminPolicy: isSet(object.adminPolicy) ? Policy.fromJSON(object.adminPolicy) : undefined,
      usedCu: isSet(object.usedCu) ? Long.fromValue(object.usedCu) : Long.UZERO,
      subscriptionPolicy: isSet(object.subscriptionPolicy) ? Policy.fromJSON(object.subscriptionPolicy) : undefined,
      snapshot: isSet(object.snapshot) ? Long.fromValue(object.snapshot) : Long.UZERO,
    };
  },

  toJSON(message: Project): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    message.subscription !== undefined && (obj.subscription = message.subscription);
    message.enabled !== undefined && (obj.enabled = message.enabled);
    if (message.projectKeys) {
      obj.projectKeys = message.projectKeys.map((e) => e ? ProjectKey.toJSON(e) : undefined);
    } else {
      obj.projectKeys = [];
    }
    message.adminPolicy !== undefined &&
      (obj.adminPolicy = message.adminPolicy ? Policy.toJSON(message.adminPolicy) : undefined);
    message.usedCu !== undefined && (obj.usedCu = (message.usedCu || Long.UZERO).toString());
    message.subscriptionPolicy !== undefined &&
      (obj.subscriptionPolicy = message.subscriptionPolicy ? Policy.toJSON(message.subscriptionPolicy) : undefined);
    message.snapshot !== undefined && (obj.snapshot = (message.snapshot || Long.UZERO).toString());
    return obj;
  },

  create<I extends Exact<DeepPartial<Project>, I>>(base?: I): Project {
    return Project.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<Project>, I>>(object: I): Project {
    const message = createBaseProject();
    message.index = object.index ?? "";
    message.subscription = object.subscription ?? "";
    message.enabled = object.enabled ?? false;
    message.projectKeys = object.projectKeys?.map((e) => ProjectKey.fromPartial(e)) || [];
    message.adminPolicy = (object.adminPolicy !== undefined && object.adminPolicy !== null)
      ? Policy.fromPartial(object.adminPolicy)
      : undefined;
    message.usedCu = (object.usedCu !== undefined && object.usedCu !== null)
      ? Long.fromValue(object.usedCu)
      : Long.UZERO;
    message.subscriptionPolicy = (object.subscriptionPolicy !== undefined && object.subscriptionPolicy !== null)
      ? Policy.fromPartial(object.subscriptionPolicy)
      : undefined;
    message.snapshot = (object.snapshot !== undefined && object.snapshot !== null)
      ? Long.fromValue(object.snapshot)
      : Long.UZERO;
    return message;
  },
};

function createBaseProjectKey(): ProjectKey {
  return { key: "", kinds: 0 };
}

export const ProjectKey = {
  encode(message: ProjectKey, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.kinds !== 0) {
      writer.uint32(32).uint32(message.kinds);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ProjectKey {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseProjectKey();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.key = reader.string();
          continue;
        case 4:
          if (tag != 32) {
            break;
          }

          message.kinds = reader.uint32();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ProjectKey {
    return { key: isSet(object.key) ? String(object.key) : "", kinds: isSet(object.kinds) ? Number(object.kinds) : 0 };
  },

  toJSON(message: ProjectKey): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.kinds !== undefined && (obj.kinds = Math.round(message.kinds));
    return obj;
  },

  create<I extends Exact<DeepPartial<ProjectKey>, I>>(base?: I): ProjectKey {
    return ProjectKey.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<ProjectKey>, I>>(object: I): ProjectKey {
    const message = createBaseProjectKey();
    message.key = object.key ?? "";
    message.kinds = object.kinds ?? 0;
    return message;
  },
};

function createBaseProtoDeveloperData(): ProtoDeveloperData {
  return { projectID: "" };
}

export const ProtoDeveloperData = {
  encode(message: ProtoDeveloperData, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.projectID !== "") {
      writer.uint32(10).string(message.projectID);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ProtoDeveloperData {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseProtoDeveloperData();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.projectID = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ProtoDeveloperData {
    return { projectID: isSet(object.projectID) ? String(object.projectID) : "" };
  },

  toJSON(message: ProtoDeveloperData): unknown {
    const obj: any = {};
    message.projectID !== undefined && (obj.projectID = message.projectID);
    return obj;
  },

  create<I extends Exact<DeepPartial<ProtoDeveloperData>, I>>(base?: I): ProtoDeveloperData {
    return ProtoDeveloperData.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<ProtoDeveloperData>, I>>(object: I): ProtoDeveloperData {
    const message = createBaseProtoDeveloperData();
    message.projectID = object.projectID ?? "";
    return message;
  },
};

function createBaseProjectData(): ProjectData {
  return { name: "", enabled: false, projectKeys: [], policy: undefined };
}

export const ProjectData = {
  encode(message: ProjectData, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.enabled === true) {
      writer.uint32(24).bool(message.enabled);
    }
    for (const v of message.projectKeys) {
      ProjectKey.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    if (message.policy !== undefined) {
      Policy.encode(message.policy, writer.uint32(42).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ProjectData {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseProjectData();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 3:
          if (tag != 24) {
            break;
          }

          message.enabled = reader.bool();
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.projectKeys.push(ProjectKey.decode(reader, reader.uint32()));
          continue;
        case 5:
          if (tag != 42) {
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

  fromJSON(object: any): ProjectData {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      enabled: isSet(object.enabled) ? Boolean(object.enabled) : false,
      projectKeys: Array.isArray(object?.projectKeys) ? object.projectKeys.map((e: any) => ProjectKey.fromJSON(e)) : [],
      policy: isSet(object.policy) ? Policy.fromJSON(object.policy) : undefined,
    };
  },

  toJSON(message: ProjectData): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.enabled !== undefined && (obj.enabled = message.enabled);
    if (message.projectKeys) {
      obj.projectKeys = message.projectKeys.map((e) => e ? ProjectKey.toJSON(e) : undefined);
    } else {
      obj.projectKeys = [];
    }
    message.policy !== undefined && (obj.policy = message.policy ? Policy.toJSON(message.policy) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<ProjectData>, I>>(base?: I): ProjectData {
    return ProjectData.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<ProjectData>, I>>(object: I): ProjectData {
    const message = createBaseProjectData();
    message.name = object.name ?? "";
    message.enabled = object.enabled ?? false;
    message.projectKeys = object.projectKeys?.map((e) => ProjectKey.fromPartial(e)) || [];
    message.policy = (object.policy !== undefined && object.policy !== null)
      ? Policy.fromPartial(object.policy)
      : undefined;
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

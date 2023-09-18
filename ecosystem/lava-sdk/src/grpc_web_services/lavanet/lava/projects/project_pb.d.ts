// package: lavanet.lava.projects
// file: lavanet/lava/projects/project.proto

import * as jspb from "google-protobuf";
import * as gogoproto_gogo_pb from "../../../gogoproto/gogo_pb";
import * as lavanet_lava_plans_policy_pb from "../../../lavanet/lava/plans/policy_pb";

export class Project extends jspb.Message {
  getIndex(): string;
  setIndex(value: string): void;

  getSubscription(): string;
  setSubscription(value: string): void;

  getEnabled(): boolean;
  setEnabled(value: boolean): void;

  clearProjectKeysList(): void;
  getProjectKeysList(): Array<ProjectKey>;
  setProjectKeysList(value: Array<ProjectKey>): void;
  addProjectKeys(value?: ProjectKey, index?: number): ProjectKey;

  hasAdminPolicy(): boolean;
  clearAdminPolicy(): void;
  getAdminPolicy(): lavanet_lava_plans_policy_pb.Policy | undefined;
  setAdminPolicy(value?: lavanet_lava_plans_policy_pb.Policy): void;

  getUsedCu(): number;
  setUsedCu(value: number): void;

  hasSubscriptionPolicy(): boolean;
  clearSubscriptionPolicy(): void;
  getSubscriptionPolicy(): lavanet_lava_plans_policy_pb.Policy | undefined;
  setSubscriptionPolicy(value?: lavanet_lava_plans_policy_pb.Policy): void;

  getSnapshot(): number;
  setSnapshot(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Project.AsObject;
  static toObject(includeInstance: boolean, msg: Project): Project.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Project, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Project;
  static deserializeBinaryFromReader(message: Project, reader: jspb.BinaryReader): Project;
}

export namespace Project {
  export type AsObject = {
    index: string,
    subscription: string,
    enabled: boolean,
    projectKeysList: Array<ProjectKey.AsObject>,
    adminPolicy?: lavanet_lava_plans_policy_pb.Policy.AsObject,
    usedCu: number,
    subscriptionPolicy?: lavanet_lava_plans_policy_pb.Policy.AsObject,
    snapshot: number,
  }
}

export class ProjectKey extends jspb.Message {
  getKey(): string;
  setKey(value: string): void;

  getKinds(): number;
  setKinds(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ProjectKey.AsObject;
  static toObject(includeInstance: boolean, msg: ProjectKey): ProjectKey.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ProjectKey, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ProjectKey;
  static deserializeBinaryFromReader(message: ProjectKey, reader: jspb.BinaryReader): ProjectKey;
}

export namespace ProjectKey {
  export type AsObject = {
    key: string,
    kinds: number,
  }

  export interface TypeMap {
    NONE: 0;
    ADMIN: 1;
    DEVELOPER: 2;
  }

  export const Type: TypeMap;
}

export class ProtoDeveloperData extends jspb.Message {
  getProjectid(): string;
  setProjectid(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ProtoDeveloperData.AsObject;
  static toObject(includeInstance: boolean, msg: ProtoDeveloperData): ProtoDeveloperData.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ProtoDeveloperData, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ProtoDeveloperData;
  static deserializeBinaryFromReader(message: ProtoDeveloperData, reader: jspb.BinaryReader): ProtoDeveloperData;
}

export namespace ProtoDeveloperData {
  export type AsObject = {
    projectid: string,
  }
}

export class ProjectData extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  getEnabled(): boolean;
  setEnabled(value: boolean): void;

  clearProjectkeysList(): void;
  getProjectkeysList(): Array<ProjectKey>;
  setProjectkeysList(value: Array<ProjectKey>): void;
  addProjectkeys(value?: ProjectKey, index?: number): ProjectKey;

  hasPolicy(): boolean;
  clearPolicy(): void;
  getPolicy(): lavanet_lava_plans_policy_pb.Policy | undefined;
  setPolicy(value?: lavanet_lava_plans_policy_pb.Policy): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ProjectData.AsObject;
  static toObject(includeInstance: boolean, msg: ProjectData): ProjectData.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ProjectData, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ProjectData;
  static deserializeBinaryFromReader(message: ProjectData, reader: jspb.BinaryReader): ProjectData;
}

export namespace ProjectData {
  export type AsObject = {
    name: string,
    enabled: boolean,
    projectkeysList: Array<ProjectKey.AsObject>,
    policy?: lavanet_lava_plans_policy_pb.Policy.AsObject,
  }
}


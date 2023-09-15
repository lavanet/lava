// package: lavanet.lava.plans
// file: lavanet/lava/plans/policy.proto

import * as jspb from "google-protobuf";
import * as gogoproto_gogo_pb from "../../../gogoproto/gogo_pb";
import * as cosmos_base_v1beta1_coin_pb from "../../../cosmos/base/v1beta1/coin_pb";
import * as lavanet_lava_spec_api_collection_pb from "../../../lavanet/lava/spec/api_collection_pb";

export class Policy extends jspb.Message {
  clearChainPoliciesList(): void;
  getChainPoliciesList(): Array<ChainPolicy>;
  setChainPoliciesList(value: Array<ChainPolicy>): void;
  addChainPolicies(value?: ChainPolicy, index?: number): ChainPolicy;

  getGeolocationProfile(): number;
  setGeolocationProfile(value: number): void;

  getTotalCuLimit(): number;
  setTotalCuLimit(value: number): void;

  getEpochCuLimit(): number;
  setEpochCuLimit(value: number): void;

  getMaxProvidersToPair(): number;
  setMaxProvidersToPair(value: number): void;

  getSelectedProvidersMode(): SELECTED_PROVIDERS_MODEMap[keyof SELECTED_PROVIDERS_MODEMap];
  setSelectedProvidersMode(value: SELECTED_PROVIDERS_MODEMap[keyof SELECTED_PROVIDERS_MODEMap]): void;

  clearSelectedProvidersList(): void;
  getSelectedProvidersList(): Array<string>;
  setSelectedProvidersList(value: Array<string>): void;
  addSelectedProviders(value: string, index?: number): string;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Policy.AsObject;
  static toObject(includeInstance: boolean, msg: Policy): Policy.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Policy, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Policy;
  static deserializeBinaryFromReader(message: Policy, reader: jspb.BinaryReader): Policy;
}

export namespace Policy {
  export type AsObject = {
    chainPoliciesList: Array<ChainPolicy.AsObject>,
    geolocationProfile: number,
    totalCuLimit: number,
    epochCuLimit: number,
    maxProvidersToPair: number,
    selectedProvidersMode: SELECTED_PROVIDERS_MODEMap[keyof SELECTED_PROVIDERS_MODEMap],
    selectedProvidersList: Array<string>,
  }
}

export class ChainPolicy extends jspb.Message {
  getChainId(): string;
  setChainId(value: string): void;

  clearApisList(): void;
  getApisList(): Array<string>;
  setApisList(value: Array<string>): void;
  addApis(value: string, index?: number): string;

  clearRequirementsList(): void;
  getRequirementsList(): Array<ChainRequirement>;
  setRequirementsList(value: Array<ChainRequirement>): void;
  addRequirements(value?: ChainRequirement, index?: number): ChainRequirement;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ChainPolicy.AsObject;
  static toObject(includeInstance: boolean, msg: ChainPolicy): ChainPolicy.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ChainPolicy, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ChainPolicy;
  static deserializeBinaryFromReader(message: ChainPolicy, reader: jspb.BinaryReader): ChainPolicy;
}

export namespace ChainPolicy {
  export type AsObject = {
    chainId: string,
    apisList: Array<string>,
    requirementsList: Array<ChainRequirement.AsObject>,
  }
}

export class ChainRequirement extends jspb.Message {
  hasCollection(): boolean;
  clearCollection(): void;
  getCollection(): lavanet_lava_spec_api_collection_pb.CollectionData | undefined;
  setCollection(value?: lavanet_lava_spec_api_collection_pb.CollectionData): void;

  clearExtensionsList(): void;
  getExtensionsList(): Array<string>;
  setExtensionsList(value: Array<string>): void;
  addExtensions(value: string, index?: number): string;

  getMixed(): boolean;
  setMixed(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ChainRequirement.AsObject;
  static toObject(includeInstance: boolean, msg: ChainRequirement): ChainRequirement.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ChainRequirement, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ChainRequirement;
  static deserializeBinaryFromReader(message: ChainRequirement, reader: jspb.BinaryReader): ChainRequirement;
}

export namespace ChainRequirement {
  export type AsObject = {
    collection?: lavanet_lava_spec_api_collection_pb.CollectionData.AsObject,
    extensionsList: Array<string>,
    mixed: boolean,
  }
}

export interface SELECTED_PROVIDERS_MODEMap {
  ALLOWED: 0;
  MIXED: 1;
  EXCLUSIVE: 2;
  DISABLED: 3;
}

export const SELECTED_PROVIDERS_MODE: SELECTED_PROVIDERS_MODEMap;


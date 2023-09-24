// package: lavanet.lava.spec
// file: lavanet/lava/spec/spec.proto

import * as jspb from "google-protobuf";
import * as gogoproto_gogo_pb from "../../../gogoproto/gogo_pb";
import * as lavanet_lava_spec_api_collection_pb from "../../../lavanet/lava/spec/api_collection_pb";
import * as cosmos_base_v1beta1_coin_pb from "../../../cosmos/base/v1beta1/coin_pb";

export class Spec extends jspb.Message {
  getIndex(): string;
  setIndex(value: string): void;

  getName(): string;
  setName(value: string): void;

  getEnabled(): boolean;
  setEnabled(value: boolean): void;

  getReliabilityThreshold(): number;
  setReliabilityThreshold(value: number): void;

  getDataReliabilityEnabled(): boolean;
  setDataReliabilityEnabled(value: boolean): void;

  getBlockDistanceForFinalizedData(): number;
  setBlockDistanceForFinalizedData(value: number): void;

  getBlocksInFinalizationProof(): number;
  setBlocksInFinalizationProof(value: number): void;

  getAverageBlockTime(): number;
  setAverageBlockTime(value: number): void;

  getAllowedBlockLagForQosSync(): number;
  setAllowedBlockLagForQosSync(value: number): void;

  getBlockLastUpdated(): number;
  setBlockLastUpdated(value: number): void;

  hasMinStakeProvider(): boolean;
  clearMinStakeProvider(): void;
  getMinStakeProvider(): cosmos_base_v1beta1_coin_pb.Coin | undefined;
  setMinStakeProvider(value?: cosmos_base_v1beta1_coin_pb.Coin): void;

  hasMinStakeClient(): boolean;
  clearMinStakeClient(): void;
  getMinStakeClient(): cosmos_base_v1beta1_coin_pb.Coin | undefined;
  setMinStakeClient(value?: cosmos_base_v1beta1_coin_pb.Coin): void;

  getProvidersTypes(): Spec.ProvidersTypesMap[keyof Spec.ProvidersTypesMap];
  setProvidersTypes(value: Spec.ProvidersTypesMap[keyof Spec.ProvidersTypesMap]): void;

  clearImportsList(): void;
  getImportsList(): Array<string>;
  setImportsList(value: Array<string>): void;
  addImports(value: string, index?: number): string;

  clearApiCollectionsList(): void;
  getApiCollectionsList(): Array<lavanet_lava_spec_api_collection_pb.ApiCollection>;
  setApiCollectionsList(value: Array<lavanet_lava_spec_api_collection_pb.ApiCollection>): void;
  addApiCollections(value?: lavanet_lava_spec_api_collection_pb.ApiCollection, index?: number): lavanet_lava_spec_api_collection_pb.ApiCollection;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Spec.AsObject;
  static toObject(includeInstance: boolean, msg: Spec): Spec.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Spec, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Spec;
  static deserializeBinaryFromReader(message: Spec, reader: jspb.BinaryReader): Spec;
}

export namespace Spec {
  export type AsObject = {
    index: string,
    name: string,
    enabled: boolean,
    reliabilityThreshold: number,
    dataReliabilityEnabled: boolean,
    blockDistanceForFinalizedData: number,
    blocksInFinalizationProof: number,
    averageBlockTime: number,
    allowedBlockLagForQosSync: number,
    blockLastUpdated: number,
    minStakeProvider?: cosmos_base_v1beta1_coin_pb.Coin.AsObject,
    minStakeClient?: cosmos_base_v1beta1_coin_pb.Coin.AsObject,
    providersTypes: Spec.ProvidersTypesMap[keyof Spec.ProvidersTypesMap],
    importsList: Array<string>,
    apiCollectionsList: Array<lavanet_lava_spec_api_collection_pb.ApiCollection.AsObject>,
  }

  export interface ProvidersTypesMap {
    DYNAMIC: 0;
    STATIC: 1;
  }

  export const ProvidersTypes: ProvidersTypesMap;
}


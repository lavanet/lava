// package: lavanet.lava.pairing
// file: lavanet/lava/pairing/query.proto

import * as jspb from "google-protobuf";
import * as gogoproto_gogo_pb from "../../../gogoproto/gogo_pb";
import * as google_api_annotations_pb from "../../../google/api/annotations_pb";
import * as cosmos_base_query_v1beta1_pagination_pb from "../../../cosmos/base/query/v1beta1/pagination_pb";
import * as lavanet_lava_pairing_params_pb from "../../../lavanet/lava/pairing/params_pb";
import * as lavanet_lava_pairing_epoch_payments_pb from "../../../lavanet/lava/pairing/epoch_payments_pb";
import * as lavanet_lava_spec_spec_pb from "../../../lavanet/lava/spec/spec_pb";
import * as lavanet_lava_plans_policy_pb from "../../../lavanet/lava/plans/policy_pb";
import * as lavanet_lava_pairing_provider_payment_storage_pb from "../../../lavanet/lava/pairing/provider_payment_storage_pb";
import * as lavanet_lava_pairing_unique_payment_storage_client_provider_pb from "../../../lavanet/lava/pairing/unique_payment_storage_client_provider_pb";
import * as lavanet_lava_epochstorage_stake_entry_pb from "../../../lavanet/lava/epochstorage/stake_entry_pb";
import * as lavanet_lava_subscription_subscription_pb from "../../../lavanet/lava/subscription/subscription_pb";
import * as lavanet_lava_projects_project_pb from "../../../lavanet/lava/projects/project_pb";

export class QueryParamsRequest extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryParamsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: QueryParamsRequest): QueryParamsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryParamsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryParamsRequest;
  static deserializeBinaryFromReader(message: QueryParamsRequest, reader: jspb.BinaryReader): QueryParamsRequest;
}

export namespace QueryParamsRequest {
  export type AsObject = {
  }
}

export class QueryParamsResponse extends jspb.Message {
  hasParams(): boolean;
  clearParams(): void;
  getParams(): lavanet_lava_pairing_params_pb.Params | undefined;
  setParams(value?: lavanet_lava_pairing_params_pb.Params): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryParamsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: QueryParamsResponse): QueryParamsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryParamsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryParamsResponse;
  static deserializeBinaryFromReader(message: QueryParamsResponse, reader: jspb.BinaryReader): QueryParamsResponse;
}

export namespace QueryParamsResponse {
  export type AsObject = {
    params?: lavanet_lava_pairing_params_pb.Params.AsObject,
  }
}

export class QueryProvidersRequest extends jspb.Message {
  getChainid(): string;
  setChainid(value: string): void;

  getShowfrozen(): boolean;
  setShowfrozen(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryProvidersRequest.AsObject;
  static toObject(includeInstance: boolean, msg: QueryProvidersRequest): QueryProvidersRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryProvidersRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryProvidersRequest;
  static deserializeBinaryFromReader(message: QueryProvidersRequest, reader: jspb.BinaryReader): QueryProvidersRequest;
}

export namespace QueryProvidersRequest {
  export type AsObject = {
    chainid: string,
    showfrozen: boolean,
  }
}

export class QueryProvidersResponse extends jspb.Message {
  clearStakeentryList(): void;
  getStakeentryList(): Array<lavanet_lava_epochstorage_stake_entry_pb.StakeEntry>;
  setStakeentryList(value: Array<lavanet_lava_epochstorage_stake_entry_pb.StakeEntry>): void;
  addStakeentry(value?: lavanet_lava_epochstorage_stake_entry_pb.StakeEntry, index?: number): lavanet_lava_epochstorage_stake_entry_pb.StakeEntry;

  getOutput(): string;
  setOutput(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryProvidersResponse.AsObject;
  static toObject(includeInstance: boolean, msg: QueryProvidersResponse): QueryProvidersResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryProvidersResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryProvidersResponse;
  static deserializeBinaryFromReader(message: QueryProvidersResponse, reader: jspb.BinaryReader): QueryProvidersResponse;
}

export namespace QueryProvidersResponse {
  export type AsObject = {
    stakeentryList: Array<lavanet_lava_epochstorage_stake_entry_pb.StakeEntry.AsObject>,
    output: string,
  }
}

export class QueryGetPairingRequest extends jspb.Message {
  getChainid(): string;
  setChainid(value: string): void;

  getClient(): string;
  setClient(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryGetPairingRequest.AsObject;
  static toObject(includeInstance: boolean, msg: QueryGetPairingRequest): QueryGetPairingRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryGetPairingRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryGetPairingRequest;
  static deserializeBinaryFromReader(message: QueryGetPairingRequest, reader: jspb.BinaryReader): QueryGetPairingRequest;
}

export namespace QueryGetPairingRequest {
  export type AsObject = {
    chainid: string,
    client: string,
  }
}

export class QueryGetPairingResponse extends jspb.Message {
  clearProvidersList(): void;
  getProvidersList(): Array<lavanet_lava_epochstorage_stake_entry_pb.StakeEntry>;
  setProvidersList(value: Array<lavanet_lava_epochstorage_stake_entry_pb.StakeEntry>): void;
  addProviders(value?: lavanet_lava_epochstorage_stake_entry_pb.StakeEntry, index?: number): lavanet_lava_epochstorage_stake_entry_pb.StakeEntry;

  getCurrentEpoch(): number;
  setCurrentEpoch(value: number): void;

  getTimeLeftToNextPairing(): number;
  setTimeLeftToNextPairing(value: number): void;

  getSpecLastUpdatedBlock(): number;
  setSpecLastUpdatedBlock(value: number): void;

  getBlockOfNextPairing(): number;
  setBlockOfNextPairing(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryGetPairingResponse.AsObject;
  static toObject(includeInstance: boolean, msg: QueryGetPairingResponse): QueryGetPairingResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryGetPairingResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryGetPairingResponse;
  static deserializeBinaryFromReader(message: QueryGetPairingResponse, reader: jspb.BinaryReader): QueryGetPairingResponse;
}

export namespace QueryGetPairingResponse {
  export type AsObject = {
    providersList: Array<lavanet_lava_epochstorage_stake_entry_pb.StakeEntry.AsObject>,
    currentEpoch: number,
    timeLeftToNextPairing: number,
    specLastUpdatedBlock: number,
    blockOfNextPairing: number,
  }
}

export class QueryVerifyPairingRequest extends jspb.Message {
  getChainid(): string;
  setChainid(value: string): void;

  getClient(): string;
  setClient(value: string): void;

  getProvider(): string;
  setProvider(value: string): void;

  getBlock(): number;
  setBlock(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryVerifyPairingRequest.AsObject;
  static toObject(includeInstance: boolean, msg: QueryVerifyPairingRequest): QueryVerifyPairingRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryVerifyPairingRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryVerifyPairingRequest;
  static deserializeBinaryFromReader(message: QueryVerifyPairingRequest, reader: jspb.BinaryReader): QueryVerifyPairingRequest;
}

export namespace QueryVerifyPairingRequest {
  export type AsObject = {
    chainid: string,
    client: string,
    provider: string,
    block: number,
  }
}

export class QueryVerifyPairingResponse extends jspb.Message {
  getValid(): boolean;
  setValid(value: boolean): void;

  getPairedProviders(): number;
  setPairedProviders(value: number): void;

  getCuPerEpoch(): number;
  setCuPerEpoch(value: number): void;

  getProjectId(): string;
  setProjectId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryVerifyPairingResponse.AsObject;
  static toObject(includeInstance: boolean, msg: QueryVerifyPairingResponse): QueryVerifyPairingResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryVerifyPairingResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryVerifyPairingResponse;
  static deserializeBinaryFromReader(message: QueryVerifyPairingResponse, reader: jspb.BinaryReader): QueryVerifyPairingResponse;
}

export namespace QueryVerifyPairingResponse {
  export type AsObject = {
    valid: boolean,
    pairedProviders: number,
    cuPerEpoch: number,
    projectId: string,
  }
}

export class QueryGetUniquePaymentStorageClientProviderRequest extends jspb.Message {
  getIndex(): string;
  setIndex(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryGetUniquePaymentStorageClientProviderRequest.AsObject;
  static toObject(includeInstance: boolean, msg: QueryGetUniquePaymentStorageClientProviderRequest): QueryGetUniquePaymentStorageClientProviderRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryGetUniquePaymentStorageClientProviderRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryGetUniquePaymentStorageClientProviderRequest;
  static deserializeBinaryFromReader(message: QueryGetUniquePaymentStorageClientProviderRequest, reader: jspb.BinaryReader): QueryGetUniquePaymentStorageClientProviderRequest;
}

export namespace QueryGetUniquePaymentStorageClientProviderRequest {
  export type AsObject = {
    index: string,
  }
}

export class QueryGetUniquePaymentStorageClientProviderResponse extends jspb.Message {
  hasUniquepaymentstorageclientprovider(): boolean;
  clearUniquepaymentstorageclientprovider(): void;
  getUniquepaymentstorageclientprovider(): lavanet_lava_pairing_unique_payment_storage_client_provider_pb.UniquePaymentStorageClientProvider | undefined;
  setUniquepaymentstorageclientprovider(value?: lavanet_lava_pairing_unique_payment_storage_client_provider_pb.UniquePaymentStorageClientProvider): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryGetUniquePaymentStorageClientProviderResponse.AsObject;
  static toObject(includeInstance: boolean, msg: QueryGetUniquePaymentStorageClientProviderResponse): QueryGetUniquePaymentStorageClientProviderResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryGetUniquePaymentStorageClientProviderResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryGetUniquePaymentStorageClientProviderResponse;
  static deserializeBinaryFromReader(message: QueryGetUniquePaymentStorageClientProviderResponse, reader: jspb.BinaryReader): QueryGetUniquePaymentStorageClientProviderResponse;
}

export namespace QueryGetUniquePaymentStorageClientProviderResponse {
  export type AsObject = {
    uniquepaymentstorageclientprovider?: lavanet_lava_pairing_unique_payment_storage_client_provider_pb.UniquePaymentStorageClientProvider.AsObject,
  }
}

export class QueryAllUniquePaymentStorageClientProviderRequest extends jspb.Message {
  hasPagination(): boolean;
  clearPagination(): void;
  getPagination(): cosmos_base_query_v1beta1_pagination_pb.PageRequest | undefined;
  setPagination(value?: cosmos_base_query_v1beta1_pagination_pb.PageRequest): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryAllUniquePaymentStorageClientProviderRequest.AsObject;
  static toObject(includeInstance: boolean, msg: QueryAllUniquePaymentStorageClientProviderRequest): QueryAllUniquePaymentStorageClientProviderRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryAllUniquePaymentStorageClientProviderRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryAllUniquePaymentStorageClientProviderRequest;
  static deserializeBinaryFromReader(message: QueryAllUniquePaymentStorageClientProviderRequest, reader: jspb.BinaryReader): QueryAllUniquePaymentStorageClientProviderRequest;
}

export namespace QueryAllUniquePaymentStorageClientProviderRequest {
  export type AsObject = {
    pagination?: cosmos_base_query_v1beta1_pagination_pb.PageRequest.AsObject,
  }
}

export class QueryAllUniquePaymentStorageClientProviderResponse extends jspb.Message {
  clearUniquepaymentstorageclientproviderList(): void;
  getUniquepaymentstorageclientproviderList(): Array<lavanet_lava_pairing_unique_payment_storage_client_provider_pb.UniquePaymentStorageClientProvider>;
  setUniquepaymentstorageclientproviderList(value: Array<lavanet_lava_pairing_unique_payment_storage_client_provider_pb.UniquePaymentStorageClientProvider>): void;
  addUniquepaymentstorageclientprovider(value?: lavanet_lava_pairing_unique_payment_storage_client_provider_pb.UniquePaymentStorageClientProvider, index?: number): lavanet_lava_pairing_unique_payment_storage_client_provider_pb.UniquePaymentStorageClientProvider;

  hasPagination(): boolean;
  clearPagination(): void;
  getPagination(): cosmos_base_query_v1beta1_pagination_pb.PageResponse | undefined;
  setPagination(value?: cosmos_base_query_v1beta1_pagination_pb.PageResponse): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryAllUniquePaymentStorageClientProviderResponse.AsObject;
  static toObject(includeInstance: boolean, msg: QueryAllUniquePaymentStorageClientProviderResponse): QueryAllUniquePaymentStorageClientProviderResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryAllUniquePaymentStorageClientProviderResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryAllUniquePaymentStorageClientProviderResponse;
  static deserializeBinaryFromReader(message: QueryAllUniquePaymentStorageClientProviderResponse, reader: jspb.BinaryReader): QueryAllUniquePaymentStorageClientProviderResponse;
}

export namespace QueryAllUniquePaymentStorageClientProviderResponse {
  export type AsObject = {
    uniquepaymentstorageclientproviderList: Array<lavanet_lava_pairing_unique_payment_storage_client_provider_pb.UniquePaymentStorageClientProvider.AsObject>,
    pagination?: cosmos_base_query_v1beta1_pagination_pb.PageResponse.AsObject,
  }
}

export class QueryGetProviderPaymentStorageRequest extends jspb.Message {
  getIndex(): string;
  setIndex(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryGetProviderPaymentStorageRequest.AsObject;
  static toObject(includeInstance: boolean, msg: QueryGetProviderPaymentStorageRequest): QueryGetProviderPaymentStorageRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryGetProviderPaymentStorageRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryGetProviderPaymentStorageRequest;
  static deserializeBinaryFromReader(message: QueryGetProviderPaymentStorageRequest, reader: jspb.BinaryReader): QueryGetProviderPaymentStorageRequest;
}

export namespace QueryGetProviderPaymentStorageRequest {
  export type AsObject = {
    index: string,
  }
}

export class QueryGetProviderPaymentStorageResponse extends jspb.Message {
  hasProviderpaymentstorage(): boolean;
  clearProviderpaymentstorage(): void;
  getProviderpaymentstorage(): lavanet_lava_pairing_provider_payment_storage_pb.ProviderPaymentStorage | undefined;
  setProviderpaymentstorage(value?: lavanet_lava_pairing_provider_payment_storage_pb.ProviderPaymentStorage): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryGetProviderPaymentStorageResponse.AsObject;
  static toObject(includeInstance: boolean, msg: QueryGetProviderPaymentStorageResponse): QueryGetProviderPaymentStorageResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryGetProviderPaymentStorageResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryGetProviderPaymentStorageResponse;
  static deserializeBinaryFromReader(message: QueryGetProviderPaymentStorageResponse, reader: jspb.BinaryReader): QueryGetProviderPaymentStorageResponse;
}

export namespace QueryGetProviderPaymentStorageResponse {
  export type AsObject = {
    providerpaymentstorage?: lavanet_lava_pairing_provider_payment_storage_pb.ProviderPaymentStorage.AsObject,
  }
}

export class QueryAllProviderPaymentStorageRequest extends jspb.Message {
  hasPagination(): boolean;
  clearPagination(): void;
  getPagination(): cosmos_base_query_v1beta1_pagination_pb.PageRequest | undefined;
  setPagination(value?: cosmos_base_query_v1beta1_pagination_pb.PageRequest): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryAllProviderPaymentStorageRequest.AsObject;
  static toObject(includeInstance: boolean, msg: QueryAllProviderPaymentStorageRequest): QueryAllProviderPaymentStorageRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryAllProviderPaymentStorageRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryAllProviderPaymentStorageRequest;
  static deserializeBinaryFromReader(message: QueryAllProviderPaymentStorageRequest, reader: jspb.BinaryReader): QueryAllProviderPaymentStorageRequest;
}

export namespace QueryAllProviderPaymentStorageRequest {
  export type AsObject = {
    pagination?: cosmos_base_query_v1beta1_pagination_pb.PageRequest.AsObject,
  }
}

export class QueryAllProviderPaymentStorageResponse extends jspb.Message {
  clearProviderpaymentstorageList(): void;
  getProviderpaymentstorageList(): Array<lavanet_lava_pairing_provider_payment_storage_pb.ProviderPaymentStorage>;
  setProviderpaymentstorageList(value: Array<lavanet_lava_pairing_provider_payment_storage_pb.ProviderPaymentStorage>): void;
  addProviderpaymentstorage(value?: lavanet_lava_pairing_provider_payment_storage_pb.ProviderPaymentStorage, index?: number): lavanet_lava_pairing_provider_payment_storage_pb.ProviderPaymentStorage;

  hasPagination(): boolean;
  clearPagination(): void;
  getPagination(): cosmos_base_query_v1beta1_pagination_pb.PageResponse | undefined;
  setPagination(value?: cosmos_base_query_v1beta1_pagination_pb.PageResponse): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryAllProviderPaymentStorageResponse.AsObject;
  static toObject(includeInstance: boolean, msg: QueryAllProviderPaymentStorageResponse): QueryAllProviderPaymentStorageResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryAllProviderPaymentStorageResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryAllProviderPaymentStorageResponse;
  static deserializeBinaryFromReader(message: QueryAllProviderPaymentStorageResponse, reader: jspb.BinaryReader): QueryAllProviderPaymentStorageResponse;
}

export namespace QueryAllProviderPaymentStorageResponse {
  export type AsObject = {
    providerpaymentstorageList: Array<lavanet_lava_pairing_provider_payment_storage_pb.ProviderPaymentStorage.AsObject>,
    pagination?: cosmos_base_query_v1beta1_pagination_pb.PageResponse.AsObject,
  }
}

export class QueryGetEpochPaymentsRequest extends jspb.Message {
  getIndex(): string;
  setIndex(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryGetEpochPaymentsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: QueryGetEpochPaymentsRequest): QueryGetEpochPaymentsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryGetEpochPaymentsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryGetEpochPaymentsRequest;
  static deserializeBinaryFromReader(message: QueryGetEpochPaymentsRequest, reader: jspb.BinaryReader): QueryGetEpochPaymentsRequest;
}

export namespace QueryGetEpochPaymentsRequest {
  export type AsObject = {
    index: string,
  }
}

export class QueryGetEpochPaymentsResponse extends jspb.Message {
  hasEpochpayments(): boolean;
  clearEpochpayments(): void;
  getEpochpayments(): lavanet_lava_pairing_epoch_payments_pb.EpochPayments | undefined;
  setEpochpayments(value?: lavanet_lava_pairing_epoch_payments_pb.EpochPayments): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryGetEpochPaymentsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: QueryGetEpochPaymentsResponse): QueryGetEpochPaymentsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryGetEpochPaymentsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryGetEpochPaymentsResponse;
  static deserializeBinaryFromReader(message: QueryGetEpochPaymentsResponse, reader: jspb.BinaryReader): QueryGetEpochPaymentsResponse;
}

export namespace QueryGetEpochPaymentsResponse {
  export type AsObject = {
    epochpayments?: lavanet_lava_pairing_epoch_payments_pb.EpochPayments.AsObject,
  }
}

export class QueryAllEpochPaymentsRequest extends jspb.Message {
  hasPagination(): boolean;
  clearPagination(): void;
  getPagination(): cosmos_base_query_v1beta1_pagination_pb.PageRequest | undefined;
  setPagination(value?: cosmos_base_query_v1beta1_pagination_pb.PageRequest): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryAllEpochPaymentsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: QueryAllEpochPaymentsRequest): QueryAllEpochPaymentsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryAllEpochPaymentsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryAllEpochPaymentsRequest;
  static deserializeBinaryFromReader(message: QueryAllEpochPaymentsRequest, reader: jspb.BinaryReader): QueryAllEpochPaymentsRequest;
}

export namespace QueryAllEpochPaymentsRequest {
  export type AsObject = {
    pagination?: cosmos_base_query_v1beta1_pagination_pb.PageRequest.AsObject,
  }
}

export class QueryAllEpochPaymentsResponse extends jspb.Message {
  clearEpochpaymentsList(): void;
  getEpochpaymentsList(): Array<lavanet_lava_pairing_epoch_payments_pb.EpochPayments>;
  setEpochpaymentsList(value: Array<lavanet_lava_pairing_epoch_payments_pb.EpochPayments>): void;
  addEpochpayments(value?: lavanet_lava_pairing_epoch_payments_pb.EpochPayments, index?: number): lavanet_lava_pairing_epoch_payments_pb.EpochPayments;

  hasPagination(): boolean;
  clearPagination(): void;
  getPagination(): cosmos_base_query_v1beta1_pagination_pb.PageResponse | undefined;
  setPagination(value?: cosmos_base_query_v1beta1_pagination_pb.PageResponse): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryAllEpochPaymentsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: QueryAllEpochPaymentsResponse): QueryAllEpochPaymentsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryAllEpochPaymentsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryAllEpochPaymentsResponse;
  static deserializeBinaryFromReader(message: QueryAllEpochPaymentsResponse, reader: jspb.BinaryReader): QueryAllEpochPaymentsResponse;
}

export namespace QueryAllEpochPaymentsResponse {
  export type AsObject = {
    epochpaymentsList: Array<lavanet_lava_pairing_epoch_payments_pb.EpochPayments.AsObject>,
    pagination?: cosmos_base_query_v1beta1_pagination_pb.PageResponse.AsObject,
  }
}

export class QueryUserEntryRequest extends jspb.Message {
  getAddress(): string;
  setAddress(value: string): void;

  getChainid(): string;
  setChainid(value: string): void;

  getBlock(): number;
  setBlock(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryUserEntryRequest.AsObject;
  static toObject(includeInstance: boolean, msg: QueryUserEntryRequest): QueryUserEntryRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryUserEntryRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryUserEntryRequest;
  static deserializeBinaryFromReader(message: QueryUserEntryRequest, reader: jspb.BinaryReader): QueryUserEntryRequest;
}

export namespace QueryUserEntryRequest {
  export type AsObject = {
    address: string,
    chainid: string,
    block: number,
  }
}

export class QueryUserEntryResponse extends jspb.Message {
  hasConsumer(): boolean;
  clearConsumer(): void;
  getConsumer(): lavanet_lava_epochstorage_stake_entry_pb.StakeEntry | undefined;
  setConsumer(value?: lavanet_lava_epochstorage_stake_entry_pb.StakeEntry): void;

  getMaxcu(): number;
  setMaxcu(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryUserEntryResponse.AsObject;
  static toObject(includeInstance: boolean, msg: QueryUserEntryResponse): QueryUserEntryResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryUserEntryResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryUserEntryResponse;
  static deserializeBinaryFromReader(message: QueryUserEntryResponse, reader: jspb.BinaryReader): QueryUserEntryResponse;
}

export namespace QueryUserEntryResponse {
  export type AsObject = {
    consumer?: lavanet_lava_epochstorage_stake_entry_pb.StakeEntry.AsObject,
    maxcu: number,
  }
}

export class QueryStaticProvidersListRequest extends jspb.Message {
  getChainid(): string;
  setChainid(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryStaticProvidersListRequest.AsObject;
  static toObject(includeInstance: boolean, msg: QueryStaticProvidersListRequest): QueryStaticProvidersListRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryStaticProvidersListRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryStaticProvidersListRequest;
  static deserializeBinaryFromReader(message: QueryStaticProvidersListRequest, reader: jspb.BinaryReader): QueryStaticProvidersListRequest;
}

export namespace QueryStaticProvidersListRequest {
  export type AsObject = {
    chainid: string,
  }
}

export class QueryStaticProvidersListResponse extends jspb.Message {
  clearProvidersList(): void;
  getProvidersList(): Array<lavanet_lava_epochstorage_stake_entry_pb.StakeEntry>;
  setProvidersList(value: Array<lavanet_lava_epochstorage_stake_entry_pb.StakeEntry>): void;
  addProviders(value?: lavanet_lava_epochstorage_stake_entry_pb.StakeEntry, index?: number): lavanet_lava_epochstorage_stake_entry_pb.StakeEntry;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryStaticProvidersListResponse.AsObject;
  static toObject(includeInstance: boolean, msg: QueryStaticProvidersListResponse): QueryStaticProvidersListResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryStaticProvidersListResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryStaticProvidersListResponse;
  static deserializeBinaryFromReader(message: QueryStaticProvidersListResponse, reader: jspb.BinaryReader): QueryStaticProvidersListResponse;
}

export namespace QueryStaticProvidersListResponse {
  export type AsObject = {
    providersList: Array<lavanet_lava_epochstorage_stake_entry_pb.StakeEntry.AsObject>,
  }
}

export class QueryAccountInfoResponse extends jspb.Message {
  clearProviderList(): void;
  getProviderList(): Array<lavanet_lava_epochstorage_stake_entry_pb.StakeEntry>;
  setProviderList(value: Array<lavanet_lava_epochstorage_stake_entry_pb.StakeEntry>): void;
  addProvider(value?: lavanet_lava_epochstorage_stake_entry_pb.StakeEntry, index?: number): lavanet_lava_epochstorage_stake_entry_pb.StakeEntry;

  clearFrozenList(): void;
  getFrozenList(): Array<lavanet_lava_epochstorage_stake_entry_pb.StakeEntry>;
  setFrozenList(value: Array<lavanet_lava_epochstorage_stake_entry_pb.StakeEntry>): void;
  addFrozen(value?: lavanet_lava_epochstorage_stake_entry_pb.StakeEntry, index?: number): lavanet_lava_epochstorage_stake_entry_pb.StakeEntry;

  clearConsumerList(): void;
  getConsumerList(): Array<lavanet_lava_epochstorage_stake_entry_pb.StakeEntry>;
  setConsumerList(value: Array<lavanet_lava_epochstorage_stake_entry_pb.StakeEntry>): void;
  addConsumer(value?: lavanet_lava_epochstorage_stake_entry_pb.StakeEntry, index?: number): lavanet_lava_epochstorage_stake_entry_pb.StakeEntry;

  clearUnstakedList(): void;
  getUnstakedList(): Array<lavanet_lava_epochstorage_stake_entry_pb.StakeEntry>;
  setUnstakedList(value: Array<lavanet_lava_epochstorage_stake_entry_pb.StakeEntry>): void;
  addUnstaked(value?: lavanet_lava_epochstorage_stake_entry_pb.StakeEntry, index?: number): lavanet_lava_epochstorage_stake_entry_pb.StakeEntry;

  hasSubscription(): boolean;
  clearSubscription(): void;
  getSubscription(): lavanet_lava_subscription_subscription_pb.Subscription | undefined;
  setSubscription(value?: lavanet_lava_subscription_subscription_pb.Subscription): void;

  hasProject(): boolean;
  clearProject(): void;
  getProject(): lavanet_lava_projects_project_pb.Project | undefined;
  setProject(value?: lavanet_lava_projects_project_pb.Project): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryAccountInfoResponse.AsObject;
  static toObject(includeInstance: boolean, msg: QueryAccountInfoResponse): QueryAccountInfoResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryAccountInfoResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryAccountInfoResponse;
  static deserializeBinaryFromReader(message: QueryAccountInfoResponse, reader: jspb.BinaryReader): QueryAccountInfoResponse;
}

export namespace QueryAccountInfoResponse {
  export type AsObject = {
    providerList: Array<lavanet_lava_epochstorage_stake_entry_pb.StakeEntry.AsObject>,
    frozenList: Array<lavanet_lava_epochstorage_stake_entry_pb.StakeEntry.AsObject>,
    consumerList: Array<lavanet_lava_epochstorage_stake_entry_pb.StakeEntry.AsObject>,
    unstakedList: Array<lavanet_lava_epochstorage_stake_entry_pb.StakeEntry.AsObject>,
    subscription?: lavanet_lava_subscription_subscription_pb.Subscription.AsObject,
    project?: lavanet_lava_projects_project_pb.Project.AsObject,
  }
}

export class QueryEffectivePolicyRequest extends jspb.Message {
  getConsumer(): string;
  setConsumer(value: string): void;

  getSpecid(): string;
  setSpecid(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryEffectivePolicyRequest.AsObject;
  static toObject(includeInstance: boolean, msg: QueryEffectivePolicyRequest): QueryEffectivePolicyRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryEffectivePolicyRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryEffectivePolicyRequest;
  static deserializeBinaryFromReader(message: QueryEffectivePolicyRequest, reader: jspb.BinaryReader): QueryEffectivePolicyRequest;
}

export namespace QueryEffectivePolicyRequest {
  export type AsObject = {
    consumer: string,
    specid: string,
  }
}

export class QueryEffectivePolicyResponse extends jspb.Message {
  hasPolicy(): boolean;
  clearPolicy(): void;
  getPolicy(): lavanet_lava_plans_policy_pb.Policy | undefined;
  setPolicy(value?: lavanet_lava_plans_policy_pb.Policy): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryEffectivePolicyResponse.AsObject;
  static toObject(includeInstance: boolean, msg: QueryEffectivePolicyResponse): QueryEffectivePolicyResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryEffectivePolicyResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryEffectivePolicyResponse;
  static deserializeBinaryFromReader(message: QueryEffectivePolicyResponse, reader: jspb.BinaryReader): QueryEffectivePolicyResponse;
}

export namespace QueryEffectivePolicyResponse {
  export type AsObject = {
    policy?: lavanet_lava_plans_policy_pb.Policy.AsObject,
  }
}

export class QuerySdkPairingResponse extends jspb.Message {
  hasPairing(): boolean;
  clearPairing(): void;
  getPairing(): QueryGetPairingResponse | undefined;
  setPairing(value?: QueryGetPairingResponse): void;

  getMaxCu(): number;
  setMaxCu(value: number): void;

  hasSpec(): boolean;
  clearSpec(): void;
  getSpec(): lavanet_lava_spec_spec_pb.Spec | undefined;
  setSpec(value?: lavanet_lava_spec_spec_pb.Spec): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QuerySdkPairingResponse.AsObject;
  static toObject(includeInstance: boolean, msg: QuerySdkPairingResponse): QuerySdkPairingResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QuerySdkPairingResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QuerySdkPairingResponse;
  static deserializeBinaryFromReader(message: QuerySdkPairingResponse, reader: jspb.BinaryReader): QuerySdkPairingResponse;
}

export namespace QuerySdkPairingResponse {
  export type AsObject = {
    pairing?: QueryGetPairingResponse.AsObject,
    maxCu: number,
    spec?: lavanet_lava_spec_spec_pb.Spec.AsObject,
  }
}


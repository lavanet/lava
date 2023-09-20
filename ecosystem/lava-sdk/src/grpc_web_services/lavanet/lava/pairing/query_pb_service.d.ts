// package: lavanet.lava.pairing
// file: lavanet/lava/pairing/query.proto

import * as lavanet_lava_pairing_query_pb from "../../../lavanet/lava/pairing/query_pb";
import {grpc} from "@improbable-eng/grpc-web";

type QueryParams = {
  readonly methodName: string;
  readonly service: typeof Query;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof lavanet_lava_pairing_query_pb.QueryParamsRequest;
  readonly responseType: typeof lavanet_lava_pairing_query_pb.QueryParamsResponse;
};

type QueryProviders = {
  readonly methodName: string;
  readonly service: typeof Query;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof lavanet_lava_pairing_query_pb.QueryProvidersRequest;
  readonly responseType: typeof lavanet_lava_pairing_query_pb.QueryProvidersResponse;
};

type QueryGetPairing = {
  readonly methodName: string;
  readonly service: typeof Query;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof lavanet_lava_pairing_query_pb.QueryGetPairingRequest;
  readonly responseType: typeof lavanet_lava_pairing_query_pb.QueryGetPairingResponse;
};

type QueryVerifyPairing = {
  readonly methodName: string;
  readonly service: typeof Query;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof lavanet_lava_pairing_query_pb.QueryVerifyPairingRequest;
  readonly responseType: typeof lavanet_lava_pairing_query_pb.QueryVerifyPairingResponse;
};

type QueryUniquePaymentStorageClientProvider = {
  readonly methodName: string;
  readonly service: typeof Query;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof lavanet_lava_pairing_query_pb.QueryGetUniquePaymentStorageClientProviderRequest;
  readonly responseType: typeof lavanet_lava_pairing_query_pb.QueryGetUniquePaymentStorageClientProviderResponse;
};

type QueryUniquePaymentStorageClientProviderAll = {
  readonly methodName: string;
  readonly service: typeof Query;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof lavanet_lava_pairing_query_pb.QueryAllUniquePaymentStorageClientProviderRequest;
  readonly responseType: typeof lavanet_lava_pairing_query_pb.QueryAllUniquePaymentStorageClientProviderResponse;
};

type QueryProviderPaymentStorage = {
  readonly methodName: string;
  readonly service: typeof Query;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof lavanet_lava_pairing_query_pb.QueryGetProviderPaymentStorageRequest;
  readonly responseType: typeof lavanet_lava_pairing_query_pb.QueryGetProviderPaymentStorageResponse;
};

type QueryProviderPaymentStorageAll = {
  readonly methodName: string;
  readonly service: typeof Query;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof lavanet_lava_pairing_query_pb.QueryAllProviderPaymentStorageRequest;
  readonly responseType: typeof lavanet_lava_pairing_query_pb.QueryAllProviderPaymentStorageResponse;
};

type QueryEpochPayments = {
  readonly methodName: string;
  readonly service: typeof Query;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof lavanet_lava_pairing_query_pb.QueryGetEpochPaymentsRequest;
  readonly responseType: typeof lavanet_lava_pairing_query_pb.QueryGetEpochPaymentsResponse;
};

type QueryEpochPaymentsAll = {
  readonly methodName: string;
  readonly service: typeof Query;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof lavanet_lava_pairing_query_pb.QueryAllEpochPaymentsRequest;
  readonly responseType: typeof lavanet_lava_pairing_query_pb.QueryAllEpochPaymentsResponse;
};

type QueryUserEntry = {
  readonly methodName: string;
  readonly service: typeof Query;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof lavanet_lava_pairing_query_pb.QueryUserEntryRequest;
  readonly responseType: typeof lavanet_lava_pairing_query_pb.QueryUserEntryResponse;
};

type QueryStaticProvidersList = {
  readonly methodName: string;
  readonly service: typeof Query;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof lavanet_lava_pairing_query_pb.QueryStaticProvidersListRequest;
  readonly responseType: typeof lavanet_lava_pairing_query_pb.QueryStaticProvidersListResponse;
};

type QueryEffectivePolicy = {
  readonly methodName: string;
  readonly service: typeof Query;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof lavanet_lava_pairing_query_pb.QueryEffectivePolicyRequest;
  readonly responseType: typeof lavanet_lava_pairing_query_pb.QueryEffectivePolicyResponse;
};

type QuerySdkPairing = {
  readonly methodName: string;
  readonly service: typeof Query;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof lavanet_lava_pairing_query_pb.QueryGetPairingRequest;
  readonly responseType: typeof lavanet_lava_pairing_query_pb.QuerySdkPairingResponse;
};

export class Query {
  static readonly serviceName: string;
  static readonly Params: QueryParams;
  static readonly Providers: QueryProviders;
  static readonly GetPairing: QueryGetPairing;
  static readonly VerifyPairing: QueryVerifyPairing;
  static readonly UniquePaymentStorageClientProvider: QueryUniquePaymentStorageClientProvider;
  static readonly UniquePaymentStorageClientProviderAll: QueryUniquePaymentStorageClientProviderAll;
  static readonly ProviderPaymentStorage: QueryProviderPaymentStorage;
  static readonly ProviderPaymentStorageAll: QueryProviderPaymentStorageAll;
  static readonly EpochPayments: QueryEpochPayments;
  static readonly EpochPaymentsAll: QueryEpochPaymentsAll;
  static readonly UserEntry: QueryUserEntry;
  static readonly StaticProvidersList: QueryStaticProvidersList;
  static readonly EffectivePolicy: QueryEffectivePolicy;
  static readonly SdkPairing: QuerySdkPairing;
}

export type ServiceError = { message: string, code: number; metadata: grpc.Metadata }
export type Status = { details: string, code: number; metadata: grpc.Metadata }

interface UnaryResponse {
  cancel(): void;
}
interface ResponseStream<T> {
  cancel(): void;
  on(type: 'data', handler: (message: T) => void): ResponseStream<T>;
  on(type: 'end', handler: (status?: Status) => void): ResponseStream<T>;
  on(type: 'status', handler: (status: Status) => void): ResponseStream<T>;
}
interface RequestStream<T> {
  write(message: T): RequestStream<T>;
  end(): void;
  cancel(): void;
  on(type: 'end', handler: (status?: Status) => void): RequestStream<T>;
  on(type: 'status', handler: (status: Status) => void): RequestStream<T>;
}
interface BidirectionalStream<ReqT, ResT> {
  write(message: ReqT): BidirectionalStream<ReqT, ResT>;
  end(): void;
  cancel(): void;
  on(type: 'data', handler: (message: ResT) => void): BidirectionalStream<ReqT, ResT>;
  on(type: 'end', handler: (status?: Status) => void): BidirectionalStream<ReqT, ResT>;
  on(type: 'status', handler: (status: Status) => void): BidirectionalStream<ReqT, ResT>;
}

export class QueryClient {
  readonly serviceHost: string;

  constructor(serviceHost: string, options?: grpc.RpcOptions);
  params(
    requestMessage: lavanet_lava_pairing_query_pb.QueryParamsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QueryParamsResponse|null) => void
  ): UnaryResponse;
  params(
    requestMessage: lavanet_lava_pairing_query_pb.QueryParamsRequest,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QueryParamsResponse|null) => void
  ): UnaryResponse;
  providers(
    requestMessage: lavanet_lava_pairing_query_pb.QueryProvidersRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QueryProvidersResponse|null) => void
  ): UnaryResponse;
  providers(
    requestMessage: lavanet_lava_pairing_query_pb.QueryProvidersRequest,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QueryProvidersResponse|null) => void
  ): UnaryResponse;
  getPairing(
    requestMessage: lavanet_lava_pairing_query_pb.QueryGetPairingRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QueryGetPairingResponse|null) => void
  ): UnaryResponse;
  getPairing(
    requestMessage: lavanet_lava_pairing_query_pb.QueryGetPairingRequest,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QueryGetPairingResponse|null) => void
  ): UnaryResponse;
  verifyPairing(
    requestMessage: lavanet_lava_pairing_query_pb.QueryVerifyPairingRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QueryVerifyPairingResponse|null) => void
  ): UnaryResponse;
  verifyPairing(
    requestMessage: lavanet_lava_pairing_query_pb.QueryVerifyPairingRequest,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QueryVerifyPairingResponse|null) => void
  ): UnaryResponse;
  uniquePaymentStorageClientProvider(
    requestMessage: lavanet_lava_pairing_query_pb.QueryGetUniquePaymentStorageClientProviderRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QueryGetUniquePaymentStorageClientProviderResponse|null) => void
  ): UnaryResponse;
  uniquePaymentStorageClientProvider(
    requestMessage: lavanet_lava_pairing_query_pb.QueryGetUniquePaymentStorageClientProviderRequest,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QueryGetUniquePaymentStorageClientProviderResponse|null) => void
  ): UnaryResponse;
  uniquePaymentStorageClientProviderAll(
    requestMessage: lavanet_lava_pairing_query_pb.QueryAllUniquePaymentStorageClientProviderRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QueryAllUniquePaymentStorageClientProviderResponse|null) => void
  ): UnaryResponse;
  uniquePaymentStorageClientProviderAll(
    requestMessage: lavanet_lava_pairing_query_pb.QueryAllUniquePaymentStorageClientProviderRequest,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QueryAllUniquePaymentStorageClientProviderResponse|null) => void
  ): UnaryResponse;
  providerPaymentStorage(
    requestMessage: lavanet_lava_pairing_query_pb.QueryGetProviderPaymentStorageRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QueryGetProviderPaymentStorageResponse|null) => void
  ): UnaryResponse;
  providerPaymentStorage(
    requestMessage: lavanet_lava_pairing_query_pb.QueryGetProviderPaymentStorageRequest,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QueryGetProviderPaymentStorageResponse|null) => void
  ): UnaryResponse;
  providerPaymentStorageAll(
    requestMessage: lavanet_lava_pairing_query_pb.QueryAllProviderPaymentStorageRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QueryAllProviderPaymentStorageResponse|null) => void
  ): UnaryResponse;
  providerPaymentStorageAll(
    requestMessage: lavanet_lava_pairing_query_pb.QueryAllProviderPaymentStorageRequest,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QueryAllProviderPaymentStorageResponse|null) => void
  ): UnaryResponse;
  epochPayments(
    requestMessage: lavanet_lava_pairing_query_pb.QueryGetEpochPaymentsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QueryGetEpochPaymentsResponse|null) => void
  ): UnaryResponse;
  epochPayments(
    requestMessage: lavanet_lava_pairing_query_pb.QueryGetEpochPaymentsRequest,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QueryGetEpochPaymentsResponse|null) => void
  ): UnaryResponse;
  epochPaymentsAll(
    requestMessage: lavanet_lava_pairing_query_pb.QueryAllEpochPaymentsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QueryAllEpochPaymentsResponse|null) => void
  ): UnaryResponse;
  epochPaymentsAll(
    requestMessage: lavanet_lava_pairing_query_pb.QueryAllEpochPaymentsRequest,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QueryAllEpochPaymentsResponse|null) => void
  ): UnaryResponse;
  userEntry(
    requestMessage: lavanet_lava_pairing_query_pb.QueryUserEntryRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QueryUserEntryResponse|null) => void
  ): UnaryResponse;
  userEntry(
    requestMessage: lavanet_lava_pairing_query_pb.QueryUserEntryRequest,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QueryUserEntryResponse|null) => void
  ): UnaryResponse;
  staticProvidersList(
    requestMessage: lavanet_lava_pairing_query_pb.QueryStaticProvidersListRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QueryStaticProvidersListResponse|null) => void
  ): UnaryResponse;
  staticProvidersList(
    requestMessage: lavanet_lava_pairing_query_pb.QueryStaticProvidersListRequest,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QueryStaticProvidersListResponse|null) => void
  ): UnaryResponse;
  effectivePolicy(
    requestMessage: lavanet_lava_pairing_query_pb.QueryEffectivePolicyRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QueryEffectivePolicyResponse|null) => void
  ): UnaryResponse;
  effectivePolicy(
    requestMessage: lavanet_lava_pairing_query_pb.QueryEffectivePolicyRequest,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QueryEffectivePolicyResponse|null) => void
  ): UnaryResponse;
  sdkPairing(
    requestMessage: lavanet_lava_pairing_query_pb.QueryGetPairingRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QuerySdkPairingResponse|null) => void
  ): UnaryResponse;
  sdkPairing(
    requestMessage: lavanet_lava_pairing_query_pb.QueryGetPairingRequest,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_pairing_query_pb.QuerySdkPairingResponse|null) => void
  ): UnaryResponse;
}


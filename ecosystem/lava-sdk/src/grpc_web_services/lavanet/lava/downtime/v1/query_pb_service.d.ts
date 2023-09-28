// package: lavanet.lava.downtime.v1
// file: lavanet/lava/downtime/v1/query.proto

import * as lavanet_lava_downtime_v1_query_pb from "../../../../lavanet/lava/downtime/v1/query_pb";
import {grpc} from "@improbable-eng/grpc-web";

type QueryQueryParams = {
  readonly methodName: string;
  readonly service: typeof Query;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof lavanet_lava_downtime_v1_query_pb.QueryParamsRequest;
  readonly responseType: typeof lavanet_lava_downtime_v1_query_pb.QueryParamsResponse;
};

type QueryQueryDowntime = {
  readonly methodName: string;
  readonly service: typeof Query;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof lavanet_lava_downtime_v1_query_pb.QueryDowntimeRequest;
  readonly responseType: typeof lavanet_lava_downtime_v1_query_pb.QueryDowntimeResponse;
};

export class Query {
  static readonly serviceName: string;
  static readonly QueryParams: QueryQueryParams;
  static readonly QueryDowntime: QueryQueryDowntime;
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
  queryParams(
    requestMessage: lavanet_lava_downtime_v1_query_pb.QueryParamsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_downtime_v1_query_pb.QueryParamsResponse|null) => void
  ): UnaryResponse;
  queryParams(
    requestMessage: lavanet_lava_downtime_v1_query_pb.QueryParamsRequest,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_downtime_v1_query_pb.QueryParamsResponse|null) => void
  ): UnaryResponse;
  queryDowntime(
    requestMessage: lavanet_lava_downtime_v1_query_pb.QueryDowntimeRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_downtime_v1_query_pb.QueryDowntimeResponse|null) => void
  ): UnaryResponse;
  queryDowntime(
    requestMessage: lavanet_lava_downtime_v1_query_pb.QueryDowntimeRequest,
    callback: (error: ServiceError|null, responseMessage: lavanet_lava_downtime_v1_query_pb.QueryDowntimeResponse|null) => void
  ): UnaryResponse;
}


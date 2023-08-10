// package: lava.pairing
// file: lava/pairing/relay.proto

import * as lava_pairing_relay_pb from "../../lava/pairing/relay_pb";
import {grpc} from "@improbable-eng/grpc-web";

type RelayerRelay = {
  readonly methodName: string;
  readonly service: typeof Relayer;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof lava_pairing_relay_pb.RelayRequest;
  readonly responseType: typeof lava_pairing_relay_pb.RelayReply;
};

type RelayerRelaySubscribe = {
  readonly methodName: string;
  readonly service: typeof Relayer;
  readonly requestStream: false;
  readonly responseStream: true;
  readonly requestType: typeof lava_pairing_relay_pb.RelayRequest;
  readonly responseType: typeof lava_pairing_relay_pb.RelayReply;
};

type RelayerProbe = {
  readonly methodName: string;
  readonly service: typeof Relayer;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof lava_pairing_relay_pb.ProbeRequest;
  readonly responseType: typeof lava_pairing_relay_pb.ProbeReply;
};

export class Relayer {
  static readonly serviceName: string;
  static readonly Relay: RelayerRelay;
  static readonly RelaySubscribe: RelayerRelaySubscribe;
  static readonly Probe: RelayerProbe;
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

export class RelayerClient {
  readonly serviceHost: string;

  constructor(serviceHost: string, options?: grpc.RpcOptions);
  relay(
    requestMessage: lava_pairing_relay_pb.RelayRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: lava_pairing_relay_pb.RelayReply|null) => void
  ): UnaryResponse;
  relay(
    requestMessage: lava_pairing_relay_pb.RelayRequest,
    callback: (error: ServiceError|null, responseMessage: lava_pairing_relay_pb.RelayReply|null) => void
  ): UnaryResponse;
  relaySubscribe(requestMessage: lava_pairing_relay_pb.RelayRequest, metadata?: grpc.Metadata): ResponseStream<lava_pairing_relay_pb.RelayReply>;
  probe(
    requestMessage: lava_pairing_relay_pb.ProbeRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: lava_pairing_relay_pb.ProbeReply|null) => void
  ): UnaryResponse;
  probe(
    requestMessage: lava_pairing_relay_pb.ProbeRequest,
    callback: (error: ServiceError|null, responseMessage: lava_pairing_relay_pb.ProbeReply|null) => void
  ): UnaryResponse;
}


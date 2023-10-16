// package: lavanet.lava.downtime.v1
// file: lavanet/lava/downtime/v1/query.proto

import * as jspb from "google-protobuf";
import * as google_protobuf_duration_pb from "google-protobuf/google/protobuf/duration_pb";
import * as lavanet_lava_downtime_v1_downtime_pb from "../../../../lavanet/lava/downtime/v1/downtime_pb";
import * as gogoproto_gogo_pb from "../../../../gogoproto/gogo_pb";
import * as google_api_annotations_pb from "../../../../google/api/annotations_pb";

export class QueryDowntimeRequest extends jspb.Message {
  getEpochStartBlock(): number;
  setEpochStartBlock(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryDowntimeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: QueryDowntimeRequest): QueryDowntimeRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryDowntimeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryDowntimeRequest;
  static deserializeBinaryFromReader(message: QueryDowntimeRequest, reader: jspb.BinaryReader): QueryDowntimeRequest;
}

export namespace QueryDowntimeRequest {
  export type AsObject = {
    epochStartBlock: number,
  }
}

export class QueryDowntimeResponse extends jspb.Message {
  hasCumulativeDowntimeDuration(): boolean;
  clearCumulativeDowntimeDuration(): void;
  getCumulativeDowntimeDuration(): google_protobuf_duration_pb.Duration | undefined;
  setCumulativeDowntimeDuration(value?: google_protobuf_duration_pb.Duration): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryDowntimeResponse.AsObject;
  static toObject(includeInstance: boolean, msg: QueryDowntimeResponse): QueryDowntimeResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryDowntimeResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryDowntimeResponse;
  static deserializeBinaryFromReader(message: QueryDowntimeResponse, reader: jspb.BinaryReader): QueryDowntimeResponse;
}

export namespace QueryDowntimeResponse {
  export type AsObject = {
    cumulativeDowntimeDuration?: google_protobuf_duration_pb.Duration.AsObject,
  }
}

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
  getParams(): lavanet_lava_downtime_v1_downtime_pb.Params | undefined;
  setParams(value?: lavanet_lava_downtime_v1_downtime_pb.Params): void;

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
    params?: lavanet_lava_downtime_v1_downtime_pb.Params.AsObject,
  }
}


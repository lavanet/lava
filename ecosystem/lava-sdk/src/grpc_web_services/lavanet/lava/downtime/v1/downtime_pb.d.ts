// package: lavanet.lava.downtime.v1
// file: lavanet/lava/downtime/v1/downtime.proto

import * as jspb from "google-protobuf";
import * as google_protobuf_duration_pb from "google-protobuf/google/protobuf/duration_pb";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";
import * as gogoproto_gogo_pb from "../../../../gogoproto/gogo_pb";

export class Params extends jspb.Message {
  hasDowntimeDuration(): boolean;
  clearDowntimeDuration(): void;
  getDowntimeDuration(): google_protobuf_duration_pb.Duration | undefined;
  setDowntimeDuration(value?: google_protobuf_duration_pb.Duration): void;

  hasEpochDuration(): boolean;
  clearEpochDuration(): void;
  getEpochDuration(): google_protobuf_duration_pb.Duration | undefined;
  setEpochDuration(value?: google_protobuf_duration_pb.Duration): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Params.AsObject;
  static toObject(includeInstance: boolean, msg: Params): Params.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Params, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Params;
  static deserializeBinaryFromReader(message: Params, reader: jspb.BinaryReader): Params;
}

export namespace Params {
  export type AsObject = {
    downtimeDuration?: google_protobuf_duration_pb.Duration.AsObject,
    epochDuration?: google_protobuf_duration_pb.Duration.AsObject,
  }
}

export class Downtime extends jspb.Message {
  getBlock(): number;
  setBlock(value: number): void;

  hasDuration(): boolean;
  clearDuration(): void;
  getDuration(): google_protobuf_duration_pb.Duration | undefined;
  setDuration(value?: google_protobuf_duration_pb.Duration): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Downtime.AsObject;
  static toObject(includeInstance: boolean, msg: Downtime): Downtime.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Downtime, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Downtime;
  static deserializeBinaryFromReader(message: Downtime, reader: jspb.BinaryReader): Downtime;
}

export namespace Downtime {
  export type AsObject = {
    block: number,
    duration?: google_protobuf_duration_pb.Duration.AsObject,
  }
}


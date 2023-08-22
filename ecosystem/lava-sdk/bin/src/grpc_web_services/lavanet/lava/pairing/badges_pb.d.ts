// package: lavanet.lava.pairing
// file: lavanet/lava/pairing/badges.proto

import * as jspb from "google-protobuf";
import * as lavanet_lava_pairing_relay_pb from "../../../lavanet/lava/pairing/relay_pb";
import * as lavanet_lava_pairing_query_pb from "../../../lavanet/lava/pairing/query_pb";
import * as gogoproto_gogo_pb from "../../../gogoproto/gogo_pb";
import * as google_protobuf_wrappers_pb from "google-protobuf/google/protobuf/wrappers_pb";
import * as lavanet_lava_epochstorage_stake_entry_pb from "../../../lavanet/lava/epochstorage/stake_entry_pb";
import * as lavanet_lava_spec_spec_pb from "../../../lavanet/lava/spec/spec_pb";

export class GenerateBadgeRequest extends jspb.Message {
  getBadgeAddress(): string;
  setBadgeAddress(value: string): void;

  getProjectId(): string;
  setProjectId(value: string): void;

  getSpecId(): string;
  setSpecId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GenerateBadgeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GenerateBadgeRequest): GenerateBadgeRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GenerateBadgeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GenerateBadgeRequest;
  static deserializeBinaryFromReader(message: GenerateBadgeRequest, reader: jspb.BinaryReader): GenerateBadgeRequest;
}

export namespace GenerateBadgeRequest {
  export type AsObject = {
    badgeAddress: string,
    projectId: string,
    specId: string,
  }
}

export class GenerateBadgeResponse extends jspb.Message {
  hasBadge(): boolean;
  clearBadge(): void;
  getBadge(): lavanet_lava_pairing_relay_pb.Badge | undefined;
  setBadge(value?: lavanet_lava_pairing_relay_pb.Badge): void;

  hasGetPairingResponse(): boolean;
  clearGetPairingResponse(): void;
  getGetPairingResponse(): lavanet_lava_pairing_query_pb.QueryGetPairingResponse | undefined;
  setGetPairingResponse(value?: lavanet_lava_pairing_query_pb.QueryGetPairingResponse): void;

  getBadgeSignerAddress(): string;
  setBadgeSignerAddress(value: string): void;

  hasSpec(): boolean;
  clearSpec(): void;
  getSpec(): lavanet_lava_spec_spec_pb.Spec | undefined;
  setSpec(value?: lavanet_lava_spec_spec_pb.Spec): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GenerateBadgeResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GenerateBadgeResponse): GenerateBadgeResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GenerateBadgeResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GenerateBadgeResponse;
  static deserializeBinaryFromReader(message: GenerateBadgeResponse, reader: jspb.BinaryReader): GenerateBadgeResponse;
}

export namespace GenerateBadgeResponse {
  export type AsObject = {
    badge?: lavanet_lava_pairing_relay_pb.Badge.AsObject,
    getPairingResponse?: lavanet_lava_pairing_query_pb.QueryGetPairingResponse.AsObject,
    badgeSignerAddress: string,
    spec?: lavanet_lava_spec_spec_pb.Spec.AsObject,
  }
}


// package: lavanet.lava.conflict
// file: lavanet/lava/conflict/conflict_data.proto

import * as jspb from "google-protobuf";
import * as gogoproto_gogo_pb from "../../../gogoproto/gogo_pb";
import * as lavanet_lava_pairing_relay_pb from "../../../lavanet/lava/pairing/relay_pb";

export class ResponseConflict extends jspb.Message {
  hasConflictrelaydata0(): boolean;
  clearConflictrelaydata0(): void;
  getConflictrelaydata0(): ConflictRelayData | undefined;
  setConflictrelaydata0(value?: ConflictRelayData): void;

  hasConflictrelaydata1(): boolean;
  clearConflictrelaydata1(): void;
  getConflictrelaydata1(): ConflictRelayData | undefined;
  setConflictrelaydata1(value?: ConflictRelayData): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ResponseConflict.AsObject;
  static toObject(includeInstance: boolean, msg: ResponseConflict): ResponseConflict.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ResponseConflict, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ResponseConflict;
  static deserializeBinaryFromReader(message: ResponseConflict, reader: jspb.BinaryReader): ResponseConflict;
}

export namespace ResponseConflict {
  export type AsObject = {
    conflictrelaydata0?: ConflictRelayData.AsObject,
    conflictrelaydata1?: ConflictRelayData.AsObject,
  }
}

export class ConflictRelayData extends jspb.Message {
  hasRequest(): boolean;
  clearRequest(): void;
  getRequest(): lavanet_lava_pairing_relay_pb.RelayRequest | undefined;
  setRequest(value?: lavanet_lava_pairing_relay_pb.RelayRequest): void;

  hasReply(): boolean;
  clearReply(): void;
  getReply(): ReplyMetadata | undefined;
  setReply(value?: ReplyMetadata): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ConflictRelayData.AsObject;
  static toObject(includeInstance: boolean, msg: ConflictRelayData): ConflictRelayData.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ConflictRelayData, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ConflictRelayData;
  static deserializeBinaryFromReader(message: ConflictRelayData, reader: jspb.BinaryReader): ConflictRelayData;
}

export namespace ConflictRelayData {
  export type AsObject = {
    request?: lavanet_lava_pairing_relay_pb.RelayRequest.AsObject,
    reply?: ReplyMetadata.AsObject,
  }
}

export class ReplyMetadata extends jspb.Message {
  getHashAllDataHash(): Uint8Array | string;
  getHashAllDataHash_asU8(): Uint8Array;
  getHashAllDataHash_asB64(): string;
  setHashAllDataHash(value: Uint8Array | string): void;

  getSig(): Uint8Array | string;
  getSig_asU8(): Uint8Array;
  getSig_asB64(): string;
  setSig(value: Uint8Array | string): void;

  getLatestBlock(): number;
  setLatestBlock(value: number): void;

  getFinalizedBlocksHashes(): Uint8Array | string;
  getFinalizedBlocksHashes_asU8(): Uint8Array;
  getFinalizedBlocksHashes_asB64(): string;
  setFinalizedBlocksHashes(value: Uint8Array | string): void;

  getSigBlocks(): Uint8Array | string;
  getSigBlocks_asU8(): Uint8Array;
  getSigBlocks_asB64(): string;
  setSigBlocks(value: Uint8Array | string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ReplyMetadata.AsObject;
  static toObject(includeInstance: boolean, msg: ReplyMetadata): ReplyMetadata.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ReplyMetadata, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ReplyMetadata;
  static deserializeBinaryFromReader(message: ReplyMetadata, reader: jspb.BinaryReader): ReplyMetadata;
}

export namespace ReplyMetadata {
  export type AsObject = {
    hashAllDataHash: Uint8Array | string,
    sig: Uint8Array | string,
    latestBlock: number,
    finalizedBlocksHashes: Uint8Array | string,
    sigBlocks: Uint8Array | string,
  }
}

export class FinalizationConflict extends jspb.Message {
  hasRelayreply0(): boolean;
  clearRelayreply0(): void;
  getRelayreply0(): lavanet_lava_pairing_relay_pb.RelayReply | undefined;
  setRelayreply0(value?: lavanet_lava_pairing_relay_pb.RelayReply): void;

  hasRelayreply1(): boolean;
  clearRelayreply1(): void;
  getRelayreply1(): lavanet_lava_pairing_relay_pb.RelayReply | undefined;
  setRelayreply1(value?: lavanet_lava_pairing_relay_pb.RelayReply): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): FinalizationConflict.AsObject;
  static toObject(includeInstance: boolean, msg: FinalizationConflict): FinalizationConflict.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: FinalizationConflict, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): FinalizationConflict;
  static deserializeBinaryFromReader(message: FinalizationConflict, reader: jspb.BinaryReader): FinalizationConflict;
}

export namespace FinalizationConflict {
  export type AsObject = {
    relayreply0?: lavanet_lava_pairing_relay_pb.RelayReply.AsObject,
    relayreply1?: lavanet_lava_pairing_relay_pb.RelayReply.AsObject,
  }
}


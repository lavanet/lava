// package: lavanet.lava.pairing
// file: lavanet/lava/pairing/relay.proto

import * as jspb from "google-protobuf";
import * as gogoproto_gogo_pb from "../../../gogoproto/gogo_pb";
import * as google_protobuf_wrappers_pb from "google-protobuf/google/protobuf/wrappers_pb";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";

export class ProbeRequest extends jspb.Message {
  getGuid(): number;
  setGuid(value: number): void;

  getSpecId(): string;
  setSpecId(value: string): void;

  getApiInterface(): string;
  setApiInterface(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ProbeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ProbeRequest): ProbeRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ProbeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ProbeRequest;
  static deserializeBinaryFromReader(message: ProbeRequest, reader: jspb.BinaryReader): ProbeRequest;
}

export namespace ProbeRequest {
  export type AsObject = {
    guid: number,
    specId: string,
    apiInterface: string,
  }
}

export class ProbeReply extends jspb.Message {
  getGuid(): number;
  setGuid(value: number): void;

  getLatestBlock(): number;
  setLatestBlock(value: number): void;

  getFinalizedBlocksHashes(): Uint8Array | string;
  getFinalizedBlocksHashes_asU8(): Uint8Array;
  getFinalizedBlocksHashes_asB64(): string;
  setFinalizedBlocksHashes(value: Uint8Array | string): void;

  getLavaEpoch(): number;
  setLavaEpoch(value: number): void;

  getLavaLatestBlock(): number;
  setLavaLatestBlock(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ProbeReply.AsObject;
  static toObject(includeInstance: boolean, msg: ProbeReply): ProbeReply.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ProbeReply, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ProbeReply;
  static deserializeBinaryFromReader(message: ProbeReply, reader: jspb.BinaryReader): ProbeReply;
}

export namespace ProbeReply {
  export type AsObject = {
    guid: number,
    latestBlock: number,
    finalizedBlocksHashes: Uint8Array | string,
    lavaEpoch: number,
    lavaLatestBlock: number,
  }
}

export class RelaySession extends jspb.Message {
  getSpecId(): string;
  setSpecId(value: string): void;

  getContentHash(): Uint8Array | string;
  getContentHash_asU8(): Uint8Array;
  getContentHash_asB64(): string;
  setContentHash(value: Uint8Array | string): void;

  getSessionId(): number;
  setSessionId(value: number): void;

  getCuSum(): number;
  setCuSum(value: number): void;

  getProvider(): string;
  setProvider(value: string): void;

  getRelayNum(): number;
  setRelayNum(value: number): void;

  hasQosReport(): boolean;
  clearQosReport(): void;
  getQosReport(): QualityOfServiceReport | undefined;
  setQosReport(value?: QualityOfServiceReport): void;

  getEpoch(): number;
  setEpoch(value: number): void;

  clearUnresponsiveProvidersList(): void;
  getUnresponsiveProvidersList(): Array<ReportedProvider>;
  setUnresponsiveProvidersList(value: Array<ReportedProvider>): void;
  addUnresponsiveProviders(value?: ReportedProvider, index?: number): ReportedProvider;

  getLavaChainId(): string;
  setLavaChainId(value: string): void;

  getSig(): Uint8Array | string;
  getSig_asU8(): Uint8Array;
  getSig_asB64(): string;
  setSig(value: Uint8Array | string): void;

  hasBadge(): boolean;
  clearBadge(): void;
  getBadge(): Badge | undefined;
  setBadge(value?: Badge): void;

  hasQosExcellenceReport(): boolean;
  clearQosExcellenceReport(): void;
  getQosExcellenceReport(): QualityOfServiceReport | undefined;
  setQosExcellenceReport(value?: QualityOfServiceReport): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RelaySession.AsObject;
  static toObject(includeInstance: boolean, msg: RelaySession): RelaySession.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RelaySession, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RelaySession;
  static deserializeBinaryFromReader(message: RelaySession, reader: jspb.BinaryReader): RelaySession;
}

export namespace RelaySession {
  export type AsObject = {
    specId: string,
    contentHash: Uint8Array | string,
    sessionId: number,
    cuSum: number,
    provider: string,
    relayNum: number,
    qosReport?: QualityOfServiceReport.AsObject,
    epoch: number,
    unresponsiveProvidersList: Array<ReportedProvider.AsObject>,
    lavaChainId: string,
    sig: Uint8Array | string,
    badge?: Badge.AsObject,
    qosExcellenceReport?: QualityOfServiceReport.AsObject,
  }
}

export class Badge extends jspb.Message {
  getCuAllocation(): number;
  setCuAllocation(value: number): void;

  getEpoch(): number;
  setEpoch(value: number): void;

  getAddress(): string;
  setAddress(value: string): void;

  getLavaChainId(): string;
  setLavaChainId(value: string): void;

  getProjectSig(): Uint8Array | string;
  getProjectSig_asU8(): Uint8Array;
  getProjectSig_asB64(): string;
  setProjectSig(value: Uint8Array | string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Badge.AsObject;
  static toObject(includeInstance: boolean, msg: Badge): Badge.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Badge, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Badge;
  static deserializeBinaryFromReader(message: Badge, reader: jspb.BinaryReader): Badge;
}

export namespace Badge {
  export type AsObject = {
    cuAllocation: number,
    epoch: number,
    address: string,
    lavaChainId: string,
    projectSig: Uint8Array | string,
  }
}

export class RelayPrivateData extends jspb.Message {
  getConnectionType(): string;
  setConnectionType(value: string): void;

  getApiUrl(): string;
  setApiUrl(value: string): void;

  getData(): Uint8Array | string;
  getData_asU8(): Uint8Array;
  getData_asB64(): string;
  setData(value: Uint8Array | string): void;

  getRequestBlock(): number;
  setRequestBlock(value: number): void;

  getApiInterface(): string;
  setApiInterface(value: string): void;

  getSalt(): Uint8Array | string;
  getSalt_asU8(): Uint8Array;
  getSalt_asB64(): string;
  setSalt(value: Uint8Array | string): void;

  clearMetadataList(): void;
  getMetadataList(): Array<Metadata>;
  setMetadataList(value: Array<Metadata>): void;
  addMetadata(value?: Metadata, index?: number): Metadata;

  getAddon(): string;
  setAddon(value: string): void;

  clearExtensionsList(): void;
  getExtensionsList(): Array<string>;
  setExtensionsList(value: Array<string>): void;
  addExtensions(value: string, index?: number): string;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RelayPrivateData.AsObject;
  static toObject(includeInstance: boolean, msg: RelayPrivateData): RelayPrivateData.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RelayPrivateData, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RelayPrivateData;
  static deserializeBinaryFromReader(message: RelayPrivateData, reader: jspb.BinaryReader): RelayPrivateData;
}

export namespace RelayPrivateData {
  export type AsObject = {
    connectionType: string,
    apiUrl: string,
    data: Uint8Array | string,
    requestBlock: number,
    apiInterface: string,
    salt: Uint8Array | string,
    metadataList: Array<Metadata.AsObject>,
    addon: string,
    extensionsList: Array<string>,
  }
}

export class ReportedProvider extends jspb.Message {
  getAddress(): string;
  setAddress(value: string): void;

  getDisconnections(): number;
  setDisconnections(value: number): void;

  getErrors(): number;
  setErrors(value: number): void;

  getTimestampS(): number;
  setTimestampS(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ReportedProvider.AsObject;
  static toObject(includeInstance: boolean, msg: ReportedProvider): ReportedProvider.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ReportedProvider, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ReportedProvider;
  static deserializeBinaryFromReader(message: ReportedProvider, reader: jspb.BinaryReader): ReportedProvider;
}

export namespace ReportedProvider {
  export type AsObject = {
    address: string,
    disconnections: number,
    errors: number,
    timestampS: number,
  }
}

export class Metadata extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  getValue(): string;
  setValue(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Metadata.AsObject;
  static toObject(includeInstance: boolean, msg: Metadata): Metadata.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Metadata, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Metadata;
  static deserializeBinaryFromReader(message: Metadata, reader: jspb.BinaryReader): Metadata;
}

export namespace Metadata {
  export type AsObject = {
    name: string,
    value: string,
  }
}

export class RelayRequest extends jspb.Message {
  hasRelaySession(): boolean;
  clearRelaySession(): void;
  getRelaySession(): RelaySession | undefined;
  setRelaySession(value?: RelaySession): void;

  hasRelayData(): boolean;
  clearRelayData(): void;
  getRelayData(): RelayPrivateData | undefined;
  setRelayData(value?: RelayPrivateData): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RelayRequest.AsObject;
  static toObject(includeInstance: boolean, msg: RelayRequest): RelayRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RelayRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RelayRequest;
  static deserializeBinaryFromReader(message: RelayRequest, reader: jspb.BinaryReader): RelayRequest;
}

export namespace RelayRequest {
  export type AsObject = {
    relaySession?: RelaySession.AsObject,
    relayData?: RelayPrivateData.AsObject,
  }
}

export class RelayReply extends jspb.Message {
  getData(): Uint8Array | string;
  getData_asU8(): Uint8Array;
  getData_asB64(): string;
  setData(value: Uint8Array | string): void;

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

  clearMetadataList(): void;
  getMetadataList(): Array<Metadata>;
  setMetadataList(value: Array<Metadata>): void;
  addMetadata(value?: Metadata, index?: number): Metadata;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RelayReply.AsObject;
  static toObject(includeInstance: boolean, msg: RelayReply): RelayReply.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RelayReply, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RelayReply;
  static deserializeBinaryFromReader(message: RelayReply, reader: jspb.BinaryReader): RelayReply;
}

export namespace RelayReply {
  export type AsObject = {
    data: Uint8Array | string,
    sig: Uint8Array | string,
    latestBlock: number,
    finalizedBlocksHashes: Uint8Array | string,
    sigBlocks: Uint8Array | string,
    metadataList: Array<Metadata.AsObject>,
  }
}

export class QualityOfServiceReport extends jspb.Message {
  getLatency(): string;
  setLatency(value: string): void;

  getAvailability(): string;
  setAvailability(value: string): void;

  getSync(): string;
  setSync(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QualityOfServiceReport.AsObject;
  static toObject(includeInstance: boolean, msg: QualityOfServiceReport): QualityOfServiceReport.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QualityOfServiceReport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QualityOfServiceReport;
  static deserializeBinaryFromReader(message: QualityOfServiceReport, reader: jspb.BinaryReader): QualityOfServiceReport;
}

export namespace QualityOfServiceReport {
  export type AsObject = {
    latency: string,
    availability: string,
    sync: string,
  }
}


// package: lavanet.lava.epochstorage
// file: epochstorage/endpoint.proto

import * as jspb from "google-protobuf";

export class Endpoint extends jspb.Message {
  getIpport(): string;
  setIpport(value: string): void;

  getUsetype(): string;
  setUsetype(value: string): void;

  getGeolocation(): number;
  setGeolocation(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Endpoint.AsObject;
  static toObject(includeInstance: boolean, msg: Endpoint): Endpoint.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Endpoint, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Endpoint;
  static deserializeBinaryFromReader(message: Endpoint, reader: jspb.BinaryReader): Endpoint;
}

export namespace Endpoint {
  export type AsObject = {
    ipport: string,
    usetype: string,
    geolocation: number,
  }
}


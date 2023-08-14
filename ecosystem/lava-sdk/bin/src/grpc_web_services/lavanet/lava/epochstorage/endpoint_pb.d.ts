// package: lavanet.lava.epochstorage
// file: lavanet/lava/epochstorage/endpoint.proto

import * as jspb from "google-protobuf";

export class Endpoint extends jspb.Message {
  getIpport(): string;
  setIpport(value: string): void;

  getGeolocation(): number;
  setGeolocation(value: number): void;

  clearAddonsList(): void;
  getAddonsList(): Array<string>;
  setAddonsList(value: Array<string>): void;
  addAddons(value: string, index?: number): string;

  clearApiInterfacesList(): void;
  getApiInterfacesList(): Array<string>;
  setApiInterfacesList(value: Array<string>): void;
  addApiInterfaces(value: string, index?: number): string;

  clearExtensionsList(): void;
  getExtensionsList(): Array<string>;
  setExtensionsList(value: Array<string>): void;
  addExtensions(value: string, index?: number): string;

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
    geolocation: number,
    addonsList: Array<string>,
    apiInterfacesList: Array<string>,
    extensionsList: Array<string>,
  }
}


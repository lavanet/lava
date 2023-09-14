// package: lavanet.lava.pairing
// file: lavanet/lava/pairing/unique_payment_storage_client_provider.proto

import * as jspb from "google-protobuf";

export class UniquePaymentStorageClientProvider extends jspb.Message {
  getIndex(): string;
  setIndex(value: string): void;

  getBlock(): number;
  setBlock(value: number): void;

  getUsedcu(): number;
  setUsedcu(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UniquePaymentStorageClientProvider.AsObject;
  static toObject(includeInstance: boolean, msg: UniquePaymentStorageClientProvider): UniquePaymentStorageClientProvider.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: UniquePaymentStorageClientProvider, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UniquePaymentStorageClientProvider;
  static deserializeBinaryFromReader(message: UniquePaymentStorageClientProvider, reader: jspb.BinaryReader): UniquePaymentStorageClientProvider;
}

export namespace UniquePaymentStorageClientProvider {
  export type AsObject = {
    index: string,
    block: number,
    usedcu: number,
  }
}


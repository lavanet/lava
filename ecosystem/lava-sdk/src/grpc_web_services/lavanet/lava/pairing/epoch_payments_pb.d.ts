// package: lavanet.lava.pairing
// file: lavanet/lava/pairing/epoch_payments.proto

import * as jspb from "google-protobuf";

export class EpochPayments extends jspb.Message {
  getIndex(): string;
  setIndex(value: string): void;

  clearProviderpaymentstoragekeysList(): void;
  getProviderpaymentstoragekeysList(): Array<string>;
  setProviderpaymentstoragekeysList(value: Array<string>): void;
  addProviderpaymentstoragekeys(value: string, index?: number): string;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): EpochPayments.AsObject;
  static toObject(includeInstance: boolean, msg: EpochPayments): EpochPayments.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: EpochPayments, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): EpochPayments;
  static deserializeBinaryFromReader(message: EpochPayments, reader: jspb.BinaryReader): EpochPayments;
}

export namespace EpochPayments {
  export type AsObject = {
    index: string,
    providerpaymentstoragekeysList: Array<string>,
  }
}


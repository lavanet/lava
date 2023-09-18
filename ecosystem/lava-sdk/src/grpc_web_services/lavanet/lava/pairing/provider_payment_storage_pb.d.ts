// package: lavanet.lava.pairing
// file: lavanet/lava/pairing/provider_payment_storage.proto

import * as jspb from "google-protobuf";
import * as lavanet_lava_pairing_unique_payment_storage_client_provider_pb from "../../../lavanet/lava/pairing/unique_payment_storage_client_provider_pb";

export class ProviderPaymentStorage extends jspb.Message {
  getIndex(): string;
  setIndex(value: string): void;

  getEpoch(): number;
  setEpoch(value: number): void;

  clearUniquepaymentstorageclientproviderkeysList(): void;
  getUniquepaymentstorageclientproviderkeysList(): Array<string>;
  setUniquepaymentstorageclientproviderkeysList(value: Array<string>): void;
  addUniquepaymentstorageclientproviderkeys(value: string, index?: number): string;

  getComplainerstotalcu(): number;
  setComplainerstotalcu(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ProviderPaymentStorage.AsObject;
  static toObject(includeInstance: boolean, msg: ProviderPaymentStorage): ProviderPaymentStorage.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ProviderPaymentStorage, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ProviderPaymentStorage;
  static deserializeBinaryFromReader(message: ProviderPaymentStorage, reader: jspb.BinaryReader): ProviderPaymentStorage;
}

export namespace ProviderPaymentStorage {
  export type AsObject = {
    index: string,
    epoch: number,
    uniquepaymentstorageclientproviderkeysList: Array<string>,
    complainerstotalcu: number,
  }
}


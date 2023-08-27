// package: cosmos.base.v1beta1
// file: cosmos/base/v1beta1/coin.proto

import * as jspb from "google-protobuf";
import * as gogoproto_gogo_pb from "../../../gogoproto/gogo_pb";
import * as cosmos_proto_cosmos_pb from "../../../cosmos_proto/cosmos_pb";
import * as amino_amino_pb from "../../../amino/amino_pb";

export class Coin extends jspb.Message {
  getDenom(): string;
  setDenom(value: string): void;

  getAmount(): string;
  setAmount(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Coin.AsObject;
  static toObject(includeInstance: boolean, msg: Coin): Coin.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Coin, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Coin;
  static deserializeBinaryFromReader(message: Coin, reader: jspb.BinaryReader): Coin;
}

export namespace Coin {
  export type AsObject = {
    denom: string,
    amount: string,
  }
}

export class DecCoin extends jspb.Message {
  getDenom(): string;
  setDenom(value: string): void;

  getAmount(): string;
  setAmount(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DecCoin.AsObject;
  static toObject(includeInstance: boolean, msg: DecCoin): DecCoin.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DecCoin, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DecCoin;
  static deserializeBinaryFromReader(message: DecCoin, reader: jspb.BinaryReader): DecCoin;
}

export namespace DecCoin {
  export type AsObject = {
    denom: string,
    amount: string,
  }
}

export class IntProto extends jspb.Message {
  getInt(): string;
  setInt(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): IntProto.AsObject;
  static toObject(includeInstance: boolean, msg: IntProto): IntProto.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: IntProto, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): IntProto;
  static deserializeBinaryFromReader(message: IntProto, reader: jspb.BinaryReader): IntProto;
}

export namespace IntProto {
  export type AsObject = {
    pb_int: string,
  }
}

export class DecProto extends jspb.Message {
  getDec(): string;
  setDec(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DecProto.AsObject;
  static toObject(includeInstance: boolean, msg: DecProto): DecProto.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DecProto, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DecProto;
  static deserializeBinaryFromReader(message: DecProto, reader: jspb.BinaryReader): DecProto;
}

export namespace DecProto {
  export type AsObject = {
    dec: string,
  }
}


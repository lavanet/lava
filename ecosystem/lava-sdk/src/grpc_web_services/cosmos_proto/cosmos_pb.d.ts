// package: cosmos_proto
// file: cosmos_proto/cosmos.proto

import * as jspb from "google-protobuf";
import * as google_protobuf_descriptor_pb from "google-protobuf/google/protobuf/descriptor_pb";

export class InterfaceDescriptor extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): InterfaceDescriptor.AsObject;
  static toObject(includeInstance: boolean, msg: InterfaceDescriptor): InterfaceDescriptor.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: InterfaceDescriptor, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): InterfaceDescriptor;
  static deserializeBinaryFromReader(message: InterfaceDescriptor, reader: jspb.BinaryReader): InterfaceDescriptor;
}

export namespace InterfaceDescriptor {
  export type AsObject = {
    name: string,
    description: string,
  }
}

export class ScalarDescriptor extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  clearFieldTypeList(): void;
  getFieldTypeList(): Array<ScalarTypeMap[keyof ScalarTypeMap]>;
  setFieldTypeList(value: Array<ScalarTypeMap[keyof ScalarTypeMap]>): void;
  addFieldType(value: ScalarTypeMap[keyof ScalarTypeMap], index?: number): ScalarTypeMap[keyof ScalarTypeMap];

  getLegacyAminoEncoding(): string;
  setLegacyAminoEncoding(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ScalarDescriptor.AsObject;
  static toObject(includeInstance: boolean, msg: ScalarDescriptor): ScalarDescriptor.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ScalarDescriptor, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ScalarDescriptor;
  static deserializeBinaryFromReader(message: ScalarDescriptor, reader: jspb.BinaryReader): ScalarDescriptor;
}

export namespace ScalarDescriptor {
  export type AsObject = {
    name: string,
    description: string,
    fieldTypeList: Array<ScalarTypeMap[keyof ScalarTypeMap]>,
    legacyAminoEncoding: string,
  }
}

  export const implementsInterface: jspb.ExtensionFieldInfo<string>;

  export const acceptsInterface: jspb.ExtensionFieldInfo<string>;

  export const scalar: jspb.ExtensionFieldInfo<string>;

  export const declareInterface: jspb.ExtensionFieldInfo<InterfaceDescriptor>;

  export const declareScalar: jspb.ExtensionFieldInfo<ScalarDescriptor>;

export interface ScalarTypeMap {
  SCALAR_TYPE_UNSPECIFIED: 0;
  SCALAR_TYPE_STRING: 1;
  SCALAR_TYPE_BYTES: 2;
}

export const ScalarType: ScalarTypeMap;


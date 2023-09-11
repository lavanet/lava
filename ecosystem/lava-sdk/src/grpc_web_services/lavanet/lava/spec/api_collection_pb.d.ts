// package: lavanet.lava.spec
// file: lavanet/lava/spec/api_collection.proto

import * as jspb from "google-protobuf";
import * as gogoproto_gogo_pb from "../../../gogoproto/gogo_pb";

export class ApiCollection extends jspb.Message {
  getEnabled(): boolean;
  setEnabled(value: boolean): void;

  hasCollectionData(): boolean;
  clearCollectionData(): void;
  getCollectionData(): CollectionData | undefined;
  setCollectionData(value?: CollectionData): void;

  clearApisList(): void;
  getApisList(): Array<Api>;
  setApisList(value: Array<Api>): void;
  addApis(value?: Api, index?: number): Api;

  clearHeadersList(): void;
  getHeadersList(): Array<Header>;
  setHeadersList(value: Array<Header>): void;
  addHeaders(value?: Header, index?: number): Header;

  clearInheritanceApisList(): void;
  getInheritanceApisList(): Array<CollectionData>;
  setInheritanceApisList(value: Array<CollectionData>): void;
  addInheritanceApis(value?: CollectionData, index?: number): CollectionData;

  clearParseDirectivesList(): void;
  getParseDirectivesList(): Array<ParseDirective>;
  setParseDirectivesList(value: Array<ParseDirective>): void;
  addParseDirectives(value?: ParseDirective, index?: number): ParseDirective;

  clearExtensionsList(): void;
  getExtensionsList(): Array<Extension>;
  setExtensionsList(value: Array<Extension>): void;
  addExtensions(value?: Extension, index?: number): Extension;

  clearVerificationsList(): void;
  getVerificationsList(): Array<Verification>;
  setVerificationsList(value: Array<Verification>): void;
  addVerifications(value?: Verification, index?: number): Verification;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ApiCollection.AsObject;
  static toObject(includeInstance: boolean, msg: ApiCollection): ApiCollection.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ApiCollection, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ApiCollection;
  static deserializeBinaryFromReader(message: ApiCollection, reader: jspb.BinaryReader): ApiCollection;
}

export namespace ApiCollection {
  export type AsObject = {
    enabled: boolean,
    collectionData?: CollectionData.AsObject,
    apisList: Array<Api.AsObject>,
    headersList: Array<Header.AsObject>,
    inheritanceApisList: Array<CollectionData.AsObject>,
    parseDirectivesList: Array<ParseDirective.AsObject>,
    extensionsList: Array<Extension.AsObject>,
    verificationsList: Array<Verification.AsObject>,
  }
}

export class Extension extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  getCuMultiplier(): number;
  setCuMultiplier(value: number): void;

  hasRule(): boolean;
  clearRule(): void;
  getRule(): Rule | undefined;
  setRule(value?: Rule): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Extension.AsObject;
  static toObject(includeInstance: boolean, msg: Extension): Extension.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Extension, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Extension;
  static deserializeBinaryFromReader(message: Extension, reader: jspb.BinaryReader): Extension;
}

export namespace Extension {
  export type AsObject = {
    name: string,
    cuMultiplier: number,
    rule?: Rule.AsObject,
  }
}

export class Rule extends jspb.Message {
  getBlock(): number;
  setBlock(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Rule.AsObject;
  static toObject(includeInstance: boolean, msg: Rule): Rule.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Rule, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Rule;
  static deserializeBinaryFromReader(message: Rule, reader: jspb.BinaryReader): Rule;
}

export namespace Rule {
  export type AsObject = {
    block: number,
  }
}

export class Verification extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  hasParseDirective(): boolean;
  clearParseDirective(): void;
  getParseDirective(): ParseDirective | undefined;
  setParseDirective(value?: ParseDirective): void;

  clearValuesList(): void;
  getValuesList(): Array<ParseValue>;
  setValuesList(value: Array<ParseValue>): void;
  addValues(value?: ParseValue, index?: number): ParseValue;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Verification.AsObject;
  static toObject(includeInstance: boolean, msg: Verification): Verification.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Verification, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Verification;
  static deserializeBinaryFromReader(message: Verification, reader: jspb.BinaryReader): Verification;
}

export namespace Verification {
  export type AsObject = {
    name: string,
    parseDirective?: ParseDirective.AsObject,
    valuesList: Array<ParseValue.AsObject>,
  }
}

export class ParseValue extends jspb.Message {
  getExtension$(): string;
  setExtension$(value: string): void;

  getExpectedValue(): string;
  setExpectedValue(value: string): void;

  getLatestDistance(): number;
  setLatestDistance(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ParseValue.AsObject;
  static toObject(includeInstance: boolean, msg: ParseValue): ParseValue.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ParseValue, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ParseValue;
  static deserializeBinaryFromReader(message: ParseValue, reader: jspb.BinaryReader): ParseValue;
}

export namespace ParseValue {
  export type AsObject = {
    extension: string,
    expectedValue: string,
    latestDistance: number,
  }
}

export class CollectionData extends jspb.Message {
  getApiInterface(): string;
  setApiInterface(value: string): void;

  getInternalPath(): string;
  setInternalPath(value: string): void;

  getType(): string;
  setType(value: string): void;

  getAddOn(): string;
  setAddOn(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CollectionData.AsObject;
  static toObject(includeInstance: boolean, msg: CollectionData): CollectionData.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: CollectionData, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CollectionData;
  static deserializeBinaryFromReader(message: CollectionData, reader: jspb.BinaryReader): CollectionData;
}

export namespace CollectionData {
  export type AsObject = {
    apiInterface: string,
    internalPath: string,
    type: string,
    addOn: string,
  }
}

export class Header extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  getKind(): Header.HeaderTypeMap[keyof Header.HeaderTypeMap];
  setKind(value: Header.HeaderTypeMap[keyof Header.HeaderTypeMap]): void;

  getFunctionTag(): FUNCTION_TAGMap[keyof FUNCTION_TAGMap];
  setFunctionTag(value: FUNCTION_TAGMap[keyof FUNCTION_TAGMap]): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Header.AsObject;
  static toObject(includeInstance: boolean, msg: Header): Header.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Header, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Header;
  static deserializeBinaryFromReader(message: Header, reader: jspb.BinaryReader): Header;
}

export namespace Header {
  export type AsObject = {
    name: string,
    kind: Header.HeaderTypeMap[keyof Header.HeaderTypeMap],
    functionTag: FUNCTION_TAGMap[keyof FUNCTION_TAGMap],
  }

  export interface HeaderTypeMap {
    PASS_SEND: 0;
    PASS_REPLY: 1;
    PASS_BOTH: 2;
    PASS_IGNORE: 3;
  }

  export const HeaderType: HeaderTypeMap;
}

export class Api extends jspb.Message {
  getEnabled(): boolean;
  setEnabled(value: boolean): void;

  getName(): string;
  setName(value: string): void;

  getComputeUnits(): number;
  setComputeUnits(value: number): void;

  getExtraComputeUnits(): number;
  setExtraComputeUnits(value: number): void;

  hasCategory(): boolean;
  clearCategory(): void;
  getCategory(): SpecCategory | undefined;
  setCategory(value?: SpecCategory): void;

  hasBlockParsing(): boolean;
  clearBlockParsing(): void;
  getBlockParsing(): BlockParser | undefined;
  setBlockParsing(value?: BlockParser): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Api.AsObject;
  static toObject(includeInstance: boolean, msg: Api): Api.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Api, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Api;
  static deserializeBinaryFromReader(message: Api, reader: jspb.BinaryReader): Api;
}

export namespace Api {
  export type AsObject = {
    enabled: boolean,
    name: string,
    computeUnits: number,
    extraComputeUnits: number,
    category?: SpecCategory.AsObject,
    blockParsing?: BlockParser.AsObject,
  }
}

export class ParseDirective extends jspb.Message {
  getFunctionTag(): FUNCTION_TAGMap[keyof FUNCTION_TAGMap];
  setFunctionTag(value: FUNCTION_TAGMap[keyof FUNCTION_TAGMap]): void;

  getFunctionTemplate(): string;
  setFunctionTemplate(value: string): void;

  hasResultParsing(): boolean;
  clearResultParsing(): void;
  getResultParsing(): BlockParser | undefined;
  setResultParsing(value?: BlockParser): void;

  getApiName(): string;
  setApiName(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ParseDirective.AsObject;
  static toObject(includeInstance: boolean, msg: ParseDirective): ParseDirective.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ParseDirective, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ParseDirective;
  static deserializeBinaryFromReader(message: ParseDirective, reader: jspb.BinaryReader): ParseDirective;
}

export namespace ParseDirective {
  export type AsObject = {
    functionTag: FUNCTION_TAGMap[keyof FUNCTION_TAGMap],
    functionTemplate: string,
    resultParsing?: BlockParser.AsObject,
    apiName: string,
  }
}

export class BlockParser extends jspb.Message {
  clearParserArgList(): void;
  getParserArgList(): Array<string>;
  setParserArgList(value: Array<string>): void;
  addParserArg(value: string, index?: number): string;

  getParserFunc(): PARSER_FUNCMap[keyof PARSER_FUNCMap];
  setParserFunc(value: PARSER_FUNCMap[keyof PARSER_FUNCMap]): void;

  getDefaultValue(): string;
  setDefaultValue(value: string): void;

  getEncoding(): string;
  setEncoding(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): BlockParser.AsObject;
  static toObject(includeInstance: boolean, msg: BlockParser): BlockParser.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: BlockParser, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): BlockParser;
  static deserializeBinaryFromReader(message: BlockParser, reader: jspb.BinaryReader): BlockParser;
}

export namespace BlockParser {
  export type AsObject = {
    parserArgList: Array<string>,
    parserFunc: PARSER_FUNCMap[keyof PARSER_FUNCMap],
    defaultValue: string,
    encoding: string,
  }
}

export class SpecCategory extends jspb.Message {
  getDeterministic(): boolean;
  setDeterministic(value: boolean): void;

  getLocal(): boolean;
  setLocal(value: boolean): void;

  getSubscription(): boolean;
  setSubscription(value: boolean): void;

  getStateful(): number;
  setStateful(value: number): void;

  getHangingApi(): boolean;
  setHangingApi(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SpecCategory.AsObject;
  static toObject(includeInstance: boolean, msg: SpecCategory): SpecCategory.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SpecCategory, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SpecCategory;
  static deserializeBinaryFromReader(message: SpecCategory, reader: jspb.BinaryReader): SpecCategory;
}

export namespace SpecCategory {
  export type AsObject = {
    deterministic: boolean,
    local: boolean,
    subscription: boolean,
    stateful: number,
    hangingApi: boolean,
  }
}

export interface EXTENSIONMap {
  NONE: 0;
  ARCHIVE: 1;
}

export const EXTENSION: EXTENSIONMap;

export interface FUNCTION_TAGMap {
  DISABLED: 0;
  GET_BLOCKNUM: 1;
  GET_BLOCK_BY_NUM: 2;
  SET_LATEST_IN_METADATA: 3;
  SET_LATEST_IN_BODY: 4;
  VERIFICATION: 5;
}

export const FUNCTION_TAG: FUNCTION_TAGMap;

export interface PARSER_FUNCMap {
  EMPTY: 0;
  PARSE_BY_ARG: 1;
  PARSE_CANONICAL: 2;
  PARSE_DICTIONARY: 3;
  PARSE_DICTIONARY_OR_ORDERED: 4;
  DEFAULT: 6;
}

export const PARSER_FUNC: PARSER_FUNCMap;


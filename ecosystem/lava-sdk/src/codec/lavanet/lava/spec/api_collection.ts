/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.spec";

export enum EXTENSION {
  NONE = 0,
  ARCHIVE = 1,
  UNRECOGNIZED = -1,
}

export function eXTENSIONFromJSON(object: any): EXTENSION {
  switch (object) {
    case 0:
    case "NONE":
      return EXTENSION.NONE;
    case 1:
    case "ARCHIVE":
      return EXTENSION.ARCHIVE;
    case -1:
    case "UNRECOGNIZED":
    default:
      return EXTENSION.UNRECOGNIZED;
  }
}

export function eXTENSIONToJSON(object: EXTENSION): string {
  switch (object) {
    case EXTENSION.NONE:
      return "NONE";
    case EXTENSION.ARCHIVE:
      return "ARCHIVE";
    case EXTENSION.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum functionTag {
  DISABLED = 0,
  GET_BLOCKNUM = 1,
  GET_BLOCK_BY_NUM = 2,
  SET_LATEST_IN_METADATA = 3,
  SET_LATEST_IN_BODY = 4,
  VERIFICATION = 5,
  UNRECOGNIZED = -1,
}

export function functionTagFromJSON(object: any): functionTag {
  switch (object) {
    case 0:
    case "DISABLED":
      return functionTag.DISABLED;
    case 1:
    case "GET_BLOCKNUM":
      return functionTag.GET_BLOCKNUM;
    case 2:
    case "GET_BLOCK_BY_NUM":
      return functionTag.GET_BLOCK_BY_NUM;
    case 3:
    case "SET_LATEST_IN_METADATA":
      return functionTag.SET_LATEST_IN_METADATA;
    case 4:
    case "SET_LATEST_IN_BODY":
      return functionTag.SET_LATEST_IN_BODY;
    case 5:
    case "VERIFICATION":
      return functionTag.VERIFICATION;
    case -1:
    case "UNRECOGNIZED":
    default:
      return functionTag.UNRECOGNIZED;
  }
}

export function functionTagToJSON(object: functionTag): string {
  switch (object) {
    case functionTag.DISABLED:
      return "DISABLED";
    case functionTag.GET_BLOCKNUM:
      return "GET_BLOCKNUM";
    case functionTag.GET_BLOCK_BY_NUM:
      return "GET_BLOCK_BY_NUM";
    case functionTag.SET_LATEST_IN_METADATA:
      return "SET_LATEST_IN_METADATA";
    case functionTag.SET_LATEST_IN_BODY:
      return "SET_LATEST_IN_BODY";
    case functionTag.VERIFICATION:
      return "VERIFICATION";
    case functionTag.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum parserFunc {
  EMPTY = 0,
  /** PARSE_BY_ARG - means parameters are ordered and flat expected arguments are: [param index] (example: PARAMS: [<#BlockNum>,"banana"]) args: 0 */
  PARSE_BY_ARG = 1,
  /** PARSE_CANONICAL - means parameters are ordered and one of them has named properties, expected arguments are: [param index to object,prop_name in object] (example: PARAMS: ["banana",{prop_name:<#BlockNum>}]) need to configure args: 1,"prop_name" */
  PARSE_CANONICAL = 2,
  /** PARSE_DICTIONARY - means parameters are named, expected arguments are [prop_name,separator] (example: PARAMS: {prop_name:<#BlockNum>,prop2:"banana"}) args: "prop_name" */
  PARSE_DICTIONARY = 3,
  /** PARSE_DICTIONARY_OR_ORDERED - means parameters are named expected arguments are [prop_name,separator,parameter order if not found] for input of: block=15&address=abc OR ?abc,15 we will do args: block,=,1 */
  PARSE_DICTIONARY_OR_ORDERED = 4,
  /** DEFAULT - reserved */
  DEFAULT = 6,
  UNRECOGNIZED = -1,
}

export function parserFuncFromJSON(object: any): parserFunc {
  switch (object) {
    case 0:
    case "EMPTY":
      return parserFunc.EMPTY;
    case 1:
    case "PARSE_BY_ARG":
      return parserFunc.PARSE_BY_ARG;
    case 2:
    case "PARSE_CANONICAL":
      return parserFunc.PARSE_CANONICAL;
    case 3:
    case "PARSE_DICTIONARY":
      return parserFunc.PARSE_DICTIONARY;
    case 4:
    case "PARSE_DICTIONARY_OR_ORDERED":
      return parserFunc.PARSE_DICTIONARY_OR_ORDERED;
    case 6:
    case "DEFAULT":
      return parserFunc.DEFAULT;
    case -1:
    case "UNRECOGNIZED":
    default:
      return parserFunc.UNRECOGNIZED;
  }
}

export function parserFuncToJSON(object: parserFunc): string {
  switch (object) {
    case parserFunc.EMPTY:
      return "EMPTY";
    case parserFunc.PARSE_BY_ARG:
      return "PARSE_BY_ARG";
    case parserFunc.PARSE_CANONICAL:
      return "PARSE_CANONICAL";
    case parserFunc.PARSE_DICTIONARY:
      return "PARSE_DICTIONARY";
    case parserFunc.PARSE_DICTIONARY_OR_ORDERED:
      return "PARSE_DICTIONARY_OR_ORDERED";
    case parserFunc.DEFAULT:
      return "DEFAULT";
    case parserFunc.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface ApiCollection {
  enabled: boolean;
  collectionData?: CollectionData;
  apis: Api[];
  headers: Header[];
  /** by collectionKey */
  inheritanceApis: CollectionData[];
  parseDirectives: ParseDirective[];
  extensions: Extension[];
  verifications: Verification[];
}

export interface Extension {
  name: string;
  cuMultiplier: number;
  rule?: Rule;
}

export interface Rule {
  block: Long;
}

export interface Verification {
  name: string;
  parseDirective?: ParseDirective;
  values: ParseValue[];
}

export interface ParseValue {
  extension: string;
  expectedValue: string;
  latestDistance: Long;
}

export interface CollectionData {
  apiInterface: string;
  internalPath: string;
  type: string;
  addOn: string;
}

export interface Header {
  name: string;
  kind: Header_HeaderType;
  functionTag: functionTag;
}

export enum Header_HeaderType {
  pass_send = 0,
  pass_reply = 1,
  pass_both = 2,
  /** pass_ignore - allows it to pass around but is not signed */
  pass_ignore = 3,
  UNRECOGNIZED = -1,
}

export function header_HeaderTypeFromJSON(object: any): Header_HeaderType {
  switch (object) {
    case 0:
    case "pass_send":
      return Header_HeaderType.pass_send;
    case 1:
    case "pass_reply":
      return Header_HeaderType.pass_reply;
    case 2:
    case "pass_both":
      return Header_HeaderType.pass_both;
    case 3:
    case "pass_ignore":
      return Header_HeaderType.pass_ignore;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Header_HeaderType.UNRECOGNIZED;
  }
}

export function header_HeaderTypeToJSON(object: Header_HeaderType): string {
  switch (object) {
    case Header_HeaderType.pass_send:
      return "pass_send";
    case Header_HeaderType.pass_reply:
      return "pass_reply";
    case Header_HeaderType.pass_both:
      return "pass_both";
    case Header_HeaderType.pass_ignore:
      return "pass_ignore";
    case Header_HeaderType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface Api {
  enabled: boolean;
  name: string;
  computeUnits: Long;
  extraComputeUnits: Long;
  category?: SpecCategory;
  blockParsing?: BlockParser;
}

export interface ParseDirective {
  functionTag: functionTag;
  functionTemplate: string;
  resultParsing?: BlockParser;
  apiName: string;
}

export interface BlockParser {
  parserArg: string[];
  parserFunc: parserFunc;
  /** default value when set allows parsing failures to assume the default value */
  defaultValue: string;
  /** used to parse byte responses: base64,hex,bech32 */
  encoding: string;
}

export interface SpecCategory {
  deterministic: boolean;
  local: boolean;
  subscription: boolean;
  stateful: number;
  hangingApi: boolean;
}

function createBaseApiCollection(): ApiCollection {
  return {
    enabled: false,
    collectionData: undefined,
    apis: [],
    headers: [],
    inheritanceApis: [],
    parseDirectives: [],
    extensions: [],
    verifications: [],
  };
}

export const ApiCollection = {
  encode(message: ApiCollection, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.enabled === true) {
      writer.uint32(8).bool(message.enabled);
    }
    if (message.collectionData !== undefined) {
      CollectionData.encode(message.collectionData, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.apis) {
      Api.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.headers) {
      Header.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.inheritanceApis) {
      CollectionData.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    for (const v of message.parseDirectives) {
      ParseDirective.encode(v!, writer.uint32(50).fork()).ldelim();
    }
    for (const v of message.extensions) {
      Extension.encode(v!, writer.uint32(58).fork()).ldelim();
    }
    for (const v of message.verifications) {
      Verification.encode(v!, writer.uint32(66).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ApiCollection {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApiCollection();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 8) {
            break;
          }

          message.enabled = reader.bool();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.collectionData = CollectionData.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.apis.push(Api.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.headers.push(Header.decode(reader, reader.uint32()));
          continue;
        case 5:
          if (tag != 42) {
            break;
          }

          message.inheritanceApis.push(CollectionData.decode(reader, reader.uint32()));
          continue;
        case 6:
          if (tag != 50) {
            break;
          }

          message.parseDirectives.push(ParseDirective.decode(reader, reader.uint32()));
          continue;
        case 7:
          if (tag != 58) {
            break;
          }

          message.extensions.push(Extension.decode(reader, reader.uint32()));
          continue;
        case 8:
          if (tag != 66) {
            break;
          }

          message.verifications.push(Verification.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ApiCollection {
    return {
      enabled: isSet(object.enabled) ? Boolean(object.enabled) : false,
      collectionData: isSet(object.collectionData) ? CollectionData.fromJSON(object.collectionData) : undefined,
      apis: Array.isArray(object?.apis) ? object.apis.map((e: any) => Api.fromJSON(e)) : [],
      headers: Array.isArray(object?.headers) ? object.headers.map((e: any) => Header.fromJSON(e)) : [],
      inheritanceApis: Array.isArray(object?.inheritanceApis)
        ? object.inheritanceApis.map((e: any) => CollectionData.fromJSON(e))
        : [],
      parseDirectives: Array.isArray(object?.parseDirectives)
        ? object.parseDirectives.map((e: any) => ParseDirective.fromJSON(e))
        : [],
      extensions: Array.isArray(object?.extensions) ? object.extensions.map((e: any) => Extension.fromJSON(e)) : [],
      verifications: Array.isArray(object?.verifications)
        ? object.verifications.map((e: any) => Verification.fromJSON(e))
        : [],
    };
  },

  toJSON(message: ApiCollection): unknown {
    const obj: any = {};
    message.enabled !== undefined && (obj.enabled = message.enabled);
    message.collectionData !== undefined &&
      (obj.collectionData = message.collectionData ? CollectionData.toJSON(message.collectionData) : undefined);
    if (message.apis) {
      obj.apis = message.apis.map((e) => e ? Api.toJSON(e) : undefined);
    } else {
      obj.apis = [];
    }
    if (message.headers) {
      obj.headers = message.headers.map((e) => e ? Header.toJSON(e) : undefined);
    } else {
      obj.headers = [];
    }
    if (message.inheritanceApis) {
      obj.inheritanceApis = message.inheritanceApis.map((e) => e ? CollectionData.toJSON(e) : undefined);
    } else {
      obj.inheritanceApis = [];
    }
    if (message.parseDirectives) {
      obj.parseDirectives = message.parseDirectives.map((e) => e ? ParseDirective.toJSON(e) : undefined);
    } else {
      obj.parseDirectives = [];
    }
    if (message.extensions) {
      obj.extensions = message.extensions.map((e) => e ? Extension.toJSON(e) : undefined);
    } else {
      obj.extensions = [];
    }
    if (message.verifications) {
      obj.verifications = message.verifications.map((e) => e ? Verification.toJSON(e) : undefined);
    } else {
      obj.verifications = [];
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<ApiCollection>, I>>(base?: I): ApiCollection {
    return ApiCollection.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<ApiCollection>, I>>(object: I): ApiCollection {
    const message = createBaseApiCollection();
    message.enabled = object.enabled ?? false;
    message.collectionData = (object.collectionData !== undefined && object.collectionData !== null)
      ? CollectionData.fromPartial(object.collectionData)
      : undefined;
    message.apis = object.apis?.map((e) => Api.fromPartial(e)) || [];
    message.headers = object.headers?.map((e) => Header.fromPartial(e)) || [];
    message.inheritanceApis = object.inheritanceApis?.map((e) => CollectionData.fromPartial(e)) || [];
    message.parseDirectives = object.parseDirectives?.map((e) => ParseDirective.fromPartial(e)) || [];
    message.extensions = object.extensions?.map((e) => Extension.fromPartial(e)) || [];
    message.verifications = object.verifications?.map((e) => Verification.fromPartial(e)) || [];
    return message;
  },
};

function createBaseExtension(): Extension {
  return { name: "", cuMultiplier: 0, rule: undefined };
}

export const Extension = {
  encode(message: Extension, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.cuMultiplier !== 0) {
      writer.uint32(21).float(message.cuMultiplier);
    }
    if (message.rule !== undefined) {
      Rule.encode(message.rule, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Extension {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseExtension();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag != 21) {
            break;
          }

          message.cuMultiplier = reader.float();
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.rule = Rule.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Extension {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      cuMultiplier: isSet(object.cuMultiplier) ? Number(object.cuMultiplier) : 0,
      rule: isSet(object.rule) ? Rule.fromJSON(object.rule) : undefined,
    };
  },

  toJSON(message: Extension): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.cuMultiplier !== undefined && (obj.cuMultiplier = message.cuMultiplier);
    message.rule !== undefined && (obj.rule = message.rule ? Rule.toJSON(message.rule) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<Extension>, I>>(base?: I): Extension {
    return Extension.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<Extension>, I>>(object: I): Extension {
    const message = createBaseExtension();
    message.name = object.name ?? "";
    message.cuMultiplier = object.cuMultiplier ?? 0;
    message.rule = (object.rule !== undefined && object.rule !== null) ? Rule.fromPartial(object.rule) : undefined;
    return message;
  },
};

function createBaseRule(): Rule {
  return { block: Long.UZERO };
}

export const Rule = {
  encode(message: Rule, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (!message.block.isZero()) {
      writer.uint32(8).uint64(message.block);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Rule {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRule();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 8) {
            break;
          }

          message.block = reader.uint64() as Long;
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Rule {
    return { block: isSet(object.block) ? Long.fromValue(object.block) : Long.UZERO };
  },

  toJSON(message: Rule): unknown {
    const obj: any = {};
    message.block !== undefined && (obj.block = (message.block || Long.UZERO).toString());
    return obj;
  },

  create<I extends Exact<DeepPartial<Rule>, I>>(base?: I): Rule {
    return Rule.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<Rule>, I>>(object: I): Rule {
    const message = createBaseRule();
    message.block = (object.block !== undefined && object.block !== null) ? Long.fromValue(object.block) : Long.UZERO;
    return message;
  },
};

function createBaseVerification(): Verification {
  return { name: "", parseDirective: undefined, values: [] };
}

export const Verification = {
  encode(message: Verification, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.parseDirective !== undefined) {
      ParseDirective.encode(message.parseDirective, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.values) {
      ParseValue.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Verification {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseVerification();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.parseDirective = ParseDirective.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.values.push(ParseValue.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Verification {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      parseDirective: isSet(object.parseDirective) ? ParseDirective.fromJSON(object.parseDirective) : undefined,
      values: Array.isArray(object?.values) ? object.values.map((e: any) => ParseValue.fromJSON(e)) : [],
    };
  },

  toJSON(message: Verification): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.parseDirective !== undefined &&
      (obj.parseDirective = message.parseDirective ? ParseDirective.toJSON(message.parseDirective) : undefined);
    if (message.values) {
      obj.values = message.values.map((e) => e ? ParseValue.toJSON(e) : undefined);
    } else {
      obj.values = [];
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Verification>, I>>(base?: I): Verification {
    return Verification.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<Verification>, I>>(object: I): Verification {
    const message = createBaseVerification();
    message.name = object.name ?? "";
    message.parseDirective = (object.parseDirective !== undefined && object.parseDirective !== null)
      ? ParseDirective.fromPartial(object.parseDirective)
      : undefined;
    message.values = object.values?.map((e) => ParseValue.fromPartial(e)) || [];
    return message;
  },
};

function createBaseParseValue(): ParseValue {
  return { extension: "", expectedValue: "", latestDistance: Long.UZERO };
}

export const ParseValue = {
  encode(message: ParseValue, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.extension !== "") {
      writer.uint32(10).string(message.extension);
    }
    if (message.expectedValue !== "") {
      writer.uint32(18).string(message.expectedValue);
    }
    if (!message.latestDistance.isZero()) {
      writer.uint32(24).uint64(message.latestDistance);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ParseValue {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseParseValue();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.extension = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.expectedValue = reader.string();
          continue;
        case 3:
          if (tag != 24) {
            break;
          }

          message.latestDistance = reader.uint64() as Long;
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ParseValue {
    return {
      extension: isSet(object.extension) ? String(object.extension) : "",
      expectedValue: isSet(object.expectedValue) ? String(object.expectedValue) : "",
      latestDistance: isSet(object.latestDistance) ? Long.fromValue(object.latestDistance) : Long.UZERO,
    };
  },

  toJSON(message: ParseValue): unknown {
    const obj: any = {};
    message.extension !== undefined && (obj.extension = message.extension);
    message.expectedValue !== undefined && (obj.expectedValue = message.expectedValue);
    message.latestDistance !== undefined && (obj.latestDistance = (message.latestDistance || Long.UZERO).toString());
    return obj;
  },

  create<I extends Exact<DeepPartial<ParseValue>, I>>(base?: I): ParseValue {
    return ParseValue.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<ParseValue>, I>>(object: I): ParseValue {
    const message = createBaseParseValue();
    message.extension = object.extension ?? "";
    message.expectedValue = object.expectedValue ?? "";
    message.latestDistance = (object.latestDistance !== undefined && object.latestDistance !== null)
      ? Long.fromValue(object.latestDistance)
      : Long.UZERO;
    return message;
  },
};

function createBaseCollectionData(): CollectionData {
  return { apiInterface: "", internalPath: "", type: "", addOn: "" };
}

export const CollectionData = {
  encode(message: CollectionData, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.apiInterface !== "") {
      writer.uint32(10).string(message.apiInterface);
    }
    if (message.internalPath !== "") {
      writer.uint32(18).string(message.internalPath);
    }
    if (message.type !== "") {
      writer.uint32(26).string(message.type);
    }
    if (message.addOn !== "") {
      writer.uint32(34).string(message.addOn);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CollectionData {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCollectionData();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.apiInterface = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.internalPath = reader.string();
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.type = reader.string();
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.addOn = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CollectionData {
    return {
      apiInterface: isSet(object.apiInterface) ? String(object.apiInterface) : "",
      internalPath: isSet(object.internalPath) ? String(object.internalPath) : "",
      type: isSet(object.type) ? String(object.type) : "",
      addOn: isSet(object.addOn) ? String(object.addOn) : "",
    };
  },

  toJSON(message: CollectionData): unknown {
    const obj: any = {};
    message.apiInterface !== undefined && (obj.apiInterface = message.apiInterface);
    message.internalPath !== undefined && (obj.internalPath = message.internalPath);
    message.type !== undefined && (obj.type = message.type);
    message.addOn !== undefined && (obj.addOn = message.addOn);
    return obj;
  },

  create<I extends Exact<DeepPartial<CollectionData>, I>>(base?: I): CollectionData {
    return CollectionData.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<CollectionData>, I>>(object: I): CollectionData {
    const message = createBaseCollectionData();
    message.apiInterface = object.apiInterface ?? "";
    message.internalPath = object.internalPath ?? "";
    message.type = object.type ?? "";
    message.addOn = object.addOn ?? "";
    return message;
  },
};

function createBaseHeader(): Header {
  return { name: "", kind: 0, functionTag: 0 };
}

export const Header = {
  encode(message: Header, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.kind !== 0) {
      writer.uint32(16).int32(message.kind);
    }
    if (message.functionTag !== 0) {
      writer.uint32(24).int32(message.functionTag);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Header {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseHeader();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag != 16) {
            break;
          }

          message.kind = reader.int32() as any;
          continue;
        case 3:
          if (tag != 24) {
            break;
          }

          message.functionTag = reader.int32() as any;
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Header {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      kind: isSet(object.kind) ? header_HeaderTypeFromJSON(object.kind) : 0,
      functionTag: isSet(object.functionTag) ? functionTagFromJSON(object.functionTag) : 0,
    };
  },

  toJSON(message: Header): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.kind !== undefined && (obj.kind = header_HeaderTypeToJSON(message.kind));
    message.functionTag !== undefined && (obj.functionTag = functionTagToJSON(message.functionTag));
    return obj;
  },

  create<I extends Exact<DeepPartial<Header>, I>>(base?: I): Header {
    return Header.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<Header>, I>>(object: I): Header {
    const message = createBaseHeader();
    message.name = object.name ?? "";
    message.kind = object.kind ?? 0;
    message.functionTag = object.functionTag ?? 0;
    return message;
  },
};

function createBaseApi(): Api {
  return {
    enabled: false,
    name: "",
    computeUnits: Long.UZERO,
    extraComputeUnits: Long.UZERO,
    category: undefined,
    blockParsing: undefined,
  };
}

export const Api = {
  encode(message: Api, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.enabled === true) {
      writer.uint32(8).bool(message.enabled);
    }
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    if (!message.computeUnits.isZero()) {
      writer.uint32(24).uint64(message.computeUnits);
    }
    if (!message.extraComputeUnits.isZero()) {
      writer.uint32(32).uint64(message.extraComputeUnits);
    }
    if (message.category !== undefined) {
      SpecCategory.encode(message.category, writer.uint32(50).fork()).ldelim();
    }
    if (message.blockParsing !== undefined) {
      BlockParser.encode(message.blockParsing, writer.uint32(58).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Api {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApi();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 8) {
            break;
          }

          message.enabled = reader.bool();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.name = reader.string();
          continue;
        case 3:
          if (tag != 24) {
            break;
          }

          message.computeUnits = reader.uint64() as Long;
          continue;
        case 4:
          if (tag != 32) {
            break;
          }

          message.extraComputeUnits = reader.uint64() as Long;
          continue;
        case 6:
          if (tag != 50) {
            break;
          }

          message.category = SpecCategory.decode(reader, reader.uint32());
          continue;
        case 7:
          if (tag != 58) {
            break;
          }

          message.blockParsing = BlockParser.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Api {
    return {
      enabled: isSet(object.enabled) ? Boolean(object.enabled) : false,
      name: isSet(object.name) ? String(object.name) : "",
      computeUnits: isSet(object.computeUnits) ? Long.fromValue(object.computeUnits) : Long.UZERO,
      extraComputeUnits: isSet(object.extraComputeUnits) ? Long.fromValue(object.extraComputeUnits) : Long.UZERO,
      category: isSet(object.category) ? SpecCategory.fromJSON(object.category) : undefined,
      blockParsing: isSet(object.blockParsing) ? BlockParser.fromJSON(object.blockParsing) : undefined,
    };
  },

  toJSON(message: Api): unknown {
    const obj: any = {};
    message.enabled !== undefined && (obj.enabled = message.enabled);
    message.name !== undefined && (obj.name = message.name);
    message.computeUnits !== undefined && (obj.computeUnits = (message.computeUnits || Long.UZERO).toString());
    message.extraComputeUnits !== undefined &&
      (obj.extraComputeUnits = (message.extraComputeUnits || Long.UZERO).toString());
    message.category !== undefined &&
      (obj.category = message.category ? SpecCategory.toJSON(message.category) : undefined);
    message.blockParsing !== undefined &&
      (obj.blockParsing = message.blockParsing ? BlockParser.toJSON(message.blockParsing) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<Api>, I>>(base?: I): Api {
    return Api.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<Api>, I>>(object: I): Api {
    const message = createBaseApi();
    message.enabled = object.enabled ?? false;
    message.name = object.name ?? "";
    message.computeUnits = (object.computeUnits !== undefined && object.computeUnits !== null)
      ? Long.fromValue(object.computeUnits)
      : Long.UZERO;
    message.extraComputeUnits = (object.extraComputeUnits !== undefined && object.extraComputeUnits !== null)
      ? Long.fromValue(object.extraComputeUnits)
      : Long.UZERO;
    message.category = (object.category !== undefined && object.category !== null)
      ? SpecCategory.fromPartial(object.category)
      : undefined;
    message.blockParsing = (object.blockParsing !== undefined && object.blockParsing !== null)
      ? BlockParser.fromPartial(object.blockParsing)
      : undefined;
    return message;
  },
};

function createBaseParseDirective(): ParseDirective {
  return { functionTag: 0, functionTemplate: "", resultParsing: undefined, apiName: "" };
}

export const ParseDirective = {
  encode(message: ParseDirective, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.functionTag !== 0) {
      writer.uint32(8).int32(message.functionTag);
    }
    if (message.functionTemplate !== "") {
      writer.uint32(18).string(message.functionTemplate);
    }
    if (message.resultParsing !== undefined) {
      BlockParser.encode(message.resultParsing, writer.uint32(26).fork()).ldelim();
    }
    if (message.apiName !== "") {
      writer.uint32(34).string(message.apiName);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ParseDirective {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseParseDirective();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 8) {
            break;
          }

          message.functionTag = reader.int32() as any;
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.functionTemplate = reader.string();
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.resultParsing = BlockParser.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.apiName = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ParseDirective {
    return {
      functionTag: isSet(object.functionTag) ? functionTagFromJSON(object.functionTag) : 0,
      functionTemplate: isSet(object.functionTemplate) ? String(object.functionTemplate) : "",
      resultParsing: isSet(object.resultParsing) ? BlockParser.fromJSON(object.resultParsing) : undefined,
      apiName: isSet(object.apiName) ? String(object.apiName) : "",
    };
  },

  toJSON(message: ParseDirective): unknown {
    const obj: any = {};
    message.functionTag !== undefined && (obj.functionTag = functionTagToJSON(message.functionTag));
    message.functionTemplate !== undefined && (obj.functionTemplate = message.functionTemplate);
    message.resultParsing !== undefined &&
      (obj.resultParsing = message.resultParsing ? BlockParser.toJSON(message.resultParsing) : undefined);
    message.apiName !== undefined && (obj.apiName = message.apiName);
    return obj;
  },

  create<I extends Exact<DeepPartial<ParseDirective>, I>>(base?: I): ParseDirective {
    return ParseDirective.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<ParseDirective>, I>>(object: I): ParseDirective {
    const message = createBaseParseDirective();
    message.functionTag = object.functionTag ?? 0;
    message.functionTemplate = object.functionTemplate ?? "";
    message.resultParsing = (object.resultParsing !== undefined && object.resultParsing !== null)
      ? BlockParser.fromPartial(object.resultParsing)
      : undefined;
    message.apiName = object.apiName ?? "";
    return message;
  },
};

function createBaseBlockParser(): BlockParser {
  return { parserArg: [], parserFunc: 0, defaultValue: "", encoding: "" };
}

export const BlockParser = {
  encode(message: BlockParser, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.parserArg) {
      writer.uint32(10).string(v!);
    }
    if (message.parserFunc !== 0) {
      writer.uint32(16).int32(message.parserFunc);
    }
    if (message.defaultValue !== "") {
      writer.uint32(26).string(message.defaultValue);
    }
    if (message.encoding !== "") {
      writer.uint32(34).string(message.encoding);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BlockParser {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBlockParser();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.parserArg.push(reader.string());
          continue;
        case 2:
          if (tag != 16) {
            break;
          }

          message.parserFunc = reader.int32() as any;
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.defaultValue = reader.string();
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.encoding = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BlockParser {
    return {
      parserArg: Array.isArray(object?.parserArg) ? object.parserArg.map((e: any) => String(e)) : [],
      parserFunc: isSet(object.parserFunc) ? parserFuncFromJSON(object.parserFunc) : 0,
      defaultValue: isSet(object.defaultValue) ? String(object.defaultValue) : "",
      encoding: isSet(object.encoding) ? String(object.encoding) : "",
    };
  },

  toJSON(message: BlockParser): unknown {
    const obj: any = {};
    if (message.parserArg) {
      obj.parserArg = message.parserArg.map((e) => e);
    } else {
      obj.parserArg = [];
    }
    message.parserFunc !== undefined && (obj.parserFunc = parserFuncToJSON(message.parserFunc));
    message.defaultValue !== undefined && (obj.defaultValue = message.defaultValue);
    message.encoding !== undefined && (obj.encoding = message.encoding);
    return obj;
  },

  create<I extends Exact<DeepPartial<BlockParser>, I>>(base?: I): BlockParser {
    return BlockParser.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<BlockParser>, I>>(object: I): BlockParser {
    const message = createBaseBlockParser();
    message.parserArg = object.parserArg?.map((e) => e) || [];
    message.parserFunc = object.parserFunc ?? 0;
    message.defaultValue = object.defaultValue ?? "";
    message.encoding = object.encoding ?? "";
    return message;
  },
};

function createBaseSpecCategory(): SpecCategory {
  return { deterministic: false, local: false, subscription: false, stateful: 0, hangingApi: false };
}

export const SpecCategory = {
  encode(message: SpecCategory, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.deterministic === true) {
      writer.uint32(8).bool(message.deterministic);
    }
    if (message.local === true) {
      writer.uint32(16).bool(message.local);
    }
    if (message.subscription === true) {
      writer.uint32(24).bool(message.subscription);
    }
    if (message.stateful !== 0) {
      writer.uint32(32).uint32(message.stateful);
    }
    if (message.hangingApi === true) {
      writer.uint32(40).bool(message.hangingApi);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SpecCategory {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSpecCategory();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 8) {
            break;
          }

          message.deterministic = reader.bool();
          continue;
        case 2:
          if (tag != 16) {
            break;
          }

          message.local = reader.bool();
          continue;
        case 3:
          if (tag != 24) {
            break;
          }

          message.subscription = reader.bool();
          continue;
        case 4:
          if (tag != 32) {
            break;
          }

          message.stateful = reader.uint32();
          continue;
        case 5:
          if (tag != 40) {
            break;
          }

          message.hangingApi = reader.bool();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SpecCategory {
    return {
      deterministic: isSet(object.deterministic) ? Boolean(object.deterministic) : false,
      local: isSet(object.local) ? Boolean(object.local) : false,
      subscription: isSet(object.subscription) ? Boolean(object.subscription) : false,
      stateful: isSet(object.stateful) ? Number(object.stateful) : 0,
      hangingApi: isSet(object.hangingApi) ? Boolean(object.hangingApi) : false,
    };
  },

  toJSON(message: SpecCategory): unknown {
    const obj: any = {};
    message.deterministic !== undefined && (obj.deterministic = message.deterministic);
    message.local !== undefined && (obj.local = message.local);
    message.subscription !== undefined && (obj.subscription = message.subscription);
    message.stateful !== undefined && (obj.stateful = Math.round(message.stateful));
    message.hangingApi !== undefined && (obj.hangingApi = message.hangingApi);
    return obj;
  },

  create<I extends Exact<DeepPartial<SpecCategory>, I>>(base?: I): SpecCategory {
    return SpecCategory.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<SpecCategory>, I>>(object: I): SpecCategory {
    const message = createBaseSpecCategory();
    message.deterministic = object.deterministic ?? false;
    message.local = object.local ?? false;
    message.subscription = object.subscription ?? false;
    message.stateful = object.stateful ?? 0;
    message.hangingApi = object.hangingApi ?? false;
    return message;
  },
};

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Long ? string | number | Long : T extends Array<infer U> ? Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

type KeysOfUnion<T> = T extends T ? keyof T : never;
export type Exact<P, I extends P> = P extends Builtin ? P
  : P & { [K in keyof P]: Exact<P[K], I[K]> } & { [K in Exclude<keyof I, KeysOfUnion<P>>]: never };

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}

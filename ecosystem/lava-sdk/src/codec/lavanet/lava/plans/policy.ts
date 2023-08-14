/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { CollectionData } from "../spec/api_collection";

export const protobufPackage = "lavanet.lava.plans";

/** the enum below determines the pairing algorithm's behaviour with the selected providers feature */
export enum selectedProvidersMode {
  /** ALLOWED - no providers restrictions */
  ALLOWED = 0,
  /** MIXED - use the selected providers mixed with randomly chosen providers */
  MIXED = 1,
  /** EXCLUSIVE - use only the selected providers */
  EXCLUSIVE = 2,
  /** DISABLED - selected providers feature is disabled */
  DISABLED = 3,
  UNRECOGNIZED = -1,
}

export function selectedProvidersModeFromJSON(object: any): selectedProvidersMode {
  switch (object) {
    case 0:
    case "ALLOWED":
      return selectedProvidersMode.ALLOWED;
    case 1:
    case "MIXED":
      return selectedProvidersMode.MIXED;
    case 2:
    case "EXCLUSIVE":
      return selectedProvidersMode.EXCLUSIVE;
    case 3:
    case "DISABLED":
      return selectedProvidersMode.DISABLED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return selectedProvidersMode.UNRECOGNIZED;
  }
}

export function selectedProvidersModeToJSON(object: selectedProvidersMode): string {
  switch (object) {
    case selectedProvidersMode.ALLOWED:
      return "ALLOWED";
    case selectedProvidersMode.MIXED:
      return "MIXED";
    case selectedProvidersMode.EXCLUSIVE:
      return "EXCLUSIVE";
    case selectedProvidersMode.DISABLED:
      return "DISABLED";
    case selectedProvidersMode.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

/** protobuf expected in YAML format: used "moretags" to simplify parsing */
export interface Policy {
  chainPolicies: ChainPolicy[];
  geolocationProfile: Long;
  totalCuLimit: Long;
  epochCuLimit: Long;
  maxProvidersToPair: Long;
  selectedProvidersMode: selectedProvidersMode;
  selectedProviders: string[];
}

export interface ChainPolicy {
  chainId: string;
  apis: string[];
  requirements: ChainRequirement[];
}

export interface ChainRequirement {
  collection?: CollectionData;
  extensions: string[];
}

function createBasePolicy(): Policy {
  return {
    chainPolicies: [],
    geolocationProfile: Long.UZERO,
    totalCuLimit: Long.UZERO,
    epochCuLimit: Long.UZERO,
    maxProvidersToPair: Long.UZERO,
    selectedProvidersMode: 0,
    selectedProviders: [],
  };
}

export const Policy = {
  encode(message: Policy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.chainPolicies) {
      ChainPolicy.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (!message.geolocationProfile.isZero()) {
      writer.uint32(16).uint64(message.geolocationProfile);
    }
    if (!message.totalCuLimit.isZero()) {
      writer.uint32(24).uint64(message.totalCuLimit);
    }
    if (!message.epochCuLimit.isZero()) {
      writer.uint32(32).uint64(message.epochCuLimit);
    }
    if (!message.maxProvidersToPair.isZero()) {
      writer.uint32(40).uint64(message.maxProvidersToPair);
    }
    if (message.selectedProvidersMode !== 0) {
      writer.uint32(48).int32(message.selectedProvidersMode);
    }
    for (const v of message.selectedProviders) {
      writer.uint32(58).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Policy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.chainPolicies.push(ChainPolicy.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag != 16) {
            break;
          }

          message.geolocationProfile = reader.uint64() as Long;
          continue;
        case 3:
          if (tag != 24) {
            break;
          }

          message.totalCuLimit = reader.uint64() as Long;
          continue;
        case 4:
          if (tag != 32) {
            break;
          }

          message.epochCuLimit = reader.uint64() as Long;
          continue;
        case 5:
          if (tag != 40) {
            break;
          }

          message.maxProvidersToPair = reader.uint64() as Long;
          continue;
        case 6:
          if (tag != 48) {
            break;
          }

          message.selectedProvidersMode = reader.int32() as any;
          continue;
        case 7:
          if (tag != 58) {
            break;
          }

          message.selectedProviders.push(reader.string());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Policy {
    return {
      chainPolicies: Array.isArray(object?.chainPolicies)
        ? object.chainPolicies.map((e: any) => ChainPolicy.fromJSON(e))
        : [],
      geolocationProfile: isSet(object.geolocationProfile) ? Long.fromValue(object.geolocationProfile) : Long.UZERO,
      totalCuLimit: isSet(object.totalCuLimit) ? Long.fromValue(object.totalCuLimit) : Long.UZERO,
      epochCuLimit: isSet(object.epochCuLimit) ? Long.fromValue(object.epochCuLimit) : Long.UZERO,
      maxProvidersToPair: isSet(object.maxProvidersToPair) ? Long.fromValue(object.maxProvidersToPair) : Long.UZERO,
      selectedProvidersMode: isSet(object.selectedProvidersMode)
        ? selectedProvidersModeFromJSON(object.selectedProvidersMode)
        : 0,
      selectedProviders: Array.isArray(object?.selectedProviders)
        ? object.selectedProviders.map((e: any) => String(e))
        : [],
    };
  },

  toJSON(message: Policy): unknown {
    const obj: any = {};
    if (message.chainPolicies) {
      obj.chainPolicies = message.chainPolicies.map((e) => e ? ChainPolicy.toJSON(e) : undefined);
    } else {
      obj.chainPolicies = [];
    }
    message.geolocationProfile !== undefined &&
      (obj.geolocationProfile = (message.geolocationProfile || Long.UZERO).toString());
    message.totalCuLimit !== undefined && (obj.totalCuLimit = (message.totalCuLimit || Long.UZERO).toString());
    message.epochCuLimit !== undefined && (obj.epochCuLimit = (message.epochCuLimit || Long.UZERO).toString());
    message.maxProvidersToPair !== undefined &&
      (obj.maxProvidersToPair = (message.maxProvidersToPair || Long.UZERO).toString());
    message.selectedProvidersMode !== undefined &&
      (obj.selectedProvidersMode = selectedProvidersModeToJSON(message.selectedProvidersMode));
    if (message.selectedProviders) {
      obj.selectedProviders = message.selectedProviders.map((e) => e);
    } else {
      obj.selectedProviders = [];
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Policy>, I>>(base?: I): Policy {
    return Policy.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<Policy>, I>>(object: I): Policy {
    const message = createBasePolicy();
    message.chainPolicies = object.chainPolicies?.map((e) => ChainPolicy.fromPartial(e)) || [];
    message.geolocationProfile = (object.geolocationProfile !== undefined && object.geolocationProfile !== null)
      ? Long.fromValue(object.geolocationProfile)
      : Long.UZERO;
    message.totalCuLimit = (object.totalCuLimit !== undefined && object.totalCuLimit !== null)
      ? Long.fromValue(object.totalCuLimit)
      : Long.UZERO;
    message.epochCuLimit = (object.epochCuLimit !== undefined && object.epochCuLimit !== null)
      ? Long.fromValue(object.epochCuLimit)
      : Long.UZERO;
    message.maxProvidersToPair = (object.maxProvidersToPair !== undefined && object.maxProvidersToPair !== null)
      ? Long.fromValue(object.maxProvidersToPair)
      : Long.UZERO;
    message.selectedProvidersMode = object.selectedProvidersMode ?? 0;
    message.selectedProviders = object.selectedProviders?.map((e) => e) || [];
    return message;
  },
};

function createBaseChainPolicy(): ChainPolicy {
  return { chainId: "", apis: [], requirements: [] };
}

export const ChainPolicy = {
  encode(message: ChainPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.chainId !== "") {
      writer.uint32(10).string(message.chainId);
    }
    for (const v of message.apis) {
      writer.uint32(18).string(v!);
    }
    for (const v of message.requirements) {
      ChainRequirement.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ChainPolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseChainPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.chainId = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.apis.push(reader.string());
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.requirements.push(ChainRequirement.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ChainPolicy {
    return {
      chainId: isSet(object.chainId) ? String(object.chainId) : "",
      apis: Array.isArray(object?.apis) ? object.apis.map((e: any) => String(e)) : [],
      requirements: Array.isArray(object?.requirements)
        ? object.requirements.map((e: any) => ChainRequirement.fromJSON(e))
        : [],
    };
  },

  toJSON(message: ChainPolicy): unknown {
    const obj: any = {};
    message.chainId !== undefined && (obj.chainId = message.chainId);
    if (message.apis) {
      obj.apis = message.apis.map((e) => e);
    } else {
      obj.apis = [];
    }
    if (message.requirements) {
      obj.requirements = message.requirements.map((e) => e ? ChainRequirement.toJSON(e) : undefined);
    } else {
      obj.requirements = [];
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<ChainPolicy>, I>>(base?: I): ChainPolicy {
    return ChainPolicy.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<ChainPolicy>, I>>(object: I): ChainPolicy {
    const message = createBaseChainPolicy();
    message.chainId = object.chainId ?? "";
    message.apis = object.apis?.map((e) => e) || [];
    message.requirements = object.requirements?.map((e) => ChainRequirement.fromPartial(e)) || [];
    return message;
  },
};

function createBaseChainRequirement(): ChainRequirement {
  return { collection: undefined, extensions: [] };
}

export const ChainRequirement = {
  encode(message: ChainRequirement, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.collection !== undefined) {
      CollectionData.encode(message.collection, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.extensions) {
      writer.uint32(18).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ChainRequirement {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseChainRequirement();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.collection = CollectionData.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.extensions.push(reader.string());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ChainRequirement {
    return {
      collection: isSet(object.collection) ? CollectionData.fromJSON(object.collection) : undefined,
      extensions: Array.isArray(object?.extensions) ? object.extensions.map((e: any) => String(e)) : [],
    };
  },

  toJSON(message: ChainRequirement): unknown {
    const obj: any = {};
    message.collection !== undefined &&
      (obj.collection = message.collection ? CollectionData.toJSON(message.collection) : undefined);
    if (message.extensions) {
      obj.extensions = message.extensions.map((e) => e);
    } else {
      obj.extensions = [];
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<ChainRequirement>, I>>(base?: I): ChainRequirement {
    return ChainRequirement.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<ChainRequirement>, I>>(object: I): ChainRequirement {
    const message = createBaseChainRequirement();
    message.collection = (object.collection !== undefined && object.collection !== null)
      ? CollectionData.fromPartial(object.collection)
      : undefined;
    message.extensions = object.extensions?.map((e) => e) || [];
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

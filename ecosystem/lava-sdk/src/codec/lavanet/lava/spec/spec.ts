/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Coin } from "../../../cosmos/base/v1beta1/coin";
import { ApiCollection } from "./api_collection";

export const protobufPackage = "lavanet.lava.spec";

export interface Spec {
  index: string;
  name: string;
  enabled: boolean;
  reliabilityThreshold: number;
  dataReliabilityEnabled: boolean;
  blockDistanceForFinalizedData: number;
  blocksInFinalizationProof: number;
  averageBlockTime: Long;
  allowedBlockLagForQosSync: Long;
  blockLastUpdated: Long;
  minStakeProvider?: Coin;
  minStakeClient?: Coin;
  providersTypes: Spec_ProvidersTypes;
  imports: string[];
  apiCollections: ApiCollection[];
}

export enum Spec_ProvidersTypes {
  dynamic = 0,
  static = 1,
  UNRECOGNIZED = -1,
}

export function spec_ProvidersTypesFromJSON(object: any): Spec_ProvidersTypes {
  switch (object) {
    case 0:
    case "dynamic":
      return Spec_ProvidersTypes.dynamic;
    case 1:
    case "static":
      return Spec_ProvidersTypes.static;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Spec_ProvidersTypes.UNRECOGNIZED;
  }
}

export function spec_ProvidersTypesToJSON(object: Spec_ProvidersTypes): string {
  switch (object) {
    case Spec_ProvidersTypes.dynamic:
      return "dynamic";
    case Spec_ProvidersTypes.static:
      return "static";
    case Spec_ProvidersTypes.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

function createBaseSpec(): Spec {
  return {
    index: "",
    name: "",
    enabled: false,
    reliabilityThreshold: 0,
    dataReliabilityEnabled: false,
    blockDistanceForFinalizedData: 0,
    blocksInFinalizationProof: 0,
    averageBlockTime: Long.ZERO,
    allowedBlockLagForQosSync: Long.ZERO,
    blockLastUpdated: Long.UZERO,
    minStakeProvider: undefined,
    minStakeClient: undefined,
    providersTypes: 0,
    imports: [],
    apiCollections: [],
  };
}

export const Spec = {
  encode(message: Spec, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    if (message.enabled === true) {
      writer.uint32(32).bool(message.enabled);
    }
    if (message.reliabilityThreshold !== 0) {
      writer.uint32(40).uint32(message.reliabilityThreshold);
    }
    if (message.dataReliabilityEnabled === true) {
      writer.uint32(48).bool(message.dataReliabilityEnabled);
    }
    if (message.blockDistanceForFinalizedData !== 0) {
      writer.uint32(56).uint32(message.blockDistanceForFinalizedData);
    }
    if (message.blocksInFinalizationProof !== 0) {
      writer.uint32(64).uint32(message.blocksInFinalizationProof);
    }
    if (!message.averageBlockTime.isZero()) {
      writer.uint32(72).int64(message.averageBlockTime);
    }
    if (!message.allowedBlockLagForQosSync.isZero()) {
      writer.uint32(80).int64(message.allowedBlockLagForQosSync);
    }
    if (!message.blockLastUpdated.isZero()) {
      writer.uint32(88).uint64(message.blockLastUpdated);
    }
    if (message.minStakeProvider !== undefined) {
      Coin.encode(message.minStakeProvider, writer.uint32(98).fork()).ldelim();
    }
    if (message.minStakeClient !== undefined) {
      Coin.encode(message.minStakeClient, writer.uint32(106).fork()).ldelim();
    }
    if (message.providersTypes !== 0) {
      writer.uint32(112).int32(message.providersTypes);
    }
    for (const v of message.imports) {
      writer.uint32(122).string(v!);
    }
    for (const v of message.apiCollections) {
      ApiCollection.encode(v!, writer.uint32(130).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Spec {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSpec();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.index = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.name = reader.string();
          continue;
        case 4:
          if (tag != 32) {
            break;
          }

          message.enabled = reader.bool();
          continue;
        case 5:
          if (tag != 40) {
            break;
          }

          message.reliabilityThreshold = reader.uint32();
          continue;
        case 6:
          if (tag != 48) {
            break;
          }

          message.dataReliabilityEnabled = reader.bool();
          continue;
        case 7:
          if (tag != 56) {
            break;
          }

          message.blockDistanceForFinalizedData = reader.uint32();
          continue;
        case 8:
          if (tag != 64) {
            break;
          }

          message.blocksInFinalizationProof = reader.uint32();
          continue;
        case 9:
          if (tag != 72) {
            break;
          }

          message.averageBlockTime = reader.int64() as Long;
          continue;
        case 10:
          if (tag != 80) {
            break;
          }

          message.allowedBlockLagForQosSync = reader.int64() as Long;
          continue;
        case 11:
          if (tag != 88) {
            break;
          }

          message.blockLastUpdated = reader.uint64() as Long;
          continue;
        case 12:
          if (tag != 98) {
            break;
          }

          message.minStakeProvider = Coin.decode(reader, reader.uint32());
          continue;
        case 13:
          if (tag != 106) {
            break;
          }

          message.minStakeClient = Coin.decode(reader, reader.uint32());
          continue;
        case 14:
          if (tag != 112) {
            break;
          }

          message.providersTypes = reader.int32() as any;
          continue;
        case 15:
          if (tag != 122) {
            break;
          }

          message.imports.push(reader.string());
          continue;
        case 16:
          if (tag != 130) {
            break;
          }

          message.apiCollections.push(ApiCollection.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Spec {
    return {
      index: isSet(object.index) ? String(object.index) : "",
      name: isSet(object.name) ? String(object.name) : "",
      enabled: isSet(object.enabled) ? Boolean(object.enabled) : false,
      reliabilityThreshold: isSet(object.reliabilityThreshold) ? Number(object.reliabilityThreshold) : 0,
      dataReliabilityEnabled: isSet(object.dataReliabilityEnabled) ? Boolean(object.dataReliabilityEnabled) : false,
      blockDistanceForFinalizedData: isSet(object.blockDistanceForFinalizedData)
        ? Number(object.blockDistanceForFinalizedData)
        : 0,
      blocksInFinalizationProof: isSet(object.blocksInFinalizationProof) ? Number(object.blocksInFinalizationProof) : 0,
      averageBlockTime: isSet(object.averageBlockTime) ? Long.fromValue(object.averageBlockTime) : Long.ZERO,
      allowedBlockLagForQosSync: isSet(object.allowedBlockLagForQosSync)
        ? Long.fromValue(object.allowedBlockLagForQosSync)
        : Long.ZERO,
      blockLastUpdated: isSet(object.blockLastUpdated) ? Long.fromValue(object.blockLastUpdated) : Long.UZERO,
      minStakeProvider: isSet(object.minStakeProvider) ? Coin.fromJSON(object.minStakeProvider) : undefined,
      minStakeClient: isSet(object.minStakeClient) ? Coin.fromJSON(object.minStakeClient) : undefined,
      providersTypes: isSet(object.providersTypes) ? spec_ProvidersTypesFromJSON(object.providersTypes) : 0,
      imports: Array.isArray(object?.imports) ? object.imports.map((e: any) => String(e)) : [],
      apiCollections: Array.isArray(object?.apiCollections)
        ? object.apiCollections.map((e: any) => ApiCollection.fromJSON(e))
        : [],
    };
  },

  toJSON(message: Spec): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    message.name !== undefined && (obj.name = message.name);
    message.enabled !== undefined && (obj.enabled = message.enabled);
    message.reliabilityThreshold !== undefined && (obj.reliabilityThreshold = Math.round(message.reliabilityThreshold));
    message.dataReliabilityEnabled !== undefined && (obj.dataReliabilityEnabled = message.dataReliabilityEnabled);
    message.blockDistanceForFinalizedData !== undefined &&
      (obj.blockDistanceForFinalizedData = Math.round(message.blockDistanceForFinalizedData));
    message.blocksInFinalizationProof !== undefined &&
      (obj.blocksInFinalizationProof = Math.round(message.blocksInFinalizationProof));
    message.averageBlockTime !== undefined &&
      (obj.averageBlockTime = (message.averageBlockTime || Long.ZERO).toString());
    message.allowedBlockLagForQosSync !== undefined &&
      (obj.allowedBlockLagForQosSync = (message.allowedBlockLagForQosSync || Long.ZERO).toString());
    message.blockLastUpdated !== undefined &&
      (obj.blockLastUpdated = (message.blockLastUpdated || Long.UZERO).toString());
    message.minStakeProvider !== undefined &&
      (obj.minStakeProvider = message.minStakeProvider ? Coin.toJSON(message.minStakeProvider) : undefined);
    message.minStakeClient !== undefined &&
      (obj.minStakeClient = message.minStakeClient ? Coin.toJSON(message.minStakeClient) : undefined);
    message.providersTypes !== undefined && (obj.providersTypes = spec_ProvidersTypesToJSON(message.providersTypes));
    if (message.imports) {
      obj.imports = message.imports.map((e) => e);
    } else {
      obj.imports = [];
    }
    if (message.apiCollections) {
      obj.apiCollections = message.apiCollections.map((e) => e ? ApiCollection.toJSON(e) : undefined);
    } else {
      obj.apiCollections = [];
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Spec>, I>>(base?: I): Spec {
    return Spec.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<Spec>, I>>(object: I): Spec {
    const message = createBaseSpec();
    message.index = object.index ?? "";
    message.name = object.name ?? "";
    message.enabled = object.enabled ?? false;
    message.reliabilityThreshold = object.reliabilityThreshold ?? 0;
    message.dataReliabilityEnabled = object.dataReliabilityEnabled ?? false;
    message.blockDistanceForFinalizedData = object.blockDistanceForFinalizedData ?? 0;
    message.blocksInFinalizationProof = object.blocksInFinalizationProof ?? 0;
    message.averageBlockTime = (object.averageBlockTime !== undefined && object.averageBlockTime !== null)
      ? Long.fromValue(object.averageBlockTime)
      : Long.ZERO;
    message.allowedBlockLagForQosSync =
      (object.allowedBlockLagForQosSync !== undefined && object.allowedBlockLagForQosSync !== null)
        ? Long.fromValue(object.allowedBlockLagForQosSync)
        : Long.ZERO;
    message.blockLastUpdated = (object.blockLastUpdated !== undefined && object.blockLastUpdated !== null)
      ? Long.fromValue(object.blockLastUpdated)
      : Long.UZERO;
    message.minStakeProvider = (object.minStakeProvider !== undefined && object.minStakeProvider !== null)
      ? Coin.fromPartial(object.minStakeProvider)
      : undefined;
    message.minStakeClient = (object.minStakeClient !== undefined && object.minStakeClient !== null)
      ? Coin.fromPartial(object.minStakeClient)
      : undefined;
    message.providersTypes = object.providersTypes ?? 0;
    message.imports = object.imports?.map((e) => e) || [];
    message.apiCollections = object.apiCollections?.map((e) => ApiCollection.fromPartial(e)) || [];
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

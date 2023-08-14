/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { PageRequest, PageResponse } from "../../../cosmos/base/query/v1beta1/pagination";
import { StakeEntry } from "../epochstorage/stake_entry";
import { Policy } from "../plans/policy";
import { Project } from "../projects/project";
import { Spec } from "../spec/spec";
import { Subscription } from "../subscription/subscription";
import { EpochPayments } from "./epoch_payments";
import { Params } from "./params";
import { ProviderPaymentStorage } from "./provider_payment_storage";
import { UniquePaymentStorageClientProvider } from "./unique_payment_storage_client_provider";

export const protobufPackage = "lavanet.lava.pairing";

/** QueryParamsRequest is request type for the Query/Params RPC method. */
export interface QueryParamsRequest {
}

/** QueryParamsResponse is response type for the Query/Params RPC method. */
export interface QueryParamsResponse {
  /** params holds all the parameters of this module. */
  params?: Params;
}

export interface QueryProvidersRequest {
  chainID: string;
  showFrozen: boolean;
}

export interface QueryProvidersResponse {
  stakeEntry: StakeEntry[];
  output: string;
}

export interface QueryGetPairingRequest {
  chainID: string;
  client: string;
}

export interface QueryGetPairingResponse {
  providers: StakeEntry[];
  currentEpoch: Long;
  timeLeftToNextPairing: Long;
  specLastUpdatedBlock: Long;
  blockOfNextPairing: Long;
}

export interface QueryVerifyPairingRequest {
  chainID: string;
  client: string;
  provider: string;
  block: Long;
}

export interface QueryVerifyPairingResponse {
  valid: boolean;
  pairedProviders: Long;
  cuPerEpoch: Long;
  projectId: string;
}

export interface QueryGetUniquePaymentStorageClientProviderRequest {
  index: string;
}

export interface QueryGetUniquePaymentStorageClientProviderResponse {
  uniquePaymentStorageClientProvider?: UniquePaymentStorageClientProvider;
}

export interface QueryAllUniquePaymentStorageClientProviderRequest {
  pagination?: PageRequest;
}

export interface QueryAllUniquePaymentStorageClientProviderResponse {
  uniquePaymentStorageClientProvider: UniquePaymentStorageClientProvider[];
  pagination?: PageResponse;
}

export interface QueryGetProviderPaymentStorageRequest {
  index: string;
}

export interface QueryGetProviderPaymentStorageResponse {
  providerPaymentStorage?: ProviderPaymentStorage;
}

export interface QueryAllProviderPaymentStorageRequest {
  pagination?: PageRequest;
}

export interface QueryAllProviderPaymentStorageResponse {
  providerPaymentStorage: ProviderPaymentStorage[];
  pagination?: PageResponse;
}

export interface QueryGetEpochPaymentsRequest {
  index: string;
}

export interface QueryGetEpochPaymentsResponse {
  epochPayments?: EpochPayments;
}

export interface QueryAllEpochPaymentsRequest {
  pagination?: PageRequest;
}

export interface QueryAllEpochPaymentsResponse {
  epochPayments: EpochPayments[];
  pagination?: PageResponse;
}

export interface QueryUserEntryRequest {
  address: string;
  chainID: string;
  block: Long;
}

export interface QueryUserEntryResponse {
  consumer?: StakeEntry;
  maxCU: Long;
}

export interface QueryStaticProvidersListRequest {
  chainID: string;
}

export interface QueryStaticProvidersListResponse {
  providers: StakeEntry[];
}

export interface QueryAccountInfoResponse {
  provider: StakeEntry[];
  frozen: StakeEntry[];
  consumer: StakeEntry[];
  unstaked: StakeEntry[];
  subscription?: Subscription;
  project?: Project;
}

export interface QueryEffectivePolicyRequest {
  consumer: string;
  specID: string;
}

export interface QueryEffectivePolicyResponse {
  policy?: Policy;
}

export interface QuerySdkPairingResponse {
  pairing?: QueryGetPairingResponse;
  maxCu: Long;
  spec?: Spec;
}

function createBaseQueryParamsRequest(): QueryParamsRequest {
  return {};
}

export const QueryParamsRequest = {
  encode(_: QueryParamsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryParamsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryParamsRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(_: any): QueryParamsRequest {
    return {};
  },

  toJSON(_: QueryParamsRequest): unknown {
    const obj: any = {};
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryParamsRequest>, I>>(base?: I): QueryParamsRequest {
    return QueryParamsRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryParamsRequest>, I>>(_: I): QueryParamsRequest {
    const message = createBaseQueryParamsRequest();
    return message;
  },
};

function createBaseQueryParamsResponse(): QueryParamsResponse {
  return { params: undefined };
}

export const QueryParamsResponse = {
  encode(message: QueryParamsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryParamsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryParamsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.params = Params.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryParamsResponse {
    return { params: isSet(object.params) ? Params.fromJSON(object.params) : undefined };
  },

  toJSON(message: QueryParamsResponse): unknown {
    const obj: any = {};
    message.params !== undefined && (obj.params = message.params ? Params.toJSON(message.params) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryParamsResponse>, I>>(base?: I): QueryParamsResponse {
    return QueryParamsResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryParamsResponse>, I>>(object: I): QueryParamsResponse {
    const message = createBaseQueryParamsResponse();
    message.params = (object.params !== undefined && object.params !== null)
      ? Params.fromPartial(object.params)
      : undefined;
    return message;
  },
};

function createBaseQueryProvidersRequest(): QueryProvidersRequest {
  return { chainID: "", showFrozen: false };
}

export const QueryProvidersRequest = {
  encode(message: QueryProvidersRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.chainID !== "") {
      writer.uint32(10).string(message.chainID);
    }
    if (message.showFrozen === true) {
      writer.uint32(16).bool(message.showFrozen);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryProvidersRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryProvidersRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.chainID = reader.string();
          continue;
        case 2:
          if (tag != 16) {
            break;
          }

          message.showFrozen = reader.bool();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryProvidersRequest {
    return {
      chainID: isSet(object.chainID) ? String(object.chainID) : "",
      showFrozen: isSet(object.showFrozen) ? Boolean(object.showFrozen) : false,
    };
  },

  toJSON(message: QueryProvidersRequest): unknown {
    const obj: any = {};
    message.chainID !== undefined && (obj.chainID = message.chainID);
    message.showFrozen !== undefined && (obj.showFrozen = message.showFrozen);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryProvidersRequest>, I>>(base?: I): QueryProvidersRequest {
    return QueryProvidersRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryProvidersRequest>, I>>(object: I): QueryProvidersRequest {
    const message = createBaseQueryProvidersRequest();
    message.chainID = object.chainID ?? "";
    message.showFrozen = object.showFrozen ?? false;
    return message;
  },
};

function createBaseQueryProvidersResponse(): QueryProvidersResponse {
  return { stakeEntry: [], output: "" };
}

export const QueryProvidersResponse = {
  encode(message: QueryProvidersResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.stakeEntry) {
      StakeEntry.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.output !== "") {
      writer.uint32(18).string(message.output);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryProvidersResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryProvidersResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.stakeEntry.push(StakeEntry.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.output = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryProvidersResponse {
    return {
      stakeEntry: Array.isArray(object?.stakeEntry) ? object.stakeEntry.map((e: any) => StakeEntry.fromJSON(e)) : [],
      output: isSet(object.output) ? String(object.output) : "",
    };
  },

  toJSON(message: QueryProvidersResponse): unknown {
    const obj: any = {};
    if (message.stakeEntry) {
      obj.stakeEntry = message.stakeEntry.map((e) => e ? StakeEntry.toJSON(e) : undefined);
    } else {
      obj.stakeEntry = [];
    }
    message.output !== undefined && (obj.output = message.output);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryProvidersResponse>, I>>(base?: I): QueryProvidersResponse {
    return QueryProvidersResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryProvidersResponse>, I>>(object: I): QueryProvidersResponse {
    const message = createBaseQueryProvidersResponse();
    message.stakeEntry = object.stakeEntry?.map((e) => StakeEntry.fromPartial(e)) || [];
    message.output = object.output ?? "";
    return message;
  },
};

function createBaseQueryGetPairingRequest(): QueryGetPairingRequest {
  return { chainID: "", client: "" };
}

export const QueryGetPairingRequest = {
  encode(message: QueryGetPairingRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.chainID !== "") {
      writer.uint32(10).string(message.chainID);
    }
    if (message.client !== "") {
      writer.uint32(18).string(message.client);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetPairingRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetPairingRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.chainID = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.client = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryGetPairingRequest {
    return {
      chainID: isSet(object.chainID) ? String(object.chainID) : "",
      client: isSet(object.client) ? String(object.client) : "",
    };
  },

  toJSON(message: QueryGetPairingRequest): unknown {
    const obj: any = {};
    message.chainID !== undefined && (obj.chainID = message.chainID);
    message.client !== undefined && (obj.client = message.client);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryGetPairingRequest>, I>>(base?: I): QueryGetPairingRequest {
    return QueryGetPairingRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetPairingRequest>, I>>(object: I): QueryGetPairingRequest {
    const message = createBaseQueryGetPairingRequest();
    message.chainID = object.chainID ?? "";
    message.client = object.client ?? "";
    return message;
  },
};

function createBaseQueryGetPairingResponse(): QueryGetPairingResponse {
  return {
    providers: [],
    currentEpoch: Long.UZERO,
    timeLeftToNextPairing: Long.UZERO,
    specLastUpdatedBlock: Long.UZERO,
    blockOfNextPairing: Long.UZERO,
  };
}

export const QueryGetPairingResponse = {
  encode(message: QueryGetPairingResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.providers) {
      StakeEntry.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (!message.currentEpoch.isZero()) {
      writer.uint32(16).uint64(message.currentEpoch);
    }
    if (!message.timeLeftToNextPairing.isZero()) {
      writer.uint32(24).uint64(message.timeLeftToNextPairing);
    }
    if (!message.specLastUpdatedBlock.isZero()) {
      writer.uint32(32).uint64(message.specLastUpdatedBlock);
    }
    if (!message.blockOfNextPairing.isZero()) {
      writer.uint32(40).uint64(message.blockOfNextPairing);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetPairingResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetPairingResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.providers.push(StakeEntry.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag != 16) {
            break;
          }

          message.currentEpoch = reader.uint64() as Long;
          continue;
        case 3:
          if (tag != 24) {
            break;
          }

          message.timeLeftToNextPairing = reader.uint64() as Long;
          continue;
        case 4:
          if (tag != 32) {
            break;
          }

          message.specLastUpdatedBlock = reader.uint64() as Long;
          continue;
        case 5:
          if (tag != 40) {
            break;
          }

          message.blockOfNextPairing = reader.uint64() as Long;
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryGetPairingResponse {
    return {
      providers: Array.isArray(object?.providers) ? object.providers.map((e: any) => StakeEntry.fromJSON(e)) : [],
      currentEpoch: isSet(object.currentEpoch) ? Long.fromValue(object.currentEpoch) : Long.UZERO,
      timeLeftToNextPairing: isSet(object.timeLeftToNextPairing)
        ? Long.fromValue(object.timeLeftToNextPairing)
        : Long.UZERO,
      specLastUpdatedBlock: isSet(object.specLastUpdatedBlock)
        ? Long.fromValue(object.specLastUpdatedBlock)
        : Long.UZERO,
      blockOfNextPairing: isSet(object.blockOfNextPairing) ? Long.fromValue(object.blockOfNextPairing) : Long.UZERO,
    };
  },

  toJSON(message: QueryGetPairingResponse): unknown {
    const obj: any = {};
    if (message.providers) {
      obj.providers = message.providers.map((e) => e ? StakeEntry.toJSON(e) : undefined);
    } else {
      obj.providers = [];
    }
    message.currentEpoch !== undefined && (obj.currentEpoch = (message.currentEpoch || Long.UZERO).toString());
    message.timeLeftToNextPairing !== undefined &&
      (obj.timeLeftToNextPairing = (message.timeLeftToNextPairing || Long.UZERO).toString());
    message.specLastUpdatedBlock !== undefined &&
      (obj.specLastUpdatedBlock = (message.specLastUpdatedBlock || Long.UZERO).toString());
    message.blockOfNextPairing !== undefined &&
      (obj.blockOfNextPairing = (message.blockOfNextPairing || Long.UZERO).toString());
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryGetPairingResponse>, I>>(base?: I): QueryGetPairingResponse {
    return QueryGetPairingResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetPairingResponse>, I>>(object: I): QueryGetPairingResponse {
    const message = createBaseQueryGetPairingResponse();
    message.providers = object.providers?.map((e) => StakeEntry.fromPartial(e)) || [];
    message.currentEpoch = (object.currentEpoch !== undefined && object.currentEpoch !== null)
      ? Long.fromValue(object.currentEpoch)
      : Long.UZERO;
    message.timeLeftToNextPairing =
      (object.timeLeftToNextPairing !== undefined && object.timeLeftToNextPairing !== null)
        ? Long.fromValue(object.timeLeftToNextPairing)
        : Long.UZERO;
    message.specLastUpdatedBlock = (object.specLastUpdatedBlock !== undefined && object.specLastUpdatedBlock !== null)
      ? Long.fromValue(object.specLastUpdatedBlock)
      : Long.UZERO;
    message.blockOfNextPairing = (object.blockOfNextPairing !== undefined && object.blockOfNextPairing !== null)
      ? Long.fromValue(object.blockOfNextPairing)
      : Long.UZERO;
    return message;
  },
};

function createBaseQueryVerifyPairingRequest(): QueryVerifyPairingRequest {
  return { chainID: "", client: "", provider: "", block: Long.UZERO };
}

export const QueryVerifyPairingRequest = {
  encode(message: QueryVerifyPairingRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.chainID !== "") {
      writer.uint32(10).string(message.chainID);
    }
    if (message.client !== "") {
      writer.uint32(18).string(message.client);
    }
    if (message.provider !== "") {
      writer.uint32(26).string(message.provider);
    }
    if (!message.block.isZero()) {
      writer.uint32(32).uint64(message.block);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryVerifyPairingRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryVerifyPairingRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.chainID = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.client = reader.string();
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.provider = reader.string();
          continue;
        case 4:
          if (tag != 32) {
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

  fromJSON(object: any): QueryVerifyPairingRequest {
    return {
      chainID: isSet(object.chainID) ? String(object.chainID) : "",
      client: isSet(object.client) ? String(object.client) : "",
      provider: isSet(object.provider) ? String(object.provider) : "",
      block: isSet(object.block) ? Long.fromValue(object.block) : Long.UZERO,
    };
  },

  toJSON(message: QueryVerifyPairingRequest): unknown {
    const obj: any = {};
    message.chainID !== undefined && (obj.chainID = message.chainID);
    message.client !== undefined && (obj.client = message.client);
    message.provider !== undefined && (obj.provider = message.provider);
    message.block !== undefined && (obj.block = (message.block || Long.UZERO).toString());
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryVerifyPairingRequest>, I>>(base?: I): QueryVerifyPairingRequest {
    return QueryVerifyPairingRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryVerifyPairingRequest>, I>>(object: I): QueryVerifyPairingRequest {
    const message = createBaseQueryVerifyPairingRequest();
    message.chainID = object.chainID ?? "";
    message.client = object.client ?? "";
    message.provider = object.provider ?? "";
    message.block = (object.block !== undefined && object.block !== null) ? Long.fromValue(object.block) : Long.UZERO;
    return message;
  },
};

function createBaseQueryVerifyPairingResponse(): QueryVerifyPairingResponse {
  return { valid: false, pairedProviders: Long.UZERO, cuPerEpoch: Long.UZERO, projectId: "" };
}

export const QueryVerifyPairingResponse = {
  encode(message: QueryVerifyPairingResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.valid === true) {
      writer.uint32(8).bool(message.valid);
    }
    if (!message.pairedProviders.isZero()) {
      writer.uint32(24).uint64(message.pairedProviders);
    }
    if (!message.cuPerEpoch.isZero()) {
      writer.uint32(32).uint64(message.cuPerEpoch);
    }
    if (message.projectId !== "") {
      writer.uint32(42).string(message.projectId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryVerifyPairingResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryVerifyPairingResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 8) {
            break;
          }

          message.valid = reader.bool();
          continue;
        case 3:
          if (tag != 24) {
            break;
          }

          message.pairedProviders = reader.uint64() as Long;
          continue;
        case 4:
          if (tag != 32) {
            break;
          }

          message.cuPerEpoch = reader.uint64() as Long;
          continue;
        case 5:
          if (tag != 42) {
            break;
          }

          message.projectId = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryVerifyPairingResponse {
    return {
      valid: isSet(object.valid) ? Boolean(object.valid) : false,
      pairedProviders: isSet(object.pairedProviders) ? Long.fromValue(object.pairedProviders) : Long.UZERO,
      cuPerEpoch: isSet(object.cuPerEpoch) ? Long.fromValue(object.cuPerEpoch) : Long.UZERO,
      projectId: isSet(object.projectId) ? String(object.projectId) : "",
    };
  },

  toJSON(message: QueryVerifyPairingResponse): unknown {
    const obj: any = {};
    message.valid !== undefined && (obj.valid = message.valid);
    message.pairedProviders !== undefined && (obj.pairedProviders = (message.pairedProviders || Long.UZERO).toString());
    message.cuPerEpoch !== undefined && (obj.cuPerEpoch = (message.cuPerEpoch || Long.UZERO).toString());
    message.projectId !== undefined && (obj.projectId = message.projectId);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryVerifyPairingResponse>, I>>(base?: I): QueryVerifyPairingResponse {
    return QueryVerifyPairingResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryVerifyPairingResponse>, I>>(object: I): QueryVerifyPairingResponse {
    const message = createBaseQueryVerifyPairingResponse();
    message.valid = object.valid ?? false;
    message.pairedProviders = (object.pairedProviders !== undefined && object.pairedProviders !== null)
      ? Long.fromValue(object.pairedProviders)
      : Long.UZERO;
    message.cuPerEpoch = (object.cuPerEpoch !== undefined && object.cuPerEpoch !== null)
      ? Long.fromValue(object.cuPerEpoch)
      : Long.UZERO;
    message.projectId = object.projectId ?? "";
    return message;
  },
};

function createBaseQueryGetUniquePaymentStorageClientProviderRequest(): QueryGetUniquePaymentStorageClientProviderRequest {
  return { index: "" };
}

export const QueryGetUniquePaymentStorageClientProviderRequest = {
  encode(
    message: QueryGetUniquePaymentStorageClientProviderRequest,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetUniquePaymentStorageClientProviderRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetUniquePaymentStorageClientProviderRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.index = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryGetUniquePaymentStorageClientProviderRequest {
    return { index: isSet(object.index) ? String(object.index) : "" };
  },

  toJSON(message: QueryGetUniquePaymentStorageClientProviderRequest): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryGetUniquePaymentStorageClientProviderRequest>, I>>(
    base?: I,
  ): QueryGetUniquePaymentStorageClientProviderRequest {
    return QueryGetUniquePaymentStorageClientProviderRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetUniquePaymentStorageClientProviderRequest>, I>>(
    object: I,
  ): QueryGetUniquePaymentStorageClientProviderRequest {
    const message = createBaseQueryGetUniquePaymentStorageClientProviderRequest();
    message.index = object.index ?? "";
    return message;
  },
};

function createBaseQueryGetUniquePaymentStorageClientProviderResponse(): QueryGetUniquePaymentStorageClientProviderResponse {
  return { uniquePaymentStorageClientProvider: undefined };
}

export const QueryGetUniquePaymentStorageClientProviderResponse = {
  encode(
    message: QueryGetUniquePaymentStorageClientProviderResponse,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.uniquePaymentStorageClientProvider !== undefined) {
      UniquePaymentStorageClientProvider.encode(message.uniquePaymentStorageClientProvider, writer.uint32(10).fork())
        .ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetUniquePaymentStorageClientProviderResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetUniquePaymentStorageClientProviderResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.uniquePaymentStorageClientProvider = UniquePaymentStorageClientProvider.decode(
            reader,
            reader.uint32(),
          );
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryGetUniquePaymentStorageClientProviderResponse {
    return {
      uniquePaymentStorageClientProvider: isSet(object.uniquePaymentStorageClientProvider)
        ? UniquePaymentStorageClientProvider.fromJSON(object.uniquePaymentStorageClientProvider)
        : undefined,
    };
  },

  toJSON(message: QueryGetUniquePaymentStorageClientProviderResponse): unknown {
    const obj: any = {};
    message.uniquePaymentStorageClientProvider !== undefined &&
      (obj.uniquePaymentStorageClientProvider = message.uniquePaymentStorageClientProvider
        ? UniquePaymentStorageClientProvider.toJSON(message.uniquePaymentStorageClientProvider)
        : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryGetUniquePaymentStorageClientProviderResponse>, I>>(
    base?: I,
  ): QueryGetUniquePaymentStorageClientProviderResponse {
    return QueryGetUniquePaymentStorageClientProviderResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetUniquePaymentStorageClientProviderResponse>, I>>(
    object: I,
  ): QueryGetUniquePaymentStorageClientProviderResponse {
    const message = createBaseQueryGetUniquePaymentStorageClientProviderResponse();
    message.uniquePaymentStorageClientProvider =
      (object.uniquePaymentStorageClientProvider !== undefined && object.uniquePaymentStorageClientProvider !== null)
        ? UniquePaymentStorageClientProvider.fromPartial(object.uniquePaymentStorageClientProvider)
        : undefined;
    return message;
  },
};

function createBaseQueryAllUniquePaymentStorageClientProviderRequest(): QueryAllUniquePaymentStorageClientProviderRequest {
  return { pagination: undefined };
}

export const QueryAllUniquePaymentStorageClientProviderRequest = {
  encode(
    message: QueryAllUniquePaymentStorageClientProviderRequest,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllUniquePaymentStorageClientProviderRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAllUniquePaymentStorageClientProviderRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.pagination = PageRequest.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryAllUniquePaymentStorageClientProviderRequest {
    return { pagination: isSet(object.pagination) ? PageRequest.fromJSON(object.pagination) : undefined };
  },

  toJSON(message: QueryAllUniquePaymentStorageClientProviderRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination ? PageRequest.toJSON(message.pagination) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryAllUniquePaymentStorageClientProviderRequest>, I>>(
    base?: I,
  ): QueryAllUniquePaymentStorageClientProviderRequest {
    return QueryAllUniquePaymentStorageClientProviderRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryAllUniquePaymentStorageClientProviderRequest>, I>>(
    object: I,
  ): QueryAllUniquePaymentStorageClientProviderRequest {
    const message = createBaseQueryAllUniquePaymentStorageClientProviderRequest();
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageRequest.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseQueryAllUniquePaymentStorageClientProviderResponse(): QueryAllUniquePaymentStorageClientProviderResponse {
  return { uniquePaymentStorageClientProvider: [], pagination: undefined };
}

export const QueryAllUniquePaymentStorageClientProviderResponse = {
  encode(
    message: QueryAllUniquePaymentStorageClientProviderResponse,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    for (const v of message.uniquePaymentStorageClientProvider) {
      UniquePaymentStorageClientProvider.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllUniquePaymentStorageClientProviderResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAllUniquePaymentStorageClientProviderResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.uniquePaymentStorageClientProvider.push(
            UniquePaymentStorageClientProvider.decode(reader, reader.uint32()),
          );
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.pagination = PageResponse.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryAllUniquePaymentStorageClientProviderResponse {
    return {
      uniquePaymentStorageClientProvider: Array.isArray(object?.uniquePaymentStorageClientProvider)
        ? object.uniquePaymentStorageClientProvider.map((e: any) => UniquePaymentStorageClientProvider.fromJSON(e))
        : [],
      pagination: isSet(object.pagination) ? PageResponse.fromJSON(object.pagination) : undefined,
    };
  },

  toJSON(message: QueryAllUniquePaymentStorageClientProviderResponse): unknown {
    const obj: any = {};
    if (message.uniquePaymentStorageClientProvider) {
      obj.uniquePaymentStorageClientProvider = message.uniquePaymentStorageClientProvider.map((e) =>
        e ? UniquePaymentStorageClientProvider.toJSON(e) : undefined
      );
    } else {
      obj.uniquePaymentStorageClientProvider = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination ? PageResponse.toJSON(message.pagination) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryAllUniquePaymentStorageClientProviderResponse>, I>>(
    base?: I,
  ): QueryAllUniquePaymentStorageClientProviderResponse {
    return QueryAllUniquePaymentStorageClientProviderResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryAllUniquePaymentStorageClientProviderResponse>, I>>(
    object: I,
  ): QueryAllUniquePaymentStorageClientProviderResponse {
    const message = createBaseQueryAllUniquePaymentStorageClientProviderResponse();
    message.uniquePaymentStorageClientProvider =
      object.uniquePaymentStorageClientProvider?.map((e) => UniquePaymentStorageClientProvider.fromPartial(e)) || [];
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageResponse.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseQueryGetProviderPaymentStorageRequest(): QueryGetProviderPaymentStorageRequest {
  return { index: "" };
}

export const QueryGetProviderPaymentStorageRequest = {
  encode(message: QueryGetProviderPaymentStorageRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetProviderPaymentStorageRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetProviderPaymentStorageRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.index = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryGetProviderPaymentStorageRequest {
    return { index: isSet(object.index) ? String(object.index) : "" };
  },

  toJSON(message: QueryGetProviderPaymentStorageRequest): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryGetProviderPaymentStorageRequest>, I>>(
    base?: I,
  ): QueryGetProviderPaymentStorageRequest {
    return QueryGetProviderPaymentStorageRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetProviderPaymentStorageRequest>, I>>(
    object: I,
  ): QueryGetProviderPaymentStorageRequest {
    const message = createBaseQueryGetProviderPaymentStorageRequest();
    message.index = object.index ?? "";
    return message;
  },
};

function createBaseQueryGetProviderPaymentStorageResponse(): QueryGetProviderPaymentStorageResponse {
  return { providerPaymentStorage: undefined };
}

export const QueryGetProviderPaymentStorageResponse = {
  encode(message: QueryGetProviderPaymentStorageResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.providerPaymentStorage !== undefined) {
      ProviderPaymentStorage.encode(message.providerPaymentStorage, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetProviderPaymentStorageResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetProviderPaymentStorageResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.providerPaymentStorage = ProviderPaymentStorage.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryGetProviderPaymentStorageResponse {
    return {
      providerPaymentStorage: isSet(object.providerPaymentStorage)
        ? ProviderPaymentStorage.fromJSON(object.providerPaymentStorage)
        : undefined,
    };
  },

  toJSON(message: QueryGetProviderPaymentStorageResponse): unknown {
    const obj: any = {};
    message.providerPaymentStorage !== undefined && (obj.providerPaymentStorage = message.providerPaymentStorage
      ? ProviderPaymentStorage.toJSON(message.providerPaymentStorage)
      : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryGetProviderPaymentStorageResponse>, I>>(
    base?: I,
  ): QueryGetProviderPaymentStorageResponse {
    return QueryGetProviderPaymentStorageResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetProviderPaymentStorageResponse>, I>>(
    object: I,
  ): QueryGetProviderPaymentStorageResponse {
    const message = createBaseQueryGetProviderPaymentStorageResponse();
    message.providerPaymentStorage =
      (object.providerPaymentStorage !== undefined && object.providerPaymentStorage !== null)
        ? ProviderPaymentStorage.fromPartial(object.providerPaymentStorage)
        : undefined;
    return message;
  },
};

function createBaseQueryAllProviderPaymentStorageRequest(): QueryAllProviderPaymentStorageRequest {
  return { pagination: undefined };
}

export const QueryAllProviderPaymentStorageRequest = {
  encode(message: QueryAllProviderPaymentStorageRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllProviderPaymentStorageRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAllProviderPaymentStorageRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.pagination = PageRequest.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryAllProviderPaymentStorageRequest {
    return { pagination: isSet(object.pagination) ? PageRequest.fromJSON(object.pagination) : undefined };
  },

  toJSON(message: QueryAllProviderPaymentStorageRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination ? PageRequest.toJSON(message.pagination) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryAllProviderPaymentStorageRequest>, I>>(
    base?: I,
  ): QueryAllProviderPaymentStorageRequest {
    return QueryAllProviderPaymentStorageRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryAllProviderPaymentStorageRequest>, I>>(
    object: I,
  ): QueryAllProviderPaymentStorageRequest {
    const message = createBaseQueryAllProviderPaymentStorageRequest();
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageRequest.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseQueryAllProviderPaymentStorageResponse(): QueryAllProviderPaymentStorageResponse {
  return { providerPaymentStorage: [], pagination: undefined };
}

export const QueryAllProviderPaymentStorageResponse = {
  encode(message: QueryAllProviderPaymentStorageResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.providerPaymentStorage) {
      ProviderPaymentStorage.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllProviderPaymentStorageResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAllProviderPaymentStorageResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.providerPaymentStorage.push(ProviderPaymentStorage.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.pagination = PageResponse.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryAllProviderPaymentStorageResponse {
    return {
      providerPaymentStorage: Array.isArray(object?.providerPaymentStorage)
        ? object.providerPaymentStorage.map((e: any) => ProviderPaymentStorage.fromJSON(e))
        : [],
      pagination: isSet(object.pagination) ? PageResponse.fromJSON(object.pagination) : undefined,
    };
  },

  toJSON(message: QueryAllProviderPaymentStorageResponse): unknown {
    const obj: any = {};
    if (message.providerPaymentStorage) {
      obj.providerPaymentStorage = message.providerPaymentStorage.map((e) =>
        e ? ProviderPaymentStorage.toJSON(e) : undefined
      );
    } else {
      obj.providerPaymentStorage = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination ? PageResponse.toJSON(message.pagination) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryAllProviderPaymentStorageResponse>, I>>(
    base?: I,
  ): QueryAllProviderPaymentStorageResponse {
    return QueryAllProviderPaymentStorageResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryAllProviderPaymentStorageResponse>, I>>(
    object: I,
  ): QueryAllProviderPaymentStorageResponse {
    const message = createBaseQueryAllProviderPaymentStorageResponse();
    message.providerPaymentStorage = object.providerPaymentStorage?.map((e) => ProviderPaymentStorage.fromPartial(e)) ||
      [];
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageResponse.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseQueryGetEpochPaymentsRequest(): QueryGetEpochPaymentsRequest {
  return { index: "" };
}

export const QueryGetEpochPaymentsRequest = {
  encode(message: QueryGetEpochPaymentsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetEpochPaymentsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetEpochPaymentsRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.index = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryGetEpochPaymentsRequest {
    return { index: isSet(object.index) ? String(object.index) : "" };
  },

  toJSON(message: QueryGetEpochPaymentsRequest): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryGetEpochPaymentsRequest>, I>>(base?: I): QueryGetEpochPaymentsRequest {
    return QueryGetEpochPaymentsRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetEpochPaymentsRequest>, I>>(object: I): QueryGetEpochPaymentsRequest {
    const message = createBaseQueryGetEpochPaymentsRequest();
    message.index = object.index ?? "";
    return message;
  },
};

function createBaseQueryGetEpochPaymentsResponse(): QueryGetEpochPaymentsResponse {
  return { epochPayments: undefined };
}

export const QueryGetEpochPaymentsResponse = {
  encode(message: QueryGetEpochPaymentsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.epochPayments !== undefined) {
      EpochPayments.encode(message.epochPayments, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetEpochPaymentsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetEpochPaymentsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.epochPayments = EpochPayments.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryGetEpochPaymentsResponse {
    return { epochPayments: isSet(object.epochPayments) ? EpochPayments.fromJSON(object.epochPayments) : undefined };
  },

  toJSON(message: QueryGetEpochPaymentsResponse): unknown {
    const obj: any = {};
    message.epochPayments !== undefined &&
      (obj.epochPayments = message.epochPayments ? EpochPayments.toJSON(message.epochPayments) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryGetEpochPaymentsResponse>, I>>(base?: I): QueryGetEpochPaymentsResponse {
    return QueryGetEpochPaymentsResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetEpochPaymentsResponse>, I>>(
    object: I,
  ): QueryGetEpochPaymentsResponse {
    const message = createBaseQueryGetEpochPaymentsResponse();
    message.epochPayments = (object.epochPayments !== undefined && object.epochPayments !== null)
      ? EpochPayments.fromPartial(object.epochPayments)
      : undefined;
    return message;
  },
};

function createBaseQueryAllEpochPaymentsRequest(): QueryAllEpochPaymentsRequest {
  return { pagination: undefined };
}

export const QueryAllEpochPaymentsRequest = {
  encode(message: QueryAllEpochPaymentsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllEpochPaymentsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAllEpochPaymentsRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.pagination = PageRequest.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryAllEpochPaymentsRequest {
    return { pagination: isSet(object.pagination) ? PageRequest.fromJSON(object.pagination) : undefined };
  },

  toJSON(message: QueryAllEpochPaymentsRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination ? PageRequest.toJSON(message.pagination) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryAllEpochPaymentsRequest>, I>>(base?: I): QueryAllEpochPaymentsRequest {
    return QueryAllEpochPaymentsRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryAllEpochPaymentsRequest>, I>>(object: I): QueryAllEpochPaymentsRequest {
    const message = createBaseQueryAllEpochPaymentsRequest();
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageRequest.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseQueryAllEpochPaymentsResponse(): QueryAllEpochPaymentsResponse {
  return { epochPayments: [], pagination: undefined };
}

export const QueryAllEpochPaymentsResponse = {
  encode(message: QueryAllEpochPaymentsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.epochPayments) {
      EpochPayments.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllEpochPaymentsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAllEpochPaymentsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.epochPayments.push(EpochPayments.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.pagination = PageResponse.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryAllEpochPaymentsResponse {
    return {
      epochPayments: Array.isArray(object?.epochPayments)
        ? object.epochPayments.map((e: any) => EpochPayments.fromJSON(e))
        : [],
      pagination: isSet(object.pagination) ? PageResponse.fromJSON(object.pagination) : undefined,
    };
  },

  toJSON(message: QueryAllEpochPaymentsResponse): unknown {
    const obj: any = {};
    if (message.epochPayments) {
      obj.epochPayments = message.epochPayments.map((e) => e ? EpochPayments.toJSON(e) : undefined);
    } else {
      obj.epochPayments = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination ? PageResponse.toJSON(message.pagination) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryAllEpochPaymentsResponse>, I>>(base?: I): QueryAllEpochPaymentsResponse {
    return QueryAllEpochPaymentsResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryAllEpochPaymentsResponse>, I>>(
    object: I,
  ): QueryAllEpochPaymentsResponse {
    const message = createBaseQueryAllEpochPaymentsResponse();
    message.epochPayments = object.epochPayments?.map((e) => EpochPayments.fromPartial(e)) || [];
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageResponse.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseQueryUserEntryRequest(): QueryUserEntryRequest {
  return { address: "", chainID: "", block: Long.UZERO };
}

export const QueryUserEntryRequest = {
  encode(message: QueryUserEntryRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.address !== "") {
      writer.uint32(10).string(message.address);
    }
    if (message.chainID !== "") {
      writer.uint32(18).string(message.chainID);
    }
    if (!message.block.isZero()) {
      writer.uint32(24).uint64(message.block);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryUserEntryRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryUserEntryRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.address = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.chainID = reader.string();
          continue;
        case 3:
          if (tag != 24) {
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

  fromJSON(object: any): QueryUserEntryRequest {
    return {
      address: isSet(object.address) ? String(object.address) : "",
      chainID: isSet(object.chainID) ? String(object.chainID) : "",
      block: isSet(object.block) ? Long.fromValue(object.block) : Long.UZERO,
    };
  },

  toJSON(message: QueryUserEntryRequest): unknown {
    const obj: any = {};
    message.address !== undefined && (obj.address = message.address);
    message.chainID !== undefined && (obj.chainID = message.chainID);
    message.block !== undefined && (obj.block = (message.block || Long.UZERO).toString());
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryUserEntryRequest>, I>>(base?: I): QueryUserEntryRequest {
    return QueryUserEntryRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryUserEntryRequest>, I>>(object: I): QueryUserEntryRequest {
    const message = createBaseQueryUserEntryRequest();
    message.address = object.address ?? "";
    message.chainID = object.chainID ?? "";
    message.block = (object.block !== undefined && object.block !== null) ? Long.fromValue(object.block) : Long.UZERO;
    return message;
  },
};

function createBaseQueryUserEntryResponse(): QueryUserEntryResponse {
  return { consumer: undefined, maxCU: Long.UZERO };
}

export const QueryUserEntryResponse = {
  encode(message: QueryUserEntryResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.consumer !== undefined) {
      StakeEntry.encode(message.consumer, writer.uint32(10).fork()).ldelim();
    }
    if (!message.maxCU.isZero()) {
      writer.uint32(16).uint64(message.maxCU);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryUserEntryResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryUserEntryResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.consumer = StakeEntry.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag != 16) {
            break;
          }

          message.maxCU = reader.uint64() as Long;
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryUserEntryResponse {
    return {
      consumer: isSet(object.consumer) ? StakeEntry.fromJSON(object.consumer) : undefined,
      maxCU: isSet(object.maxCU) ? Long.fromValue(object.maxCU) : Long.UZERO,
    };
  },

  toJSON(message: QueryUserEntryResponse): unknown {
    const obj: any = {};
    message.consumer !== undefined &&
      (obj.consumer = message.consumer ? StakeEntry.toJSON(message.consumer) : undefined);
    message.maxCU !== undefined && (obj.maxCU = (message.maxCU || Long.UZERO).toString());
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryUserEntryResponse>, I>>(base?: I): QueryUserEntryResponse {
    return QueryUserEntryResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryUserEntryResponse>, I>>(object: I): QueryUserEntryResponse {
    const message = createBaseQueryUserEntryResponse();
    message.consumer = (object.consumer !== undefined && object.consumer !== null)
      ? StakeEntry.fromPartial(object.consumer)
      : undefined;
    message.maxCU = (object.maxCU !== undefined && object.maxCU !== null) ? Long.fromValue(object.maxCU) : Long.UZERO;
    return message;
  },
};

function createBaseQueryStaticProvidersListRequest(): QueryStaticProvidersListRequest {
  return { chainID: "" };
}

export const QueryStaticProvidersListRequest = {
  encode(message: QueryStaticProvidersListRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.chainID !== "") {
      writer.uint32(10).string(message.chainID);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryStaticProvidersListRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryStaticProvidersListRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.chainID = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryStaticProvidersListRequest {
    return { chainID: isSet(object.chainID) ? String(object.chainID) : "" };
  },

  toJSON(message: QueryStaticProvidersListRequest): unknown {
    const obj: any = {};
    message.chainID !== undefined && (obj.chainID = message.chainID);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryStaticProvidersListRequest>, I>>(base?: I): QueryStaticProvidersListRequest {
    return QueryStaticProvidersListRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryStaticProvidersListRequest>, I>>(
    object: I,
  ): QueryStaticProvidersListRequest {
    const message = createBaseQueryStaticProvidersListRequest();
    message.chainID = object.chainID ?? "";
    return message;
  },
};

function createBaseQueryStaticProvidersListResponse(): QueryStaticProvidersListResponse {
  return { providers: [] };
}

export const QueryStaticProvidersListResponse = {
  encode(message: QueryStaticProvidersListResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.providers) {
      StakeEntry.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryStaticProvidersListResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryStaticProvidersListResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.providers.push(StakeEntry.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryStaticProvidersListResponse {
    return {
      providers: Array.isArray(object?.providers) ? object.providers.map((e: any) => StakeEntry.fromJSON(e)) : [],
    };
  },

  toJSON(message: QueryStaticProvidersListResponse): unknown {
    const obj: any = {};
    if (message.providers) {
      obj.providers = message.providers.map((e) => e ? StakeEntry.toJSON(e) : undefined);
    } else {
      obj.providers = [];
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryStaticProvidersListResponse>, I>>(
    base?: I,
  ): QueryStaticProvidersListResponse {
    return QueryStaticProvidersListResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryStaticProvidersListResponse>, I>>(
    object: I,
  ): QueryStaticProvidersListResponse {
    const message = createBaseQueryStaticProvidersListResponse();
    message.providers = object.providers?.map((e) => StakeEntry.fromPartial(e)) || [];
    return message;
  },
};

function createBaseQueryAccountInfoResponse(): QueryAccountInfoResponse {
  return { provider: [], frozen: [], consumer: [], unstaked: [], subscription: undefined, project: undefined };
}

export const QueryAccountInfoResponse = {
  encode(message: QueryAccountInfoResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.provider) {
      StakeEntry.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.frozen) {
      StakeEntry.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.consumer) {
      StakeEntry.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.unstaked) {
      StakeEntry.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    if (message.subscription !== undefined) {
      Subscription.encode(message.subscription, writer.uint32(42).fork()).ldelim();
    }
    if (message.project !== undefined) {
      Project.encode(message.project, writer.uint32(50).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAccountInfoResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAccountInfoResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.provider.push(StakeEntry.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.frozen.push(StakeEntry.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.consumer.push(StakeEntry.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.unstaked.push(StakeEntry.decode(reader, reader.uint32()));
          continue;
        case 5:
          if (tag != 42) {
            break;
          }

          message.subscription = Subscription.decode(reader, reader.uint32());
          continue;
        case 6:
          if (tag != 50) {
            break;
          }

          message.project = Project.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryAccountInfoResponse {
    return {
      provider: Array.isArray(object?.provider) ? object.provider.map((e: any) => StakeEntry.fromJSON(e)) : [],
      frozen: Array.isArray(object?.frozen) ? object.frozen.map((e: any) => StakeEntry.fromJSON(e)) : [],
      consumer: Array.isArray(object?.consumer) ? object.consumer.map((e: any) => StakeEntry.fromJSON(e)) : [],
      unstaked: Array.isArray(object?.unstaked) ? object.unstaked.map((e: any) => StakeEntry.fromJSON(e)) : [],
      subscription: isSet(object.subscription) ? Subscription.fromJSON(object.subscription) : undefined,
      project: isSet(object.project) ? Project.fromJSON(object.project) : undefined,
    };
  },

  toJSON(message: QueryAccountInfoResponse): unknown {
    const obj: any = {};
    if (message.provider) {
      obj.provider = message.provider.map((e) => e ? StakeEntry.toJSON(e) : undefined);
    } else {
      obj.provider = [];
    }
    if (message.frozen) {
      obj.frozen = message.frozen.map((e) => e ? StakeEntry.toJSON(e) : undefined);
    } else {
      obj.frozen = [];
    }
    if (message.consumer) {
      obj.consumer = message.consumer.map((e) => e ? StakeEntry.toJSON(e) : undefined);
    } else {
      obj.consumer = [];
    }
    if (message.unstaked) {
      obj.unstaked = message.unstaked.map((e) => e ? StakeEntry.toJSON(e) : undefined);
    } else {
      obj.unstaked = [];
    }
    message.subscription !== undefined &&
      (obj.subscription = message.subscription ? Subscription.toJSON(message.subscription) : undefined);
    message.project !== undefined && (obj.project = message.project ? Project.toJSON(message.project) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryAccountInfoResponse>, I>>(base?: I): QueryAccountInfoResponse {
    return QueryAccountInfoResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryAccountInfoResponse>, I>>(object: I): QueryAccountInfoResponse {
    const message = createBaseQueryAccountInfoResponse();
    message.provider = object.provider?.map((e) => StakeEntry.fromPartial(e)) || [];
    message.frozen = object.frozen?.map((e) => StakeEntry.fromPartial(e)) || [];
    message.consumer = object.consumer?.map((e) => StakeEntry.fromPartial(e)) || [];
    message.unstaked = object.unstaked?.map((e) => StakeEntry.fromPartial(e)) || [];
    message.subscription = (object.subscription !== undefined && object.subscription !== null)
      ? Subscription.fromPartial(object.subscription)
      : undefined;
    message.project = (object.project !== undefined && object.project !== null)
      ? Project.fromPartial(object.project)
      : undefined;
    return message;
  },
};

function createBaseQueryEffectivePolicyRequest(): QueryEffectivePolicyRequest {
  return { consumer: "", specID: "" };
}

export const QueryEffectivePolicyRequest = {
  encode(message: QueryEffectivePolicyRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.consumer !== "") {
      writer.uint32(10).string(message.consumer);
    }
    if (message.specID !== "") {
      writer.uint32(18).string(message.specID);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryEffectivePolicyRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryEffectivePolicyRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.consumer = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.specID = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryEffectivePolicyRequest {
    return {
      consumer: isSet(object.consumer) ? String(object.consumer) : "",
      specID: isSet(object.specID) ? String(object.specID) : "",
    };
  },

  toJSON(message: QueryEffectivePolicyRequest): unknown {
    const obj: any = {};
    message.consumer !== undefined && (obj.consumer = message.consumer);
    message.specID !== undefined && (obj.specID = message.specID);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryEffectivePolicyRequest>, I>>(base?: I): QueryEffectivePolicyRequest {
    return QueryEffectivePolicyRequest.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryEffectivePolicyRequest>, I>>(object: I): QueryEffectivePolicyRequest {
    const message = createBaseQueryEffectivePolicyRequest();
    message.consumer = object.consumer ?? "";
    message.specID = object.specID ?? "";
    return message;
  },
};

function createBaseQueryEffectivePolicyResponse(): QueryEffectivePolicyResponse {
  return { policy: undefined };
}

export const QueryEffectivePolicyResponse = {
  encode(message: QueryEffectivePolicyResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.policy !== undefined) {
      Policy.encode(message.policy, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryEffectivePolicyResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryEffectivePolicyResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.policy = Policy.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryEffectivePolicyResponse {
    return { policy: isSet(object.policy) ? Policy.fromJSON(object.policy) : undefined };
  },

  toJSON(message: QueryEffectivePolicyResponse): unknown {
    const obj: any = {};
    message.policy !== undefined && (obj.policy = message.policy ? Policy.toJSON(message.policy) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryEffectivePolicyResponse>, I>>(base?: I): QueryEffectivePolicyResponse {
    return QueryEffectivePolicyResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QueryEffectivePolicyResponse>, I>>(object: I): QueryEffectivePolicyResponse {
    const message = createBaseQueryEffectivePolicyResponse();
    message.policy = (object.policy !== undefined && object.policy !== null)
      ? Policy.fromPartial(object.policy)
      : undefined;
    return message;
  },
};

function createBaseQuerySdkPairingResponse(): QuerySdkPairingResponse {
  return { pairing: undefined, maxCu: Long.UZERO, spec: undefined };
}

export const QuerySdkPairingResponse = {
  encode(message: QuerySdkPairingResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pairing !== undefined) {
      QueryGetPairingResponse.encode(message.pairing, writer.uint32(10).fork()).ldelim();
    }
    if (!message.maxCu.isZero()) {
      writer.uint32(16).uint64(message.maxCu);
    }
    if (message.spec !== undefined) {
      Spec.encode(message.spec, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QuerySdkPairingResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQuerySdkPairingResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.pairing = QueryGetPairingResponse.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag != 16) {
            break;
          }

          message.maxCu = reader.uint64() as Long;
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.spec = Spec.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QuerySdkPairingResponse {
    return {
      pairing: isSet(object.pairing) ? QueryGetPairingResponse.fromJSON(object.pairing) : undefined,
      maxCu: isSet(object.maxCu) ? Long.fromValue(object.maxCu) : Long.UZERO,
      spec: isSet(object.spec) ? Spec.fromJSON(object.spec) : undefined,
    };
  },

  toJSON(message: QuerySdkPairingResponse): unknown {
    const obj: any = {};
    message.pairing !== undefined &&
      (obj.pairing = message.pairing ? QueryGetPairingResponse.toJSON(message.pairing) : undefined);
    message.maxCu !== undefined && (obj.maxCu = (message.maxCu || Long.UZERO).toString());
    message.spec !== undefined && (obj.spec = message.spec ? Spec.toJSON(message.spec) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<QuerySdkPairingResponse>, I>>(base?: I): QuerySdkPairingResponse {
    return QuerySdkPairingResponse.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<QuerySdkPairingResponse>, I>>(object: I): QuerySdkPairingResponse {
    const message = createBaseQuerySdkPairingResponse();
    message.pairing = (object.pairing !== undefined && object.pairing !== null)
      ? QueryGetPairingResponse.fromPartial(object.pairing)
      : undefined;
    message.maxCu = (object.maxCu !== undefined && object.maxCu !== null) ? Long.fromValue(object.maxCu) : Long.UZERO;
    message.spec = (object.spec !== undefined && object.spec !== null) ? Spec.fromPartial(object.spec) : undefined;
    return message;
  },
};

/** Query defines the gRPC querier service. */
export interface Query {
  /** Parameters queries the parameters of the module. */
  Params(request: QueryParamsRequest): Promise<QueryParamsResponse>;
  /** Queries a list of Providers items. */
  Providers(request: QueryProvidersRequest): Promise<QueryProvidersResponse>;
  /** Queries a list of GetPairing items. */
  GetPairing(request: QueryGetPairingRequest): Promise<QueryGetPairingResponse>;
  /** Queries a list of VerifyPairing items. */
  VerifyPairing(request: QueryVerifyPairingRequest): Promise<QueryVerifyPairingResponse>;
  /** Queries a UniquePaymentStorageClientProvider by index. */
  UniquePaymentStorageClientProvider(
    request: QueryGetUniquePaymentStorageClientProviderRequest,
  ): Promise<QueryGetUniquePaymentStorageClientProviderResponse>;
  /** Queries a list of UniquePaymentStorageClientProvider items. */
  UniquePaymentStorageClientProviderAll(
    request: QueryAllUniquePaymentStorageClientProviderRequest,
  ): Promise<QueryAllUniquePaymentStorageClientProviderResponse>;
  /** Queries a ProviderPaymentStorage by index. */
  ProviderPaymentStorage(
    request: QueryGetProviderPaymentStorageRequest,
  ): Promise<QueryGetProviderPaymentStorageResponse>;
  /** Queries a list of ProviderPaymentStorage items. */
  ProviderPaymentStorageAll(
    request: QueryAllProviderPaymentStorageRequest,
  ): Promise<QueryAllProviderPaymentStorageResponse>;
  /** Queries a EpochPayments by index. */
  EpochPayments(request: QueryGetEpochPaymentsRequest): Promise<QueryGetEpochPaymentsResponse>;
  /** Queries a list of EpochPayments items. */
  EpochPaymentsAll(request: QueryAllEpochPaymentsRequest): Promise<QueryAllEpochPaymentsResponse>;
  /** Queries a UserEntry items. */
  UserEntry(request: QueryUserEntryRequest): Promise<QueryUserEntryResponse>;
  /** Queries a list of StaticProvidersList items. */
  StaticProvidersList(request: QueryStaticProvidersListRequest): Promise<QueryStaticProvidersListResponse>;
  /** Queries a list of EffectivePolicy items. */
  EffectivePolicy(request: QueryEffectivePolicyRequest): Promise<QueryEffectivePolicyResponse>;
}

export class QueryClientImpl implements Query {
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || "lavanet.lava.pairing.Query";
    this.rpc = rpc;
    this.Params = this.Params.bind(this);
    this.Providers = this.Providers.bind(this);
    this.GetPairing = this.GetPairing.bind(this);
    this.VerifyPairing = this.VerifyPairing.bind(this);
    this.UniquePaymentStorageClientProvider = this.UniquePaymentStorageClientProvider.bind(this);
    this.UniquePaymentStorageClientProviderAll = this.UniquePaymentStorageClientProviderAll.bind(this);
    this.ProviderPaymentStorage = this.ProviderPaymentStorage.bind(this);
    this.ProviderPaymentStorageAll = this.ProviderPaymentStorageAll.bind(this);
    this.EpochPayments = this.EpochPayments.bind(this);
    this.EpochPaymentsAll = this.EpochPaymentsAll.bind(this);
    this.UserEntry = this.UserEntry.bind(this);
    this.StaticProvidersList = this.StaticProvidersList.bind(this);
    this.EffectivePolicy = this.EffectivePolicy.bind(this);
  }
  Params(request: QueryParamsRequest): Promise<QueryParamsResponse> {
    const data = QueryParamsRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "Params", data);
    return promise.then((data) => QueryParamsResponse.decode(_m0.Reader.create(data)));
  }

  Providers(request: QueryProvidersRequest): Promise<QueryProvidersResponse> {
    const data = QueryProvidersRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "Providers", data);
    return promise.then((data) => QueryProvidersResponse.decode(_m0.Reader.create(data)));
  }

  GetPairing(request: QueryGetPairingRequest): Promise<QueryGetPairingResponse> {
    const data = QueryGetPairingRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "GetPairing", data);
    return promise.then((data) => QueryGetPairingResponse.decode(_m0.Reader.create(data)));
  }

  VerifyPairing(request: QueryVerifyPairingRequest): Promise<QueryVerifyPairingResponse> {
    const data = QueryVerifyPairingRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "VerifyPairing", data);
    return promise.then((data) => QueryVerifyPairingResponse.decode(_m0.Reader.create(data)));
  }

  UniquePaymentStorageClientProvider(
    request: QueryGetUniquePaymentStorageClientProviderRequest,
  ): Promise<QueryGetUniquePaymentStorageClientProviderResponse> {
    const data = QueryGetUniquePaymentStorageClientProviderRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "UniquePaymentStorageClientProvider", data);
    return promise.then((data) => QueryGetUniquePaymentStorageClientProviderResponse.decode(_m0.Reader.create(data)));
  }

  UniquePaymentStorageClientProviderAll(
    request: QueryAllUniquePaymentStorageClientProviderRequest,
  ): Promise<QueryAllUniquePaymentStorageClientProviderResponse> {
    const data = QueryAllUniquePaymentStorageClientProviderRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "UniquePaymentStorageClientProviderAll", data);
    return promise.then((data) => QueryAllUniquePaymentStorageClientProviderResponse.decode(_m0.Reader.create(data)));
  }

  ProviderPaymentStorage(
    request: QueryGetProviderPaymentStorageRequest,
  ): Promise<QueryGetProviderPaymentStorageResponse> {
    const data = QueryGetProviderPaymentStorageRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "ProviderPaymentStorage", data);
    return promise.then((data) => QueryGetProviderPaymentStorageResponse.decode(_m0.Reader.create(data)));
  }

  ProviderPaymentStorageAll(
    request: QueryAllProviderPaymentStorageRequest,
  ): Promise<QueryAllProviderPaymentStorageResponse> {
    const data = QueryAllProviderPaymentStorageRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "ProviderPaymentStorageAll", data);
    return promise.then((data) => QueryAllProviderPaymentStorageResponse.decode(_m0.Reader.create(data)));
  }

  EpochPayments(request: QueryGetEpochPaymentsRequest): Promise<QueryGetEpochPaymentsResponse> {
    const data = QueryGetEpochPaymentsRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "EpochPayments", data);
    return promise.then((data) => QueryGetEpochPaymentsResponse.decode(_m0.Reader.create(data)));
  }

  EpochPaymentsAll(request: QueryAllEpochPaymentsRequest): Promise<QueryAllEpochPaymentsResponse> {
    const data = QueryAllEpochPaymentsRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "EpochPaymentsAll", data);
    return promise.then((data) => QueryAllEpochPaymentsResponse.decode(_m0.Reader.create(data)));
  }

  UserEntry(request: QueryUserEntryRequest): Promise<QueryUserEntryResponse> {
    const data = QueryUserEntryRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "UserEntry", data);
    return promise.then((data) => QueryUserEntryResponse.decode(_m0.Reader.create(data)));
  }

  StaticProvidersList(request: QueryStaticProvidersListRequest): Promise<QueryStaticProvidersListResponse> {
    const data = QueryStaticProvidersListRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "StaticProvidersList", data);
    return promise.then((data) => QueryStaticProvidersListResponse.decode(_m0.Reader.create(data)));
  }

  EffectivePolicy(request: QueryEffectivePolicyRequest): Promise<QueryEffectivePolicyResponse> {
    const data = QueryEffectivePolicyRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "EffectivePolicy", data);
    return promise.then((data) => QueryEffectivePolicyResponse.decode(_m0.Reader.create(data)));
  }
}

interface Rpc {
  request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}

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

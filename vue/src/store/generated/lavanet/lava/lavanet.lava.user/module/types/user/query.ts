/* eslint-disable */
import { Reader, util, configure, Writer } from "protobufjs/minimal";
import * as Long from "long";
import { Params } from "../user/params";
import { UserStake } from "../user/user_stake";
import {
  PageRequest,
  PageResponse,
} from "../cosmos/base/query/v1beta1/pagination";
import { SpecStakeStorage } from "../user/spec_stake_storage";
import { BlockDeadlineForCallback } from "../user/block_deadline_for_callback";
import { UnstakingUsersAllSpecs } from "../user/unstaking_users_all_specs";
import { StakeStorage } from "../user/stake_storage";

export const protobufPackage = "lavanet.lava.user";

/** QueryParamsRequest is request type for the Query/Params RPC method. */
export interface QueryParamsRequest {}

/** QueryParamsResponse is response type for the Query/Params RPC method. */
export interface QueryParamsResponse {
  /** params holds all the parameters of this module. */
  params: Params | undefined;
}

export interface QueryGetUserStakeRequest {
  index: string;
}

export interface QueryGetUserStakeResponse {
  userStake: UserStake | undefined;
}

export interface QueryAllUserStakeRequest {
  pagination: PageRequest | undefined;
}

export interface QueryAllUserStakeResponse {
  userStake: UserStake[];
  pagination: PageResponse | undefined;
}

export interface QueryGetSpecStakeStorageRequest {
  index: string;
}

export interface QueryGetSpecStakeStorageResponse {
  specStakeStorage: SpecStakeStorage | undefined;
}

export interface QueryAllSpecStakeStorageRequest {
  pagination: PageRequest | undefined;
}

export interface QueryAllSpecStakeStorageResponse {
  specStakeStorage: SpecStakeStorage[];
  pagination: PageResponse | undefined;
}

export interface QueryGetBlockDeadlineForCallbackRequest {}

export interface QueryGetBlockDeadlineForCallbackResponse {
  BlockDeadlineForCallback: BlockDeadlineForCallback | undefined;
}

export interface QueryGetUnstakingUsersAllSpecsRequest {
  id: number;
}

export interface QueryGetUnstakingUsersAllSpecsResponse {
  UnstakingUsersAllSpecs: UnstakingUsersAllSpecs | undefined;
}

export interface QueryAllUnstakingUsersAllSpecsRequest {
  pagination: PageRequest | undefined;
}

export interface QueryAllUnstakingUsersAllSpecsResponse {
  UnstakingUsersAllSpecs: UnstakingUsersAllSpecs[];
  pagination: PageResponse | undefined;
}

export interface QueryStakedUsersRequest {
  specName: string;
}

export interface QueryStakedUsersResponse {
  stakeStorage: StakeStorage | undefined;
  output: string;
}

const baseQueryParamsRequest: object = {};

export const QueryParamsRequest = {
  encode(_: QueryParamsRequest, writer: Writer = Writer.create()): Writer {
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryParamsRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseQueryParamsRequest } as QueryParamsRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(_: any): QueryParamsRequest {
    const message = { ...baseQueryParamsRequest } as QueryParamsRequest;
    return message;
  },

  toJSON(_: QueryParamsRequest): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(_: DeepPartial<QueryParamsRequest>): QueryParamsRequest {
    const message = { ...baseQueryParamsRequest } as QueryParamsRequest;
    return message;
  },
};

const baseQueryParamsResponse: object = {};

export const QueryParamsResponse = {
  encode(
    message: QueryParamsResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryParamsResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseQueryParamsResponse } as QueryParamsResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.params = Params.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryParamsResponse {
    const message = { ...baseQueryParamsResponse } as QueryParamsResponse;
    if (object.params !== undefined && object.params !== null) {
      message.params = Params.fromJSON(object.params);
    } else {
      message.params = undefined;
    }
    return message;
  },

  toJSON(message: QueryParamsResponse): unknown {
    const obj: any = {};
    message.params !== undefined &&
      (obj.params = message.params ? Params.toJSON(message.params) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<QueryParamsResponse>): QueryParamsResponse {
    const message = { ...baseQueryParamsResponse } as QueryParamsResponse;
    if (object.params !== undefined && object.params !== null) {
      message.params = Params.fromPartial(object.params);
    } else {
      message.params = undefined;
    }
    return message;
  },
};

const baseQueryGetUserStakeRequest: object = { index: "" };

export const QueryGetUserStakeRequest = {
  encode(
    message: QueryGetUserStakeRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetUserStakeRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetUserStakeRequest,
    } as QueryGetUserStakeRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.index = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetUserStakeRequest {
    const message = {
      ...baseQueryGetUserStakeRequest,
    } as QueryGetUserStakeRequest;
    if (object.index !== undefined && object.index !== null) {
      message.index = String(object.index);
    } else {
      message.index = "";
    }
    return message;
  },

  toJSON(message: QueryGetUserStakeRequest): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetUserStakeRequest>
  ): QueryGetUserStakeRequest {
    const message = {
      ...baseQueryGetUserStakeRequest,
    } as QueryGetUserStakeRequest;
    if (object.index !== undefined && object.index !== null) {
      message.index = object.index;
    } else {
      message.index = "";
    }
    return message;
  },
};

const baseQueryGetUserStakeResponse: object = {};

export const QueryGetUserStakeResponse = {
  encode(
    message: QueryGetUserStakeResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.userStake !== undefined) {
      UserStake.encode(message.userStake, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetUserStakeResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetUserStakeResponse,
    } as QueryGetUserStakeResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.userStake = UserStake.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetUserStakeResponse {
    const message = {
      ...baseQueryGetUserStakeResponse,
    } as QueryGetUserStakeResponse;
    if (object.userStake !== undefined && object.userStake !== null) {
      message.userStake = UserStake.fromJSON(object.userStake);
    } else {
      message.userStake = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetUserStakeResponse): unknown {
    const obj: any = {};
    message.userStake !== undefined &&
      (obj.userStake = message.userStake
        ? UserStake.toJSON(message.userStake)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetUserStakeResponse>
  ): QueryGetUserStakeResponse {
    const message = {
      ...baseQueryGetUserStakeResponse,
    } as QueryGetUserStakeResponse;
    if (object.userStake !== undefined && object.userStake !== null) {
      message.userStake = UserStake.fromPartial(object.userStake);
    } else {
      message.userStake = undefined;
    }
    return message;
  },
};

const baseQueryAllUserStakeRequest: object = {};

export const QueryAllUserStakeRequest = {
  encode(
    message: QueryAllUserStakeRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryAllUserStakeRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllUserStakeRequest,
    } as QueryAllUserStakeRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.pagination = PageRequest.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryAllUserStakeRequest {
    const message = {
      ...baseQueryAllUserStakeRequest,
    } as QueryAllUserStakeRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllUserStakeRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllUserStakeRequest>
  ): QueryAllUserStakeRequest {
    const message = {
      ...baseQueryAllUserStakeRequest,
    } as QueryAllUserStakeRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryAllUserStakeResponse: object = {};

export const QueryAllUserStakeResponse = {
  encode(
    message: QueryAllUserStakeResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.userStake) {
      UserStake.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(
        message.pagination,
        writer.uint32(18).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryAllUserStakeResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllUserStakeResponse,
    } as QueryAllUserStakeResponse;
    message.userStake = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.userStake.push(UserStake.decode(reader, reader.uint32()));
          break;
        case 2:
          message.pagination = PageResponse.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryAllUserStakeResponse {
    const message = {
      ...baseQueryAllUserStakeResponse,
    } as QueryAllUserStakeResponse;
    message.userStake = [];
    if (object.userStake !== undefined && object.userStake !== null) {
      for (const e of object.userStake) {
        message.userStake.push(UserStake.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllUserStakeResponse): unknown {
    const obj: any = {};
    if (message.userStake) {
      obj.userStake = message.userStake.map((e) =>
        e ? UserStake.toJSON(e) : undefined
      );
    } else {
      obj.userStake = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllUserStakeResponse>
  ): QueryAllUserStakeResponse {
    const message = {
      ...baseQueryAllUserStakeResponse,
    } as QueryAllUserStakeResponse;
    message.userStake = [];
    if (object.userStake !== undefined && object.userStake !== null) {
      for (const e of object.userStake) {
        message.userStake.push(UserStake.fromPartial(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryGetSpecStakeStorageRequest: object = { index: "" };

export const QueryGetSpecStakeStorageRequest = {
  encode(
    message: QueryGetSpecStakeStorageRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetSpecStakeStorageRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetSpecStakeStorageRequest,
    } as QueryGetSpecStakeStorageRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.index = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetSpecStakeStorageRequest {
    const message = {
      ...baseQueryGetSpecStakeStorageRequest,
    } as QueryGetSpecStakeStorageRequest;
    if (object.index !== undefined && object.index !== null) {
      message.index = String(object.index);
    } else {
      message.index = "";
    }
    return message;
  },

  toJSON(message: QueryGetSpecStakeStorageRequest): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetSpecStakeStorageRequest>
  ): QueryGetSpecStakeStorageRequest {
    const message = {
      ...baseQueryGetSpecStakeStorageRequest,
    } as QueryGetSpecStakeStorageRequest;
    if (object.index !== undefined && object.index !== null) {
      message.index = object.index;
    } else {
      message.index = "";
    }
    return message;
  },
};

const baseQueryGetSpecStakeStorageResponse: object = {};

export const QueryGetSpecStakeStorageResponse = {
  encode(
    message: QueryGetSpecStakeStorageResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.specStakeStorage !== undefined) {
      SpecStakeStorage.encode(
        message.specStakeStorage,
        writer.uint32(10).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetSpecStakeStorageResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetSpecStakeStorageResponse,
    } as QueryGetSpecStakeStorageResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.specStakeStorage = SpecStakeStorage.decode(
            reader,
            reader.uint32()
          );
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetSpecStakeStorageResponse {
    const message = {
      ...baseQueryGetSpecStakeStorageResponse,
    } as QueryGetSpecStakeStorageResponse;
    if (
      object.specStakeStorage !== undefined &&
      object.specStakeStorage !== null
    ) {
      message.specStakeStorage = SpecStakeStorage.fromJSON(
        object.specStakeStorage
      );
    } else {
      message.specStakeStorage = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetSpecStakeStorageResponse): unknown {
    const obj: any = {};
    message.specStakeStorage !== undefined &&
      (obj.specStakeStorage = message.specStakeStorage
        ? SpecStakeStorage.toJSON(message.specStakeStorage)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetSpecStakeStorageResponse>
  ): QueryGetSpecStakeStorageResponse {
    const message = {
      ...baseQueryGetSpecStakeStorageResponse,
    } as QueryGetSpecStakeStorageResponse;
    if (
      object.specStakeStorage !== undefined &&
      object.specStakeStorage !== null
    ) {
      message.specStakeStorage = SpecStakeStorage.fromPartial(
        object.specStakeStorage
      );
    } else {
      message.specStakeStorage = undefined;
    }
    return message;
  },
};

const baseQueryAllSpecStakeStorageRequest: object = {};

export const QueryAllSpecStakeStorageRequest = {
  encode(
    message: QueryAllSpecStakeStorageRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryAllSpecStakeStorageRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllSpecStakeStorageRequest,
    } as QueryAllSpecStakeStorageRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.pagination = PageRequest.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryAllSpecStakeStorageRequest {
    const message = {
      ...baseQueryAllSpecStakeStorageRequest,
    } as QueryAllSpecStakeStorageRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllSpecStakeStorageRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllSpecStakeStorageRequest>
  ): QueryAllSpecStakeStorageRequest {
    const message = {
      ...baseQueryAllSpecStakeStorageRequest,
    } as QueryAllSpecStakeStorageRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryAllSpecStakeStorageResponse: object = {};

export const QueryAllSpecStakeStorageResponse = {
  encode(
    message: QueryAllSpecStakeStorageResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.specStakeStorage) {
      SpecStakeStorage.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(
        message.pagination,
        writer.uint32(18).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryAllSpecStakeStorageResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllSpecStakeStorageResponse,
    } as QueryAllSpecStakeStorageResponse;
    message.specStakeStorage = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.specStakeStorage.push(
            SpecStakeStorage.decode(reader, reader.uint32())
          );
          break;
        case 2:
          message.pagination = PageResponse.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryAllSpecStakeStorageResponse {
    const message = {
      ...baseQueryAllSpecStakeStorageResponse,
    } as QueryAllSpecStakeStorageResponse;
    message.specStakeStorage = [];
    if (
      object.specStakeStorage !== undefined &&
      object.specStakeStorage !== null
    ) {
      for (const e of object.specStakeStorage) {
        message.specStakeStorage.push(SpecStakeStorage.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllSpecStakeStorageResponse): unknown {
    const obj: any = {};
    if (message.specStakeStorage) {
      obj.specStakeStorage = message.specStakeStorage.map((e) =>
        e ? SpecStakeStorage.toJSON(e) : undefined
      );
    } else {
      obj.specStakeStorage = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllSpecStakeStorageResponse>
  ): QueryAllSpecStakeStorageResponse {
    const message = {
      ...baseQueryAllSpecStakeStorageResponse,
    } as QueryAllSpecStakeStorageResponse;
    message.specStakeStorage = [];
    if (
      object.specStakeStorage !== undefined &&
      object.specStakeStorage !== null
    ) {
      for (const e of object.specStakeStorage) {
        message.specStakeStorage.push(SpecStakeStorage.fromPartial(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryGetBlockDeadlineForCallbackRequest: object = {};

export const QueryGetBlockDeadlineForCallbackRequest = {
  encode(
    _: QueryGetBlockDeadlineForCallbackRequest,
    writer: Writer = Writer.create()
  ): Writer {
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetBlockDeadlineForCallbackRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetBlockDeadlineForCallbackRequest,
    } as QueryGetBlockDeadlineForCallbackRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(_: any): QueryGetBlockDeadlineForCallbackRequest {
    const message = {
      ...baseQueryGetBlockDeadlineForCallbackRequest,
    } as QueryGetBlockDeadlineForCallbackRequest;
    return message;
  },

  toJSON(_: QueryGetBlockDeadlineForCallbackRequest): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(
    _: DeepPartial<QueryGetBlockDeadlineForCallbackRequest>
  ): QueryGetBlockDeadlineForCallbackRequest {
    const message = {
      ...baseQueryGetBlockDeadlineForCallbackRequest,
    } as QueryGetBlockDeadlineForCallbackRequest;
    return message;
  },
};

const baseQueryGetBlockDeadlineForCallbackResponse: object = {};

export const QueryGetBlockDeadlineForCallbackResponse = {
  encode(
    message: QueryGetBlockDeadlineForCallbackResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.BlockDeadlineForCallback !== undefined) {
      BlockDeadlineForCallback.encode(
        message.BlockDeadlineForCallback,
        writer.uint32(10).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetBlockDeadlineForCallbackResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetBlockDeadlineForCallbackResponse,
    } as QueryGetBlockDeadlineForCallbackResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.BlockDeadlineForCallback = BlockDeadlineForCallback.decode(
            reader,
            reader.uint32()
          );
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetBlockDeadlineForCallbackResponse {
    const message = {
      ...baseQueryGetBlockDeadlineForCallbackResponse,
    } as QueryGetBlockDeadlineForCallbackResponse;
    if (
      object.BlockDeadlineForCallback !== undefined &&
      object.BlockDeadlineForCallback !== null
    ) {
      message.BlockDeadlineForCallback = BlockDeadlineForCallback.fromJSON(
        object.BlockDeadlineForCallback
      );
    } else {
      message.BlockDeadlineForCallback = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetBlockDeadlineForCallbackResponse): unknown {
    const obj: any = {};
    message.BlockDeadlineForCallback !== undefined &&
      (obj.BlockDeadlineForCallback = message.BlockDeadlineForCallback
        ? BlockDeadlineForCallback.toJSON(message.BlockDeadlineForCallback)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetBlockDeadlineForCallbackResponse>
  ): QueryGetBlockDeadlineForCallbackResponse {
    const message = {
      ...baseQueryGetBlockDeadlineForCallbackResponse,
    } as QueryGetBlockDeadlineForCallbackResponse;
    if (
      object.BlockDeadlineForCallback !== undefined &&
      object.BlockDeadlineForCallback !== null
    ) {
      message.BlockDeadlineForCallback = BlockDeadlineForCallback.fromPartial(
        object.BlockDeadlineForCallback
      );
    } else {
      message.BlockDeadlineForCallback = undefined;
    }
    return message;
  },
};

const baseQueryGetUnstakingUsersAllSpecsRequest: object = { id: 0 };

export const QueryGetUnstakingUsersAllSpecsRequest = {
  encode(
    message: QueryGetUnstakingUsersAllSpecsRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.id !== 0) {
      writer.uint32(8).uint64(message.id);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetUnstakingUsersAllSpecsRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetUnstakingUsersAllSpecsRequest,
    } as QueryGetUnstakingUsersAllSpecsRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = longToNumber(reader.uint64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetUnstakingUsersAllSpecsRequest {
    const message = {
      ...baseQueryGetUnstakingUsersAllSpecsRequest,
    } as QueryGetUnstakingUsersAllSpecsRequest;
    if (object.id !== undefined && object.id !== null) {
      message.id = Number(object.id);
    } else {
      message.id = 0;
    }
    return message;
  },

  toJSON(message: QueryGetUnstakingUsersAllSpecsRequest): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetUnstakingUsersAllSpecsRequest>
  ): QueryGetUnstakingUsersAllSpecsRequest {
    const message = {
      ...baseQueryGetUnstakingUsersAllSpecsRequest,
    } as QueryGetUnstakingUsersAllSpecsRequest;
    if (object.id !== undefined && object.id !== null) {
      message.id = object.id;
    } else {
      message.id = 0;
    }
    return message;
  },
};

const baseQueryGetUnstakingUsersAllSpecsResponse: object = {};

export const QueryGetUnstakingUsersAllSpecsResponse = {
  encode(
    message: QueryGetUnstakingUsersAllSpecsResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.UnstakingUsersAllSpecs !== undefined) {
      UnstakingUsersAllSpecs.encode(
        message.UnstakingUsersAllSpecs,
        writer.uint32(10).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetUnstakingUsersAllSpecsResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetUnstakingUsersAllSpecsResponse,
    } as QueryGetUnstakingUsersAllSpecsResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.UnstakingUsersAllSpecs = UnstakingUsersAllSpecs.decode(
            reader,
            reader.uint32()
          );
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetUnstakingUsersAllSpecsResponse {
    const message = {
      ...baseQueryGetUnstakingUsersAllSpecsResponse,
    } as QueryGetUnstakingUsersAllSpecsResponse;
    if (
      object.UnstakingUsersAllSpecs !== undefined &&
      object.UnstakingUsersAllSpecs !== null
    ) {
      message.UnstakingUsersAllSpecs = UnstakingUsersAllSpecs.fromJSON(
        object.UnstakingUsersAllSpecs
      );
    } else {
      message.UnstakingUsersAllSpecs = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetUnstakingUsersAllSpecsResponse): unknown {
    const obj: any = {};
    message.UnstakingUsersAllSpecs !== undefined &&
      (obj.UnstakingUsersAllSpecs = message.UnstakingUsersAllSpecs
        ? UnstakingUsersAllSpecs.toJSON(message.UnstakingUsersAllSpecs)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetUnstakingUsersAllSpecsResponse>
  ): QueryGetUnstakingUsersAllSpecsResponse {
    const message = {
      ...baseQueryGetUnstakingUsersAllSpecsResponse,
    } as QueryGetUnstakingUsersAllSpecsResponse;
    if (
      object.UnstakingUsersAllSpecs !== undefined &&
      object.UnstakingUsersAllSpecs !== null
    ) {
      message.UnstakingUsersAllSpecs = UnstakingUsersAllSpecs.fromPartial(
        object.UnstakingUsersAllSpecs
      );
    } else {
      message.UnstakingUsersAllSpecs = undefined;
    }
    return message;
  },
};

const baseQueryAllUnstakingUsersAllSpecsRequest: object = {};

export const QueryAllUnstakingUsersAllSpecsRequest = {
  encode(
    message: QueryAllUnstakingUsersAllSpecsRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryAllUnstakingUsersAllSpecsRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllUnstakingUsersAllSpecsRequest,
    } as QueryAllUnstakingUsersAllSpecsRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.pagination = PageRequest.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryAllUnstakingUsersAllSpecsRequest {
    const message = {
      ...baseQueryAllUnstakingUsersAllSpecsRequest,
    } as QueryAllUnstakingUsersAllSpecsRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllUnstakingUsersAllSpecsRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllUnstakingUsersAllSpecsRequest>
  ): QueryAllUnstakingUsersAllSpecsRequest {
    const message = {
      ...baseQueryAllUnstakingUsersAllSpecsRequest,
    } as QueryAllUnstakingUsersAllSpecsRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryAllUnstakingUsersAllSpecsResponse: object = {};

export const QueryAllUnstakingUsersAllSpecsResponse = {
  encode(
    message: QueryAllUnstakingUsersAllSpecsResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.UnstakingUsersAllSpecs) {
      UnstakingUsersAllSpecs.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(
        message.pagination,
        writer.uint32(18).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryAllUnstakingUsersAllSpecsResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllUnstakingUsersAllSpecsResponse,
    } as QueryAllUnstakingUsersAllSpecsResponse;
    message.UnstakingUsersAllSpecs = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.UnstakingUsersAllSpecs.push(
            UnstakingUsersAllSpecs.decode(reader, reader.uint32())
          );
          break;
        case 2:
          message.pagination = PageResponse.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryAllUnstakingUsersAllSpecsResponse {
    const message = {
      ...baseQueryAllUnstakingUsersAllSpecsResponse,
    } as QueryAllUnstakingUsersAllSpecsResponse;
    message.UnstakingUsersAllSpecs = [];
    if (
      object.UnstakingUsersAllSpecs !== undefined &&
      object.UnstakingUsersAllSpecs !== null
    ) {
      for (const e of object.UnstakingUsersAllSpecs) {
        message.UnstakingUsersAllSpecs.push(UnstakingUsersAllSpecs.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllUnstakingUsersAllSpecsResponse): unknown {
    const obj: any = {};
    if (message.UnstakingUsersAllSpecs) {
      obj.UnstakingUsersAllSpecs = message.UnstakingUsersAllSpecs.map((e) =>
        e ? UnstakingUsersAllSpecs.toJSON(e) : undefined
      );
    } else {
      obj.UnstakingUsersAllSpecs = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllUnstakingUsersAllSpecsResponse>
  ): QueryAllUnstakingUsersAllSpecsResponse {
    const message = {
      ...baseQueryAllUnstakingUsersAllSpecsResponse,
    } as QueryAllUnstakingUsersAllSpecsResponse;
    message.UnstakingUsersAllSpecs = [];
    if (
      object.UnstakingUsersAllSpecs !== undefined &&
      object.UnstakingUsersAllSpecs !== null
    ) {
      for (const e of object.UnstakingUsersAllSpecs) {
        message.UnstakingUsersAllSpecs.push(
          UnstakingUsersAllSpecs.fromPartial(e)
        );
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryStakedUsersRequest: object = { specName: "" };

export const QueryStakedUsersRequest = {
  encode(
    message: QueryStakedUsersRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.specName !== "") {
      writer.uint32(10).string(message.specName);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryStakedUsersRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryStakedUsersRequest,
    } as QueryStakedUsersRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.specName = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryStakedUsersRequest {
    const message = {
      ...baseQueryStakedUsersRequest,
    } as QueryStakedUsersRequest;
    if (object.specName !== undefined && object.specName !== null) {
      message.specName = String(object.specName);
    } else {
      message.specName = "";
    }
    return message;
  },

  toJSON(message: QueryStakedUsersRequest): unknown {
    const obj: any = {};
    message.specName !== undefined && (obj.specName = message.specName);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryStakedUsersRequest>
  ): QueryStakedUsersRequest {
    const message = {
      ...baseQueryStakedUsersRequest,
    } as QueryStakedUsersRequest;
    if (object.specName !== undefined && object.specName !== null) {
      message.specName = object.specName;
    } else {
      message.specName = "";
    }
    return message;
  },
};

const baseQueryStakedUsersResponse: object = { output: "" };

export const QueryStakedUsersResponse = {
  encode(
    message: QueryStakedUsersResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.stakeStorage !== undefined) {
      StakeStorage.encode(
        message.stakeStorage,
        writer.uint32(10).fork()
      ).ldelim();
    }
    if (message.output !== "") {
      writer.uint32(18).string(message.output);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryStakedUsersResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryStakedUsersResponse,
    } as QueryStakedUsersResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.stakeStorage = StakeStorage.decode(reader, reader.uint32());
          break;
        case 2:
          message.output = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryStakedUsersResponse {
    const message = {
      ...baseQueryStakedUsersResponse,
    } as QueryStakedUsersResponse;
    if (object.stakeStorage !== undefined && object.stakeStorage !== null) {
      message.stakeStorage = StakeStorage.fromJSON(object.stakeStorage);
    } else {
      message.stakeStorage = undefined;
    }
    if (object.output !== undefined && object.output !== null) {
      message.output = String(object.output);
    } else {
      message.output = "";
    }
    return message;
  },

  toJSON(message: QueryStakedUsersResponse): unknown {
    const obj: any = {};
    message.stakeStorage !== undefined &&
      (obj.stakeStorage = message.stakeStorage
        ? StakeStorage.toJSON(message.stakeStorage)
        : undefined);
    message.output !== undefined && (obj.output = message.output);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryStakedUsersResponse>
  ): QueryStakedUsersResponse {
    const message = {
      ...baseQueryStakedUsersResponse,
    } as QueryStakedUsersResponse;
    if (object.stakeStorage !== undefined && object.stakeStorage !== null) {
      message.stakeStorage = StakeStorage.fromPartial(object.stakeStorage);
    } else {
      message.stakeStorage = undefined;
    }
    if (object.output !== undefined && object.output !== null) {
      message.output = object.output;
    } else {
      message.output = "";
    }
    return message;
  },
};

/** Query defines the gRPC querier service. */
export interface Query {
  /** Parameters queries the parameters of the module. */
  Params(request: QueryParamsRequest): Promise<QueryParamsResponse>;
  /** Queries a UserStake by index. */
  UserStake(
    request: QueryGetUserStakeRequest
  ): Promise<QueryGetUserStakeResponse>;
  /** Queries a list of UserStake items. */
  UserStakeAll(
    request: QueryAllUserStakeRequest
  ): Promise<QueryAllUserStakeResponse>;
  /** Queries a SpecStakeStorage by index. */
  SpecStakeStorage(
    request: QueryGetSpecStakeStorageRequest
  ): Promise<QueryGetSpecStakeStorageResponse>;
  /** Queries a list of SpecStakeStorage items. */
  SpecStakeStorageAll(
    request: QueryAllSpecStakeStorageRequest
  ): Promise<QueryAllSpecStakeStorageResponse>;
  /** Queries a BlockDeadlineForCallback by index. */
  BlockDeadlineForCallback(
    request: QueryGetBlockDeadlineForCallbackRequest
  ): Promise<QueryGetBlockDeadlineForCallbackResponse>;
  /** Queries a UnstakingUsersAllSpecs by id. */
  UnstakingUsersAllSpecs(
    request: QueryGetUnstakingUsersAllSpecsRequest
  ): Promise<QueryGetUnstakingUsersAllSpecsResponse>;
  /** Queries a list of UnstakingUsersAllSpecs items. */
  UnstakingUsersAllSpecsAll(
    request: QueryAllUnstakingUsersAllSpecsRequest
  ): Promise<QueryAllUnstakingUsersAllSpecsResponse>;
  /** Queries a list of StakedUsers items. */
  StakedUsers(
    request: QueryStakedUsersRequest
  ): Promise<QueryStakedUsersResponse>;
}

export class QueryClientImpl implements Query {
  private readonly rpc: Rpc;
  constructor(rpc: Rpc) {
    this.rpc = rpc;
  }
  Params(request: QueryParamsRequest): Promise<QueryParamsResponse> {
    const data = QueryParamsRequest.encode(request).finish();
    const promise = this.rpc.request("lavanet.lava.user.Query", "Params", data);
    return promise.then((data) => QueryParamsResponse.decode(new Reader(data)));
  }

  UserStake(
    request: QueryGetUserStakeRequest
  ): Promise<QueryGetUserStakeResponse> {
    const data = QueryGetUserStakeRequest.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.user.Query",
      "UserStake",
      data
    );
    return promise.then((data) =>
      QueryGetUserStakeResponse.decode(new Reader(data))
    );
  }

  UserStakeAll(
    request: QueryAllUserStakeRequest
  ): Promise<QueryAllUserStakeResponse> {
    const data = QueryAllUserStakeRequest.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.user.Query",
      "UserStakeAll",
      data
    );
    return promise.then((data) =>
      QueryAllUserStakeResponse.decode(new Reader(data))
    );
  }

  SpecStakeStorage(
    request: QueryGetSpecStakeStorageRequest
  ): Promise<QueryGetSpecStakeStorageResponse> {
    const data = QueryGetSpecStakeStorageRequest.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.user.Query",
      "SpecStakeStorage",
      data
    );
    return promise.then((data) =>
      QueryGetSpecStakeStorageResponse.decode(new Reader(data))
    );
  }

  SpecStakeStorageAll(
    request: QueryAllSpecStakeStorageRequest
  ): Promise<QueryAllSpecStakeStorageResponse> {
    const data = QueryAllSpecStakeStorageRequest.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.user.Query",
      "SpecStakeStorageAll",
      data
    );
    return promise.then((data) =>
      QueryAllSpecStakeStorageResponse.decode(new Reader(data))
    );
  }

  BlockDeadlineForCallback(
    request: QueryGetBlockDeadlineForCallbackRequest
  ): Promise<QueryGetBlockDeadlineForCallbackResponse> {
    const data = QueryGetBlockDeadlineForCallbackRequest.encode(
      request
    ).finish();
    const promise = this.rpc.request(
      "lavanet.lava.user.Query",
      "BlockDeadlineForCallback",
      data
    );
    return promise.then((data) =>
      QueryGetBlockDeadlineForCallbackResponse.decode(new Reader(data))
    );
  }

  UnstakingUsersAllSpecs(
    request: QueryGetUnstakingUsersAllSpecsRequest
  ): Promise<QueryGetUnstakingUsersAllSpecsResponse> {
    const data = QueryGetUnstakingUsersAllSpecsRequest.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.user.Query",
      "UnstakingUsersAllSpecs",
      data
    );
    return promise.then((data) =>
      QueryGetUnstakingUsersAllSpecsResponse.decode(new Reader(data))
    );
  }

  UnstakingUsersAllSpecsAll(
    request: QueryAllUnstakingUsersAllSpecsRequest
  ): Promise<QueryAllUnstakingUsersAllSpecsResponse> {
    const data = QueryAllUnstakingUsersAllSpecsRequest.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.user.Query",
      "UnstakingUsersAllSpecsAll",
      data
    );
    return promise.then((data) =>
      QueryAllUnstakingUsersAllSpecsResponse.decode(new Reader(data))
    );
  }

  StakedUsers(
    request: QueryStakedUsersRequest
  ): Promise<QueryStakedUsersResponse> {
    const data = QueryStakedUsersRequest.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.user.Query",
      "StakedUsers",
      data
    );
    return promise.then((data) =>
      QueryStakedUsersResponse.decode(new Reader(data))
    );
  }
}

interface Rpc {
  request(
    service: string,
    method: string,
    data: Uint8Array
  ): Promise<Uint8Array>;
}

declare var self: any | undefined;
declare var window: any | undefined;
var globalThis: any = (() => {
  if (typeof globalThis !== "undefined") return globalThis;
  if (typeof self !== "undefined") return self;
  if (typeof window !== "undefined") return window;
  if (typeof global !== "undefined") return global;
  throw "Unable to locate global object";
})();

type Builtin = Date | Function | Uint8Array | string | number | undefined;
export type DeepPartial<T> = T extends Builtin
  ? T
  : T extends Array<infer U>
  ? Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U>
  ? ReadonlyArray<DeepPartial<U>>
  : T extends {}
  ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function longToNumber(long: Long): number {
  if (long.gt(Number.MAX_SAFE_INTEGER)) {
    throw new globalThis.Error("Value is larger than Number.MAX_SAFE_INTEGER");
  }
  return long.toNumber();
}

if (util.Long !== Long) {
  util.Long = Long as any;
  configure();
}

import { Reader, Writer } from "protobufjs/minimal";
import { Params } from "../user/params";
import { UserStake } from "../user/user_stake";
import { PageRequest, PageResponse } from "../cosmos/base/query/v1beta1/pagination";
import { SpecStakeStorage } from "../user/spec_stake_storage";
import { BlockDeadlineForCallback } from "../user/block_deadline_for_callback";
import { UnstakingUsersAllSpecs } from "../user/unstaking_users_all_specs";
import { StakeStorage } from "../user/stake_storage";
export declare const protobufPackage = "lavanet.lava.user";
/** QueryParamsRequest is request type for the Query/Params RPC method. */
export interface QueryParamsRequest {
}
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
export interface QueryGetBlockDeadlineForCallbackRequest {
}
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
export declare const QueryParamsRequest: {
    encode(_: QueryParamsRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryParamsRequest;
    fromJSON(_: any): QueryParamsRequest;
    toJSON(_: QueryParamsRequest): unknown;
    fromPartial(_: DeepPartial<QueryParamsRequest>): QueryParamsRequest;
};
export declare const QueryParamsResponse: {
    encode(message: QueryParamsResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryParamsResponse;
    fromJSON(object: any): QueryParamsResponse;
    toJSON(message: QueryParamsResponse): unknown;
    fromPartial(object: DeepPartial<QueryParamsResponse>): QueryParamsResponse;
};
export declare const QueryGetUserStakeRequest: {
    encode(message: QueryGetUserStakeRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetUserStakeRequest;
    fromJSON(object: any): QueryGetUserStakeRequest;
    toJSON(message: QueryGetUserStakeRequest): unknown;
    fromPartial(object: DeepPartial<QueryGetUserStakeRequest>): QueryGetUserStakeRequest;
};
export declare const QueryGetUserStakeResponse: {
    encode(message: QueryGetUserStakeResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetUserStakeResponse;
    fromJSON(object: any): QueryGetUserStakeResponse;
    toJSON(message: QueryGetUserStakeResponse): unknown;
    fromPartial(object: DeepPartial<QueryGetUserStakeResponse>): QueryGetUserStakeResponse;
};
export declare const QueryAllUserStakeRequest: {
    encode(message: QueryAllUserStakeRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryAllUserStakeRequest;
    fromJSON(object: any): QueryAllUserStakeRequest;
    toJSON(message: QueryAllUserStakeRequest): unknown;
    fromPartial(object: DeepPartial<QueryAllUserStakeRequest>): QueryAllUserStakeRequest;
};
export declare const QueryAllUserStakeResponse: {
    encode(message: QueryAllUserStakeResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryAllUserStakeResponse;
    fromJSON(object: any): QueryAllUserStakeResponse;
    toJSON(message: QueryAllUserStakeResponse): unknown;
    fromPartial(object: DeepPartial<QueryAllUserStakeResponse>): QueryAllUserStakeResponse;
};
export declare const QueryGetSpecStakeStorageRequest: {
    encode(message: QueryGetSpecStakeStorageRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetSpecStakeStorageRequest;
    fromJSON(object: any): QueryGetSpecStakeStorageRequest;
    toJSON(message: QueryGetSpecStakeStorageRequest): unknown;
    fromPartial(object: DeepPartial<QueryGetSpecStakeStorageRequest>): QueryGetSpecStakeStorageRequest;
};
export declare const QueryGetSpecStakeStorageResponse: {
    encode(message: QueryGetSpecStakeStorageResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetSpecStakeStorageResponse;
    fromJSON(object: any): QueryGetSpecStakeStorageResponse;
    toJSON(message: QueryGetSpecStakeStorageResponse): unknown;
    fromPartial(object: DeepPartial<QueryGetSpecStakeStorageResponse>): QueryGetSpecStakeStorageResponse;
};
export declare const QueryAllSpecStakeStorageRequest: {
    encode(message: QueryAllSpecStakeStorageRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryAllSpecStakeStorageRequest;
    fromJSON(object: any): QueryAllSpecStakeStorageRequest;
    toJSON(message: QueryAllSpecStakeStorageRequest): unknown;
    fromPartial(object: DeepPartial<QueryAllSpecStakeStorageRequest>): QueryAllSpecStakeStorageRequest;
};
export declare const QueryAllSpecStakeStorageResponse: {
    encode(message: QueryAllSpecStakeStorageResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryAllSpecStakeStorageResponse;
    fromJSON(object: any): QueryAllSpecStakeStorageResponse;
    toJSON(message: QueryAllSpecStakeStorageResponse): unknown;
    fromPartial(object: DeepPartial<QueryAllSpecStakeStorageResponse>): QueryAllSpecStakeStorageResponse;
};
export declare const QueryGetBlockDeadlineForCallbackRequest: {
    encode(_: QueryGetBlockDeadlineForCallbackRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetBlockDeadlineForCallbackRequest;
    fromJSON(_: any): QueryGetBlockDeadlineForCallbackRequest;
    toJSON(_: QueryGetBlockDeadlineForCallbackRequest): unknown;
    fromPartial(_: DeepPartial<QueryGetBlockDeadlineForCallbackRequest>): QueryGetBlockDeadlineForCallbackRequest;
};
export declare const QueryGetBlockDeadlineForCallbackResponse: {
    encode(message: QueryGetBlockDeadlineForCallbackResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetBlockDeadlineForCallbackResponse;
    fromJSON(object: any): QueryGetBlockDeadlineForCallbackResponse;
    toJSON(message: QueryGetBlockDeadlineForCallbackResponse): unknown;
    fromPartial(object: DeepPartial<QueryGetBlockDeadlineForCallbackResponse>): QueryGetBlockDeadlineForCallbackResponse;
};
export declare const QueryGetUnstakingUsersAllSpecsRequest: {
    encode(message: QueryGetUnstakingUsersAllSpecsRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetUnstakingUsersAllSpecsRequest;
    fromJSON(object: any): QueryGetUnstakingUsersAllSpecsRequest;
    toJSON(message: QueryGetUnstakingUsersAllSpecsRequest): unknown;
    fromPartial(object: DeepPartial<QueryGetUnstakingUsersAllSpecsRequest>): QueryGetUnstakingUsersAllSpecsRequest;
};
export declare const QueryGetUnstakingUsersAllSpecsResponse: {
    encode(message: QueryGetUnstakingUsersAllSpecsResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetUnstakingUsersAllSpecsResponse;
    fromJSON(object: any): QueryGetUnstakingUsersAllSpecsResponse;
    toJSON(message: QueryGetUnstakingUsersAllSpecsResponse): unknown;
    fromPartial(object: DeepPartial<QueryGetUnstakingUsersAllSpecsResponse>): QueryGetUnstakingUsersAllSpecsResponse;
};
export declare const QueryAllUnstakingUsersAllSpecsRequest: {
    encode(message: QueryAllUnstakingUsersAllSpecsRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryAllUnstakingUsersAllSpecsRequest;
    fromJSON(object: any): QueryAllUnstakingUsersAllSpecsRequest;
    toJSON(message: QueryAllUnstakingUsersAllSpecsRequest): unknown;
    fromPartial(object: DeepPartial<QueryAllUnstakingUsersAllSpecsRequest>): QueryAllUnstakingUsersAllSpecsRequest;
};
export declare const QueryAllUnstakingUsersAllSpecsResponse: {
    encode(message: QueryAllUnstakingUsersAllSpecsResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryAllUnstakingUsersAllSpecsResponse;
    fromJSON(object: any): QueryAllUnstakingUsersAllSpecsResponse;
    toJSON(message: QueryAllUnstakingUsersAllSpecsResponse): unknown;
    fromPartial(object: DeepPartial<QueryAllUnstakingUsersAllSpecsResponse>): QueryAllUnstakingUsersAllSpecsResponse;
};
export declare const QueryStakedUsersRequest: {
    encode(message: QueryStakedUsersRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryStakedUsersRequest;
    fromJSON(object: any): QueryStakedUsersRequest;
    toJSON(message: QueryStakedUsersRequest): unknown;
    fromPartial(object: DeepPartial<QueryStakedUsersRequest>): QueryStakedUsersRequest;
};
export declare const QueryStakedUsersResponse: {
    encode(message: QueryStakedUsersResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryStakedUsersResponse;
    fromJSON(object: any): QueryStakedUsersResponse;
    toJSON(message: QueryStakedUsersResponse): unknown;
    fromPartial(object: DeepPartial<QueryStakedUsersResponse>): QueryStakedUsersResponse;
};
/** Query defines the gRPC querier service. */
export interface Query {
    /** Parameters queries the parameters of the module. */
    Params(request: QueryParamsRequest): Promise<QueryParamsResponse>;
    /** Queries a UserStake by index. */
    UserStake(request: QueryGetUserStakeRequest): Promise<QueryGetUserStakeResponse>;
    /** Queries a list of UserStake items. */
    UserStakeAll(request: QueryAllUserStakeRequest): Promise<QueryAllUserStakeResponse>;
    /** Queries a SpecStakeStorage by index. */
    SpecStakeStorage(request: QueryGetSpecStakeStorageRequest): Promise<QueryGetSpecStakeStorageResponse>;
    /** Queries a list of SpecStakeStorage items. */
    SpecStakeStorageAll(request: QueryAllSpecStakeStorageRequest): Promise<QueryAllSpecStakeStorageResponse>;
    /** Queries a BlockDeadlineForCallback by index. */
    BlockDeadlineForCallback(request: QueryGetBlockDeadlineForCallbackRequest): Promise<QueryGetBlockDeadlineForCallbackResponse>;
    /** Queries a UnstakingUsersAllSpecs by id. */
    UnstakingUsersAllSpecs(request: QueryGetUnstakingUsersAllSpecsRequest): Promise<QueryGetUnstakingUsersAllSpecsResponse>;
    /** Queries a list of UnstakingUsersAllSpecs items. */
    UnstakingUsersAllSpecsAll(request: QueryAllUnstakingUsersAllSpecsRequest): Promise<QueryAllUnstakingUsersAllSpecsResponse>;
    /** Queries a list of StakedUsers items. */
    StakedUsers(request: QueryStakedUsersRequest): Promise<QueryStakedUsersResponse>;
}
export declare class QueryClientImpl implements Query {
    private readonly rpc;
    constructor(rpc: Rpc);
    Params(request: QueryParamsRequest): Promise<QueryParamsResponse>;
    UserStake(request: QueryGetUserStakeRequest): Promise<QueryGetUserStakeResponse>;
    UserStakeAll(request: QueryAllUserStakeRequest): Promise<QueryAllUserStakeResponse>;
    SpecStakeStorage(request: QueryGetSpecStakeStorageRequest): Promise<QueryGetSpecStakeStorageResponse>;
    SpecStakeStorageAll(request: QueryAllSpecStakeStorageRequest): Promise<QueryAllSpecStakeStorageResponse>;
    BlockDeadlineForCallback(request: QueryGetBlockDeadlineForCallbackRequest): Promise<QueryGetBlockDeadlineForCallbackResponse>;
    UnstakingUsersAllSpecs(request: QueryGetUnstakingUsersAllSpecsRequest): Promise<QueryGetUnstakingUsersAllSpecsResponse>;
    UnstakingUsersAllSpecsAll(request: QueryAllUnstakingUsersAllSpecsRequest): Promise<QueryAllUnstakingUsersAllSpecsResponse>;
    StakedUsers(request: QueryStakedUsersRequest): Promise<QueryStakedUsersResponse>;
}
interface Rpc {
    request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};

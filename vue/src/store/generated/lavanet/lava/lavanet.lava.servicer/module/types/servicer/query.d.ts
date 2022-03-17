import { Reader, Writer } from "protobufjs/minimal";
import { Params } from "../servicer/params";
import { StakeMap } from "../servicer/stake_map";
import { PageRequest, PageResponse } from "../cosmos/base/query/v1beta1/pagination";
import { SpecStakeStorage } from "../servicer/spec_stake_storage";
import { StakeStorage } from "../servicer/stake_storage";
import { BlockDeadlineForCallback } from "../servicer/block_deadline_for_callback";
import { UnstakingServicersAllSpecs } from "../servicer/unstaking_servicers_all_specs";
export declare const protobufPackage = "lavanet.lava.servicer";
/** QueryParamsRequest is request type for the Query/Params RPC method. */
export interface QueryParamsRequest {
}
/** QueryParamsResponse is response type for the Query/Params RPC method. */
export interface QueryParamsResponse {
    /** params holds all the parameters of this module. */
    params: Params | undefined;
}
export interface QueryGetStakeMapRequest {
    index: string;
}
export interface QueryGetStakeMapResponse {
    stakeMap: StakeMap | undefined;
}
export interface QueryAllStakeMapRequest {
    pagination: PageRequest | undefined;
}
export interface QueryAllStakeMapResponse {
    stakeMap: StakeMap[];
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
export interface QueryStakedServicersRequest {
    specName: string;
}
export interface QueryStakedServicersResponse {
    stakeStorage: StakeStorage | undefined;
    output: string;
}
export interface QueryGetBlockDeadlineForCallbackRequest {
}
export interface QueryGetBlockDeadlineForCallbackResponse {
    BlockDeadlineForCallback: BlockDeadlineForCallback | undefined;
}
export interface QueryGetUnstakingServicersAllSpecsRequest {
    id: number;
}
export interface QueryGetUnstakingServicersAllSpecsResponse {
    UnstakingServicersAllSpecs: UnstakingServicersAllSpecs | undefined;
}
export interface QueryAllUnstakingServicersAllSpecsRequest {
    pagination: PageRequest | undefined;
}
export interface QueryAllUnstakingServicersAllSpecsResponse {
    UnstakingServicersAllSpecs: UnstakingServicersAllSpecs[];
    pagination: PageResponse | undefined;
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
export declare const QueryGetStakeMapRequest: {
    encode(message: QueryGetStakeMapRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetStakeMapRequest;
    fromJSON(object: any): QueryGetStakeMapRequest;
    toJSON(message: QueryGetStakeMapRequest): unknown;
    fromPartial(object: DeepPartial<QueryGetStakeMapRequest>): QueryGetStakeMapRequest;
};
export declare const QueryGetStakeMapResponse: {
    encode(message: QueryGetStakeMapResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetStakeMapResponse;
    fromJSON(object: any): QueryGetStakeMapResponse;
    toJSON(message: QueryGetStakeMapResponse): unknown;
    fromPartial(object: DeepPartial<QueryGetStakeMapResponse>): QueryGetStakeMapResponse;
};
export declare const QueryAllStakeMapRequest: {
    encode(message: QueryAllStakeMapRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryAllStakeMapRequest;
    fromJSON(object: any): QueryAllStakeMapRequest;
    toJSON(message: QueryAllStakeMapRequest): unknown;
    fromPartial(object: DeepPartial<QueryAllStakeMapRequest>): QueryAllStakeMapRequest;
};
export declare const QueryAllStakeMapResponse: {
    encode(message: QueryAllStakeMapResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryAllStakeMapResponse;
    fromJSON(object: any): QueryAllStakeMapResponse;
    toJSON(message: QueryAllStakeMapResponse): unknown;
    fromPartial(object: DeepPartial<QueryAllStakeMapResponse>): QueryAllStakeMapResponse;
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
export declare const QueryStakedServicersRequest: {
    encode(message: QueryStakedServicersRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryStakedServicersRequest;
    fromJSON(object: any): QueryStakedServicersRequest;
    toJSON(message: QueryStakedServicersRequest): unknown;
    fromPartial(object: DeepPartial<QueryStakedServicersRequest>): QueryStakedServicersRequest;
};
export declare const QueryStakedServicersResponse: {
    encode(message: QueryStakedServicersResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryStakedServicersResponse;
    fromJSON(object: any): QueryStakedServicersResponse;
    toJSON(message: QueryStakedServicersResponse): unknown;
    fromPartial(object: DeepPartial<QueryStakedServicersResponse>): QueryStakedServicersResponse;
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
export declare const QueryGetUnstakingServicersAllSpecsRequest: {
    encode(message: QueryGetUnstakingServicersAllSpecsRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetUnstakingServicersAllSpecsRequest;
    fromJSON(object: any): QueryGetUnstakingServicersAllSpecsRequest;
    toJSON(message: QueryGetUnstakingServicersAllSpecsRequest): unknown;
    fromPartial(object: DeepPartial<QueryGetUnstakingServicersAllSpecsRequest>): QueryGetUnstakingServicersAllSpecsRequest;
};
export declare const QueryGetUnstakingServicersAllSpecsResponse: {
    encode(message: QueryGetUnstakingServicersAllSpecsResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetUnstakingServicersAllSpecsResponse;
    fromJSON(object: any): QueryGetUnstakingServicersAllSpecsResponse;
    toJSON(message: QueryGetUnstakingServicersAllSpecsResponse): unknown;
    fromPartial(object: DeepPartial<QueryGetUnstakingServicersAllSpecsResponse>): QueryGetUnstakingServicersAllSpecsResponse;
};
export declare const QueryAllUnstakingServicersAllSpecsRequest: {
    encode(message: QueryAllUnstakingServicersAllSpecsRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryAllUnstakingServicersAllSpecsRequest;
    fromJSON(object: any): QueryAllUnstakingServicersAllSpecsRequest;
    toJSON(message: QueryAllUnstakingServicersAllSpecsRequest): unknown;
    fromPartial(object: DeepPartial<QueryAllUnstakingServicersAllSpecsRequest>): QueryAllUnstakingServicersAllSpecsRequest;
};
export declare const QueryAllUnstakingServicersAllSpecsResponse: {
    encode(message: QueryAllUnstakingServicersAllSpecsResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryAllUnstakingServicersAllSpecsResponse;
    fromJSON(object: any): QueryAllUnstakingServicersAllSpecsResponse;
    toJSON(message: QueryAllUnstakingServicersAllSpecsResponse): unknown;
    fromPartial(object: DeepPartial<QueryAllUnstakingServicersAllSpecsResponse>): QueryAllUnstakingServicersAllSpecsResponse;
};
/** Query defines the gRPC querier service. */
export interface Query {
    /** Parameters queries the parameters of the module. */
    Params(request: QueryParamsRequest): Promise<QueryParamsResponse>;
    /** Queries a StakeMap by index. */
    StakeMap(request: QueryGetStakeMapRequest): Promise<QueryGetStakeMapResponse>;
    /** Queries a list of StakeMap items. */
    StakeMapAll(request: QueryAllStakeMapRequest): Promise<QueryAllStakeMapResponse>;
    /** Queries a SpecStakeStorage by index. */
    SpecStakeStorage(request: QueryGetSpecStakeStorageRequest): Promise<QueryGetSpecStakeStorageResponse>;
    /** Queries a list of SpecStakeStorage items. */
    SpecStakeStorageAll(request: QueryAllSpecStakeStorageRequest): Promise<QueryAllSpecStakeStorageResponse>;
    /** Queries a list of StakedServicers items. */
    StakedServicers(request: QueryStakedServicersRequest): Promise<QueryStakedServicersResponse>;
    /** Queries a BlockDeadlineForCallback by index. */
    BlockDeadlineForCallback(request: QueryGetBlockDeadlineForCallbackRequest): Promise<QueryGetBlockDeadlineForCallbackResponse>;
    /** Queries a UnstakingServicersAllSpecs by id. */
    UnstakingServicersAllSpecs(request: QueryGetUnstakingServicersAllSpecsRequest): Promise<QueryGetUnstakingServicersAllSpecsResponse>;
    /** Queries a list of UnstakingServicersAllSpecs items. */
    UnstakingServicersAllSpecsAll(request: QueryAllUnstakingServicersAllSpecsRequest): Promise<QueryAllUnstakingServicersAllSpecsResponse>;
}
export declare class QueryClientImpl implements Query {
    private readonly rpc;
    constructor(rpc: Rpc);
    Params(request: QueryParamsRequest): Promise<QueryParamsResponse>;
    StakeMap(request: QueryGetStakeMapRequest): Promise<QueryGetStakeMapResponse>;
    StakeMapAll(request: QueryAllStakeMapRequest): Promise<QueryAllStakeMapResponse>;
    SpecStakeStorage(request: QueryGetSpecStakeStorageRequest): Promise<QueryGetSpecStakeStorageResponse>;
    SpecStakeStorageAll(request: QueryAllSpecStakeStorageRequest): Promise<QueryAllSpecStakeStorageResponse>;
    StakedServicers(request: QueryStakedServicersRequest): Promise<QueryStakedServicersResponse>;
    BlockDeadlineForCallback(request: QueryGetBlockDeadlineForCallbackRequest): Promise<QueryGetBlockDeadlineForCallbackResponse>;
    UnstakingServicersAllSpecs(request: QueryGetUnstakingServicersAllSpecsRequest): Promise<QueryGetUnstakingServicersAllSpecsResponse>;
    UnstakingServicersAllSpecsAll(request: QueryAllUnstakingServicersAllSpecsRequest): Promise<QueryAllUnstakingServicersAllSpecsResponse>;
}
interface Rpc {
    request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};

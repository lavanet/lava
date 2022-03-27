import { Reader, Writer } from "protobufjs/minimal";
import { Params } from "../servicer/params";
import { StakeMap } from "../servicer/stake_map";
import { PageRequest, PageResponse } from "../cosmos/base/query/v1beta1/pagination";
import { SpecStakeStorage } from "../servicer/spec_stake_storage";
import { StakeStorage } from "../servicer/stake_storage";
import { BlockDeadlineForCallback } from "../servicer/block_deadline_for_callback";
import { UnstakingServicersAllSpecs } from "../servicer/unstaking_servicers_all_specs";
import { CurrentSessionStart } from "../servicer/current_session_start";
import { PreviousSessionBlocks } from "../servicer/previous_session_blocks";
import { SessionStorageForSpec } from "../servicer/session_storage_for_spec";
import { EarliestSessionStart } from "../servicer/earliest_session_start";
import { UniquePaymentStorageUserServicer } from "../servicer/unique_payment_storage_user_servicer";
import { UserPaymentStorage } from "../servicer/user_payment_storage";
import { SessionPayments } from "../servicer/session_payments";
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
export interface QueryGetPairingRequest {
    specName: string;
    userAddr: string;
}
export interface QueryGetPairingResponse {
    servicers: StakeStorage | undefined;
}
export interface QueryGetCurrentSessionStartRequest {
}
export interface QueryGetCurrentSessionStartResponse {
    CurrentSessionStart: CurrentSessionStart | undefined;
}
export interface QueryGetPreviousSessionBlocksRequest {
}
export interface QueryGetPreviousSessionBlocksResponse {
    PreviousSessionBlocks: PreviousSessionBlocks | undefined;
}
export interface QueryGetSessionStorageForSpecRequest {
    index: string;
}
export interface QueryGetSessionStorageForSpecResponse {
    sessionStorageForSpec: SessionStorageForSpec | undefined;
}
export interface QueryAllSessionStorageForSpecRequest {
    pagination: PageRequest | undefined;
}
export interface QueryAllSessionStorageForSpecResponse {
    sessionStorageForSpec: SessionStorageForSpec[];
    pagination: PageResponse | undefined;
}
export interface QuerySessionStorageForAllSpecsRequest {
    blockNum: number;
}
export interface QuerySessionStorageForAllSpecsResponse {
    servicers: StakeStorage | undefined;
}
export interface QueryAllSessionStoragesForSpecRequest {
    specName: string;
}
export interface QueryAllSessionStoragesForSpecResponse {
    storages: SessionStorageForSpec[];
}
export interface QueryGetEarliestSessionStartRequest {
}
export interface QueryGetEarliestSessionStartResponse {
    EarliestSessionStart: EarliestSessionStart | undefined;
}
export interface QueryVerifyPairingRequest {
    spec: number;
    userAddr: string;
    servicerAddr: string;
    blockNum: number;
}
export interface QueryVerifyPairingResponse {
    valid: boolean;
    overlap: boolean;
}
export interface QueryGetUniquePaymentStorageUserServicerRequest {
    index: string;
}
export interface QueryGetUniquePaymentStorageUserServicerResponse {
    uniquePaymentStorageUserServicer: UniquePaymentStorageUserServicer | undefined;
}
export interface QueryAllUniquePaymentStorageUserServicerRequest {
    pagination: PageRequest | undefined;
}
export interface QueryAllUniquePaymentStorageUserServicerResponse {
    uniquePaymentStorageUserServicer: UniquePaymentStorageUserServicer[];
    pagination: PageResponse | undefined;
}
export interface QueryGetUserPaymentStorageRequest {
    index: string;
}
export interface QueryGetUserPaymentStorageResponse {
    userPaymentStorage: UserPaymentStorage | undefined;
}
export interface QueryAllUserPaymentStorageRequest {
    pagination: PageRequest | undefined;
}
export interface QueryAllUserPaymentStorageResponse {
    userPaymentStorage: UserPaymentStorage[];
    pagination: PageResponse | undefined;
}
export interface QueryGetSessionPaymentsRequest {
    index: string;
}
export interface QueryGetSessionPaymentsResponse {
    sessionPayments: SessionPayments | undefined;
}
export interface QueryAllSessionPaymentsRequest {
    pagination: PageRequest | undefined;
}
export interface QueryAllSessionPaymentsResponse {
    sessionPayments: SessionPayments[];
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
export declare const QueryGetPairingRequest: {
    encode(message: QueryGetPairingRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetPairingRequest;
    fromJSON(object: any): QueryGetPairingRequest;
    toJSON(message: QueryGetPairingRequest): unknown;
    fromPartial(object: DeepPartial<QueryGetPairingRequest>): QueryGetPairingRequest;
};
export declare const QueryGetPairingResponse: {
    encode(message: QueryGetPairingResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetPairingResponse;
    fromJSON(object: any): QueryGetPairingResponse;
    toJSON(message: QueryGetPairingResponse): unknown;
    fromPartial(object: DeepPartial<QueryGetPairingResponse>): QueryGetPairingResponse;
};
export declare const QueryGetCurrentSessionStartRequest: {
    encode(_: QueryGetCurrentSessionStartRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetCurrentSessionStartRequest;
    fromJSON(_: any): QueryGetCurrentSessionStartRequest;
    toJSON(_: QueryGetCurrentSessionStartRequest): unknown;
    fromPartial(_: DeepPartial<QueryGetCurrentSessionStartRequest>): QueryGetCurrentSessionStartRequest;
};
export declare const QueryGetCurrentSessionStartResponse: {
    encode(message: QueryGetCurrentSessionStartResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetCurrentSessionStartResponse;
    fromJSON(object: any): QueryGetCurrentSessionStartResponse;
    toJSON(message: QueryGetCurrentSessionStartResponse): unknown;
    fromPartial(object: DeepPartial<QueryGetCurrentSessionStartResponse>): QueryGetCurrentSessionStartResponse;
};
export declare const QueryGetPreviousSessionBlocksRequest: {
    encode(_: QueryGetPreviousSessionBlocksRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetPreviousSessionBlocksRequest;
    fromJSON(_: any): QueryGetPreviousSessionBlocksRequest;
    toJSON(_: QueryGetPreviousSessionBlocksRequest): unknown;
    fromPartial(_: DeepPartial<QueryGetPreviousSessionBlocksRequest>): QueryGetPreviousSessionBlocksRequest;
};
export declare const QueryGetPreviousSessionBlocksResponse: {
    encode(message: QueryGetPreviousSessionBlocksResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetPreviousSessionBlocksResponse;
    fromJSON(object: any): QueryGetPreviousSessionBlocksResponse;
    toJSON(message: QueryGetPreviousSessionBlocksResponse): unknown;
    fromPartial(object: DeepPartial<QueryGetPreviousSessionBlocksResponse>): QueryGetPreviousSessionBlocksResponse;
};
export declare const QueryGetSessionStorageForSpecRequest: {
    encode(message: QueryGetSessionStorageForSpecRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetSessionStorageForSpecRequest;
    fromJSON(object: any): QueryGetSessionStorageForSpecRequest;
    toJSON(message: QueryGetSessionStorageForSpecRequest): unknown;
    fromPartial(object: DeepPartial<QueryGetSessionStorageForSpecRequest>): QueryGetSessionStorageForSpecRequest;
};
export declare const QueryGetSessionStorageForSpecResponse: {
    encode(message: QueryGetSessionStorageForSpecResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetSessionStorageForSpecResponse;
    fromJSON(object: any): QueryGetSessionStorageForSpecResponse;
    toJSON(message: QueryGetSessionStorageForSpecResponse): unknown;
    fromPartial(object: DeepPartial<QueryGetSessionStorageForSpecResponse>): QueryGetSessionStorageForSpecResponse;
};
export declare const QueryAllSessionStorageForSpecRequest: {
    encode(message: QueryAllSessionStorageForSpecRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryAllSessionStorageForSpecRequest;
    fromJSON(object: any): QueryAllSessionStorageForSpecRequest;
    toJSON(message: QueryAllSessionStorageForSpecRequest): unknown;
    fromPartial(object: DeepPartial<QueryAllSessionStorageForSpecRequest>): QueryAllSessionStorageForSpecRequest;
};
export declare const QueryAllSessionStorageForSpecResponse: {
    encode(message: QueryAllSessionStorageForSpecResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryAllSessionStorageForSpecResponse;
    fromJSON(object: any): QueryAllSessionStorageForSpecResponse;
    toJSON(message: QueryAllSessionStorageForSpecResponse): unknown;
    fromPartial(object: DeepPartial<QueryAllSessionStorageForSpecResponse>): QueryAllSessionStorageForSpecResponse;
};
export declare const QuerySessionStorageForAllSpecsRequest: {
    encode(message: QuerySessionStorageForAllSpecsRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QuerySessionStorageForAllSpecsRequest;
    fromJSON(object: any): QuerySessionStorageForAllSpecsRequest;
    toJSON(message: QuerySessionStorageForAllSpecsRequest): unknown;
    fromPartial(object: DeepPartial<QuerySessionStorageForAllSpecsRequest>): QuerySessionStorageForAllSpecsRequest;
};
export declare const QuerySessionStorageForAllSpecsResponse: {
    encode(message: QuerySessionStorageForAllSpecsResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QuerySessionStorageForAllSpecsResponse;
    fromJSON(object: any): QuerySessionStorageForAllSpecsResponse;
    toJSON(message: QuerySessionStorageForAllSpecsResponse): unknown;
    fromPartial(object: DeepPartial<QuerySessionStorageForAllSpecsResponse>): QuerySessionStorageForAllSpecsResponse;
};
export declare const QueryAllSessionStoragesForSpecRequest: {
    encode(message: QueryAllSessionStoragesForSpecRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryAllSessionStoragesForSpecRequest;
    fromJSON(object: any): QueryAllSessionStoragesForSpecRequest;
    toJSON(message: QueryAllSessionStoragesForSpecRequest): unknown;
    fromPartial(object: DeepPartial<QueryAllSessionStoragesForSpecRequest>): QueryAllSessionStoragesForSpecRequest;
};
export declare const QueryAllSessionStoragesForSpecResponse: {
    encode(message: QueryAllSessionStoragesForSpecResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryAllSessionStoragesForSpecResponse;
    fromJSON(object: any): QueryAllSessionStoragesForSpecResponse;
    toJSON(message: QueryAllSessionStoragesForSpecResponse): unknown;
    fromPartial(object: DeepPartial<QueryAllSessionStoragesForSpecResponse>): QueryAllSessionStoragesForSpecResponse;
};
export declare const QueryGetEarliestSessionStartRequest: {
    encode(_: QueryGetEarliestSessionStartRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetEarliestSessionStartRequest;
    fromJSON(_: any): QueryGetEarliestSessionStartRequest;
    toJSON(_: QueryGetEarliestSessionStartRequest): unknown;
    fromPartial(_: DeepPartial<QueryGetEarliestSessionStartRequest>): QueryGetEarliestSessionStartRequest;
};
export declare const QueryGetEarliestSessionStartResponse: {
    encode(message: QueryGetEarliestSessionStartResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetEarliestSessionStartResponse;
    fromJSON(object: any): QueryGetEarliestSessionStartResponse;
    toJSON(message: QueryGetEarliestSessionStartResponse): unknown;
    fromPartial(object: DeepPartial<QueryGetEarliestSessionStartResponse>): QueryGetEarliestSessionStartResponse;
};
export declare const QueryVerifyPairingRequest: {
    encode(message: QueryVerifyPairingRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryVerifyPairingRequest;
    fromJSON(object: any): QueryVerifyPairingRequest;
    toJSON(message: QueryVerifyPairingRequest): unknown;
    fromPartial(object: DeepPartial<QueryVerifyPairingRequest>): QueryVerifyPairingRequest;
};
export declare const QueryVerifyPairingResponse: {
    encode(message: QueryVerifyPairingResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryVerifyPairingResponse;
    fromJSON(object: any): QueryVerifyPairingResponse;
    toJSON(message: QueryVerifyPairingResponse): unknown;
    fromPartial(object: DeepPartial<QueryVerifyPairingResponse>): QueryVerifyPairingResponse;
};
export declare const QueryGetUniquePaymentStorageUserServicerRequest: {
    encode(message: QueryGetUniquePaymentStorageUserServicerRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetUniquePaymentStorageUserServicerRequest;
    fromJSON(object: any): QueryGetUniquePaymentStorageUserServicerRequest;
    toJSON(message: QueryGetUniquePaymentStorageUserServicerRequest): unknown;
    fromPartial(object: DeepPartial<QueryGetUniquePaymentStorageUserServicerRequest>): QueryGetUniquePaymentStorageUserServicerRequest;
};
export declare const QueryGetUniquePaymentStorageUserServicerResponse: {
    encode(message: QueryGetUniquePaymentStorageUserServicerResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetUniquePaymentStorageUserServicerResponse;
    fromJSON(object: any): QueryGetUniquePaymentStorageUserServicerResponse;
    toJSON(message: QueryGetUniquePaymentStorageUserServicerResponse): unknown;
    fromPartial(object: DeepPartial<QueryGetUniquePaymentStorageUserServicerResponse>): QueryGetUniquePaymentStorageUserServicerResponse;
};
export declare const QueryAllUniquePaymentStorageUserServicerRequest: {
    encode(message: QueryAllUniquePaymentStorageUserServicerRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryAllUniquePaymentStorageUserServicerRequest;
    fromJSON(object: any): QueryAllUniquePaymentStorageUserServicerRequest;
    toJSON(message: QueryAllUniquePaymentStorageUserServicerRequest): unknown;
    fromPartial(object: DeepPartial<QueryAllUniquePaymentStorageUserServicerRequest>): QueryAllUniquePaymentStorageUserServicerRequest;
};
export declare const QueryAllUniquePaymentStorageUserServicerResponse: {
    encode(message: QueryAllUniquePaymentStorageUserServicerResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryAllUniquePaymentStorageUserServicerResponse;
    fromJSON(object: any): QueryAllUniquePaymentStorageUserServicerResponse;
    toJSON(message: QueryAllUniquePaymentStorageUserServicerResponse): unknown;
    fromPartial(object: DeepPartial<QueryAllUniquePaymentStorageUserServicerResponse>): QueryAllUniquePaymentStorageUserServicerResponse;
};
export declare const QueryGetUserPaymentStorageRequest: {
    encode(message: QueryGetUserPaymentStorageRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetUserPaymentStorageRequest;
    fromJSON(object: any): QueryGetUserPaymentStorageRequest;
    toJSON(message: QueryGetUserPaymentStorageRequest): unknown;
    fromPartial(object: DeepPartial<QueryGetUserPaymentStorageRequest>): QueryGetUserPaymentStorageRequest;
};
export declare const QueryGetUserPaymentStorageResponse: {
    encode(message: QueryGetUserPaymentStorageResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetUserPaymentStorageResponse;
    fromJSON(object: any): QueryGetUserPaymentStorageResponse;
    toJSON(message: QueryGetUserPaymentStorageResponse): unknown;
    fromPartial(object: DeepPartial<QueryGetUserPaymentStorageResponse>): QueryGetUserPaymentStorageResponse;
};
export declare const QueryAllUserPaymentStorageRequest: {
    encode(message: QueryAllUserPaymentStorageRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryAllUserPaymentStorageRequest;
    fromJSON(object: any): QueryAllUserPaymentStorageRequest;
    toJSON(message: QueryAllUserPaymentStorageRequest): unknown;
    fromPartial(object: DeepPartial<QueryAllUserPaymentStorageRequest>): QueryAllUserPaymentStorageRequest;
};
export declare const QueryAllUserPaymentStorageResponse: {
    encode(message: QueryAllUserPaymentStorageResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryAllUserPaymentStorageResponse;
    fromJSON(object: any): QueryAllUserPaymentStorageResponse;
    toJSON(message: QueryAllUserPaymentStorageResponse): unknown;
    fromPartial(object: DeepPartial<QueryAllUserPaymentStorageResponse>): QueryAllUserPaymentStorageResponse;
};
export declare const QueryGetSessionPaymentsRequest: {
    encode(message: QueryGetSessionPaymentsRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetSessionPaymentsRequest;
    fromJSON(object: any): QueryGetSessionPaymentsRequest;
    toJSON(message: QueryGetSessionPaymentsRequest): unknown;
    fromPartial(object: DeepPartial<QueryGetSessionPaymentsRequest>): QueryGetSessionPaymentsRequest;
};
export declare const QueryGetSessionPaymentsResponse: {
    encode(message: QueryGetSessionPaymentsResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetSessionPaymentsResponse;
    fromJSON(object: any): QueryGetSessionPaymentsResponse;
    toJSON(message: QueryGetSessionPaymentsResponse): unknown;
    fromPartial(object: DeepPartial<QueryGetSessionPaymentsResponse>): QueryGetSessionPaymentsResponse;
};
export declare const QueryAllSessionPaymentsRequest: {
    encode(message: QueryAllSessionPaymentsRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryAllSessionPaymentsRequest;
    fromJSON(object: any): QueryAllSessionPaymentsRequest;
    toJSON(message: QueryAllSessionPaymentsRequest): unknown;
    fromPartial(object: DeepPartial<QueryAllSessionPaymentsRequest>): QueryAllSessionPaymentsRequest;
};
export declare const QueryAllSessionPaymentsResponse: {
    encode(message: QueryAllSessionPaymentsResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryAllSessionPaymentsResponse;
    fromJSON(object: any): QueryAllSessionPaymentsResponse;
    toJSON(message: QueryAllSessionPaymentsResponse): unknown;
    fromPartial(object: DeepPartial<QueryAllSessionPaymentsResponse>): QueryAllSessionPaymentsResponse;
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
    /** Queries a list of GetPairing items. */
    GetPairing(request: QueryGetPairingRequest): Promise<QueryGetPairingResponse>;
    /** Queries a CurrentSessionStart by index. */
    CurrentSessionStart(request: QueryGetCurrentSessionStartRequest): Promise<QueryGetCurrentSessionStartResponse>;
    /** Queries a PreviousSessionBlocks by index. */
    PreviousSessionBlocks(request: QueryGetPreviousSessionBlocksRequest): Promise<QueryGetPreviousSessionBlocksResponse>;
    /** Queries a SessionStorageForSpec by index. */
    SessionStorageForSpec(request: QueryGetSessionStorageForSpecRequest): Promise<QueryGetSessionStorageForSpecResponse>;
    /** Queries a list of SessionStorageForSpec items. */
    SessionStorageForSpecAll(request: QueryAllSessionStorageForSpecRequest): Promise<QueryAllSessionStorageForSpecResponse>;
    /** Queries a list of SessionStorageForAllSpecs items. */
    SessionStorageForAllSpecs(request: QuerySessionStorageForAllSpecsRequest): Promise<QuerySessionStorageForAllSpecsResponse>;
    /** Queries a list of AllSessionStoragesForSpec items. */
    AllSessionStoragesForSpec(request: QueryAllSessionStoragesForSpecRequest): Promise<QueryAllSessionStoragesForSpecResponse>;
    /** Queries a EarliestSessionStart by index. */
    EarliestSessionStart(request: QueryGetEarliestSessionStartRequest): Promise<QueryGetEarliestSessionStartResponse>;
    /** Queries a list of VerifyPairing items. */
    VerifyPairing(request: QueryVerifyPairingRequest): Promise<QueryVerifyPairingResponse>;
    /** Queries a UniquePaymentStorageUserServicer by index. */
    UniquePaymentStorageUserServicer(request: QueryGetUniquePaymentStorageUserServicerRequest): Promise<QueryGetUniquePaymentStorageUserServicerResponse>;
    /** Queries a list of UniquePaymentStorageUserServicer items. */
    UniquePaymentStorageUserServicerAll(request: QueryAllUniquePaymentStorageUserServicerRequest): Promise<QueryAllUniquePaymentStorageUserServicerResponse>;
    /** Queries a UserPaymentStorage by index. */
    UserPaymentStorage(request: QueryGetUserPaymentStorageRequest): Promise<QueryGetUserPaymentStorageResponse>;
    /** Queries a list of UserPaymentStorage items. */
    UserPaymentStorageAll(request: QueryAllUserPaymentStorageRequest): Promise<QueryAllUserPaymentStorageResponse>;
    /** Queries a SessionPayments by index. */
    SessionPayments(request: QueryGetSessionPaymentsRequest): Promise<QueryGetSessionPaymentsResponse>;
    /** Queries a list of SessionPayments items. */
    SessionPaymentsAll(request: QueryAllSessionPaymentsRequest): Promise<QueryAllSessionPaymentsResponse>;
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
    GetPairing(request: QueryGetPairingRequest): Promise<QueryGetPairingResponse>;
    CurrentSessionStart(request: QueryGetCurrentSessionStartRequest): Promise<QueryGetCurrentSessionStartResponse>;
    PreviousSessionBlocks(request: QueryGetPreviousSessionBlocksRequest): Promise<QueryGetPreviousSessionBlocksResponse>;
    SessionStorageForSpec(request: QueryGetSessionStorageForSpecRequest): Promise<QueryGetSessionStorageForSpecResponse>;
    SessionStorageForSpecAll(request: QueryAllSessionStorageForSpecRequest): Promise<QueryAllSessionStorageForSpecResponse>;
    SessionStorageForAllSpecs(request: QuerySessionStorageForAllSpecsRequest): Promise<QuerySessionStorageForAllSpecsResponse>;
    AllSessionStoragesForSpec(request: QueryAllSessionStoragesForSpecRequest): Promise<QueryAllSessionStoragesForSpecResponse>;
    EarliestSessionStart(request: QueryGetEarliestSessionStartRequest): Promise<QueryGetEarliestSessionStartResponse>;
    VerifyPairing(request: QueryVerifyPairingRequest): Promise<QueryVerifyPairingResponse>;
    UniquePaymentStorageUserServicer(request: QueryGetUniquePaymentStorageUserServicerRequest): Promise<QueryGetUniquePaymentStorageUserServicerResponse>;
    UniquePaymentStorageUserServicerAll(request: QueryAllUniquePaymentStorageUserServicerRequest): Promise<QueryAllUniquePaymentStorageUserServicerResponse>;
    UserPaymentStorage(request: QueryGetUserPaymentStorageRequest): Promise<QueryGetUserPaymentStorageResponse>;
    UserPaymentStorageAll(request: QueryAllUserPaymentStorageRequest): Promise<QueryAllUserPaymentStorageResponse>;
    SessionPayments(request: QueryGetSessionPaymentsRequest): Promise<QueryGetSessionPaymentsResponse>;
    SessionPaymentsAll(request: QueryAllSessionPaymentsRequest): Promise<QueryAllSessionPaymentsResponse>;
}
interface Rpc {
    request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};

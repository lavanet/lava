import Long from "long";
import _m0 from "protobufjs/minimal";
import { PageRequest, PageResponse } from "../cosmos/base/query/v1beta1/pagination";
import { Params } from "./params";
import { Spec } from "./spec";
export declare const protobufPackage = "lavanet.lava.spec";
/** QueryParamsRequest is request type for the Query/Params RPC method. */
export interface QueryParamsRequest {
}
/** QueryParamsResponse is response type for the Query/Params RPC method. */
export interface QueryParamsResponse {
    /** params holds all the parameters of this module. */
    params?: Params;
}
export interface QueryGetSpecRequest {
    ChainID: string;
}
export interface QueryGetSpecResponse {
    Spec?: Spec;
}
export interface QueryAllSpecRequest {
    pagination?: PageRequest;
}
export interface QueryAllSpecResponse {
    Spec: Spec[];
    pagination?: PageResponse;
}
export interface QueryShowAllChainsRequest {
}
export interface QueryShowAllChainsResponse {
    chainInfoList: ShowAllChainsInfoStruct[];
}
export interface ShowAllChainsInfoStruct {
    chainName: string;
    chainID: string;
    enabledApiInterfaces: string[];
    apiCount: Long;
}
export interface QueryShowChainInfoRequest {
    chainName: string;
}
export interface ApiList {
    interface: string;
    supportedApis: string[];
}
export interface QueryShowChainInfoResponse {
    chainID: string;
    interfaces: string[];
    supportedApisInterfaceList: ApiList[];
}
export declare const QueryParamsRequest: {
    encode(_: QueryParamsRequest, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): QueryParamsRequest;
    fromJSON(_: any): QueryParamsRequest;
    toJSON(_: QueryParamsRequest): unknown;
    create<I extends {} & {} & { [K in Exclude<keyof I, never>]: never; }>(base?: I | undefined): QueryParamsRequest;
    fromPartial<I_1 extends {} & {} & { [K_1 in Exclude<keyof I_1, never>]: never; }>(_: I_1): QueryParamsRequest;
};
export declare const QueryParamsResponse: {
    encode(message: QueryParamsResponse, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): QueryParamsResponse;
    fromJSON(object: any): QueryParamsResponse;
    toJSON(message: QueryParamsResponse): unknown;
    create<I extends {
        params?: {
            geolocationCount?: string | number | Long | undefined;
            maxCU?: string | number | Long | undefined;
        } | undefined;
    } & {
        params?: ({
            geolocationCount?: string | number | Long | undefined;
            maxCU?: string | number | Long | undefined;
        } & {
            geolocationCount?: string | number | (Long & {
                high: number;
                low: number;
                unsigned: boolean;
                add: (addend: string | number | Long) => Long;
                and: (other: string | number | Long) => Long;
                compare: (other: string | number | Long) => number;
                comp: (other: string | number | Long) => number;
                divide: (divisor: string | number | Long) => Long;
                div: (divisor: string | number | Long) => Long;
                equals: (other: string | number | Long) => boolean;
                eq: (other: string | number | Long) => boolean;
                getHighBits: () => number;
                getHighBitsUnsigned: () => number;
                getLowBits: () => number;
                getLowBitsUnsigned: () => number;
                getNumBitsAbs: () => number;
                greaterThan: (other: string | number | Long) => boolean;
                gt: (other: string | number | Long) => boolean;
                greaterThanOrEqual: (other: string | number | Long) => boolean;
                gte: (other: string | number | Long) => boolean;
                ge: (other: string | number | Long) => boolean;
                isEven: () => boolean;
                isNegative: () => boolean;
                isOdd: () => boolean;
                isPositive: () => boolean;
                isZero: () => boolean;
                eqz: () => boolean;
                lessThan: (other: string | number | Long) => boolean;
                lt: (other: string | number | Long) => boolean;
                lessThanOrEqual: (other: string | number | Long) => boolean;
                lte: (other: string | number | Long) => boolean;
                le: (other: string | number | Long) => boolean;
                modulo: (other: string | number | Long) => Long;
                mod: (other: string | number | Long) => Long;
                rem: (other: string | number | Long) => Long;
                multiply: (multiplier: string | number | Long) => Long;
                mul: (multiplier: string | number | Long) => Long;
                negate: () => Long;
                neg: () => Long;
                not: () => Long;
                countLeadingZeros: () => number;
                clz: () => number;
                countTrailingZeros: () => number;
                ctz: () => number;
                notEquals: (other: string | number | Long) => boolean;
                neq: (other: string | number | Long) => boolean;
                ne: (other: string | number | Long) => boolean;
                or: (other: string | number | Long) => Long;
                shiftLeft: (numBits: number | Long) => Long;
                shl: (numBits: number | Long) => Long;
                shiftRight: (numBits: number | Long) => Long;
                shr: (numBits: number | Long) => Long;
                shiftRightUnsigned: (numBits: number | Long) => Long;
                shru: (numBits: number | Long) => Long;
                shr_u: (numBits: number | Long) => Long;
                rotateLeft: (numBits: number | Long) => Long;
                rotl: (numBits: number | Long) => Long;
                rotateRight: (numBits: number | Long) => Long;
                rotr: (numBits: number | Long) => Long;
                subtract: (subtrahend: string | number | Long) => Long;
                sub: (subtrahend: string | number | Long) => Long;
                toInt: () => number;
                toNumber: () => number;
                toBytes: (le?: boolean | undefined) => number[];
                toBytesLE: () => number[];
                toBytesBE: () => number[];
                toSigned: () => Long;
                toString: (radix?: number | undefined) => string;
                toUnsigned: () => Long;
                xor: (other: string | number | Long) => Long;
            } & { [K in Exclude<keyof I["params"]["geolocationCount"], keyof Long>]: never; }) | undefined;
            maxCU?: string | number | (Long & {
                high: number;
                low: number;
                unsigned: boolean;
                add: (addend: string | number | Long) => Long;
                and: (other: string | number | Long) => Long;
                compare: (other: string | number | Long) => number;
                comp: (other: string | number | Long) => number;
                divide: (divisor: string | number | Long) => Long;
                div: (divisor: string | number | Long) => Long;
                equals: (other: string | number | Long) => boolean;
                eq: (other: string | number | Long) => boolean;
                getHighBits: () => number;
                getHighBitsUnsigned: () => number;
                getLowBits: () => number;
                getLowBitsUnsigned: () => number;
                getNumBitsAbs: () => number;
                greaterThan: (other: string | number | Long) => boolean;
                gt: (other: string | number | Long) => boolean;
                greaterThanOrEqual: (other: string | number | Long) => boolean;
                gte: (other: string | number | Long) => boolean;
                ge: (other: string | number | Long) => boolean;
                isEven: () => boolean;
                isNegative: () => boolean;
                isOdd: () => boolean;
                isPositive: () => boolean;
                isZero: () => boolean;
                eqz: () => boolean;
                lessThan: (other: string | number | Long) => boolean;
                lt: (other: string | number | Long) => boolean;
                lessThanOrEqual: (other: string | number | Long) => boolean;
                lte: (other: string | number | Long) => boolean;
                le: (other: string | number | Long) => boolean;
                modulo: (other: string | number | Long) => Long;
                mod: (other: string | number | Long) => Long;
                rem: (other: string | number | Long) => Long;
                multiply: (multiplier: string | number | Long) => Long;
                mul: (multiplier: string | number | Long) => Long;
                negate: () => Long;
                neg: () => Long;
                not: () => Long;
                countLeadingZeros: () => number;
                clz: () => number;
                countTrailingZeros: () => number;
                ctz: () => number;
                notEquals: (other: string | number | Long) => boolean;
                neq: (other: string | number | Long) => boolean;
                ne: (other: string | number | Long) => boolean;
                or: (other: string | number | Long) => Long;
                shiftLeft: (numBits: number | Long) => Long;
                shl: (numBits: number | Long) => Long;
                shiftRight: (numBits: number | Long) => Long;
                shr: (numBits: number | Long) => Long;
                shiftRightUnsigned: (numBits: number | Long) => Long;
                shru: (numBits: number | Long) => Long;
                shr_u: (numBits: number | Long) => Long;
                rotateLeft: (numBits: number | Long) => Long;
                rotl: (numBits: number | Long) => Long;
                rotateRight: (numBits: number | Long) => Long;
                rotr: (numBits: number | Long) => Long;
                subtract: (subtrahend: string | number | Long) => Long;
                sub: (subtrahend: string | number | Long) => Long;
                toInt: () => number;
                toNumber: () => number;
                toBytes: (le?: boolean | undefined) => number[];
                toBytesLE: () => number[];
                toBytesBE: () => number[];
                toSigned: () => Long;
                toString: (radix?: number | undefined) => string;
                toUnsigned: () => Long;
                xor: (other: string | number | Long) => Long;
            } & { [K_1 in Exclude<keyof I["params"]["maxCU"], keyof Long>]: never; }) | undefined;
        } & { [K_2 in Exclude<keyof I["params"], keyof Params>]: never; }) | undefined;
    } & { [K_3 in Exclude<keyof I, "params">]: never; }>(base?: I | undefined): QueryParamsResponse;
    fromPartial<I_1 extends {
        params?: {
            geolocationCount?: string | number | Long | undefined;
            maxCU?: string | number | Long | undefined;
        } | undefined;
    } & {
        params?: ({
            geolocationCount?: string | number | Long | undefined;
            maxCU?: string | number | Long | undefined;
        } & {
            geolocationCount?: string | number | (Long & {
                high: number;
                low: number;
                unsigned: boolean;
                add: (addend: string | number | Long) => Long;
                and: (other: string | number | Long) => Long;
                compare: (other: string | number | Long) => number;
                comp: (other: string | number | Long) => number;
                divide: (divisor: string | number | Long) => Long;
                div: (divisor: string | number | Long) => Long;
                equals: (other: string | number | Long) => boolean;
                eq: (other: string | number | Long) => boolean;
                getHighBits: () => number;
                getHighBitsUnsigned: () => number;
                getLowBits: () => number;
                getLowBitsUnsigned: () => number;
                getNumBitsAbs: () => number;
                greaterThan: (other: string | number | Long) => boolean;
                gt: (other: string | number | Long) => boolean;
                greaterThanOrEqual: (other: string | number | Long) => boolean;
                gte: (other: string | number | Long) => boolean;
                ge: (other: string | number | Long) => boolean;
                isEven: () => boolean;
                isNegative: () => boolean;
                isOdd: () => boolean;
                isPositive: () => boolean;
                isZero: () => boolean;
                eqz: () => boolean;
                lessThan: (other: string | number | Long) => boolean;
                lt: (other: string | number | Long) => boolean;
                lessThanOrEqual: (other: string | number | Long) => boolean;
                lte: (other: string | number | Long) => boolean;
                le: (other: string | number | Long) => boolean;
                modulo: (other: string | number | Long) => Long;
                mod: (other: string | number | Long) => Long;
                rem: (other: string | number | Long) => Long;
                multiply: (multiplier: string | number | Long) => Long;
                mul: (multiplier: string | number | Long) => Long;
                negate: () => Long;
                neg: () => Long;
                not: () => Long;
                countLeadingZeros: () => number;
                clz: () => number;
                countTrailingZeros: () => number;
                ctz: () => number;
                notEquals: (other: string | number | Long) => boolean;
                neq: (other: string | number | Long) => boolean;
                ne: (other: string | number | Long) => boolean;
                or: (other: string | number | Long) => Long;
                shiftLeft: (numBits: number | Long) => Long;
                shl: (numBits: number | Long) => Long;
                shiftRight: (numBits: number | Long) => Long;
                shr: (numBits: number | Long) => Long;
                shiftRightUnsigned: (numBits: number | Long) => Long;
                shru: (numBits: number | Long) => Long;
                shr_u: (numBits: number | Long) => Long;
                rotateLeft: (numBits: number | Long) => Long;
                rotl: (numBits: number | Long) => Long;
                rotateRight: (numBits: number | Long) => Long;
                rotr: (numBits: number | Long) => Long;
                subtract: (subtrahend: string | number | Long) => Long;
                sub: (subtrahend: string | number | Long) => Long;
                toInt: () => number;
                toNumber: () => number;
                toBytes: (le?: boolean | undefined) => number[];
                toBytesLE: () => number[];
                toBytesBE: () => number[];
                toSigned: () => Long;
                toString: (radix?: number | undefined) => string;
                toUnsigned: () => Long;
                xor: (other: string | number | Long) => Long;
            } & { [K_4 in Exclude<keyof I_1["params"]["geolocationCount"], keyof Long>]: never; }) | undefined;
            maxCU?: string | number | (Long & {
                high: number;
                low: number;
                unsigned: boolean;
                add: (addend: string | number | Long) => Long;
                and: (other: string | number | Long) => Long;
                compare: (other: string | number | Long) => number;
                comp: (other: string | number | Long) => number;
                divide: (divisor: string | number | Long) => Long;
                div: (divisor: string | number | Long) => Long;
                equals: (other: string | number | Long) => boolean;
                eq: (other: string | number | Long) => boolean;
                getHighBits: () => number;
                getHighBitsUnsigned: () => number;
                getLowBits: () => number;
                getLowBitsUnsigned: () => number;
                getNumBitsAbs: () => number;
                greaterThan: (other: string | number | Long) => boolean;
                gt: (other: string | number | Long) => boolean;
                greaterThanOrEqual: (other: string | number | Long) => boolean;
                gte: (other: string | number | Long) => boolean;
                ge: (other: string | number | Long) => boolean;
                isEven: () => boolean;
                isNegative: () => boolean;
                isOdd: () => boolean;
                isPositive: () => boolean;
                isZero: () => boolean;
                eqz: () => boolean;
                lessThan: (other: string | number | Long) => boolean;
                lt: (other: string | number | Long) => boolean;
                lessThanOrEqual: (other: string | number | Long) => boolean;
                lte: (other: string | number | Long) => boolean;
                le: (other: string | number | Long) => boolean;
                modulo: (other: string | number | Long) => Long;
                mod: (other: string | number | Long) => Long;
                rem: (other: string | number | Long) => Long;
                multiply: (multiplier: string | number | Long) => Long;
                mul: (multiplier: string | number | Long) => Long;
                negate: () => Long;
                neg: () => Long;
                not: () => Long;
                countLeadingZeros: () => number;
                clz: () => number;
                countTrailingZeros: () => number;
                ctz: () => number;
                notEquals: (other: string | number | Long) => boolean;
                neq: (other: string | number | Long) => boolean;
                ne: (other: string | number | Long) => boolean;
                or: (other: string | number | Long) => Long;
                shiftLeft: (numBits: number | Long) => Long;
                shl: (numBits: number | Long) => Long;
                shiftRight: (numBits: number | Long) => Long;
                shr: (numBits: number | Long) => Long;
                shiftRightUnsigned: (numBits: number | Long) => Long;
                shru: (numBits: number | Long) => Long;
                shr_u: (numBits: number | Long) => Long;
                rotateLeft: (numBits: number | Long) => Long;
                rotl: (numBits: number | Long) => Long;
                rotateRight: (numBits: number | Long) => Long;
                rotr: (numBits: number | Long) => Long;
                subtract: (subtrahend: string | number | Long) => Long;
                sub: (subtrahend: string | number | Long) => Long;
                toInt: () => number;
                toNumber: () => number;
                toBytes: (le?: boolean | undefined) => number[];
                toBytesLE: () => number[];
                toBytesBE: () => number[];
                toSigned: () => Long;
                toString: (radix?: number | undefined) => string;
                toUnsigned: () => Long;
                xor: (other: string | number | Long) => Long;
            } & { [K_5 in Exclude<keyof I_1["params"]["maxCU"], keyof Long>]: never; }) | undefined;
        } & { [K_6 in Exclude<keyof I_1["params"], keyof Params>]: never; }) | undefined;
    } & { [K_7 in Exclude<keyof I_1, "params">]: never; }>(object: I_1): QueryParamsResponse;
};
export declare const QueryGetSpecRequest: {
    encode(message: QueryGetSpecRequest, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetSpecRequest;
    fromJSON(object: any): QueryGetSpecRequest;
    toJSON(message: QueryGetSpecRequest): unknown;
    create<I extends {
        ChainID?: string | undefined;
    } & {
        ChainID?: string | undefined;
    } & { [K in Exclude<keyof I, "ChainID">]: never; }>(base?: I | undefined): QueryGetSpecRequest;
    fromPartial<I_1 extends {
        ChainID?: string | undefined;
    } & {
        ChainID?: string | undefined;
    } & { [K_1 in Exclude<keyof I_1, "ChainID">]: never; }>(object: I_1): QueryGetSpecRequest;
};
export declare const QueryGetSpecResponse: {
    encode(message: QueryGetSpecResponse, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetSpecResponse;
    fromJSON(object: any): QueryGetSpecResponse;
    toJSON(message: QueryGetSpecResponse): unknown;
    create<I extends {
        Spec?: {
            index?: string | undefined;
            name?: string | undefined;
            enabled?: boolean | undefined;
            reliabilityThreshold?: number | undefined;
            dataReliabilityEnabled?: boolean | undefined;
            blockDistanceForFinalizedData?: number | undefined;
            blocksInFinalizationProof?: number | undefined;
            averageBlockTime?: string | number | Long | undefined;
            allowedBlockLagForQosSync?: string | number | Long | undefined;
            blockLastUpdated?: string | number | Long | undefined;
            minStakeProvider?: {
                denom?: string | undefined;
                amount?: string | undefined;
            } | undefined;
            minStakeClient?: {
                denom?: string | undefined;
                amount?: string | undefined;
            } | undefined;
            providersTypes?: import("./spec").Spec_ProvidersTypes | undefined;
            imports?: string[] | undefined;
            apiCollections?: {
                enabled?: boolean | undefined;
                collectionData?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } | undefined;
                apis?: {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] | undefined;
                headers?: {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] | undefined;
                inheritanceApis?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] | undefined;
                parseDirectives?: {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] | undefined;
            }[] | undefined;
        } | undefined;
    } & {
        Spec?: ({
            index?: string | undefined;
            name?: string | undefined;
            enabled?: boolean | undefined;
            reliabilityThreshold?: number | undefined;
            dataReliabilityEnabled?: boolean | undefined;
            blockDistanceForFinalizedData?: number | undefined;
            blocksInFinalizationProof?: number | undefined;
            averageBlockTime?: string | number | Long | undefined;
            allowedBlockLagForQosSync?: string | number | Long | undefined;
            blockLastUpdated?: string | number | Long | undefined;
            minStakeProvider?: {
                denom?: string | undefined;
                amount?: string | undefined;
            } | undefined;
            minStakeClient?: {
                denom?: string | undefined;
                amount?: string | undefined;
            } | undefined;
            providersTypes?: import("./spec").Spec_ProvidersTypes | undefined;
            imports?: string[] | undefined;
            apiCollections?: {
                enabled?: boolean | undefined;
                collectionData?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } | undefined;
                apis?: {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] | undefined;
                headers?: {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] | undefined;
                inheritanceApis?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] | undefined;
                parseDirectives?: {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] | undefined;
            }[] | undefined;
        } & {
            index?: string | undefined;
            name?: string | undefined;
            enabled?: boolean | undefined;
            reliabilityThreshold?: number | undefined;
            dataReliabilityEnabled?: boolean | undefined;
            blockDistanceForFinalizedData?: number | undefined;
            blocksInFinalizationProof?: number | undefined;
            averageBlockTime?: string | number | (Long & {
                high: number;
                low: number;
                unsigned: boolean;
                add: (addend: string | number | Long) => Long;
                and: (other: string | number | Long) => Long;
                compare: (other: string | number | Long) => number;
                comp: (other: string | number | Long) => number;
                divide: (divisor: string | number | Long) => Long;
                div: (divisor: string | number | Long) => Long;
                equals: (other: string | number | Long) => boolean;
                eq: (other: string | number | Long) => boolean;
                getHighBits: () => number;
                getHighBitsUnsigned: () => number;
                getLowBits: () => number;
                getLowBitsUnsigned: () => number;
                getNumBitsAbs: () => number;
                greaterThan: (other: string | number | Long) => boolean;
                gt: (other: string | number | Long) => boolean;
                greaterThanOrEqual: (other: string | number | Long) => boolean;
                gte: (other: string | number | Long) => boolean;
                ge: (other: string | number | Long) => boolean;
                isEven: () => boolean;
                isNegative: () => boolean;
                isOdd: () => boolean;
                isPositive: () => boolean;
                isZero: () => boolean;
                eqz: () => boolean;
                lessThan: (other: string | number | Long) => boolean;
                lt: (other: string | number | Long) => boolean;
                lessThanOrEqual: (other: string | number | Long) => boolean;
                lte: (other: string | number | Long) => boolean;
                le: (other: string | number | Long) => boolean;
                modulo: (other: string | number | Long) => Long;
                mod: (other: string | number | Long) => Long;
                rem: (other: string | number | Long) => Long;
                multiply: (multiplier: string | number | Long) => Long;
                mul: (multiplier: string | number | Long) => Long;
                negate: () => Long;
                neg: () => Long;
                not: () => Long;
                countLeadingZeros: () => number;
                clz: () => number;
                countTrailingZeros: () => number;
                ctz: () => number;
                notEquals: (other: string | number | Long) => boolean;
                neq: (other: string | number | Long) => boolean;
                ne: (other: string | number | Long) => boolean;
                or: (other: string | number | Long) => Long;
                shiftLeft: (numBits: number | Long) => Long;
                shl: (numBits: number | Long) => Long;
                shiftRight: (numBits: number | Long) => Long;
                shr: (numBits: number | Long) => Long;
                shiftRightUnsigned: (numBits: number | Long) => Long;
                shru: (numBits: number | Long) => Long;
                shr_u: (numBits: number | Long) => Long;
                rotateLeft: (numBits: number | Long) => Long;
                rotl: (numBits: number | Long) => Long;
                rotateRight: (numBits: number | Long) => Long;
                rotr: (numBits: number | Long) => Long;
                subtract: (subtrahend: string | number | Long) => Long;
                sub: (subtrahend: string | number | Long) => Long;
                toInt: () => number;
                toNumber: () => number;
                toBytes: (le?: boolean | undefined) => number[];
                toBytesLE: () => number[];
                toBytesBE: () => number[];
                toSigned: () => Long;
                toString: (radix?: number | undefined) => string;
                toUnsigned: () => Long;
                xor: (other: string | number | Long) => Long;
            } & { [K in Exclude<keyof I["Spec"]["averageBlockTime"], keyof Long>]: never; }) | undefined;
            allowedBlockLagForQosSync?: string | number | (Long & {
                high: number;
                low: number;
                unsigned: boolean;
                add: (addend: string | number | Long) => Long;
                and: (other: string | number | Long) => Long;
                compare: (other: string | number | Long) => number;
                comp: (other: string | number | Long) => number;
                divide: (divisor: string | number | Long) => Long;
                div: (divisor: string | number | Long) => Long;
                equals: (other: string | number | Long) => boolean;
                eq: (other: string | number | Long) => boolean;
                getHighBits: () => number;
                getHighBitsUnsigned: () => number;
                getLowBits: () => number;
                getLowBitsUnsigned: () => number;
                getNumBitsAbs: () => number;
                greaterThan: (other: string | number | Long) => boolean;
                gt: (other: string | number | Long) => boolean;
                greaterThanOrEqual: (other: string | number | Long) => boolean;
                gte: (other: string | number | Long) => boolean;
                ge: (other: string | number | Long) => boolean;
                isEven: () => boolean;
                isNegative: () => boolean;
                isOdd: () => boolean;
                isPositive: () => boolean;
                isZero: () => boolean;
                eqz: () => boolean;
                lessThan: (other: string | number | Long) => boolean;
                lt: (other: string | number | Long) => boolean;
                lessThanOrEqual: (other: string | number | Long) => boolean;
                lte: (other: string | number | Long) => boolean;
                le: (other: string | number | Long) => boolean;
                modulo: (other: string | number | Long) => Long;
                mod: (other: string | number | Long) => Long;
                rem: (other: string | number | Long) => Long;
                multiply: (multiplier: string | number | Long) => Long;
                mul: (multiplier: string | number | Long) => Long;
                negate: () => Long;
                neg: () => Long;
                not: () => Long;
                countLeadingZeros: () => number;
                clz: () => number;
                countTrailingZeros: () => number;
                ctz: () => number;
                notEquals: (other: string | number | Long) => boolean;
                neq: (other: string | number | Long) => boolean;
                ne: (other: string | number | Long) => boolean;
                or: (other: string | number | Long) => Long;
                shiftLeft: (numBits: number | Long) => Long;
                shl: (numBits: number | Long) => Long;
                shiftRight: (numBits: number | Long) => Long;
                shr: (numBits: number | Long) => Long;
                shiftRightUnsigned: (numBits: number | Long) => Long;
                shru: (numBits: number | Long) => Long;
                shr_u: (numBits: number | Long) => Long;
                rotateLeft: (numBits: number | Long) => Long;
                rotl: (numBits: number | Long) => Long;
                rotateRight: (numBits: number | Long) => Long;
                rotr: (numBits: number | Long) => Long;
                subtract: (subtrahend: string | number | Long) => Long;
                sub: (subtrahend: string | number | Long) => Long;
                toInt: () => number;
                toNumber: () => number;
                toBytes: (le?: boolean | undefined) => number[];
                toBytesLE: () => number[];
                toBytesBE: () => number[];
                toSigned: () => Long;
                toString: (radix?: number | undefined) => string;
                toUnsigned: () => Long;
                xor: (other: string | number | Long) => Long;
            } & { [K_1 in Exclude<keyof I["Spec"]["allowedBlockLagForQosSync"], keyof Long>]: never; }) | undefined;
            blockLastUpdated?: string | number | (Long & {
                high: number;
                low: number;
                unsigned: boolean;
                add: (addend: string | number | Long) => Long;
                and: (other: string | number | Long) => Long;
                compare: (other: string | number | Long) => number;
                comp: (other: string | number | Long) => number;
                divide: (divisor: string | number | Long) => Long;
                div: (divisor: string | number | Long) => Long;
                equals: (other: string | number | Long) => boolean;
                eq: (other: string | number | Long) => boolean;
                getHighBits: () => number;
                getHighBitsUnsigned: () => number;
                getLowBits: () => number;
                getLowBitsUnsigned: () => number;
                getNumBitsAbs: () => number;
                greaterThan: (other: string | number | Long) => boolean;
                gt: (other: string | number | Long) => boolean;
                greaterThanOrEqual: (other: string | number | Long) => boolean;
                gte: (other: string | number | Long) => boolean;
                ge: (other: string | number | Long) => boolean;
                isEven: () => boolean;
                isNegative: () => boolean;
                isOdd: () => boolean;
                isPositive: () => boolean;
                isZero: () => boolean;
                eqz: () => boolean;
                lessThan: (other: string | number | Long) => boolean;
                lt: (other: string | number | Long) => boolean;
                lessThanOrEqual: (other: string | number | Long) => boolean;
                lte: (other: string | number | Long) => boolean;
                le: (other: string | number | Long) => boolean;
                modulo: (other: string | number | Long) => Long;
                mod: (other: string | number | Long) => Long;
                rem: (other: string | number | Long) => Long;
                multiply: (multiplier: string | number | Long) => Long;
                mul: (multiplier: string | number | Long) => Long;
                negate: () => Long;
                neg: () => Long;
                not: () => Long;
                countLeadingZeros: () => number;
                clz: () => number;
                countTrailingZeros: () => number;
                ctz: () => number;
                notEquals: (other: string | number | Long) => boolean;
                neq: (other: string | number | Long) => boolean;
                ne: (other: string | number | Long) => boolean;
                or: (other: string | number | Long) => Long;
                shiftLeft: (numBits: number | Long) => Long;
                shl: (numBits: number | Long) => Long;
                shiftRight: (numBits: number | Long) => Long;
                shr: (numBits: number | Long) => Long;
                shiftRightUnsigned: (numBits: number | Long) => Long;
                shru: (numBits: number | Long) => Long;
                shr_u: (numBits: number | Long) => Long;
                rotateLeft: (numBits: number | Long) => Long;
                rotl: (numBits: number | Long) => Long;
                rotateRight: (numBits: number | Long) => Long;
                rotr: (numBits: number | Long) => Long;
                subtract: (subtrahend: string | number | Long) => Long;
                sub: (subtrahend: string | number | Long) => Long;
                toInt: () => number;
                toNumber: () => number;
                toBytes: (le?: boolean | undefined) => number[];
                toBytesLE: () => number[];
                toBytesBE: () => number[];
                toSigned: () => Long;
                toString: (radix?: number | undefined) => string;
                toUnsigned: () => Long;
                xor: (other: string | number | Long) => Long;
            } & { [K_2 in Exclude<keyof I["Spec"]["blockLastUpdated"], keyof Long>]: never; }) | undefined;
            minStakeProvider?: ({
                denom?: string | undefined;
                amount?: string | undefined;
            } & {
                denom?: string | undefined;
                amount?: string | undefined;
            } & { [K_3 in Exclude<keyof I["Spec"]["minStakeProvider"], keyof import("../cosmos/base/v1beta1/coin").Coin>]: never; }) | undefined;
            minStakeClient?: ({
                denom?: string | undefined;
                amount?: string | undefined;
            } & {
                denom?: string | undefined;
                amount?: string | undefined;
            } & { [K_4 in Exclude<keyof I["Spec"]["minStakeClient"], keyof import("../cosmos/base/v1beta1/coin").Coin>]: never; }) | undefined;
            providersTypes?: import("./spec").Spec_ProvidersTypes | undefined;
            imports?: (string[] & string[] & { [K_5 in Exclude<keyof I["Spec"]["imports"], keyof string[]>]: never; }) | undefined;
            apiCollections?: ({
                enabled?: boolean | undefined;
                collectionData?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } | undefined;
                apis?: {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] | undefined;
                headers?: {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] | undefined;
                inheritanceApis?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] | undefined;
                parseDirectives?: {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] | undefined;
            }[] & ({
                enabled?: boolean | undefined;
                collectionData?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } | undefined;
                apis?: {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] | undefined;
                headers?: {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] | undefined;
                inheritanceApis?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] | undefined;
                parseDirectives?: {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] | undefined;
            } & {
                enabled?: boolean | undefined;
                collectionData?: ({
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } & {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } & { [K_6 in Exclude<keyof I["Spec"]["apiCollections"][number]["collectionData"], keyof import("./api_collection").CollectionData>]: never; }) | undefined;
                apis?: ({
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] & ({
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                } & {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | (Long & {
                        high: number;
                        low: number;
                        unsigned: boolean;
                        add: (addend: string | number | Long) => Long;
                        and: (other: string | number | Long) => Long;
                        compare: (other: string | number | Long) => number;
                        comp: (other: string | number | Long) => number;
                        divide: (divisor: string | number | Long) => Long;
                        div: (divisor: string | number | Long) => Long;
                        equals: (other: string | number | Long) => boolean;
                        eq: (other: string | number | Long) => boolean;
                        getHighBits: () => number;
                        getHighBitsUnsigned: () => number;
                        getLowBits: () => number;
                        getLowBitsUnsigned: () => number;
                        getNumBitsAbs: () => number;
                        greaterThan: (other: string | number | Long) => boolean;
                        gt: (other: string | number | Long) => boolean;
                        greaterThanOrEqual: (other: string | number | Long) => boolean;
                        gte: (other: string | number | Long) => boolean;
                        ge: (other: string | number | Long) => boolean;
                        isEven: () => boolean;
                        isNegative: () => boolean;
                        isOdd: () => boolean;
                        isPositive: () => boolean;
                        isZero: () => boolean;
                        eqz: () => boolean;
                        lessThan: (other: string | number | Long) => boolean;
                        lt: (other: string | number | Long) => boolean;
                        lessThanOrEqual: (other: string | number | Long) => boolean;
                        lte: (other: string | number | Long) => boolean;
                        le: (other: string | number | Long) => boolean;
                        modulo: (other: string | number | Long) => Long;
                        mod: (other: string | number | Long) => Long;
                        rem: (other: string | number | Long) => Long;
                        multiply: (multiplier: string | number | Long) => Long;
                        mul: (multiplier: string | number | Long) => Long;
                        negate: () => Long;
                        neg: () => Long;
                        not: () => Long;
                        countLeadingZeros: () => number;
                        clz: () => number;
                        countTrailingZeros: () => number;
                        ctz: () => number;
                        notEquals: (other: string | number | Long) => boolean;
                        neq: (other: string | number | Long) => boolean;
                        ne: (other: string | number | Long) => boolean;
                        or: (other: string | number | Long) => Long;
                        shiftLeft: (numBits: number | Long) => Long;
                        shl: (numBits: number | Long) => Long;
                        shiftRight: (numBits: number | Long) => Long;
                        shr: (numBits: number | Long) => Long;
                        shiftRightUnsigned: (numBits: number | Long) => Long;
                        shru: (numBits: number | Long) => Long;
                        shr_u: (numBits: number | Long) => Long;
                        rotateLeft: (numBits: number | Long) => Long;
                        rotl: (numBits: number | Long) => Long;
                        rotateRight: (numBits: number | Long) => Long;
                        rotr: (numBits: number | Long) => Long;
                        subtract: (subtrahend: string | number | Long) => Long;
                        sub: (subtrahend: string | number | Long) => Long;
                        toInt: () => number;
                        toNumber: () => number;
                        toBytes: (le?: boolean | undefined) => number[];
                        toBytesLE: () => number[];
                        toBytesBE: () => number[];
                        toSigned: () => Long;
                        toString: (radix?: number | undefined) => string;
                        toUnsigned: () => Long;
                        xor: (other: string | number | Long) => Long;
                    } & { [K_7 in Exclude<keyof I["Spec"]["apiCollections"][number]["apis"][number]["computeUnits"], keyof Long>]: never; }) | undefined;
                    extraComputeUnits?: string | number | (Long & {
                        high: number;
                        low: number;
                        unsigned: boolean;
                        add: (addend: string | number | Long) => Long;
                        and: (other: string | number | Long) => Long;
                        compare: (other: string | number | Long) => number;
                        comp: (other: string | number | Long) => number;
                        divide: (divisor: string | number | Long) => Long;
                        div: (divisor: string | number | Long) => Long;
                        equals: (other: string | number | Long) => boolean;
                        eq: (other: string | number | Long) => boolean;
                        getHighBits: () => number;
                        getHighBitsUnsigned: () => number;
                        getLowBits: () => number;
                        getLowBitsUnsigned: () => number;
                        getNumBitsAbs: () => number;
                        greaterThan: (other: string | number | Long) => boolean;
                        gt: (other: string | number | Long) => boolean;
                        greaterThanOrEqual: (other: string | number | Long) => boolean;
                        gte: (other: string | number | Long) => boolean;
                        ge: (other: string | number | Long) => boolean;
                        isEven: () => boolean;
                        isNegative: () => boolean;
                        isOdd: () => boolean;
                        isPositive: () => boolean;
                        isZero: () => boolean;
                        eqz: () => boolean;
                        lessThan: (other: string | number | Long) => boolean;
                        lt: (other: string | number | Long) => boolean;
                        lessThanOrEqual: (other: string | number | Long) => boolean;
                        lte: (other: string | number | Long) => boolean;
                        le: (other: string | number | Long) => boolean;
                        modulo: (other: string | number | Long) => Long;
                        mod: (other: string | number | Long) => Long;
                        rem: (other: string | number | Long) => Long;
                        multiply: (multiplier: string | number | Long) => Long;
                        mul: (multiplier: string | number | Long) => Long;
                        negate: () => Long;
                        neg: () => Long;
                        not: () => Long;
                        countLeadingZeros: () => number;
                        clz: () => number;
                        countTrailingZeros: () => number;
                        ctz: () => number;
                        notEquals: (other: string | number | Long) => boolean;
                        neq: (other: string | number | Long) => boolean;
                        ne: (other: string | number | Long) => boolean;
                        or: (other: string | number | Long) => Long;
                        shiftLeft: (numBits: number | Long) => Long;
                        shl: (numBits: number | Long) => Long;
                        shiftRight: (numBits: number | Long) => Long;
                        shr: (numBits: number | Long) => Long;
                        shiftRightUnsigned: (numBits: number | Long) => Long;
                        shru: (numBits: number | Long) => Long;
                        shr_u: (numBits: number | Long) => Long;
                        rotateLeft: (numBits: number | Long) => Long;
                        rotl: (numBits: number | Long) => Long;
                        rotateRight: (numBits: number | Long) => Long;
                        rotr: (numBits: number | Long) => Long;
                        subtract: (subtrahend: string | number | Long) => Long;
                        sub: (subtrahend: string | number | Long) => Long;
                        toInt: () => number;
                        toNumber: () => number;
                        toBytes: (le?: boolean | undefined) => number[];
                        toBytesLE: () => number[];
                        toBytesBE: () => number[];
                        toSigned: () => Long;
                        toString: (radix?: number | undefined) => string;
                        toUnsigned: () => Long;
                        xor: (other: string | number | Long) => Long;
                    } & { [K_8 in Exclude<keyof I["Spec"]["apiCollections"][number]["apis"][number]["extraComputeUnits"], keyof Long>]: never; }) | undefined;
                    category?: ({
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } & {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } & { [K_9 in Exclude<keyof I["Spec"]["apiCollections"][number]["apis"][number]["category"], keyof import("./api_collection").SpecCategory>]: never; }) | undefined;
                    blockParsing?: ({
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } & {
                        parserArg?: (string[] & string[] & { [K_10 in Exclude<keyof I["Spec"]["apiCollections"][number]["apis"][number]["blockParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } & { [K_11 in Exclude<keyof I["Spec"]["apiCollections"][number]["apis"][number]["blockParsing"], keyof import("./api_collection").BlockParser>]: never; }) | undefined;
                } & { [K_12 in Exclude<keyof I["Spec"]["apiCollections"][number]["apis"][number], keyof import("./api_collection").Api>]: never; })[] & { [K_13 in Exclude<keyof I["Spec"]["apiCollections"][number]["apis"], keyof {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[]>]: never; }) | undefined;
                headers?: ({
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] & ({
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                } & {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                } & { [K_14 in Exclude<keyof I["Spec"]["apiCollections"][number]["headers"][number], keyof import("./api_collection").Header>]: never; })[] & { [K_15 in Exclude<keyof I["Spec"]["apiCollections"][number]["headers"], keyof {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[]>]: never; }) | undefined;
                inheritanceApis?: ({
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] & ({
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } & {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } & { [K_16 in Exclude<keyof I["Spec"]["apiCollections"][number]["inheritanceApis"][number], keyof import("./api_collection").CollectionData>]: never; })[] & { [K_17 in Exclude<keyof I["Spec"]["apiCollections"][number]["inheritanceApis"], keyof {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[]>]: never; }) | undefined;
                parseDirectives?: ({
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] & ({
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                } & {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: ({
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } & {
                        parserArg?: (string[] & string[] & { [K_18 in Exclude<keyof I["Spec"]["apiCollections"][number]["parseDirectives"][number]["resultParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } & { [K_19 in Exclude<keyof I["Spec"]["apiCollections"][number]["parseDirectives"][number]["resultParsing"], keyof import("./api_collection").BlockParser>]: never; }) | undefined;
                    apiName?: string | undefined;
                } & { [K_20 in Exclude<keyof I["Spec"]["apiCollections"][number]["parseDirectives"][number], keyof import("./api_collection").ParseDirective>]: never; })[] & { [K_21 in Exclude<keyof I["Spec"]["apiCollections"][number]["parseDirectives"], keyof {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[]>]: never; }) | undefined;
            } & { [K_22 in Exclude<keyof I["Spec"]["apiCollections"][number], keyof import("./api_collection").ApiCollection>]: never; })[] & { [K_23 in Exclude<keyof I["Spec"]["apiCollections"], keyof {
                enabled?: boolean | undefined;
                collectionData?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } | undefined;
                apis?: {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] | undefined;
                headers?: {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] | undefined;
                inheritanceApis?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] | undefined;
                parseDirectives?: {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] | undefined;
            }[]>]: never; }) | undefined;
        } & { [K_24 in Exclude<keyof I["Spec"], keyof Spec>]: never; }) | undefined;
    } & { [K_25 in Exclude<keyof I, "Spec">]: never; }>(base?: I | undefined): QueryGetSpecResponse;
    fromPartial<I_1 extends {
        Spec?: {
            index?: string | undefined;
            name?: string | undefined;
            enabled?: boolean | undefined;
            reliabilityThreshold?: number | undefined;
            dataReliabilityEnabled?: boolean | undefined;
            blockDistanceForFinalizedData?: number | undefined;
            blocksInFinalizationProof?: number | undefined;
            averageBlockTime?: string | number | Long | undefined;
            allowedBlockLagForQosSync?: string | number | Long | undefined;
            blockLastUpdated?: string | number | Long | undefined;
            minStakeProvider?: {
                denom?: string | undefined;
                amount?: string | undefined;
            } | undefined;
            minStakeClient?: {
                denom?: string | undefined;
                amount?: string | undefined;
            } | undefined;
            providersTypes?: import("./spec").Spec_ProvidersTypes | undefined;
            imports?: string[] | undefined;
            apiCollections?: {
                enabled?: boolean | undefined;
                collectionData?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } | undefined;
                apis?: {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] | undefined;
                headers?: {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] | undefined;
                inheritanceApis?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] | undefined;
                parseDirectives?: {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] | undefined;
            }[] | undefined;
        } | undefined;
    } & {
        Spec?: ({
            index?: string | undefined;
            name?: string | undefined;
            enabled?: boolean | undefined;
            reliabilityThreshold?: number | undefined;
            dataReliabilityEnabled?: boolean | undefined;
            blockDistanceForFinalizedData?: number | undefined;
            blocksInFinalizationProof?: number | undefined;
            averageBlockTime?: string | number | Long | undefined;
            allowedBlockLagForQosSync?: string | number | Long | undefined;
            blockLastUpdated?: string | number | Long | undefined;
            minStakeProvider?: {
                denom?: string | undefined;
                amount?: string | undefined;
            } | undefined;
            minStakeClient?: {
                denom?: string | undefined;
                amount?: string | undefined;
            } | undefined;
            providersTypes?: import("./spec").Spec_ProvidersTypes | undefined;
            imports?: string[] | undefined;
            apiCollections?: {
                enabled?: boolean | undefined;
                collectionData?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } | undefined;
                apis?: {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] | undefined;
                headers?: {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] | undefined;
                inheritanceApis?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] | undefined;
                parseDirectives?: {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] | undefined;
            }[] | undefined;
        } & {
            index?: string | undefined;
            name?: string | undefined;
            enabled?: boolean | undefined;
            reliabilityThreshold?: number | undefined;
            dataReliabilityEnabled?: boolean | undefined;
            blockDistanceForFinalizedData?: number | undefined;
            blocksInFinalizationProof?: number | undefined;
            averageBlockTime?: string | number | (Long & {
                high: number;
                low: number;
                unsigned: boolean;
                add: (addend: string | number | Long) => Long;
                and: (other: string | number | Long) => Long;
                compare: (other: string | number | Long) => number;
                comp: (other: string | number | Long) => number;
                divide: (divisor: string | number | Long) => Long;
                div: (divisor: string | number | Long) => Long;
                equals: (other: string | number | Long) => boolean;
                eq: (other: string | number | Long) => boolean;
                getHighBits: () => number;
                getHighBitsUnsigned: () => number;
                getLowBits: () => number;
                getLowBitsUnsigned: () => number;
                getNumBitsAbs: () => number;
                greaterThan: (other: string | number | Long) => boolean;
                gt: (other: string | number | Long) => boolean;
                greaterThanOrEqual: (other: string | number | Long) => boolean;
                gte: (other: string | number | Long) => boolean;
                ge: (other: string | number | Long) => boolean;
                isEven: () => boolean;
                isNegative: () => boolean;
                isOdd: () => boolean;
                isPositive: () => boolean;
                isZero: () => boolean;
                eqz: () => boolean;
                lessThan: (other: string | number | Long) => boolean;
                lt: (other: string | number | Long) => boolean;
                lessThanOrEqual: (other: string | number | Long) => boolean;
                lte: (other: string | number | Long) => boolean;
                le: (other: string | number | Long) => boolean;
                modulo: (other: string | number | Long) => Long;
                mod: (other: string | number | Long) => Long;
                rem: (other: string | number | Long) => Long;
                multiply: (multiplier: string | number | Long) => Long;
                mul: (multiplier: string | number | Long) => Long;
                negate: () => Long;
                neg: () => Long;
                not: () => Long;
                countLeadingZeros: () => number;
                clz: () => number;
                countTrailingZeros: () => number;
                ctz: () => number;
                notEquals: (other: string | number | Long) => boolean;
                neq: (other: string | number | Long) => boolean;
                ne: (other: string | number | Long) => boolean;
                or: (other: string | number | Long) => Long;
                shiftLeft: (numBits: number | Long) => Long;
                shl: (numBits: number | Long) => Long;
                shiftRight: (numBits: number | Long) => Long;
                shr: (numBits: number | Long) => Long;
                shiftRightUnsigned: (numBits: number | Long) => Long;
                shru: (numBits: number | Long) => Long;
                shr_u: (numBits: number | Long) => Long;
                rotateLeft: (numBits: number | Long) => Long;
                rotl: (numBits: number | Long) => Long;
                rotateRight: (numBits: number | Long) => Long;
                rotr: (numBits: number | Long) => Long;
                subtract: (subtrahend: string | number | Long) => Long;
                sub: (subtrahend: string | number | Long) => Long;
                toInt: () => number;
                toNumber: () => number;
                toBytes: (le?: boolean | undefined) => number[];
                toBytesLE: () => number[];
                toBytesBE: () => number[];
                toSigned: () => Long;
                toString: (radix?: number | undefined) => string;
                toUnsigned: () => Long;
                xor: (other: string | number | Long) => Long;
            } & { [K_26 in Exclude<keyof I_1["Spec"]["averageBlockTime"], keyof Long>]: never; }) | undefined;
            allowedBlockLagForQosSync?: string | number | (Long & {
                high: number;
                low: number;
                unsigned: boolean;
                add: (addend: string | number | Long) => Long;
                and: (other: string | number | Long) => Long;
                compare: (other: string | number | Long) => number;
                comp: (other: string | number | Long) => number;
                divide: (divisor: string | number | Long) => Long;
                div: (divisor: string | number | Long) => Long;
                equals: (other: string | number | Long) => boolean;
                eq: (other: string | number | Long) => boolean;
                getHighBits: () => number;
                getHighBitsUnsigned: () => number;
                getLowBits: () => number;
                getLowBitsUnsigned: () => number;
                getNumBitsAbs: () => number;
                greaterThan: (other: string | number | Long) => boolean;
                gt: (other: string | number | Long) => boolean;
                greaterThanOrEqual: (other: string | number | Long) => boolean;
                gte: (other: string | number | Long) => boolean;
                ge: (other: string | number | Long) => boolean;
                isEven: () => boolean;
                isNegative: () => boolean;
                isOdd: () => boolean;
                isPositive: () => boolean;
                isZero: () => boolean;
                eqz: () => boolean;
                lessThan: (other: string | number | Long) => boolean;
                lt: (other: string | number | Long) => boolean;
                lessThanOrEqual: (other: string | number | Long) => boolean;
                lte: (other: string | number | Long) => boolean;
                le: (other: string | number | Long) => boolean;
                modulo: (other: string | number | Long) => Long;
                mod: (other: string | number | Long) => Long;
                rem: (other: string | number | Long) => Long;
                multiply: (multiplier: string | number | Long) => Long;
                mul: (multiplier: string | number | Long) => Long;
                negate: () => Long;
                neg: () => Long;
                not: () => Long;
                countLeadingZeros: () => number;
                clz: () => number;
                countTrailingZeros: () => number;
                ctz: () => number;
                notEquals: (other: string | number | Long) => boolean;
                neq: (other: string | number | Long) => boolean;
                ne: (other: string | number | Long) => boolean;
                or: (other: string | number | Long) => Long;
                shiftLeft: (numBits: number | Long) => Long;
                shl: (numBits: number | Long) => Long;
                shiftRight: (numBits: number | Long) => Long;
                shr: (numBits: number | Long) => Long;
                shiftRightUnsigned: (numBits: number | Long) => Long;
                shru: (numBits: number | Long) => Long;
                shr_u: (numBits: number | Long) => Long;
                rotateLeft: (numBits: number | Long) => Long;
                rotl: (numBits: number | Long) => Long;
                rotateRight: (numBits: number | Long) => Long;
                rotr: (numBits: number | Long) => Long;
                subtract: (subtrahend: string | number | Long) => Long;
                sub: (subtrahend: string | number | Long) => Long;
                toInt: () => number;
                toNumber: () => number;
                toBytes: (le?: boolean | undefined) => number[];
                toBytesLE: () => number[];
                toBytesBE: () => number[];
                toSigned: () => Long;
                toString: (radix?: number | undefined) => string;
                toUnsigned: () => Long;
                xor: (other: string | number | Long) => Long;
            } & { [K_27 in Exclude<keyof I_1["Spec"]["allowedBlockLagForQosSync"], keyof Long>]: never; }) | undefined;
            blockLastUpdated?: string | number | (Long & {
                high: number;
                low: number;
                unsigned: boolean;
                add: (addend: string | number | Long) => Long;
                and: (other: string | number | Long) => Long;
                compare: (other: string | number | Long) => number;
                comp: (other: string | number | Long) => number;
                divide: (divisor: string | number | Long) => Long;
                div: (divisor: string | number | Long) => Long;
                equals: (other: string | number | Long) => boolean;
                eq: (other: string | number | Long) => boolean;
                getHighBits: () => number;
                getHighBitsUnsigned: () => number;
                getLowBits: () => number;
                getLowBitsUnsigned: () => number;
                getNumBitsAbs: () => number;
                greaterThan: (other: string | number | Long) => boolean;
                gt: (other: string | number | Long) => boolean;
                greaterThanOrEqual: (other: string | number | Long) => boolean;
                gte: (other: string | number | Long) => boolean;
                ge: (other: string | number | Long) => boolean;
                isEven: () => boolean;
                isNegative: () => boolean;
                isOdd: () => boolean;
                isPositive: () => boolean;
                isZero: () => boolean;
                eqz: () => boolean;
                lessThan: (other: string | number | Long) => boolean;
                lt: (other: string | number | Long) => boolean;
                lessThanOrEqual: (other: string | number | Long) => boolean;
                lte: (other: string | number | Long) => boolean;
                le: (other: string | number | Long) => boolean;
                modulo: (other: string | number | Long) => Long;
                mod: (other: string | number | Long) => Long;
                rem: (other: string | number | Long) => Long;
                multiply: (multiplier: string | number | Long) => Long;
                mul: (multiplier: string | number | Long) => Long;
                negate: () => Long;
                neg: () => Long;
                not: () => Long;
                countLeadingZeros: () => number;
                clz: () => number;
                countTrailingZeros: () => number;
                ctz: () => number;
                notEquals: (other: string | number | Long) => boolean;
                neq: (other: string | number | Long) => boolean;
                ne: (other: string | number | Long) => boolean;
                or: (other: string | number | Long) => Long;
                shiftLeft: (numBits: number | Long) => Long;
                shl: (numBits: number | Long) => Long;
                shiftRight: (numBits: number | Long) => Long;
                shr: (numBits: number | Long) => Long;
                shiftRightUnsigned: (numBits: number | Long) => Long;
                shru: (numBits: number | Long) => Long;
                shr_u: (numBits: number | Long) => Long;
                rotateLeft: (numBits: number | Long) => Long;
                rotl: (numBits: number | Long) => Long;
                rotateRight: (numBits: number | Long) => Long;
                rotr: (numBits: number | Long) => Long;
                subtract: (subtrahend: string | number | Long) => Long;
                sub: (subtrahend: string | number | Long) => Long;
                toInt: () => number;
                toNumber: () => number;
                toBytes: (le?: boolean | undefined) => number[];
                toBytesLE: () => number[];
                toBytesBE: () => number[];
                toSigned: () => Long;
                toString: (radix?: number | undefined) => string;
                toUnsigned: () => Long;
                xor: (other: string | number | Long) => Long;
            } & { [K_28 in Exclude<keyof I_1["Spec"]["blockLastUpdated"], keyof Long>]: never; }) | undefined;
            minStakeProvider?: ({
                denom?: string | undefined;
                amount?: string | undefined;
            } & {
                denom?: string | undefined;
                amount?: string | undefined;
            } & { [K_29 in Exclude<keyof I_1["Spec"]["minStakeProvider"], keyof import("../cosmos/base/v1beta1/coin").Coin>]: never; }) | undefined;
            minStakeClient?: ({
                denom?: string | undefined;
                amount?: string | undefined;
            } & {
                denom?: string | undefined;
                amount?: string | undefined;
            } & { [K_30 in Exclude<keyof I_1["Spec"]["minStakeClient"], keyof import("../cosmos/base/v1beta1/coin").Coin>]: never; }) | undefined;
            providersTypes?: import("./spec").Spec_ProvidersTypes | undefined;
            imports?: (string[] & string[] & { [K_31 in Exclude<keyof I_1["Spec"]["imports"], keyof string[]>]: never; }) | undefined;
            apiCollections?: ({
                enabled?: boolean | undefined;
                collectionData?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } | undefined;
                apis?: {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] | undefined;
                headers?: {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] | undefined;
                inheritanceApis?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] | undefined;
                parseDirectives?: {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] | undefined;
            }[] & ({
                enabled?: boolean | undefined;
                collectionData?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } | undefined;
                apis?: {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] | undefined;
                headers?: {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] | undefined;
                inheritanceApis?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] | undefined;
                parseDirectives?: {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] | undefined;
            } & {
                enabled?: boolean | undefined;
                collectionData?: ({
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } & {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } & { [K_32 in Exclude<keyof I_1["Spec"]["apiCollections"][number]["collectionData"], keyof import("./api_collection").CollectionData>]: never; }) | undefined;
                apis?: ({
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] & ({
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                } & {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | (Long & {
                        high: number;
                        low: number;
                        unsigned: boolean;
                        add: (addend: string | number | Long) => Long;
                        and: (other: string | number | Long) => Long;
                        compare: (other: string | number | Long) => number;
                        comp: (other: string | number | Long) => number;
                        divide: (divisor: string | number | Long) => Long;
                        div: (divisor: string | number | Long) => Long;
                        equals: (other: string | number | Long) => boolean;
                        eq: (other: string | number | Long) => boolean;
                        getHighBits: () => number;
                        getHighBitsUnsigned: () => number;
                        getLowBits: () => number;
                        getLowBitsUnsigned: () => number;
                        getNumBitsAbs: () => number;
                        greaterThan: (other: string | number | Long) => boolean;
                        gt: (other: string | number | Long) => boolean;
                        greaterThanOrEqual: (other: string | number | Long) => boolean;
                        gte: (other: string | number | Long) => boolean;
                        ge: (other: string | number | Long) => boolean;
                        isEven: () => boolean;
                        isNegative: () => boolean;
                        isOdd: () => boolean;
                        isPositive: () => boolean;
                        isZero: () => boolean;
                        eqz: () => boolean;
                        lessThan: (other: string | number | Long) => boolean;
                        lt: (other: string | number | Long) => boolean;
                        lessThanOrEqual: (other: string | number | Long) => boolean;
                        lte: (other: string | number | Long) => boolean;
                        le: (other: string | number | Long) => boolean;
                        modulo: (other: string | number | Long) => Long;
                        mod: (other: string | number | Long) => Long;
                        rem: (other: string | number | Long) => Long;
                        multiply: (multiplier: string | number | Long) => Long;
                        mul: (multiplier: string | number | Long) => Long;
                        negate: () => Long;
                        neg: () => Long;
                        not: () => Long;
                        countLeadingZeros: () => number;
                        clz: () => number;
                        countTrailingZeros: () => number;
                        ctz: () => number;
                        notEquals: (other: string | number | Long) => boolean;
                        neq: (other: string | number | Long) => boolean;
                        ne: (other: string | number | Long) => boolean;
                        or: (other: string | number | Long) => Long;
                        shiftLeft: (numBits: number | Long) => Long;
                        shl: (numBits: number | Long) => Long;
                        shiftRight: (numBits: number | Long) => Long;
                        shr: (numBits: number | Long) => Long;
                        shiftRightUnsigned: (numBits: number | Long) => Long;
                        shru: (numBits: number | Long) => Long;
                        shr_u: (numBits: number | Long) => Long;
                        rotateLeft: (numBits: number | Long) => Long;
                        rotl: (numBits: number | Long) => Long;
                        rotateRight: (numBits: number | Long) => Long;
                        rotr: (numBits: number | Long) => Long;
                        subtract: (subtrahend: string | number | Long) => Long;
                        sub: (subtrahend: string | number | Long) => Long;
                        toInt: () => number;
                        toNumber: () => number;
                        toBytes: (le?: boolean | undefined) => number[];
                        toBytesLE: () => number[];
                        toBytesBE: () => number[];
                        toSigned: () => Long;
                        toString: (radix?: number | undefined) => string;
                        toUnsigned: () => Long;
                        xor: (other: string | number | Long) => Long;
                    } & { [K_33 in Exclude<keyof I_1["Spec"]["apiCollections"][number]["apis"][number]["computeUnits"], keyof Long>]: never; }) | undefined;
                    extraComputeUnits?: string | number | (Long & {
                        high: number;
                        low: number;
                        unsigned: boolean;
                        add: (addend: string | number | Long) => Long;
                        and: (other: string | number | Long) => Long;
                        compare: (other: string | number | Long) => number;
                        comp: (other: string | number | Long) => number;
                        divide: (divisor: string | number | Long) => Long;
                        div: (divisor: string | number | Long) => Long;
                        equals: (other: string | number | Long) => boolean;
                        eq: (other: string | number | Long) => boolean;
                        getHighBits: () => number;
                        getHighBitsUnsigned: () => number;
                        getLowBits: () => number;
                        getLowBitsUnsigned: () => number;
                        getNumBitsAbs: () => number;
                        greaterThan: (other: string | number | Long) => boolean;
                        gt: (other: string | number | Long) => boolean;
                        greaterThanOrEqual: (other: string | number | Long) => boolean;
                        gte: (other: string | number | Long) => boolean;
                        ge: (other: string | number | Long) => boolean;
                        isEven: () => boolean;
                        isNegative: () => boolean;
                        isOdd: () => boolean;
                        isPositive: () => boolean;
                        isZero: () => boolean;
                        eqz: () => boolean;
                        lessThan: (other: string | number | Long) => boolean;
                        lt: (other: string | number | Long) => boolean;
                        lessThanOrEqual: (other: string | number | Long) => boolean;
                        lte: (other: string | number | Long) => boolean;
                        le: (other: string | number | Long) => boolean;
                        modulo: (other: string | number | Long) => Long;
                        mod: (other: string | number | Long) => Long;
                        rem: (other: string | number | Long) => Long;
                        multiply: (multiplier: string | number | Long) => Long;
                        mul: (multiplier: string | number | Long) => Long;
                        negate: () => Long;
                        neg: () => Long;
                        not: () => Long;
                        countLeadingZeros: () => number;
                        clz: () => number;
                        countTrailingZeros: () => number;
                        ctz: () => number;
                        notEquals: (other: string | number | Long) => boolean;
                        neq: (other: string | number | Long) => boolean;
                        ne: (other: string | number | Long) => boolean;
                        or: (other: string | number | Long) => Long;
                        shiftLeft: (numBits: number | Long) => Long;
                        shl: (numBits: number | Long) => Long;
                        shiftRight: (numBits: number | Long) => Long;
                        shr: (numBits: number | Long) => Long;
                        shiftRightUnsigned: (numBits: number | Long) => Long;
                        shru: (numBits: number | Long) => Long;
                        shr_u: (numBits: number | Long) => Long;
                        rotateLeft: (numBits: number | Long) => Long;
                        rotl: (numBits: number | Long) => Long;
                        rotateRight: (numBits: number | Long) => Long;
                        rotr: (numBits: number | Long) => Long;
                        subtract: (subtrahend: string | number | Long) => Long;
                        sub: (subtrahend: string | number | Long) => Long;
                        toInt: () => number;
                        toNumber: () => number;
                        toBytes: (le?: boolean | undefined) => number[];
                        toBytesLE: () => number[];
                        toBytesBE: () => number[];
                        toSigned: () => Long;
                        toString: (radix?: number | undefined) => string;
                        toUnsigned: () => Long;
                        xor: (other: string | number | Long) => Long;
                    } & { [K_34 in Exclude<keyof I_1["Spec"]["apiCollections"][number]["apis"][number]["extraComputeUnits"], keyof Long>]: never; }) | undefined;
                    category?: ({
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } & {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } & { [K_35 in Exclude<keyof I_1["Spec"]["apiCollections"][number]["apis"][number]["category"], keyof import("./api_collection").SpecCategory>]: never; }) | undefined;
                    blockParsing?: ({
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } & {
                        parserArg?: (string[] & string[] & { [K_36 in Exclude<keyof I_1["Spec"]["apiCollections"][number]["apis"][number]["blockParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } & { [K_37 in Exclude<keyof I_1["Spec"]["apiCollections"][number]["apis"][number]["blockParsing"], keyof import("./api_collection").BlockParser>]: never; }) | undefined;
                } & { [K_38 in Exclude<keyof I_1["Spec"]["apiCollections"][number]["apis"][number], keyof import("./api_collection").Api>]: never; })[] & { [K_39 in Exclude<keyof I_1["Spec"]["apiCollections"][number]["apis"], keyof {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[]>]: never; }) | undefined;
                headers?: ({
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] & ({
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                } & {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                } & { [K_40 in Exclude<keyof I_1["Spec"]["apiCollections"][number]["headers"][number], keyof import("./api_collection").Header>]: never; })[] & { [K_41 in Exclude<keyof I_1["Spec"]["apiCollections"][number]["headers"], keyof {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[]>]: never; }) | undefined;
                inheritanceApis?: ({
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] & ({
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } & {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } & { [K_42 in Exclude<keyof I_1["Spec"]["apiCollections"][number]["inheritanceApis"][number], keyof import("./api_collection").CollectionData>]: never; })[] & { [K_43 in Exclude<keyof I_1["Spec"]["apiCollections"][number]["inheritanceApis"], keyof {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[]>]: never; }) | undefined;
                parseDirectives?: ({
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] & ({
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                } & {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: ({
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } & {
                        parserArg?: (string[] & string[] & { [K_44 in Exclude<keyof I_1["Spec"]["apiCollections"][number]["parseDirectives"][number]["resultParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } & { [K_45 in Exclude<keyof I_1["Spec"]["apiCollections"][number]["parseDirectives"][number]["resultParsing"], keyof import("./api_collection").BlockParser>]: never; }) | undefined;
                    apiName?: string | undefined;
                } & { [K_46 in Exclude<keyof I_1["Spec"]["apiCollections"][number]["parseDirectives"][number], keyof import("./api_collection").ParseDirective>]: never; })[] & { [K_47 in Exclude<keyof I_1["Spec"]["apiCollections"][number]["parseDirectives"], keyof {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[]>]: never; }) | undefined;
            } & { [K_48 in Exclude<keyof I_1["Spec"]["apiCollections"][number], keyof import("./api_collection").ApiCollection>]: never; })[] & { [K_49 in Exclude<keyof I_1["Spec"]["apiCollections"], keyof {
                enabled?: boolean | undefined;
                collectionData?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } | undefined;
                apis?: {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] | undefined;
                headers?: {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] | undefined;
                inheritanceApis?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] | undefined;
                parseDirectives?: {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] | undefined;
            }[]>]: never; }) | undefined;
        } & { [K_50 in Exclude<keyof I_1["Spec"], keyof Spec>]: never; }) | undefined;
    } & { [K_51 in Exclude<keyof I_1, "Spec">]: never; }>(object: I_1): QueryGetSpecResponse;
};
export declare const QueryAllSpecRequest: {
    encode(message: QueryAllSpecRequest, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllSpecRequest;
    fromJSON(object: any): QueryAllSpecRequest;
    toJSON(message: QueryAllSpecRequest): unknown;
    create<I extends {
        pagination?: {
            key?: Uint8Array | undefined;
            offset?: string | number | Long | undefined;
            limit?: string | number | Long | undefined;
            countTotal?: boolean | undefined;
            reverse?: boolean | undefined;
        } | undefined;
    } & {
        pagination?: ({
            key?: Uint8Array | undefined;
            offset?: string | number | Long | undefined;
            limit?: string | number | Long | undefined;
            countTotal?: boolean | undefined;
            reverse?: boolean | undefined;
        } & {
            key?: Uint8Array | undefined;
            offset?: string | number | (Long & {
                high: number;
                low: number;
                unsigned: boolean;
                add: (addend: string | number | Long) => Long;
                and: (other: string | number | Long) => Long;
                compare: (other: string | number | Long) => number;
                comp: (other: string | number | Long) => number;
                divide: (divisor: string | number | Long) => Long;
                div: (divisor: string | number | Long) => Long;
                equals: (other: string | number | Long) => boolean;
                eq: (other: string | number | Long) => boolean;
                getHighBits: () => number;
                getHighBitsUnsigned: () => number;
                getLowBits: () => number;
                getLowBitsUnsigned: () => number;
                getNumBitsAbs: () => number;
                greaterThan: (other: string | number | Long) => boolean;
                gt: (other: string | number | Long) => boolean;
                greaterThanOrEqual: (other: string | number | Long) => boolean;
                gte: (other: string | number | Long) => boolean;
                ge: (other: string | number | Long) => boolean;
                isEven: () => boolean;
                isNegative: () => boolean;
                isOdd: () => boolean;
                isPositive: () => boolean;
                isZero: () => boolean;
                eqz: () => boolean;
                lessThan: (other: string | number | Long) => boolean;
                lt: (other: string | number | Long) => boolean;
                lessThanOrEqual: (other: string | number | Long) => boolean;
                lte: (other: string | number | Long) => boolean;
                le: (other: string | number | Long) => boolean;
                modulo: (other: string | number | Long) => Long;
                mod: (other: string | number | Long) => Long;
                rem: (other: string | number | Long) => Long;
                multiply: (multiplier: string | number | Long) => Long;
                mul: (multiplier: string | number | Long) => Long;
                negate: () => Long;
                neg: () => Long;
                not: () => Long;
                countLeadingZeros: () => number;
                clz: () => number;
                countTrailingZeros: () => number;
                ctz: () => number;
                notEquals: (other: string | number | Long) => boolean;
                neq: (other: string | number | Long) => boolean;
                ne: (other: string | number | Long) => boolean;
                or: (other: string | number | Long) => Long;
                shiftLeft: (numBits: number | Long) => Long;
                shl: (numBits: number | Long) => Long;
                shiftRight: (numBits: number | Long) => Long;
                shr: (numBits: number | Long) => Long;
                shiftRightUnsigned: (numBits: number | Long) => Long;
                shru: (numBits: number | Long) => Long;
                shr_u: (numBits: number | Long) => Long;
                rotateLeft: (numBits: number | Long) => Long;
                rotl: (numBits: number | Long) => Long;
                rotateRight: (numBits: number | Long) => Long;
                rotr: (numBits: number | Long) => Long;
                subtract: (subtrahend: string | number | Long) => Long;
                sub: (subtrahend: string | number | Long) => Long;
                toInt: () => number;
                toNumber: () => number;
                toBytes: (le?: boolean | undefined) => number[];
                toBytesLE: () => number[];
                toBytesBE: () => number[];
                toSigned: () => Long;
                toString: (radix?: number | undefined) => string;
                toUnsigned: () => Long;
                xor: (other: string | number | Long) => Long;
            } & { [K in Exclude<keyof I["pagination"]["offset"], keyof Long>]: never; }) | undefined;
            limit?: string | number | (Long & {
                high: number;
                low: number;
                unsigned: boolean;
                add: (addend: string | number | Long) => Long;
                and: (other: string | number | Long) => Long;
                compare: (other: string | number | Long) => number;
                comp: (other: string | number | Long) => number;
                divide: (divisor: string | number | Long) => Long;
                div: (divisor: string | number | Long) => Long;
                equals: (other: string | number | Long) => boolean;
                eq: (other: string | number | Long) => boolean;
                getHighBits: () => number;
                getHighBitsUnsigned: () => number;
                getLowBits: () => number;
                getLowBitsUnsigned: () => number;
                getNumBitsAbs: () => number;
                greaterThan: (other: string | number | Long) => boolean;
                gt: (other: string | number | Long) => boolean;
                greaterThanOrEqual: (other: string | number | Long) => boolean;
                gte: (other: string | number | Long) => boolean;
                ge: (other: string | number | Long) => boolean;
                isEven: () => boolean;
                isNegative: () => boolean;
                isOdd: () => boolean;
                isPositive: () => boolean;
                isZero: () => boolean;
                eqz: () => boolean;
                lessThan: (other: string | number | Long) => boolean;
                lt: (other: string | number | Long) => boolean;
                lessThanOrEqual: (other: string | number | Long) => boolean;
                lte: (other: string | number | Long) => boolean;
                le: (other: string | number | Long) => boolean;
                modulo: (other: string | number | Long) => Long;
                mod: (other: string | number | Long) => Long;
                rem: (other: string | number | Long) => Long;
                multiply: (multiplier: string | number | Long) => Long;
                mul: (multiplier: string | number | Long) => Long;
                negate: () => Long;
                neg: () => Long;
                not: () => Long;
                countLeadingZeros: () => number;
                clz: () => number;
                countTrailingZeros: () => number;
                ctz: () => number;
                notEquals: (other: string | number | Long) => boolean;
                neq: (other: string | number | Long) => boolean;
                ne: (other: string | number | Long) => boolean;
                or: (other: string | number | Long) => Long;
                shiftLeft: (numBits: number | Long) => Long;
                shl: (numBits: number | Long) => Long;
                shiftRight: (numBits: number | Long) => Long;
                shr: (numBits: number | Long) => Long;
                shiftRightUnsigned: (numBits: number | Long) => Long;
                shru: (numBits: number | Long) => Long;
                shr_u: (numBits: number | Long) => Long;
                rotateLeft: (numBits: number | Long) => Long;
                rotl: (numBits: number | Long) => Long;
                rotateRight: (numBits: number | Long) => Long;
                rotr: (numBits: number | Long) => Long;
                subtract: (subtrahend: string | number | Long) => Long;
                sub: (subtrahend: string | number | Long) => Long;
                toInt: () => number;
                toNumber: () => number;
                toBytes: (le?: boolean | undefined) => number[];
                toBytesLE: () => number[];
                toBytesBE: () => number[];
                toSigned: () => Long;
                toString: (radix?: number | undefined) => string;
                toUnsigned: () => Long;
                xor: (other: string | number | Long) => Long;
            } & { [K_1 in Exclude<keyof I["pagination"]["limit"], keyof Long>]: never; }) | undefined;
            countTotal?: boolean | undefined;
            reverse?: boolean | undefined;
        } & { [K_2 in Exclude<keyof I["pagination"], keyof PageRequest>]: never; }) | undefined;
    } & { [K_3 in Exclude<keyof I, "pagination">]: never; }>(base?: I | undefined): QueryAllSpecRequest;
    fromPartial<I_1 extends {
        pagination?: {
            key?: Uint8Array | undefined;
            offset?: string | number | Long | undefined;
            limit?: string | number | Long | undefined;
            countTotal?: boolean | undefined;
            reverse?: boolean | undefined;
        } | undefined;
    } & {
        pagination?: ({
            key?: Uint8Array | undefined;
            offset?: string | number | Long | undefined;
            limit?: string | number | Long | undefined;
            countTotal?: boolean | undefined;
            reverse?: boolean | undefined;
        } & {
            key?: Uint8Array | undefined;
            offset?: string | number | (Long & {
                high: number;
                low: number;
                unsigned: boolean;
                add: (addend: string | number | Long) => Long;
                and: (other: string | number | Long) => Long;
                compare: (other: string | number | Long) => number;
                comp: (other: string | number | Long) => number;
                divide: (divisor: string | number | Long) => Long;
                div: (divisor: string | number | Long) => Long;
                equals: (other: string | number | Long) => boolean;
                eq: (other: string | number | Long) => boolean;
                getHighBits: () => number;
                getHighBitsUnsigned: () => number;
                getLowBits: () => number;
                getLowBitsUnsigned: () => number;
                getNumBitsAbs: () => number;
                greaterThan: (other: string | number | Long) => boolean;
                gt: (other: string | number | Long) => boolean;
                greaterThanOrEqual: (other: string | number | Long) => boolean;
                gte: (other: string | number | Long) => boolean;
                ge: (other: string | number | Long) => boolean;
                isEven: () => boolean;
                isNegative: () => boolean;
                isOdd: () => boolean;
                isPositive: () => boolean;
                isZero: () => boolean;
                eqz: () => boolean;
                lessThan: (other: string | number | Long) => boolean;
                lt: (other: string | number | Long) => boolean;
                lessThanOrEqual: (other: string | number | Long) => boolean;
                lte: (other: string | number | Long) => boolean;
                le: (other: string | number | Long) => boolean;
                modulo: (other: string | number | Long) => Long;
                mod: (other: string | number | Long) => Long;
                rem: (other: string | number | Long) => Long;
                multiply: (multiplier: string | number | Long) => Long;
                mul: (multiplier: string | number | Long) => Long;
                negate: () => Long;
                neg: () => Long;
                not: () => Long;
                countLeadingZeros: () => number;
                clz: () => number;
                countTrailingZeros: () => number;
                ctz: () => number;
                notEquals: (other: string | number | Long) => boolean;
                neq: (other: string | number | Long) => boolean;
                ne: (other: string | number | Long) => boolean;
                or: (other: string | number | Long) => Long;
                shiftLeft: (numBits: number | Long) => Long;
                shl: (numBits: number | Long) => Long;
                shiftRight: (numBits: number | Long) => Long;
                shr: (numBits: number | Long) => Long;
                shiftRightUnsigned: (numBits: number | Long) => Long;
                shru: (numBits: number | Long) => Long;
                shr_u: (numBits: number | Long) => Long;
                rotateLeft: (numBits: number | Long) => Long;
                rotl: (numBits: number | Long) => Long;
                rotateRight: (numBits: number | Long) => Long;
                rotr: (numBits: number | Long) => Long;
                subtract: (subtrahend: string | number | Long) => Long;
                sub: (subtrahend: string | number | Long) => Long;
                toInt: () => number;
                toNumber: () => number;
                toBytes: (le?: boolean | undefined) => number[];
                toBytesLE: () => number[];
                toBytesBE: () => number[];
                toSigned: () => Long;
                toString: (radix?: number | undefined) => string;
                toUnsigned: () => Long;
                xor: (other: string | number | Long) => Long;
            } & { [K_4 in Exclude<keyof I_1["pagination"]["offset"], keyof Long>]: never; }) | undefined;
            limit?: string | number | (Long & {
                high: number;
                low: number;
                unsigned: boolean;
                add: (addend: string | number | Long) => Long;
                and: (other: string | number | Long) => Long;
                compare: (other: string | number | Long) => number;
                comp: (other: string | number | Long) => number;
                divide: (divisor: string | number | Long) => Long;
                div: (divisor: string | number | Long) => Long;
                equals: (other: string | number | Long) => boolean;
                eq: (other: string | number | Long) => boolean;
                getHighBits: () => number;
                getHighBitsUnsigned: () => number;
                getLowBits: () => number;
                getLowBitsUnsigned: () => number;
                getNumBitsAbs: () => number;
                greaterThan: (other: string | number | Long) => boolean;
                gt: (other: string | number | Long) => boolean;
                greaterThanOrEqual: (other: string | number | Long) => boolean;
                gte: (other: string | number | Long) => boolean;
                ge: (other: string | number | Long) => boolean;
                isEven: () => boolean;
                isNegative: () => boolean;
                isOdd: () => boolean;
                isPositive: () => boolean;
                isZero: () => boolean;
                eqz: () => boolean;
                lessThan: (other: string | number | Long) => boolean;
                lt: (other: string | number | Long) => boolean;
                lessThanOrEqual: (other: string | number | Long) => boolean;
                lte: (other: string | number | Long) => boolean;
                le: (other: string | number | Long) => boolean;
                modulo: (other: string | number | Long) => Long;
                mod: (other: string | number | Long) => Long;
                rem: (other: string | number | Long) => Long;
                multiply: (multiplier: string | number | Long) => Long;
                mul: (multiplier: string | number | Long) => Long;
                negate: () => Long;
                neg: () => Long;
                not: () => Long;
                countLeadingZeros: () => number;
                clz: () => number;
                countTrailingZeros: () => number;
                ctz: () => number;
                notEquals: (other: string | number | Long) => boolean;
                neq: (other: string | number | Long) => boolean;
                ne: (other: string | number | Long) => boolean;
                or: (other: string | number | Long) => Long;
                shiftLeft: (numBits: number | Long) => Long;
                shl: (numBits: number | Long) => Long;
                shiftRight: (numBits: number | Long) => Long;
                shr: (numBits: number | Long) => Long;
                shiftRightUnsigned: (numBits: number | Long) => Long;
                shru: (numBits: number | Long) => Long;
                shr_u: (numBits: number | Long) => Long;
                rotateLeft: (numBits: number | Long) => Long;
                rotl: (numBits: number | Long) => Long;
                rotateRight: (numBits: number | Long) => Long;
                rotr: (numBits: number | Long) => Long;
                subtract: (subtrahend: string | number | Long) => Long;
                sub: (subtrahend: string | number | Long) => Long;
                toInt: () => number;
                toNumber: () => number;
                toBytes: (le?: boolean | undefined) => number[];
                toBytesLE: () => number[];
                toBytesBE: () => number[];
                toSigned: () => Long;
                toString: (radix?: number | undefined) => string;
                toUnsigned: () => Long;
                xor: (other: string | number | Long) => Long;
            } & { [K_5 in Exclude<keyof I_1["pagination"]["limit"], keyof Long>]: never; }) | undefined;
            countTotal?: boolean | undefined;
            reverse?: boolean | undefined;
        } & { [K_6 in Exclude<keyof I_1["pagination"], keyof PageRequest>]: never; }) | undefined;
    } & { [K_7 in Exclude<keyof I_1, "pagination">]: never; }>(object: I_1): QueryAllSpecRequest;
};
export declare const QueryAllSpecResponse: {
    encode(message: QueryAllSpecResponse, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllSpecResponse;
    fromJSON(object: any): QueryAllSpecResponse;
    toJSON(message: QueryAllSpecResponse): unknown;
    create<I extends {
        Spec?: {
            index?: string | undefined;
            name?: string | undefined;
            enabled?: boolean | undefined;
            reliabilityThreshold?: number | undefined;
            dataReliabilityEnabled?: boolean | undefined;
            blockDistanceForFinalizedData?: number | undefined;
            blocksInFinalizationProof?: number | undefined;
            averageBlockTime?: string | number | Long | undefined;
            allowedBlockLagForQosSync?: string | number | Long | undefined;
            blockLastUpdated?: string | number | Long | undefined;
            minStakeProvider?: {
                denom?: string | undefined;
                amount?: string | undefined;
            } | undefined;
            minStakeClient?: {
                denom?: string | undefined;
                amount?: string | undefined;
            } | undefined;
            providersTypes?: import("./spec").Spec_ProvidersTypes | undefined;
            imports?: string[] | undefined;
            apiCollections?: {
                enabled?: boolean | undefined;
                collectionData?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } | undefined;
                apis?: {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] | undefined;
                headers?: {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] | undefined;
                inheritanceApis?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] | undefined;
                parseDirectives?: {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] | undefined;
            }[] | undefined;
        }[] | undefined;
        pagination?: {
            nextKey?: Uint8Array | undefined;
            total?: string | number | Long | undefined;
        } | undefined;
    } & {
        Spec?: ({
            index?: string | undefined;
            name?: string | undefined;
            enabled?: boolean | undefined;
            reliabilityThreshold?: number | undefined;
            dataReliabilityEnabled?: boolean | undefined;
            blockDistanceForFinalizedData?: number | undefined;
            blocksInFinalizationProof?: number | undefined;
            averageBlockTime?: string | number | Long | undefined;
            allowedBlockLagForQosSync?: string | number | Long | undefined;
            blockLastUpdated?: string | number | Long | undefined;
            minStakeProvider?: {
                denom?: string | undefined;
                amount?: string | undefined;
            } | undefined;
            minStakeClient?: {
                denom?: string | undefined;
                amount?: string | undefined;
            } | undefined;
            providersTypes?: import("./spec").Spec_ProvidersTypes | undefined;
            imports?: string[] | undefined;
            apiCollections?: {
                enabled?: boolean | undefined;
                collectionData?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } | undefined;
                apis?: {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] | undefined;
                headers?: {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] | undefined;
                inheritanceApis?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] | undefined;
                parseDirectives?: {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] | undefined;
            }[] | undefined;
        }[] & ({
            index?: string | undefined;
            name?: string | undefined;
            enabled?: boolean | undefined;
            reliabilityThreshold?: number | undefined;
            dataReliabilityEnabled?: boolean | undefined;
            blockDistanceForFinalizedData?: number | undefined;
            blocksInFinalizationProof?: number | undefined;
            averageBlockTime?: string | number | Long | undefined;
            allowedBlockLagForQosSync?: string | number | Long | undefined;
            blockLastUpdated?: string | number | Long | undefined;
            minStakeProvider?: {
                denom?: string | undefined;
                amount?: string | undefined;
            } | undefined;
            minStakeClient?: {
                denom?: string | undefined;
                amount?: string | undefined;
            } | undefined;
            providersTypes?: import("./spec").Spec_ProvidersTypes | undefined;
            imports?: string[] | undefined;
            apiCollections?: {
                enabled?: boolean | undefined;
                collectionData?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } | undefined;
                apis?: {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] | undefined;
                headers?: {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] | undefined;
                inheritanceApis?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] | undefined;
                parseDirectives?: {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] | undefined;
            }[] | undefined;
        } & {
            index?: string | undefined;
            name?: string | undefined;
            enabled?: boolean | undefined;
            reliabilityThreshold?: number | undefined;
            dataReliabilityEnabled?: boolean | undefined;
            blockDistanceForFinalizedData?: number | undefined;
            blocksInFinalizationProof?: number | undefined;
            averageBlockTime?: string | number | (Long & {
                high: number;
                low: number;
                unsigned: boolean;
                add: (addend: string | number | Long) => Long;
                and: (other: string | number | Long) => Long;
                compare: (other: string | number | Long) => number;
                comp: (other: string | number | Long) => number;
                divide: (divisor: string | number | Long) => Long;
                div: (divisor: string | number | Long) => Long;
                equals: (other: string | number | Long) => boolean;
                eq: (other: string | number | Long) => boolean;
                getHighBits: () => number;
                getHighBitsUnsigned: () => number;
                getLowBits: () => number;
                getLowBitsUnsigned: () => number;
                getNumBitsAbs: () => number;
                greaterThan: (other: string | number | Long) => boolean;
                gt: (other: string | number | Long) => boolean;
                greaterThanOrEqual: (other: string | number | Long) => boolean;
                gte: (other: string | number | Long) => boolean;
                ge: (other: string | number | Long) => boolean;
                isEven: () => boolean;
                isNegative: () => boolean;
                isOdd: () => boolean;
                isPositive: () => boolean;
                isZero: () => boolean;
                eqz: () => boolean;
                lessThan: (other: string | number | Long) => boolean;
                lt: (other: string | number | Long) => boolean;
                lessThanOrEqual: (other: string | number | Long) => boolean;
                lte: (other: string | number | Long) => boolean;
                le: (other: string | number | Long) => boolean;
                modulo: (other: string | number | Long) => Long;
                mod: (other: string | number | Long) => Long;
                rem: (other: string | number | Long) => Long;
                multiply: (multiplier: string | number | Long) => Long;
                mul: (multiplier: string | number | Long) => Long;
                negate: () => Long;
                neg: () => Long;
                not: () => Long;
                countLeadingZeros: () => number;
                clz: () => number;
                countTrailingZeros: () => number;
                ctz: () => number;
                notEquals: (other: string | number | Long) => boolean;
                neq: (other: string | number | Long) => boolean;
                ne: (other: string | number | Long) => boolean;
                or: (other: string | number | Long) => Long;
                shiftLeft: (numBits: number | Long) => Long;
                shl: (numBits: number | Long) => Long;
                shiftRight: (numBits: number | Long) => Long;
                shr: (numBits: number | Long) => Long;
                shiftRightUnsigned: (numBits: number | Long) => Long;
                shru: (numBits: number | Long) => Long;
                shr_u: (numBits: number | Long) => Long;
                rotateLeft: (numBits: number | Long) => Long;
                rotl: (numBits: number | Long) => Long;
                rotateRight: (numBits: number | Long) => Long;
                rotr: (numBits: number | Long) => Long;
                subtract: (subtrahend: string | number | Long) => Long;
                sub: (subtrahend: string | number | Long) => Long;
                toInt: () => number;
                toNumber: () => number;
                toBytes: (le?: boolean | undefined) => number[];
                toBytesLE: () => number[];
                toBytesBE: () => number[];
                toSigned: () => Long;
                toString: (radix?: number | undefined) => string;
                toUnsigned: () => Long;
                xor: (other: string | number | Long) => Long;
            } & { [K in Exclude<keyof I["Spec"][number]["averageBlockTime"], keyof Long>]: never; }) | undefined;
            allowedBlockLagForQosSync?: string | number | (Long & {
                high: number;
                low: number;
                unsigned: boolean;
                add: (addend: string | number | Long) => Long;
                and: (other: string | number | Long) => Long;
                compare: (other: string | number | Long) => number;
                comp: (other: string | number | Long) => number;
                divide: (divisor: string | number | Long) => Long;
                div: (divisor: string | number | Long) => Long;
                equals: (other: string | number | Long) => boolean;
                eq: (other: string | number | Long) => boolean;
                getHighBits: () => number;
                getHighBitsUnsigned: () => number;
                getLowBits: () => number;
                getLowBitsUnsigned: () => number;
                getNumBitsAbs: () => number;
                greaterThan: (other: string | number | Long) => boolean;
                gt: (other: string | number | Long) => boolean;
                greaterThanOrEqual: (other: string | number | Long) => boolean;
                gte: (other: string | number | Long) => boolean;
                ge: (other: string | number | Long) => boolean;
                isEven: () => boolean;
                isNegative: () => boolean;
                isOdd: () => boolean;
                isPositive: () => boolean;
                isZero: () => boolean;
                eqz: () => boolean;
                lessThan: (other: string | number | Long) => boolean;
                lt: (other: string | number | Long) => boolean;
                lessThanOrEqual: (other: string | number | Long) => boolean;
                lte: (other: string | number | Long) => boolean;
                le: (other: string | number | Long) => boolean;
                modulo: (other: string | number | Long) => Long;
                mod: (other: string | number | Long) => Long;
                rem: (other: string | number | Long) => Long;
                multiply: (multiplier: string | number | Long) => Long;
                mul: (multiplier: string | number | Long) => Long;
                negate: () => Long;
                neg: () => Long;
                not: () => Long;
                countLeadingZeros: () => number;
                clz: () => number;
                countTrailingZeros: () => number;
                ctz: () => number;
                notEquals: (other: string | number | Long) => boolean;
                neq: (other: string | number | Long) => boolean;
                ne: (other: string | number | Long) => boolean;
                or: (other: string | number | Long) => Long;
                shiftLeft: (numBits: number | Long) => Long;
                shl: (numBits: number | Long) => Long;
                shiftRight: (numBits: number | Long) => Long;
                shr: (numBits: number | Long) => Long;
                shiftRightUnsigned: (numBits: number | Long) => Long;
                shru: (numBits: number | Long) => Long;
                shr_u: (numBits: number | Long) => Long;
                rotateLeft: (numBits: number | Long) => Long;
                rotl: (numBits: number | Long) => Long;
                rotateRight: (numBits: number | Long) => Long;
                rotr: (numBits: number | Long) => Long;
                subtract: (subtrahend: string | number | Long) => Long;
                sub: (subtrahend: string | number | Long) => Long;
                toInt: () => number;
                toNumber: () => number;
                toBytes: (le?: boolean | undefined) => number[];
                toBytesLE: () => number[];
                toBytesBE: () => number[];
                toSigned: () => Long;
                toString: (radix?: number | undefined) => string;
                toUnsigned: () => Long;
                xor: (other: string | number | Long) => Long;
            } & { [K_1 in Exclude<keyof I["Spec"][number]["allowedBlockLagForQosSync"], keyof Long>]: never; }) | undefined;
            blockLastUpdated?: string | number | (Long & {
                high: number;
                low: number;
                unsigned: boolean;
                add: (addend: string | number | Long) => Long;
                and: (other: string | number | Long) => Long;
                compare: (other: string | number | Long) => number;
                comp: (other: string | number | Long) => number;
                divide: (divisor: string | number | Long) => Long;
                div: (divisor: string | number | Long) => Long;
                equals: (other: string | number | Long) => boolean;
                eq: (other: string | number | Long) => boolean;
                getHighBits: () => number;
                getHighBitsUnsigned: () => number;
                getLowBits: () => number;
                getLowBitsUnsigned: () => number;
                getNumBitsAbs: () => number;
                greaterThan: (other: string | number | Long) => boolean;
                gt: (other: string | number | Long) => boolean;
                greaterThanOrEqual: (other: string | number | Long) => boolean;
                gte: (other: string | number | Long) => boolean;
                ge: (other: string | number | Long) => boolean;
                isEven: () => boolean;
                isNegative: () => boolean;
                isOdd: () => boolean;
                isPositive: () => boolean;
                isZero: () => boolean;
                eqz: () => boolean;
                lessThan: (other: string | number | Long) => boolean;
                lt: (other: string | number | Long) => boolean;
                lessThanOrEqual: (other: string | number | Long) => boolean;
                lte: (other: string | number | Long) => boolean;
                le: (other: string | number | Long) => boolean;
                modulo: (other: string | number | Long) => Long;
                mod: (other: string | number | Long) => Long;
                rem: (other: string | number | Long) => Long;
                multiply: (multiplier: string | number | Long) => Long;
                mul: (multiplier: string | number | Long) => Long;
                negate: () => Long;
                neg: () => Long;
                not: () => Long;
                countLeadingZeros: () => number;
                clz: () => number;
                countTrailingZeros: () => number;
                ctz: () => number;
                notEquals: (other: string | number | Long) => boolean;
                neq: (other: string | number | Long) => boolean;
                ne: (other: string | number | Long) => boolean;
                or: (other: string | number | Long) => Long;
                shiftLeft: (numBits: number | Long) => Long;
                shl: (numBits: number | Long) => Long;
                shiftRight: (numBits: number | Long) => Long;
                shr: (numBits: number | Long) => Long;
                shiftRightUnsigned: (numBits: number | Long) => Long;
                shru: (numBits: number | Long) => Long;
                shr_u: (numBits: number | Long) => Long;
                rotateLeft: (numBits: number | Long) => Long;
                rotl: (numBits: number | Long) => Long;
                rotateRight: (numBits: number | Long) => Long;
                rotr: (numBits: number | Long) => Long;
                subtract: (subtrahend: string | number | Long) => Long;
                sub: (subtrahend: string | number | Long) => Long;
                toInt: () => number;
                toNumber: () => number;
                toBytes: (le?: boolean | undefined) => number[];
                toBytesLE: () => number[];
                toBytesBE: () => number[];
                toSigned: () => Long;
                toString: (radix?: number | undefined) => string;
                toUnsigned: () => Long;
                xor: (other: string | number | Long) => Long;
            } & { [K_2 in Exclude<keyof I["Spec"][number]["blockLastUpdated"], keyof Long>]: never; }) | undefined;
            minStakeProvider?: ({
                denom?: string | undefined;
                amount?: string | undefined;
            } & {
                denom?: string | undefined;
                amount?: string | undefined;
            } & { [K_3 in Exclude<keyof I["Spec"][number]["minStakeProvider"], keyof import("../cosmos/base/v1beta1/coin").Coin>]: never; }) | undefined;
            minStakeClient?: ({
                denom?: string | undefined;
                amount?: string | undefined;
            } & {
                denom?: string | undefined;
                amount?: string | undefined;
            } & { [K_4 in Exclude<keyof I["Spec"][number]["minStakeClient"], keyof import("../cosmos/base/v1beta1/coin").Coin>]: never; }) | undefined;
            providersTypes?: import("./spec").Spec_ProvidersTypes | undefined;
            imports?: (string[] & string[] & { [K_5 in Exclude<keyof I["Spec"][number]["imports"], keyof string[]>]: never; }) | undefined;
            apiCollections?: ({
                enabled?: boolean | undefined;
                collectionData?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } | undefined;
                apis?: {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] | undefined;
                headers?: {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] | undefined;
                inheritanceApis?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] | undefined;
                parseDirectives?: {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] | undefined;
            }[] & ({
                enabled?: boolean | undefined;
                collectionData?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } | undefined;
                apis?: {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] | undefined;
                headers?: {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] | undefined;
                inheritanceApis?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] | undefined;
                parseDirectives?: {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] | undefined;
            } & {
                enabled?: boolean | undefined;
                collectionData?: ({
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } & {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } & { [K_6 in Exclude<keyof I["Spec"][number]["apiCollections"][number]["collectionData"], keyof import("./api_collection").CollectionData>]: never; }) | undefined;
                apis?: ({
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] & ({
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                } & {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | (Long & {
                        high: number;
                        low: number;
                        unsigned: boolean;
                        add: (addend: string | number | Long) => Long;
                        and: (other: string | number | Long) => Long;
                        compare: (other: string | number | Long) => number;
                        comp: (other: string | number | Long) => number;
                        divide: (divisor: string | number | Long) => Long;
                        div: (divisor: string | number | Long) => Long;
                        equals: (other: string | number | Long) => boolean;
                        eq: (other: string | number | Long) => boolean;
                        getHighBits: () => number;
                        getHighBitsUnsigned: () => number;
                        getLowBits: () => number;
                        getLowBitsUnsigned: () => number;
                        getNumBitsAbs: () => number;
                        greaterThan: (other: string | number | Long) => boolean;
                        gt: (other: string | number | Long) => boolean;
                        greaterThanOrEqual: (other: string | number | Long) => boolean;
                        gte: (other: string | number | Long) => boolean;
                        ge: (other: string | number | Long) => boolean;
                        isEven: () => boolean;
                        isNegative: () => boolean;
                        isOdd: () => boolean;
                        isPositive: () => boolean;
                        isZero: () => boolean;
                        eqz: () => boolean;
                        lessThan: (other: string | number | Long) => boolean;
                        lt: (other: string | number | Long) => boolean;
                        lessThanOrEqual: (other: string | number | Long) => boolean;
                        lte: (other: string | number | Long) => boolean;
                        le: (other: string | number | Long) => boolean;
                        modulo: (other: string | number | Long) => Long;
                        mod: (other: string | number | Long) => Long;
                        rem: (other: string | number | Long) => Long;
                        multiply: (multiplier: string | number | Long) => Long;
                        mul: (multiplier: string | number | Long) => Long;
                        negate: () => Long;
                        neg: () => Long;
                        not: () => Long;
                        countLeadingZeros: () => number;
                        clz: () => number;
                        countTrailingZeros: () => number;
                        ctz: () => number;
                        notEquals: (other: string | number | Long) => boolean;
                        neq: (other: string | number | Long) => boolean;
                        ne: (other: string | number | Long) => boolean;
                        or: (other: string | number | Long) => Long;
                        shiftLeft: (numBits: number | Long) => Long;
                        shl: (numBits: number | Long) => Long;
                        shiftRight: (numBits: number | Long) => Long;
                        shr: (numBits: number | Long) => Long;
                        shiftRightUnsigned: (numBits: number | Long) => Long;
                        shru: (numBits: number | Long) => Long;
                        shr_u: (numBits: number | Long) => Long;
                        rotateLeft: (numBits: number | Long) => Long;
                        rotl: (numBits: number | Long) => Long;
                        rotateRight: (numBits: number | Long) => Long;
                        rotr: (numBits: number | Long) => Long;
                        subtract: (subtrahend: string | number | Long) => Long;
                        sub: (subtrahend: string | number | Long) => Long;
                        toInt: () => number;
                        toNumber: () => number;
                        toBytes: (le?: boolean | undefined) => number[];
                        toBytesLE: () => number[];
                        toBytesBE: () => number[];
                        toSigned: () => Long;
                        toString: (radix?: number | undefined) => string;
                        toUnsigned: () => Long;
                        xor: (other: string | number | Long) => Long;
                    } & { [K_7 in Exclude<keyof I["Spec"][number]["apiCollections"][number]["apis"][number]["computeUnits"], keyof Long>]: never; }) | undefined;
                    extraComputeUnits?: string | number | (Long & {
                        high: number;
                        low: number;
                        unsigned: boolean;
                        add: (addend: string | number | Long) => Long;
                        and: (other: string | number | Long) => Long;
                        compare: (other: string | number | Long) => number;
                        comp: (other: string | number | Long) => number;
                        divide: (divisor: string | number | Long) => Long;
                        div: (divisor: string | number | Long) => Long;
                        equals: (other: string | number | Long) => boolean;
                        eq: (other: string | number | Long) => boolean;
                        getHighBits: () => number;
                        getHighBitsUnsigned: () => number;
                        getLowBits: () => number;
                        getLowBitsUnsigned: () => number;
                        getNumBitsAbs: () => number;
                        greaterThan: (other: string | number | Long) => boolean;
                        gt: (other: string | number | Long) => boolean;
                        greaterThanOrEqual: (other: string | number | Long) => boolean;
                        gte: (other: string | number | Long) => boolean;
                        ge: (other: string | number | Long) => boolean;
                        isEven: () => boolean;
                        isNegative: () => boolean;
                        isOdd: () => boolean;
                        isPositive: () => boolean;
                        isZero: () => boolean;
                        eqz: () => boolean;
                        lessThan: (other: string | number | Long) => boolean;
                        lt: (other: string | number | Long) => boolean;
                        lessThanOrEqual: (other: string | number | Long) => boolean;
                        lte: (other: string | number | Long) => boolean;
                        le: (other: string | number | Long) => boolean;
                        modulo: (other: string | number | Long) => Long;
                        mod: (other: string | number | Long) => Long;
                        rem: (other: string | number | Long) => Long;
                        multiply: (multiplier: string | number | Long) => Long;
                        mul: (multiplier: string | number | Long) => Long;
                        negate: () => Long;
                        neg: () => Long;
                        not: () => Long;
                        countLeadingZeros: () => number;
                        clz: () => number;
                        countTrailingZeros: () => number;
                        ctz: () => number;
                        notEquals: (other: string | number | Long) => boolean;
                        neq: (other: string | number | Long) => boolean;
                        ne: (other: string | number | Long) => boolean;
                        or: (other: string | number | Long) => Long;
                        shiftLeft: (numBits: number | Long) => Long;
                        shl: (numBits: number | Long) => Long;
                        shiftRight: (numBits: number | Long) => Long;
                        shr: (numBits: number | Long) => Long;
                        shiftRightUnsigned: (numBits: number | Long) => Long;
                        shru: (numBits: number | Long) => Long;
                        shr_u: (numBits: number | Long) => Long;
                        rotateLeft: (numBits: number | Long) => Long;
                        rotl: (numBits: number | Long) => Long;
                        rotateRight: (numBits: number | Long) => Long;
                        rotr: (numBits: number | Long) => Long;
                        subtract: (subtrahend: string | number | Long) => Long;
                        sub: (subtrahend: string | number | Long) => Long;
                        toInt: () => number;
                        toNumber: () => number;
                        toBytes: (le?: boolean | undefined) => number[];
                        toBytesLE: () => number[];
                        toBytesBE: () => number[];
                        toSigned: () => Long;
                        toString: (radix?: number | undefined) => string;
                        toUnsigned: () => Long;
                        xor: (other: string | number | Long) => Long;
                    } & { [K_8 in Exclude<keyof I["Spec"][number]["apiCollections"][number]["apis"][number]["extraComputeUnits"], keyof Long>]: never; }) | undefined;
                    category?: ({
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } & {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } & { [K_9 in Exclude<keyof I["Spec"][number]["apiCollections"][number]["apis"][number]["category"], keyof import("./api_collection").SpecCategory>]: never; }) | undefined;
                    blockParsing?: ({
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } & {
                        parserArg?: (string[] & string[] & { [K_10 in Exclude<keyof I["Spec"][number]["apiCollections"][number]["apis"][number]["blockParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } & { [K_11 in Exclude<keyof I["Spec"][number]["apiCollections"][number]["apis"][number]["blockParsing"], keyof import("./api_collection").BlockParser>]: never; }) | undefined;
                } & { [K_12 in Exclude<keyof I["Spec"][number]["apiCollections"][number]["apis"][number], keyof import("./api_collection").Api>]: never; })[] & { [K_13 in Exclude<keyof I["Spec"][number]["apiCollections"][number]["apis"], keyof {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[]>]: never; }) | undefined;
                headers?: ({
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] & ({
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                } & {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                } & { [K_14 in Exclude<keyof I["Spec"][number]["apiCollections"][number]["headers"][number], keyof import("./api_collection").Header>]: never; })[] & { [K_15 in Exclude<keyof I["Spec"][number]["apiCollections"][number]["headers"], keyof {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[]>]: never; }) | undefined;
                inheritanceApis?: ({
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] & ({
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } & {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } & { [K_16 in Exclude<keyof I["Spec"][number]["apiCollections"][number]["inheritanceApis"][number], keyof import("./api_collection").CollectionData>]: never; })[] & { [K_17 in Exclude<keyof I["Spec"][number]["apiCollections"][number]["inheritanceApis"], keyof {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[]>]: never; }) | undefined;
                parseDirectives?: ({
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] & ({
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                } & {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: ({
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } & {
                        parserArg?: (string[] & string[] & { [K_18 in Exclude<keyof I["Spec"][number]["apiCollections"][number]["parseDirectives"][number]["resultParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } & { [K_19 in Exclude<keyof I["Spec"][number]["apiCollections"][number]["parseDirectives"][number]["resultParsing"], keyof import("./api_collection").BlockParser>]: never; }) | undefined;
                    apiName?: string | undefined;
                } & { [K_20 in Exclude<keyof I["Spec"][number]["apiCollections"][number]["parseDirectives"][number], keyof import("./api_collection").ParseDirective>]: never; })[] & { [K_21 in Exclude<keyof I["Spec"][number]["apiCollections"][number]["parseDirectives"], keyof {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[]>]: never; }) | undefined;
            } & { [K_22 in Exclude<keyof I["Spec"][number]["apiCollections"][number], keyof import("./api_collection").ApiCollection>]: never; })[] & { [K_23 in Exclude<keyof I["Spec"][number]["apiCollections"], keyof {
                enabled?: boolean | undefined;
                collectionData?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } | undefined;
                apis?: {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] | undefined;
                headers?: {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] | undefined;
                inheritanceApis?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] | undefined;
                parseDirectives?: {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] | undefined;
            }[]>]: never; }) | undefined;
        } & { [K_24 in Exclude<keyof I["Spec"][number], keyof Spec>]: never; })[] & { [K_25 in Exclude<keyof I["Spec"], keyof {
            index?: string | undefined;
            name?: string | undefined;
            enabled?: boolean | undefined;
            reliabilityThreshold?: number | undefined;
            dataReliabilityEnabled?: boolean | undefined;
            blockDistanceForFinalizedData?: number | undefined;
            blocksInFinalizationProof?: number | undefined;
            averageBlockTime?: string | number | Long | undefined;
            allowedBlockLagForQosSync?: string | number | Long | undefined;
            blockLastUpdated?: string | number | Long | undefined;
            minStakeProvider?: {
                denom?: string | undefined;
                amount?: string | undefined;
            } | undefined;
            minStakeClient?: {
                denom?: string | undefined;
                amount?: string | undefined;
            } | undefined;
            providersTypes?: import("./spec").Spec_ProvidersTypes | undefined;
            imports?: string[] | undefined;
            apiCollections?: {
                enabled?: boolean | undefined;
                collectionData?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } | undefined;
                apis?: {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] | undefined;
                headers?: {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] | undefined;
                inheritanceApis?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] | undefined;
                parseDirectives?: {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] | undefined;
            }[] | undefined;
        }[]>]: never; }) | undefined;
        pagination?: ({
            nextKey?: Uint8Array | undefined;
            total?: string | number | Long | undefined;
        } & {
            nextKey?: Uint8Array | undefined;
            total?: string | number | (Long & {
                high: number;
                low: number;
                unsigned: boolean;
                add: (addend: string | number | Long) => Long;
                and: (other: string | number | Long) => Long;
                compare: (other: string | number | Long) => number;
                comp: (other: string | number | Long) => number;
                divide: (divisor: string | number | Long) => Long;
                div: (divisor: string | number | Long) => Long;
                equals: (other: string | number | Long) => boolean;
                eq: (other: string | number | Long) => boolean;
                getHighBits: () => number;
                getHighBitsUnsigned: () => number;
                getLowBits: () => number;
                getLowBitsUnsigned: () => number;
                getNumBitsAbs: () => number;
                greaterThan: (other: string | number | Long) => boolean;
                gt: (other: string | number | Long) => boolean;
                greaterThanOrEqual: (other: string | number | Long) => boolean;
                gte: (other: string | number | Long) => boolean;
                ge: (other: string | number | Long) => boolean;
                isEven: () => boolean;
                isNegative: () => boolean;
                isOdd: () => boolean;
                isPositive: () => boolean;
                isZero: () => boolean;
                eqz: () => boolean;
                lessThan: (other: string | number | Long) => boolean;
                lt: (other: string | number | Long) => boolean;
                lessThanOrEqual: (other: string | number | Long) => boolean;
                lte: (other: string | number | Long) => boolean;
                le: (other: string | number | Long) => boolean;
                modulo: (other: string | number | Long) => Long;
                mod: (other: string | number | Long) => Long;
                rem: (other: string | number | Long) => Long;
                multiply: (multiplier: string | number | Long) => Long;
                mul: (multiplier: string | number | Long) => Long;
                negate: () => Long;
                neg: () => Long;
                not: () => Long;
                countLeadingZeros: () => number;
                clz: () => number;
                countTrailingZeros: () => number;
                ctz: () => number;
                notEquals: (other: string | number | Long) => boolean;
                neq: (other: string | number | Long) => boolean;
                ne: (other: string | number | Long) => boolean;
                or: (other: string | number | Long) => Long;
                shiftLeft: (numBits: number | Long) => Long;
                shl: (numBits: number | Long) => Long;
                shiftRight: (numBits: number | Long) => Long;
                shr: (numBits: number | Long) => Long;
                shiftRightUnsigned: (numBits: number | Long) => Long;
                shru: (numBits: number | Long) => Long;
                shr_u: (numBits: number | Long) => Long;
                rotateLeft: (numBits: number | Long) => Long;
                rotl: (numBits: number | Long) => Long;
                rotateRight: (numBits: number | Long) => Long;
                rotr: (numBits: number | Long) => Long;
                subtract: (subtrahend: string | number | Long) => Long;
                sub: (subtrahend: string | number | Long) => Long;
                toInt: () => number;
                toNumber: () => number;
                toBytes: (le?: boolean | undefined) => number[];
                toBytesLE: () => number[];
                toBytesBE: () => number[];
                toSigned: () => Long;
                toString: (radix?: number | undefined) => string;
                toUnsigned: () => Long;
                xor: (other: string | number | Long) => Long;
            } & { [K_26 in Exclude<keyof I["pagination"]["total"], keyof Long>]: never; }) | undefined;
        } & { [K_27 in Exclude<keyof I["pagination"], keyof PageResponse>]: never; }) | undefined;
    } & { [K_28 in Exclude<keyof I, keyof QueryAllSpecResponse>]: never; }>(base?: I | undefined): QueryAllSpecResponse;
    fromPartial<I_1 extends {
        Spec?: {
            index?: string | undefined;
            name?: string | undefined;
            enabled?: boolean | undefined;
            reliabilityThreshold?: number | undefined;
            dataReliabilityEnabled?: boolean | undefined;
            blockDistanceForFinalizedData?: number | undefined;
            blocksInFinalizationProof?: number | undefined;
            averageBlockTime?: string | number | Long | undefined;
            allowedBlockLagForQosSync?: string | number | Long | undefined;
            blockLastUpdated?: string | number | Long | undefined;
            minStakeProvider?: {
                denom?: string | undefined;
                amount?: string | undefined;
            } | undefined;
            minStakeClient?: {
                denom?: string | undefined;
                amount?: string | undefined;
            } | undefined;
            providersTypes?: import("./spec").Spec_ProvidersTypes | undefined;
            imports?: string[] | undefined;
            apiCollections?: {
                enabled?: boolean | undefined;
                collectionData?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } | undefined;
                apis?: {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] | undefined;
                headers?: {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] | undefined;
                inheritanceApis?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] | undefined;
                parseDirectives?: {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] | undefined;
            }[] | undefined;
        }[] | undefined;
        pagination?: {
            nextKey?: Uint8Array | undefined;
            total?: string | number | Long | undefined;
        } | undefined;
    } & {
        Spec?: ({
            index?: string | undefined;
            name?: string | undefined;
            enabled?: boolean | undefined;
            reliabilityThreshold?: number | undefined;
            dataReliabilityEnabled?: boolean | undefined;
            blockDistanceForFinalizedData?: number | undefined;
            blocksInFinalizationProof?: number | undefined;
            averageBlockTime?: string | number | Long | undefined;
            allowedBlockLagForQosSync?: string | number | Long | undefined;
            blockLastUpdated?: string | number | Long | undefined;
            minStakeProvider?: {
                denom?: string | undefined;
                amount?: string | undefined;
            } | undefined;
            minStakeClient?: {
                denom?: string | undefined;
                amount?: string | undefined;
            } | undefined;
            providersTypes?: import("./spec").Spec_ProvidersTypes | undefined;
            imports?: string[] | undefined;
            apiCollections?: {
                enabled?: boolean | undefined;
                collectionData?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } | undefined;
                apis?: {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] | undefined;
                headers?: {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] | undefined;
                inheritanceApis?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] | undefined;
                parseDirectives?: {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] | undefined;
            }[] | undefined;
        }[] & ({
            index?: string | undefined;
            name?: string | undefined;
            enabled?: boolean | undefined;
            reliabilityThreshold?: number | undefined;
            dataReliabilityEnabled?: boolean | undefined;
            blockDistanceForFinalizedData?: number | undefined;
            blocksInFinalizationProof?: number | undefined;
            averageBlockTime?: string | number | Long | undefined;
            allowedBlockLagForQosSync?: string | number | Long | undefined;
            blockLastUpdated?: string | number | Long | undefined;
            minStakeProvider?: {
                denom?: string | undefined;
                amount?: string | undefined;
            } | undefined;
            minStakeClient?: {
                denom?: string | undefined;
                amount?: string | undefined;
            } | undefined;
            providersTypes?: import("./spec").Spec_ProvidersTypes | undefined;
            imports?: string[] | undefined;
            apiCollections?: {
                enabled?: boolean | undefined;
                collectionData?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } | undefined;
                apis?: {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] | undefined;
                headers?: {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] | undefined;
                inheritanceApis?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] | undefined;
                parseDirectives?: {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] | undefined;
            }[] | undefined;
        } & {
            index?: string | undefined;
            name?: string | undefined;
            enabled?: boolean | undefined;
            reliabilityThreshold?: number | undefined;
            dataReliabilityEnabled?: boolean | undefined;
            blockDistanceForFinalizedData?: number | undefined;
            blocksInFinalizationProof?: number | undefined;
            averageBlockTime?: string | number | (Long & {
                high: number;
                low: number;
                unsigned: boolean;
                add: (addend: string | number | Long) => Long;
                and: (other: string | number | Long) => Long;
                compare: (other: string | number | Long) => number;
                comp: (other: string | number | Long) => number;
                divide: (divisor: string | number | Long) => Long;
                div: (divisor: string | number | Long) => Long;
                equals: (other: string | number | Long) => boolean;
                eq: (other: string | number | Long) => boolean;
                getHighBits: () => number;
                getHighBitsUnsigned: () => number;
                getLowBits: () => number;
                getLowBitsUnsigned: () => number;
                getNumBitsAbs: () => number;
                greaterThan: (other: string | number | Long) => boolean;
                gt: (other: string | number | Long) => boolean;
                greaterThanOrEqual: (other: string | number | Long) => boolean;
                gte: (other: string | number | Long) => boolean;
                ge: (other: string | number | Long) => boolean;
                isEven: () => boolean;
                isNegative: () => boolean;
                isOdd: () => boolean;
                isPositive: () => boolean;
                isZero: () => boolean;
                eqz: () => boolean;
                lessThan: (other: string | number | Long) => boolean;
                lt: (other: string | number | Long) => boolean;
                lessThanOrEqual: (other: string | number | Long) => boolean;
                lte: (other: string | number | Long) => boolean;
                le: (other: string | number | Long) => boolean;
                modulo: (other: string | number | Long) => Long;
                mod: (other: string | number | Long) => Long;
                rem: (other: string | number | Long) => Long;
                multiply: (multiplier: string | number | Long) => Long;
                mul: (multiplier: string | number | Long) => Long;
                negate: () => Long;
                neg: () => Long;
                not: () => Long;
                countLeadingZeros: () => number;
                clz: () => number;
                countTrailingZeros: () => number;
                ctz: () => number;
                notEquals: (other: string | number | Long) => boolean;
                neq: (other: string | number | Long) => boolean;
                ne: (other: string | number | Long) => boolean;
                or: (other: string | number | Long) => Long;
                shiftLeft: (numBits: number | Long) => Long;
                shl: (numBits: number | Long) => Long;
                shiftRight: (numBits: number | Long) => Long;
                shr: (numBits: number | Long) => Long;
                shiftRightUnsigned: (numBits: number | Long) => Long;
                shru: (numBits: number | Long) => Long;
                shr_u: (numBits: number | Long) => Long;
                rotateLeft: (numBits: number | Long) => Long;
                rotl: (numBits: number | Long) => Long;
                rotateRight: (numBits: number | Long) => Long;
                rotr: (numBits: number | Long) => Long;
                subtract: (subtrahend: string | number | Long) => Long;
                sub: (subtrahend: string | number | Long) => Long;
                toInt: () => number;
                toNumber: () => number;
                toBytes: (le?: boolean | undefined) => number[];
                toBytesLE: () => number[];
                toBytesBE: () => number[];
                toSigned: () => Long;
                toString: (radix?: number | undefined) => string;
                toUnsigned: () => Long;
                xor: (other: string | number | Long) => Long;
            } & { [K_29 in Exclude<keyof I_1["Spec"][number]["averageBlockTime"], keyof Long>]: never; }) | undefined;
            allowedBlockLagForQosSync?: string | number | (Long & {
                high: number;
                low: number;
                unsigned: boolean;
                add: (addend: string | number | Long) => Long;
                and: (other: string | number | Long) => Long;
                compare: (other: string | number | Long) => number;
                comp: (other: string | number | Long) => number;
                divide: (divisor: string | number | Long) => Long;
                div: (divisor: string | number | Long) => Long;
                equals: (other: string | number | Long) => boolean;
                eq: (other: string | number | Long) => boolean;
                getHighBits: () => number;
                getHighBitsUnsigned: () => number;
                getLowBits: () => number;
                getLowBitsUnsigned: () => number;
                getNumBitsAbs: () => number;
                greaterThan: (other: string | number | Long) => boolean;
                gt: (other: string | number | Long) => boolean;
                greaterThanOrEqual: (other: string | number | Long) => boolean;
                gte: (other: string | number | Long) => boolean;
                ge: (other: string | number | Long) => boolean;
                isEven: () => boolean;
                isNegative: () => boolean;
                isOdd: () => boolean;
                isPositive: () => boolean;
                isZero: () => boolean;
                eqz: () => boolean;
                lessThan: (other: string | number | Long) => boolean;
                lt: (other: string | number | Long) => boolean;
                lessThanOrEqual: (other: string | number | Long) => boolean;
                lte: (other: string | number | Long) => boolean;
                le: (other: string | number | Long) => boolean;
                modulo: (other: string | number | Long) => Long;
                mod: (other: string | number | Long) => Long;
                rem: (other: string | number | Long) => Long;
                multiply: (multiplier: string | number | Long) => Long;
                mul: (multiplier: string | number | Long) => Long;
                negate: () => Long;
                neg: () => Long;
                not: () => Long;
                countLeadingZeros: () => number;
                clz: () => number;
                countTrailingZeros: () => number;
                ctz: () => number;
                notEquals: (other: string | number | Long) => boolean;
                neq: (other: string | number | Long) => boolean;
                ne: (other: string | number | Long) => boolean;
                or: (other: string | number | Long) => Long;
                shiftLeft: (numBits: number | Long) => Long;
                shl: (numBits: number | Long) => Long;
                shiftRight: (numBits: number | Long) => Long;
                shr: (numBits: number | Long) => Long;
                shiftRightUnsigned: (numBits: number | Long) => Long;
                shru: (numBits: number | Long) => Long;
                shr_u: (numBits: number | Long) => Long;
                rotateLeft: (numBits: number | Long) => Long;
                rotl: (numBits: number | Long) => Long;
                rotateRight: (numBits: number | Long) => Long;
                rotr: (numBits: number | Long) => Long;
                subtract: (subtrahend: string | number | Long) => Long;
                sub: (subtrahend: string | number | Long) => Long;
                toInt: () => number;
                toNumber: () => number;
                toBytes: (le?: boolean | undefined) => number[];
                toBytesLE: () => number[];
                toBytesBE: () => number[];
                toSigned: () => Long;
                toString: (radix?: number | undefined) => string;
                toUnsigned: () => Long;
                xor: (other: string | number | Long) => Long;
            } & { [K_30 in Exclude<keyof I_1["Spec"][number]["allowedBlockLagForQosSync"], keyof Long>]: never; }) | undefined;
            blockLastUpdated?: string | number | (Long & {
                high: number;
                low: number;
                unsigned: boolean;
                add: (addend: string | number | Long) => Long;
                and: (other: string | number | Long) => Long;
                compare: (other: string | number | Long) => number;
                comp: (other: string | number | Long) => number;
                divide: (divisor: string | number | Long) => Long;
                div: (divisor: string | number | Long) => Long;
                equals: (other: string | number | Long) => boolean;
                eq: (other: string | number | Long) => boolean;
                getHighBits: () => number;
                getHighBitsUnsigned: () => number;
                getLowBits: () => number;
                getLowBitsUnsigned: () => number;
                getNumBitsAbs: () => number;
                greaterThan: (other: string | number | Long) => boolean;
                gt: (other: string | number | Long) => boolean;
                greaterThanOrEqual: (other: string | number | Long) => boolean;
                gte: (other: string | number | Long) => boolean;
                ge: (other: string | number | Long) => boolean;
                isEven: () => boolean;
                isNegative: () => boolean;
                isOdd: () => boolean;
                isPositive: () => boolean;
                isZero: () => boolean;
                eqz: () => boolean;
                lessThan: (other: string | number | Long) => boolean;
                lt: (other: string | number | Long) => boolean;
                lessThanOrEqual: (other: string | number | Long) => boolean;
                lte: (other: string | number | Long) => boolean;
                le: (other: string | number | Long) => boolean;
                modulo: (other: string | number | Long) => Long;
                mod: (other: string | number | Long) => Long;
                rem: (other: string | number | Long) => Long;
                multiply: (multiplier: string | number | Long) => Long;
                mul: (multiplier: string | number | Long) => Long;
                negate: () => Long;
                neg: () => Long;
                not: () => Long;
                countLeadingZeros: () => number;
                clz: () => number;
                countTrailingZeros: () => number;
                ctz: () => number;
                notEquals: (other: string | number | Long) => boolean;
                neq: (other: string | number | Long) => boolean;
                ne: (other: string | number | Long) => boolean;
                or: (other: string | number | Long) => Long;
                shiftLeft: (numBits: number | Long) => Long;
                shl: (numBits: number | Long) => Long;
                shiftRight: (numBits: number | Long) => Long;
                shr: (numBits: number | Long) => Long;
                shiftRightUnsigned: (numBits: number | Long) => Long;
                shru: (numBits: number | Long) => Long;
                shr_u: (numBits: number | Long) => Long;
                rotateLeft: (numBits: number | Long) => Long;
                rotl: (numBits: number | Long) => Long;
                rotateRight: (numBits: number | Long) => Long;
                rotr: (numBits: number | Long) => Long;
                subtract: (subtrahend: string | number | Long) => Long;
                sub: (subtrahend: string | number | Long) => Long;
                toInt: () => number;
                toNumber: () => number;
                toBytes: (le?: boolean | undefined) => number[];
                toBytesLE: () => number[];
                toBytesBE: () => number[];
                toSigned: () => Long;
                toString: (radix?: number | undefined) => string;
                toUnsigned: () => Long;
                xor: (other: string | number | Long) => Long;
            } & { [K_31 in Exclude<keyof I_1["Spec"][number]["blockLastUpdated"], keyof Long>]: never; }) | undefined;
            minStakeProvider?: ({
                denom?: string | undefined;
                amount?: string | undefined;
            } & {
                denom?: string | undefined;
                amount?: string | undefined;
            } & { [K_32 in Exclude<keyof I_1["Spec"][number]["minStakeProvider"], keyof import("../cosmos/base/v1beta1/coin").Coin>]: never; }) | undefined;
            minStakeClient?: ({
                denom?: string | undefined;
                amount?: string | undefined;
            } & {
                denom?: string | undefined;
                amount?: string | undefined;
            } & { [K_33 in Exclude<keyof I_1["Spec"][number]["minStakeClient"], keyof import("../cosmos/base/v1beta1/coin").Coin>]: never; }) | undefined;
            providersTypes?: import("./spec").Spec_ProvidersTypes | undefined;
            imports?: (string[] & string[] & { [K_34 in Exclude<keyof I_1["Spec"][number]["imports"], keyof string[]>]: never; }) | undefined;
            apiCollections?: ({
                enabled?: boolean | undefined;
                collectionData?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } | undefined;
                apis?: {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] | undefined;
                headers?: {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] | undefined;
                inheritanceApis?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] | undefined;
                parseDirectives?: {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] | undefined;
            }[] & ({
                enabled?: boolean | undefined;
                collectionData?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } | undefined;
                apis?: {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] | undefined;
                headers?: {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] | undefined;
                inheritanceApis?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] | undefined;
                parseDirectives?: {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] | undefined;
            } & {
                enabled?: boolean | undefined;
                collectionData?: ({
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } & {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } & { [K_35 in Exclude<keyof I_1["Spec"][number]["apiCollections"][number]["collectionData"], keyof import("./api_collection").CollectionData>]: never; }) | undefined;
                apis?: ({
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] & ({
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                } & {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | (Long & {
                        high: number;
                        low: number;
                        unsigned: boolean;
                        add: (addend: string | number | Long) => Long;
                        and: (other: string | number | Long) => Long;
                        compare: (other: string | number | Long) => number;
                        comp: (other: string | number | Long) => number;
                        divide: (divisor: string | number | Long) => Long;
                        div: (divisor: string | number | Long) => Long;
                        equals: (other: string | number | Long) => boolean;
                        eq: (other: string | number | Long) => boolean;
                        getHighBits: () => number;
                        getHighBitsUnsigned: () => number;
                        getLowBits: () => number;
                        getLowBitsUnsigned: () => number;
                        getNumBitsAbs: () => number;
                        greaterThan: (other: string | number | Long) => boolean;
                        gt: (other: string | number | Long) => boolean;
                        greaterThanOrEqual: (other: string | number | Long) => boolean;
                        gte: (other: string | number | Long) => boolean;
                        ge: (other: string | number | Long) => boolean;
                        isEven: () => boolean;
                        isNegative: () => boolean;
                        isOdd: () => boolean;
                        isPositive: () => boolean;
                        isZero: () => boolean;
                        eqz: () => boolean;
                        lessThan: (other: string | number | Long) => boolean;
                        lt: (other: string | number | Long) => boolean;
                        lessThanOrEqual: (other: string | number | Long) => boolean;
                        lte: (other: string | number | Long) => boolean;
                        le: (other: string | number | Long) => boolean;
                        modulo: (other: string | number | Long) => Long;
                        mod: (other: string | number | Long) => Long;
                        rem: (other: string | number | Long) => Long;
                        multiply: (multiplier: string | number | Long) => Long;
                        mul: (multiplier: string | number | Long) => Long;
                        negate: () => Long;
                        neg: () => Long;
                        not: () => Long;
                        countLeadingZeros: () => number;
                        clz: () => number;
                        countTrailingZeros: () => number;
                        ctz: () => number;
                        notEquals: (other: string | number | Long) => boolean;
                        neq: (other: string | number | Long) => boolean;
                        ne: (other: string | number | Long) => boolean;
                        or: (other: string | number | Long) => Long;
                        shiftLeft: (numBits: number | Long) => Long;
                        shl: (numBits: number | Long) => Long;
                        shiftRight: (numBits: number | Long) => Long;
                        shr: (numBits: number | Long) => Long;
                        shiftRightUnsigned: (numBits: number | Long) => Long;
                        shru: (numBits: number | Long) => Long;
                        shr_u: (numBits: number | Long) => Long;
                        rotateLeft: (numBits: number | Long) => Long;
                        rotl: (numBits: number | Long) => Long;
                        rotateRight: (numBits: number | Long) => Long;
                        rotr: (numBits: number | Long) => Long;
                        subtract: (subtrahend: string | number | Long) => Long;
                        sub: (subtrahend: string | number | Long) => Long;
                        toInt: () => number;
                        toNumber: () => number;
                        toBytes: (le?: boolean | undefined) => number[];
                        toBytesLE: () => number[];
                        toBytesBE: () => number[];
                        toSigned: () => Long;
                        toString: (radix?: number | undefined) => string;
                        toUnsigned: () => Long;
                        xor: (other: string | number | Long) => Long;
                    } & { [K_36 in Exclude<keyof I_1["Spec"][number]["apiCollections"][number]["apis"][number]["computeUnits"], keyof Long>]: never; }) | undefined;
                    extraComputeUnits?: string | number | (Long & {
                        high: number;
                        low: number;
                        unsigned: boolean;
                        add: (addend: string | number | Long) => Long;
                        and: (other: string | number | Long) => Long;
                        compare: (other: string | number | Long) => number;
                        comp: (other: string | number | Long) => number;
                        divide: (divisor: string | number | Long) => Long;
                        div: (divisor: string | number | Long) => Long;
                        equals: (other: string | number | Long) => boolean;
                        eq: (other: string | number | Long) => boolean;
                        getHighBits: () => number;
                        getHighBitsUnsigned: () => number;
                        getLowBits: () => number;
                        getLowBitsUnsigned: () => number;
                        getNumBitsAbs: () => number;
                        greaterThan: (other: string | number | Long) => boolean;
                        gt: (other: string | number | Long) => boolean;
                        greaterThanOrEqual: (other: string | number | Long) => boolean;
                        gte: (other: string | number | Long) => boolean;
                        ge: (other: string | number | Long) => boolean;
                        isEven: () => boolean;
                        isNegative: () => boolean;
                        isOdd: () => boolean;
                        isPositive: () => boolean;
                        isZero: () => boolean;
                        eqz: () => boolean;
                        lessThan: (other: string | number | Long) => boolean;
                        lt: (other: string | number | Long) => boolean;
                        lessThanOrEqual: (other: string | number | Long) => boolean;
                        lte: (other: string | number | Long) => boolean;
                        le: (other: string | number | Long) => boolean;
                        modulo: (other: string | number | Long) => Long;
                        mod: (other: string | number | Long) => Long;
                        rem: (other: string | number | Long) => Long;
                        multiply: (multiplier: string | number | Long) => Long;
                        mul: (multiplier: string | number | Long) => Long;
                        negate: () => Long;
                        neg: () => Long;
                        not: () => Long;
                        countLeadingZeros: () => number;
                        clz: () => number;
                        countTrailingZeros: () => number;
                        ctz: () => number;
                        notEquals: (other: string | number | Long) => boolean;
                        neq: (other: string | number | Long) => boolean;
                        ne: (other: string | number | Long) => boolean;
                        or: (other: string | number | Long) => Long;
                        shiftLeft: (numBits: number | Long) => Long;
                        shl: (numBits: number | Long) => Long;
                        shiftRight: (numBits: number | Long) => Long;
                        shr: (numBits: number | Long) => Long;
                        shiftRightUnsigned: (numBits: number | Long) => Long;
                        shru: (numBits: number | Long) => Long;
                        shr_u: (numBits: number | Long) => Long;
                        rotateLeft: (numBits: number | Long) => Long;
                        rotl: (numBits: number | Long) => Long;
                        rotateRight: (numBits: number | Long) => Long;
                        rotr: (numBits: number | Long) => Long;
                        subtract: (subtrahend: string | number | Long) => Long;
                        sub: (subtrahend: string | number | Long) => Long;
                        toInt: () => number;
                        toNumber: () => number;
                        toBytes: (le?: boolean | undefined) => number[];
                        toBytesLE: () => number[];
                        toBytesBE: () => number[];
                        toSigned: () => Long;
                        toString: (radix?: number | undefined) => string;
                        toUnsigned: () => Long;
                        xor: (other: string | number | Long) => Long;
                    } & { [K_37 in Exclude<keyof I_1["Spec"][number]["apiCollections"][number]["apis"][number]["extraComputeUnits"], keyof Long>]: never; }) | undefined;
                    category?: ({
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } & {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } & { [K_38 in Exclude<keyof I_1["Spec"][number]["apiCollections"][number]["apis"][number]["category"], keyof import("./api_collection").SpecCategory>]: never; }) | undefined;
                    blockParsing?: ({
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } & {
                        parserArg?: (string[] & string[] & { [K_39 in Exclude<keyof I_1["Spec"][number]["apiCollections"][number]["apis"][number]["blockParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } & { [K_40 in Exclude<keyof I_1["Spec"][number]["apiCollections"][number]["apis"][number]["blockParsing"], keyof import("./api_collection").BlockParser>]: never; }) | undefined;
                } & { [K_41 in Exclude<keyof I_1["Spec"][number]["apiCollections"][number]["apis"][number], keyof import("./api_collection").Api>]: never; })[] & { [K_42 in Exclude<keyof I_1["Spec"][number]["apiCollections"][number]["apis"], keyof {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[]>]: never; }) | undefined;
                headers?: ({
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] & ({
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                } & {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                } & { [K_43 in Exclude<keyof I_1["Spec"][number]["apiCollections"][number]["headers"][number], keyof import("./api_collection").Header>]: never; })[] & { [K_44 in Exclude<keyof I_1["Spec"][number]["apiCollections"][number]["headers"], keyof {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[]>]: never; }) | undefined;
                inheritanceApis?: ({
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] & ({
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } & {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } & { [K_45 in Exclude<keyof I_1["Spec"][number]["apiCollections"][number]["inheritanceApis"][number], keyof import("./api_collection").CollectionData>]: never; })[] & { [K_46 in Exclude<keyof I_1["Spec"][number]["apiCollections"][number]["inheritanceApis"], keyof {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[]>]: never; }) | undefined;
                parseDirectives?: ({
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] & ({
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                } & {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: ({
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } & {
                        parserArg?: (string[] & string[] & { [K_47 in Exclude<keyof I_1["Spec"][number]["apiCollections"][number]["parseDirectives"][number]["resultParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } & { [K_48 in Exclude<keyof I_1["Spec"][number]["apiCollections"][number]["parseDirectives"][number]["resultParsing"], keyof import("./api_collection").BlockParser>]: never; }) | undefined;
                    apiName?: string | undefined;
                } & { [K_49 in Exclude<keyof I_1["Spec"][number]["apiCollections"][number]["parseDirectives"][number], keyof import("./api_collection").ParseDirective>]: never; })[] & { [K_50 in Exclude<keyof I_1["Spec"][number]["apiCollections"][number]["parseDirectives"], keyof {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[]>]: never; }) | undefined;
            } & { [K_51 in Exclude<keyof I_1["Spec"][number]["apiCollections"][number], keyof import("./api_collection").ApiCollection>]: never; })[] & { [K_52 in Exclude<keyof I_1["Spec"][number]["apiCollections"], keyof {
                enabled?: boolean | undefined;
                collectionData?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } | undefined;
                apis?: {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] | undefined;
                headers?: {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] | undefined;
                inheritanceApis?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] | undefined;
                parseDirectives?: {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] | undefined;
            }[]>]: never; }) | undefined;
        } & { [K_53 in Exclude<keyof I_1["Spec"][number], keyof Spec>]: never; })[] & { [K_54 in Exclude<keyof I_1["Spec"], keyof {
            index?: string | undefined;
            name?: string | undefined;
            enabled?: boolean | undefined;
            reliabilityThreshold?: number | undefined;
            dataReliabilityEnabled?: boolean | undefined;
            blockDistanceForFinalizedData?: number | undefined;
            blocksInFinalizationProof?: number | undefined;
            averageBlockTime?: string | number | Long | undefined;
            allowedBlockLagForQosSync?: string | number | Long | undefined;
            blockLastUpdated?: string | number | Long | undefined;
            minStakeProvider?: {
                denom?: string | undefined;
                amount?: string | undefined;
            } | undefined;
            minStakeClient?: {
                denom?: string | undefined;
                amount?: string | undefined;
            } | undefined;
            providersTypes?: import("./spec").Spec_ProvidersTypes | undefined;
            imports?: string[] | undefined;
            apiCollections?: {
                enabled?: boolean | undefined;
                collectionData?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                } | undefined;
                apis?: {
                    enabled?: boolean | undefined;
                    name?: string | undefined;
                    computeUnits?: string | number | Long | undefined;
                    extraComputeUnits?: string | number | Long | undefined;
                    category?: {
                        deterministic?: boolean | undefined;
                        local?: boolean | undefined;
                        subscription?: boolean | undefined;
                        stateful?: number | undefined;
                        hangingApi?: boolean | undefined;
                    } | undefined;
                    blockParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                }[] | undefined;
                headers?: {
                    name?: string | undefined;
                    kind?: import("./api_collection").Header_HeaderType | undefined;
                    functionTag?: import("./api_collection").functionTag | undefined;
                }[] | undefined;
                inheritanceApis?: {
                    apiInterface?: string | undefined;
                    internalPath?: string | undefined;
                    type?: string | undefined;
                    addOn?: string | undefined;
                }[] | undefined;
                parseDirectives?: {
                    functionTag?: import("./api_collection").functionTag | undefined;
                    functionTemplate?: string | undefined;
                    resultParsing?: {
                        parserArg?: string[] | undefined;
                        parserFunc?: import("./api_collection").parserFunc | undefined;
                        defaultValue?: string | undefined;
                        encoding?: string | undefined;
                    } | undefined;
                    apiName?: string | undefined;
                }[] | undefined;
            }[] | undefined;
        }[]>]: never; }) | undefined;
        pagination?: ({
            nextKey?: Uint8Array | undefined;
            total?: string | number | Long | undefined;
        } & {
            nextKey?: Uint8Array | undefined;
            total?: string | number | (Long & {
                high: number;
                low: number;
                unsigned: boolean;
                add: (addend: string | number | Long) => Long;
                and: (other: string | number | Long) => Long;
                compare: (other: string | number | Long) => number;
                comp: (other: string | number | Long) => number;
                divide: (divisor: string | number | Long) => Long;
                div: (divisor: string | number | Long) => Long;
                equals: (other: string | number | Long) => boolean;
                eq: (other: string | number | Long) => boolean;
                getHighBits: () => number;
                getHighBitsUnsigned: () => number;
                getLowBits: () => number;
                getLowBitsUnsigned: () => number;
                getNumBitsAbs: () => number;
                greaterThan: (other: string | number | Long) => boolean;
                gt: (other: string | number | Long) => boolean;
                greaterThanOrEqual: (other: string | number | Long) => boolean;
                gte: (other: string | number | Long) => boolean;
                ge: (other: string | number | Long) => boolean;
                isEven: () => boolean;
                isNegative: () => boolean;
                isOdd: () => boolean;
                isPositive: () => boolean;
                isZero: () => boolean;
                eqz: () => boolean;
                lessThan: (other: string | number | Long) => boolean;
                lt: (other: string | number | Long) => boolean;
                lessThanOrEqual: (other: string | number | Long) => boolean;
                lte: (other: string | number | Long) => boolean;
                le: (other: string | number | Long) => boolean;
                modulo: (other: string | number | Long) => Long;
                mod: (other: string | number | Long) => Long;
                rem: (other: string | number | Long) => Long;
                multiply: (multiplier: string | number | Long) => Long;
                mul: (multiplier: string | number | Long) => Long;
                negate: () => Long;
                neg: () => Long;
                not: () => Long;
                countLeadingZeros: () => number;
                clz: () => number;
                countTrailingZeros: () => number;
                ctz: () => number;
                notEquals: (other: string | number | Long) => boolean;
                neq: (other: string | number | Long) => boolean;
                ne: (other: string | number | Long) => boolean;
                or: (other: string | number | Long) => Long;
                shiftLeft: (numBits: number | Long) => Long;
                shl: (numBits: number | Long) => Long;
                shiftRight: (numBits: number | Long) => Long;
                shr: (numBits: number | Long) => Long;
                shiftRightUnsigned: (numBits: number | Long) => Long;
                shru: (numBits: number | Long) => Long;
                shr_u: (numBits: number | Long) => Long;
                rotateLeft: (numBits: number | Long) => Long;
                rotl: (numBits: number | Long) => Long;
                rotateRight: (numBits: number | Long) => Long;
                rotr: (numBits: number | Long) => Long;
                subtract: (subtrahend: string | number | Long) => Long;
                sub: (subtrahend: string | number | Long) => Long;
                toInt: () => number;
                toNumber: () => number;
                toBytes: (le?: boolean | undefined) => number[];
                toBytesLE: () => number[];
                toBytesBE: () => number[];
                toSigned: () => Long;
                toString: (radix?: number | undefined) => string;
                toUnsigned: () => Long;
                xor: (other: string | number | Long) => Long;
            } & { [K_55 in Exclude<keyof I_1["pagination"]["total"], keyof Long>]: never; }) | undefined;
        } & { [K_56 in Exclude<keyof I_1["pagination"], keyof PageResponse>]: never; }) | undefined;
    } & { [K_57 in Exclude<keyof I_1, keyof QueryAllSpecResponse>]: never; }>(object: I_1): QueryAllSpecResponse;
};
export declare const QueryShowAllChainsRequest: {
    encode(_: QueryShowAllChainsRequest, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): QueryShowAllChainsRequest;
    fromJSON(_: any): QueryShowAllChainsRequest;
    toJSON(_: QueryShowAllChainsRequest): unknown;
    create<I extends {} & {} & { [K in Exclude<keyof I, never>]: never; }>(base?: I | undefined): QueryShowAllChainsRequest;
    fromPartial<I_1 extends {} & {} & { [K_1 in Exclude<keyof I_1, never>]: never; }>(_: I_1): QueryShowAllChainsRequest;
};
export declare const QueryShowAllChainsResponse: {
    encode(message: QueryShowAllChainsResponse, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): QueryShowAllChainsResponse;
    fromJSON(object: any): QueryShowAllChainsResponse;
    toJSON(message: QueryShowAllChainsResponse): unknown;
    create<I extends {
        chainInfoList?: {
            chainName?: string | undefined;
            chainID?: string | undefined;
            enabledApiInterfaces?: string[] | undefined;
            apiCount?: string | number | Long | undefined;
        }[] | undefined;
    } & {
        chainInfoList?: ({
            chainName?: string | undefined;
            chainID?: string | undefined;
            enabledApiInterfaces?: string[] | undefined;
            apiCount?: string | number | Long | undefined;
        }[] & ({
            chainName?: string | undefined;
            chainID?: string | undefined;
            enabledApiInterfaces?: string[] | undefined;
            apiCount?: string | number | Long | undefined;
        } & {
            chainName?: string | undefined;
            chainID?: string | undefined;
            enabledApiInterfaces?: (string[] & string[] & { [K in Exclude<keyof I["chainInfoList"][number]["enabledApiInterfaces"], keyof string[]>]: never; }) | undefined;
            apiCount?: string | number | (Long & {
                high: number;
                low: number;
                unsigned: boolean;
                add: (addend: string | number | Long) => Long;
                and: (other: string | number | Long) => Long;
                compare: (other: string | number | Long) => number;
                comp: (other: string | number | Long) => number;
                divide: (divisor: string | number | Long) => Long;
                div: (divisor: string | number | Long) => Long;
                equals: (other: string | number | Long) => boolean;
                eq: (other: string | number | Long) => boolean;
                getHighBits: () => number;
                getHighBitsUnsigned: () => number;
                getLowBits: () => number;
                getLowBitsUnsigned: () => number;
                getNumBitsAbs: () => number;
                greaterThan: (other: string | number | Long) => boolean;
                gt: (other: string | number | Long) => boolean;
                greaterThanOrEqual: (other: string | number | Long) => boolean;
                gte: (other: string | number | Long) => boolean;
                ge: (other: string | number | Long) => boolean;
                isEven: () => boolean;
                isNegative: () => boolean;
                isOdd: () => boolean;
                isPositive: () => boolean;
                isZero: () => boolean;
                eqz: () => boolean;
                lessThan: (other: string | number | Long) => boolean;
                lt: (other: string | number | Long) => boolean;
                lessThanOrEqual: (other: string | number | Long) => boolean;
                lte: (other: string | number | Long) => boolean;
                le: (other: string | number | Long) => boolean;
                modulo: (other: string | number | Long) => Long;
                mod: (other: string | number | Long) => Long;
                rem: (other: string | number | Long) => Long;
                multiply: (multiplier: string | number | Long) => Long;
                mul: (multiplier: string | number | Long) => Long;
                negate: () => Long;
                neg: () => Long;
                not: () => Long;
                countLeadingZeros: () => number;
                clz: () => number;
                countTrailingZeros: () => number;
                ctz: () => number;
                notEquals: (other: string | number | Long) => boolean;
                neq: (other: string | number | Long) => boolean;
                ne: (other: string | number | Long) => boolean;
                or: (other: string | number | Long) => Long;
                shiftLeft: (numBits: number | Long) => Long;
                shl: (numBits: number | Long) => Long;
                shiftRight: (numBits: number | Long) => Long;
                shr: (numBits: number | Long) => Long;
                shiftRightUnsigned: (numBits: number | Long) => Long;
                shru: (numBits: number | Long) => Long;
                shr_u: (numBits: number | Long) => Long;
                rotateLeft: (numBits: number | Long) => Long;
                rotl: (numBits: number | Long) => Long;
                rotateRight: (numBits: number | Long) => Long;
                rotr: (numBits: number | Long) => Long;
                subtract: (subtrahend: string | number | Long) => Long;
                sub: (subtrahend: string | number | Long) => Long;
                toInt: () => number;
                toNumber: () => number;
                toBytes: (le?: boolean | undefined) => number[];
                toBytesLE: () => number[];
                toBytesBE: () => number[];
                toSigned: () => Long;
                toString: (radix?: number | undefined) => string;
                toUnsigned: () => Long;
                xor: (other: string | number | Long) => Long;
            } & { [K_1 in Exclude<keyof I["chainInfoList"][number]["apiCount"], keyof Long>]: never; }) | undefined;
        } & { [K_2 in Exclude<keyof I["chainInfoList"][number], keyof ShowAllChainsInfoStruct>]: never; })[] & { [K_3 in Exclude<keyof I["chainInfoList"], keyof {
            chainName?: string | undefined;
            chainID?: string | undefined;
            enabledApiInterfaces?: string[] | undefined;
            apiCount?: string | number | Long | undefined;
        }[]>]: never; }) | undefined;
    } & { [K_4 in Exclude<keyof I, "chainInfoList">]: never; }>(base?: I | undefined): QueryShowAllChainsResponse;
    fromPartial<I_1 extends {
        chainInfoList?: {
            chainName?: string | undefined;
            chainID?: string | undefined;
            enabledApiInterfaces?: string[] | undefined;
            apiCount?: string | number | Long | undefined;
        }[] | undefined;
    } & {
        chainInfoList?: ({
            chainName?: string | undefined;
            chainID?: string | undefined;
            enabledApiInterfaces?: string[] | undefined;
            apiCount?: string | number | Long | undefined;
        }[] & ({
            chainName?: string | undefined;
            chainID?: string | undefined;
            enabledApiInterfaces?: string[] | undefined;
            apiCount?: string | number | Long | undefined;
        } & {
            chainName?: string | undefined;
            chainID?: string | undefined;
            enabledApiInterfaces?: (string[] & string[] & { [K_5 in Exclude<keyof I_1["chainInfoList"][number]["enabledApiInterfaces"], keyof string[]>]: never; }) | undefined;
            apiCount?: string | number | (Long & {
                high: number;
                low: number;
                unsigned: boolean;
                add: (addend: string | number | Long) => Long;
                and: (other: string | number | Long) => Long;
                compare: (other: string | number | Long) => number;
                comp: (other: string | number | Long) => number;
                divide: (divisor: string | number | Long) => Long;
                div: (divisor: string | number | Long) => Long;
                equals: (other: string | number | Long) => boolean;
                eq: (other: string | number | Long) => boolean;
                getHighBits: () => number;
                getHighBitsUnsigned: () => number;
                getLowBits: () => number;
                getLowBitsUnsigned: () => number;
                getNumBitsAbs: () => number;
                greaterThan: (other: string | number | Long) => boolean;
                gt: (other: string | number | Long) => boolean;
                greaterThanOrEqual: (other: string | number | Long) => boolean;
                gte: (other: string | number | Long) => boolean;
                ge: (other: string | number | Long) => boolean;
                isEven: () => boolean;
                isNegative: () => boolean;
                isOdd: () => boolean;
                isPositive: () => boolean;
                isZero: () => boolean;
                eqz: () => boolean;
                lessThan: (other: string | number | Long) => boolean;
                lt: (other: string | number | Long) => boolean;
                lessThanOrEqual: (other: string | number | Long) => boolean;
                lte: (other: string | number | Long) => boolean;
                le: (other: string | number | Long) => boolean;
                modulo: (other: string | number | Long) => Long;
                mod: (other: string | number | Long) => Long;
                rem: (other: string | number | Long) => Long;
                multiply: (multiplier: string | number | Long) => Long;
                mul: (multiplier: string | number | Long) => Long;
                negate: () => Long;
                neg: () => Long;
                not: () => Long;
                countLeadingZeros: () => number;
                clz: () => number;
                countTrailingZeros: () => number;
                ctz: () => number;
                notEquals: (other: string | number | Long) => boolean;
                neq: (other: string | number | Long) => boolean;
                ne: (other: string | number | Long) => boolean;
                or: (other: string | number | Long) => Long;
                shiftLeft: (numBits: number | Long) => Long;
                shl: (numBits: number | Long) => Long;
                shiftRight: (numBits: number | Long) => Long;
                shr: (numBits: number | Long) => Long;
                shiftRightUnsigned: (numBits: number | Long) => Long;
                shru: (numBits: number | Long) => Long;
                shr_u: (numBits: number | Long) => Long;
                rotateLeft: (numBits: number | Long) => Long;
                rotl: (numBits: number | Long) => Long;
                rotateRight: (numBits: number | Long) => Long;
                rotr: (numBits: number | Long) => Long;
                subtract: (subtrahend: string | number | Long) => Long;
                sub: (subtrahend: string | number | Long) => Long;
                toInt: () => number;
                toNumber: () => number;
                toBytes: (le?: boolean | undefined) => number[];
                toBytesLE: () => number[];
                toBytesBE: () => number[];
                toSigned: () => Long;
                toString: (radix?: number | undefined) => string;
                toUnsigned: () => Long;
                xor: (other: string | number | Long) => Long;
            } & { [K_6 in Exclude<keyof I_1["chainInfoList"][number]["apiCount"], keyof Long>]: never; }) | undefined;
        } & { [K_7 in Exclude<keyof I_1["chainInfoList"][number], keyof ShowAllChainsInfoStruct>]: never; })[] & { [K_8 in Exclude<keyof I_1["chainInfoList"], keyof {
            chainName?: string | undefined;
            chainID?: string | undefined;
            enabledApiInterfaces?: string[] | undefined;
            apiCount?: string | number | Long | undefined;
        }[]>]: never; }) | undefined;
    } & { [K_9 in Exclude<keyof I_1, "chainInfoList">]: never; }>(object: I_1): QueryShowAllChainsResponse;
};
export declare const ShowAllChainsInfoStruct: {
    encode(message: ShowAllChainsInfoStruct, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): ShowAllChainsInfoStruct;
    fromJSON(object: any): ShowAllChainsInfoStruct;
    toJSON(message: ShowAllChainsInfoStruct): unknown;
    create<I extends {
        chainName?: string | undefined;
        chainID?: string | undefined;
        enabledApiInterfaces?: string[] | undefined;
        apiCount?: string | number | Long | undefined;
    } & {
        chainName?: string | undefined;
        chainID?: string | undefined;
        enabledApiInterfaces?: (string[] & string[] & { [K in Exclude<keyof I["enabledApiInterfaces"], keyof string[]>]: never; }) | undefined;
        apiCount?: string | number | (Long & {
            high: number;
            low: number;
            unsigned: boolean;
            add: (addend: string | number | Long) => Long;
            and: (other: string | number | Long) => Long;
            compare: (other: string | number | Long) => number;
            comp: (other: string | number | Long) => number;
            divide: (divisor: string | number | Long) => Long;
            div: (divisor: string | number | Long) => Long;
            equals: (other: string | number | Long) => boolean;
            eq: (other: string | number | Long) => boolean;
            getHighBits: () => number;
            getHighBitsUnsigned: () => number;
            getLowBits: () => number;
            getLowBitsUnsigned: () => number;
            getNumBitsAbs: () => number;
            greaterThan: (other: string | number | Long) => boolean;
            gt: (other: string | number | Long) => boolean;
            greaterThanOrEqual: (other: string | number | Long) => boolean;
            gte: (other: string | number | Long) => boolean;
            ge: (other: string | number | Long) => boolean;
            isEven: () => boolean;
            isNegative: () => boolean;
            isOdd: () => boolean;
            isPositive: () => boolean;
            isZero: () => boolean;
            eqz: () => boolean;
            lessThan: (other: string | number | Long) => boolean;
            lt: (other: string | number | Long) => boolean;
            lessThanOrEqual: (other: string | number | Long) => boolean;
            lte: (other: string | number | Long) => boolean;
            le: (other: string | number | Long) => boolean;
            modulo: (other: string | number | Long) => Long;
            mod: (other: string | number | Long) => Long;
            rem: (other: string | number | Long) => Long;
            multiply: (multiplier: string | number | Long) => Long;
            mul: (multiplier: string | number | Long) => Long;
            negate: () => Long;
            neg: () => Long;
            not: () => Long;
            countLeadingZeros: () => number;
            clz: () => number;
            countTrailingZeros: () => number;
            ctz: () => number;
            notEquals: (other: string | number | Long) => boolean;
            neq: (other: string | number | Long) => boolean;
            ne: (other: string | number | Long) => boolean;
            or: (other: string | number | Long) => Long;
            shiftLeft: (numBits: number | Long) => Long;
            shl: (numBits: number | Long) => Long;
            shiftRight: (numBits: number | Long) => Long;
            shr: (numBits: number | Long) => Long;
            shiftRightUnsigned: (numBits: number | Long) => Long;
            shru: (numBits: number | Long) => Long;
            shr_u: (numBits: number | Long) => Long;
            rotateLeft: (numBits: number | Long) => Long;
            rotl: (numBits: number | Long) => Long;
            rotateRight: (numBits: number | Long) => Long;
            rotr: (numBits: number | Long) => Long;
            subtract: (subtrahend: string | number | Long) => Long;
            sub: (subtrahend: string | number | Long) => Long;
            toInt: () => number;
            toNumber: () => number;
            toBytes: (le?: boolean | undefined) => number[];
            toBytesLE: () => number[];
            toBytesBE: () => number[];
            toSigned: () => Long;
            toString: (radix?: number | undefined) => string;
            toUnsigned: () => Long;
            xor: (other: string | number | Long) => Long;
        } & { [K_1 in Exclude<keyof I["apiCount"], keyof Long>]: never; }) | undefined;
    } & { [K_2 in Exclude<keyof I, keyof ShowAllChainsInfoStruct>]: never; }>(base?: I | undefined): ShowAllChainsInfoStruct;
    fromPartial<I_1 extends {
        chainName?: string | undefined;
        chainID?: string | undefined;
        enabledApiInterfaces?: string[] | undefined;
        apiCount?: string | number | Long | undefined;
    } & {
        chainName?: string | undefined;
        chainID?: string | undefined;
        enabledApiInterfaces?: (string[] & string[] & { [K_3 in Exclude<keyof I_1["enabledApiInterfaces"], keyof string[]>]: never; }) | undefined;
        apiCount?: string | number | (Long & {
            high: number;
            low: number;
            unsigned: boolean;
            add: (addend: string | number | Long) => Long;
            and: (other: string | number | Long) => Long;
            compare: (other: string | number | Long) => number;
            comp: (other: string | number | Long) => number;
            divide: (divisor: string | number | Long) => Long;
            div: (divisor: string | number | Long) => Long;
            equals: (other: string | number | Long) => boolean;
            eq: (other: string | number | Long) => boolean;
            getHighBits: () => number;
            getHighBitsUnsigned: () => number;
            getLowBits: () => number;
            getLowBitsUnsigned: () => number;
            getNumBitsAbs: () => number;
            greaterThan: (other: string | number | Long) => boolean;
            gt: (other: string | number | Long) => boolean;
            greaterThanOrEqual: (other: string | number | Long) => boolean;
            gte: (other: string | number | Long) => boolean;
            ge: (other: string | number | Long) => boolean;
            isEven: () => boolean;
            isNegative: () => boolean;
            isOdd: () => boolean;
            isPositive: () => boolean;
            isZero: () => boolean;
            eqz: () => boolean;
            lessThan: (other: string | number | Long) => boolean;
            lt: (other: string | number | Long) => boolean;
            lessThanOrEqual: (other: string | number | Long) => boolean;
            lte: (other: string | number | Long) => boolean;
            le: (other: string | number | Long) => boolean;
            modulo: (other: string | number | Long) => Long;
            mod: (other: string | number | Long) => Long;
            rem: (other: string | number | Long) => Long;
            multiply: (multiplier: string | number | Long) => Long;
            mul: (multiplier: string | number | Long) => Long;
            negate: () => Long;
            neg: () => Long;
            not: () => Long;
            countLeadingZeros: () => number;
            clz: () => number;
            countTrailingZeros: () => number;
            ctz: () => number;
            notEquals: (other: string | number | Long) => boolean;
            neq: (other: string | number | Long) => boolean;
            ne: (other: string | number | Long) => boolean;
            or: (other: string | number | Long) => Long;
            shiftLeft: (numBits: number | Long) => Long;
            shl: (numBits: number | Long) => Long;
            shiftRight: (numBits: number | Long) => Long;
            shr: (numBits: number | Long) => Long;
            shiftRightUnsigned: (numBits: number | Long) => Long;
            shru: (numBits: number | Long) => Long;
            shr_u: (numBits: number | Long) => Long;
            rotateLeft: (numBits: number | Long) => Long;
            rotl: (numBits: number | Long) => Long;
            rotateRight: (numBits: number | Long) => Long;
            rotr: (numBits: number | Long) => Long;
            subtract: (subtrahend: string | number | Long) => Long;
            sub: (subtrahend: string | number | Long) => Long;
            toInt: () => number;
            toNumber: () => number;
            toBytes: (le?: boolean | undefined) => number[];
            toBytesLE: () => number[];
            toBytesBE: () => number[];
            toSigned: () => Long;
            toString: (radix?: number | undefined) => string;
            toUnsigned: () => Long;
            xor: (other: string | number | Long) => Long;
        } & { [K_4 in Exclude<keyof I_1["apiCount"], keyof Long>]: never; }) | undefined;
    } & { [K_5 in Exclude<keyof I_1, keyof ShowAllChainsInfoStruct>]: never; }>(object: I_1): ShowAllChainsInfoStruct;
};
export declare const QueryShowChainInfoRequest: {
    encode(message: QueryShowChainInfoRequest, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): QueryShowChainInfoRequest;
    fromJSON(object: any): QueryShowChainInfoRequest;
    toJSON(message: QueryShowChainInfoRequest): unknown;
    create<I extends {
        chainName?: string | undefined;
    } & {
        chainName?: string | undefined;
    } & { [K in Exclude<keyof I, "chainName">]: never; }>(base?: I | undefined): QueryShowChainInfoRequest;
    fromPartial<I_1 extends {
        chainName?: string | undefined;
    } & {
        chainName?: string | undefined;
    } & { [K_1 in Exclude<keyof I_1, "chainName">]: never; }>(object: I_1): QueryShowChainInfoRequest;
};
export declare const ApiList: {
    encode(message: ApiList, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): ApiList;
    fromJSON(object: any): ApiList;
    toJSON(message: ApiList): unknown;
    create<I extends {
        interface?: string | undefined;
        supportedApis?: string[] | undefined;
    } & {
        interface?: string | undefined;
        supportedApis?: (string[] & string[] & { [K in Exclude<keyof I["supportedApis"], keyof string[]>]: never; }) | undefined;
    } & { [K_1 in Exclude<keyof I, keyof ApiList>]: never; }>(base?: I | undefined): ApiList;
    fromPartial<I_1 extends {
        interface?: string | undefined;
        supportedApis?: string[] | undefined;
    } & {
        interface?: string | undefined;
        supportedApis?: (string[] & string[] & { [K_2 in Exclude<keyof I_1["supportedApis"], keyof string[]>]: never; }) | undefined;
    } & { [K_3 in Exclude<keyof I_1, keyof ApiList>]: never; }>(object: I_1): ApiList;
};
export declare const QueryShowChainInfoResponse: {
    encode(message: QueryShowChainInfoResponse, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): QueryShowChainInfoResponse;
    fromJSON(object: any): QueryShowChainInfoResponse;
    toJSON(message: QueryShowChainInfoResponse): unknown;
    create<I extends {
        chainID?: string | undefined;
        interfaces?: string[] | undefined;
        supportedApisInterfaceList?: {
            interface?: string | undefined;
            supportedApis?: string[] | undefined;
        }[] | undefined;
    } & {
        chainID?: string | undefined;
        interfaces?: (string[] & string[] & { [K in Exclude<keyof I["interfaces"], keyof string[]>]: never; }) | undefined;
        supportedApisInterfaceList?: ({
            interface?: string | undefined;
            supportedApis?: string[] | undefined;
        }[] & ({
            interface?: string | undefined;
            supportedApis?: string[] | undefined;
        } & {
            interface?: string | undefined;
            supportedApis?: (string[] & string[] & { [K_1 in Exclude<keyof I["supportedApisInterfaceList"][number]["supportedApis"], keyof string[]>]: never; }) | undefined;
        } & { [K_2 in Exclude<keyof I["supportedApisInterfaceList"][number], keyof ApiList>]: never; })[] & { [K_3 in Exclude<keyof I["supportedApisInterfaceList"], keyof {
            interface?: string | undefined;
            supportedApis?: string[] | undefined;
        }[]>]: never; }) | undefined;
    } & { [K_4 in Exclude<keyof I, keyof QueryShowChainInfoResponse>]: never; }>(base?: I | undefined): QueryShowChainInfoResponse;
    fromPartial<I_1 extends {
        chainID?: string | undefined;
        interfaces?: string[] | undefined;
        supportedApisInterfaceList?: {
            interface?: string | undefined;
            supportedApis?: string[] | undefined;
        }[] | undefined;
    } & {
        chainID?: string | undefined;
        interfaces?: (string[] & string[] & { [K_5 in Exclude<keyof I_1["interfaces"], keyof string[]>]: never; }) | undefined;
        supportedApisInterfaceList?: ({
            interface?: string | undefined;
            supportedApis?: string[] | undefined;
        }[] & ({
            interface?: string | undefined;
            supportedApis?: string[] | undefined;
        } & {
            interface?: string | undefined;
            supportedApis?: (string[] & string[] & { [K_6 in Exclude<keyof I_1["supportedApisInterfaceList"][number]["supportedApis"], keyof string[]>]: never; }) | undefined;
        } & { [K_7 in Exclude<keyof I_1["supportedApisInterfaceList"][number], keyof ApiList>]: never; })[] & { [K_8 in Exclude<keyof I_1["supportedApisInterfaceList"], keyof {
            interface?: string | undefined;
            supportedApis?: string[] | undefined;
        }[]>]: never; }) | undefined;
    } & { [K_9 in Exclude<keyof I_1, keyof QueryShowChainInfoResponse>]: never; }>(object: I_1): QueryShowChainInfoResponse;
};
/** Query defines the gRPC querier service. */
export interface Query {
    /** Parameters queries the parameters of the module. */
    Params(request: QueryParamsRequest): Promise<QueryParamsResponse>;
    /** Queries a Spec by id. */
    Spec(request: QueryGetSpecRequest): Promise<QueryGetSpecResponse>;
    /** Queries a list of Spec items. */
    SpecAll(request: QueryAllSpecRequest): Promise<QueryAllSpecResponse>;
    /** Queries a Spec by id (raw form). */
    SpecRaw(request: QueryGetSpecRequest): Promise<QueryGetSpecResponse>;
    /** Queries a list of Spec items (raw form). */
    SpecAllRaw(request: QueryAllSpecRequest): Promise<QueryAllSpecResponse>;
    /** Queries a list of ShowAllChains items. */
    ShowAllChains(request: QueryShowAllChainsRequest): Promise<QueryShowAllChainsResponse>;
    /** Queries a list of ShowChainInfo items. */
    ShowChainInfo(request: QueryShowChainInfoRequest): Promise<QueryShowChainInfoResponse>;
}
export declare class QueryClientImpl implements Query {
    private readonly rpc;
    private readonly service;
    constructor(rpc: Rpc, opts?: {
        service?: string;
    });
    Params(request: QueryParamsRequest): Promise<QueryParamsResponse>;
    Spec(request: QueryGetSpecRequest): Promise<QueryGetSpecResponse>;
    SpecAll(request: QueryAllSpecRequest): Promise<QueryAllSpecResponse>;
    SpecRaw(request: QueryGetSpecRequest): Promise<QueryGetSpecResponse>;
    SpecAllRaw(request: QueryAllSpecRequest): Promise<QueryAllSpecResponse>;
    ShowAllChains(request: QueryShowAllChainsRequest): Promise<QueryShowAllChainsResponse>;
    ShowChainInfo(request: QueryShowChainInfoRequest): Promise<QueryShowChainInfoResponse>;
}
interface Rpc {
    request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}
declare type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Long ? string | number | Long : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
declare type KeysOfUnion<T> = T extends T ? keyof T : never;
export declare type Exact<P, I extends P> = P extends Builtin ? P : P & {
    [K in keyof P]: Exact<P[K], I[K]>;
} & {
    [K in Exclude<keyof I, KeysOfUnion<P>>]: never;
};
export {};

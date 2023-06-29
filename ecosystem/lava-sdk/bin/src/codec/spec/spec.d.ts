import Long from "long";
import _m0 from "protobufjs/minimal";
import { Coin } from "../cosmos/base/v1beta1/coin";
import { ServiceApi } from "./service_api";
export declare const protobufPackage = "lavanet.lava.spec";
export interface Spec {
    index: string;
    name: string;
    imports: string[];
    apis: ServiceApi[];
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
}
export declare enum Spec_ProvidersTypes {
    dynamic = 0,
    static = 1,
    UNRECOGNIZED = -1
}
export declare function spec_ProvidersTypesFromJSON(object: any): Spec_ProvidersTypes;
export declare function spec_ProvidersTypesToJSON(object: Spec_ProvidersTypes): string;
export declare const Spec: {
    encode(message: Spec, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): Spec;
    fromJSON(object: any): Spec;
    toJSON(message: Spec): unknown;
    create<I extends {
        index?: string | undefined;
        name?: string | undefined;
        imports?: string[] | undefined;
        apis?: {
            name?: string | undefined;
            blockParsing?: {
                parserArg?: string[] | undefined;
                parserFunc?: import("./service_api").parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } | undefined;
            computeUnits?: string | number | Long | undefined;
            enabled?: boolean | undefined;
            apiInterfaces?: {
                interface?: string | undefined;
                type?: string | undefined;
                extraComputeUnits?: string | number | Long | undefined;
                category?: {
                    deterministic?: boolean | undefined;
                    local?: boolean | undefined;
                    subscription?: boolean | undefined;
                    stateful?: number | undefined;
                    hangingApi?: boolean | undefined;
                } | undefined;
                overwriteBlockParsing?: {
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } | undefined;
            }[] | undefined;
            reserved?: {
                deterministic?: boolean | undefined;
                local?: boolean | undefined;
                subscription?: boolean | undefined;
                stateful?: number | undefined;
                hangingApi?: boolean | undefined;
            } | undefined;
            parsing?: {
                functionTag?: string | undefined;
                functionTemplate?: string | undefined;
                resultParsing?: {
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } | undefined;
            } | undefined;
            internalPath?: string | undefined;
        }[] | undefined;
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
        providersTypes?: Spec_ProvidersTypes | undefined;
    } & {
        index?: string | undefined;
        name?: string | undefined;
        imports?: (string[] & string[] & { [K in Exclude<keyof I["imports"], keyof string[]>]: never; }) | undefined;
        apis?: ({
            name?: string | undefined;
            blockParsing?: {
                parserArg?: string[] | undefined;
                parserFunc?: import("./service_api").parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } | undefined;
            computeUnits?: string | number | Long | undefined;
            enabled?: boolean | undefined;
            apiInterfaces?: {
                interface?: string | undefined;
                type?: string | undefined;
                extraComputeUnits?: string | number | Long | undefined;
                category?: {
                    deterministic?: boolean | undefined;
                    local?: boolean | undefined;
                    subscription?: boolean | undefined;
                    stateful?: number | undefined;
                    hangingApi?: boolean | undefined;
                } | undefined;
                overwriteBlockParsing?: {
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } | undefined;
            }[] | undefined;
            reserved?: {
                deterministic?: boolean | undefined;
                local?: boolean | undefined;
                subscription?: boolean | undefined;
                stateful?: number | undefined;
                hangingApi?: boolean | undefined;
            } | undefined;
            parsing?: {
                functionTag?: string | undefined;
                functionTemplate?: string | undefined;
                resultParsing?: {
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } | undefined;
            } | undefined;
            internalPath?: string | undefined;
        }[] & ({
            name?: string | undefined;
            blockParsing?: {
                parserArg?: string[] | undefined;
                parserFunc?: import("./service_api").parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } | undefined;
            computeUnits?: string | number | Long | undefined;
            enabled?: boolean | undefined;
            apiInterfaces?: {
                interface?: string | undefined;
                type?: string | undefined;
                extraComputeUnits?: string | number | Long | undefined;
                category?: {
                    deterministic?: boolean | undefined;
                    local?: boolean | undefined;
                    subscription?: boolean | undefined;
                    stateful?: number | undefined;
                    hangingApi?: boolean | undefined;
                } | undefined;
                overwriteBlockParsing?: {
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } | undefined;
            }[] | undefined;
            reserved?: {
                deterministic?: boolean | undefined;
                local?: boolean | undefined;
                subscription?: boolean | undefined;
                stateful?: number | undefined;
                hangingApi?: boolean | undefined;
            } | undefined;
            parsing?: {
                functionTag?: string | undefined;
                functionTemplate?: string | undefined;
                resultParsing?: {
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } | undefined;
            } | undefined;
            internalPath?: string | undefined;
        } & {
            name?: string | undefined;
            blockParsing?: ({
                parserArg?: string[] | undefined;
                parserFunc?: import("./service_api").parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } & {
                parserArg?: (string[] & string[] & { [K_1 in Exclude<keyof I["apis"][number]["blockParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
                parserFunc?: import("./service_api").parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } & { [K_2 in Exclude<keyof I["apis"][number]["blockParsing"], keyof import("./service_api").BlockParser>]: never; }) | undefined;
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
            } & { [K_3 in Exclude<keyof I["apis"][number]["computeUnits"], keyof Long>]: never; }) | undefined;
            enabled?: boolean | undefined;
            apiInterfaces?: ({
                interface?: string | undefined;
                type?: string | undefined;
                extraComputeUnits?: string | number | Long | undefined;
                category?: {
                    deterministic?: boolean | undefined;
                    local?: boolean | undefined;
                    subscription?: boolean | undefined;
                    stateful?: number | undefined;
                    hangingApi?: boolean | undefined;
                } | undefined;
                overwriteBlockParsing?: {
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } | undefined;
            }[] & ({
                interface?: string | undefined;
                type?: string | undefined;
                extraComputeUnits?: string | number | Long | undefined;
                category?: {
                    deterministic?: boolean | undefined;
                    local?: boolean | undefined;
                    subscription?: boolean | undefined;
                    stateful?: number | undefined;
                    hangingApi?: boolean | undefined;
                } | undefined;
                overwriteBlockParsing?: {
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } | undefined;
            } & {
                interface?: string | undefined;
                type?: string | undefined;
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
                } & { [K_4 in Exclude<keyof I["apis"][number]["apiInterfaces"][number]["extraComputeUnits"], keyof Long>]: never; }) | undefined;
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
                } & { [K_5 in Exclude<keyof I["apis"][number]["apiInterfaces"][number]["category"], keyof import("./service_api").SpecCategory>]: never; }) | undefined;
                overwriteBlockParsing?: ({
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } & {
                    parserArg?: (string[] & string[] & { [K_6 in Exclude<keyof I["apis"][number]["apiInterfaces"][number]["overwriteBlockParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } & { [K_7 in Exclude<keyof I["apis"][number]["apiInterfaces"][number]["overwriteBlockParsing"], keyof import("./service_api").BlockParser>]: never; }) | undefined;
            } & { [K_8 in Exclude<keyof I["apis"][number]["apiInterfaces"][number], keyof import("./service_api").ApiInterface>]: never; })[] & { [K_9 in Exclude<keyof I["apis"][number]["apiInterfaces"], keyof {
                interface?: string | undefined;
                type?: string | undefined;
                extraComputeUnits?: string | number | Long | undefined;
                category?: {
                    deterministic?: boolean | undefined;
                    local?: boolean | undefined;
                    subscription?: boolean | undefined;
                    stateful?: number | undefined;
                    hangingApi?: boolean | undefined;
                } | undefined;
                overwriteBlockParsing?: {
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } | undefined;
            }[]>]: never; }) | undefined;
            reserved?: ({
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
            } & { [K_10 in Exclude<keyof I["apis"][number]["reserved"], keyof import("./service_api").SpecCategory>]: never; }) | undefined;
            parsing?: ({
                functionTag?: string | undefined;
                functionTemplate?: string | undefined;
                resultParsing?: {
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } | undefined;
            } & {
                functionTag?: string | undefined;
                functionTemplate?: string | undefined;
                resultParsing?: ({
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } & {
                    parserArg?: (string[] & string[] & { [K_11 in Exclude<keyof I["apis"][number]["parsing"]["resultParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } & { [K_12 in Exclude<keyof I["apis"][number]["parsing"]["resultParsing"], keyof import("./service_api").BlockParser>]: never; }) | undefined;
            } & { [K_13 in Exclude<keyof I["apis"][number]["parsing"], keyof import("./service_api").Parsing>]: never; }) | undefined;
            internalPath?: string | undefined;
        } & { [K_14 in Exclude<keyof I["apis"][number], keyof ServiceApi>]: never; })[] & { [K_15 in Exclude<keyof I["apis"], keyof {
            name?: string | undefined;
            blockParsing?: {
                parserArg?: string[] | undefined;
                parserFunc?: import("./service_api").parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } | undefined;
            computeUnits?: string | number | Long | undefined;
            enabled?: boolean | undefined;
            apiInterfaces?: {
                interface?: string | undefined;
                type?: string | undefined;
                extraComputeUnits?: string | number | Long | undefined;
                category?: {
                    deterministic?: boolean | undefined;
                    local?: boolean | undefined;
                    subscription?: boolean | undefined;
                    stateful?: number | undefined;
                    hangingApi?: boolean | undefined;
                } | undefined;
                overwriteBlockParsing?: {
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } | undefined;
            }[] | undefined;
            reserved?: {
                deterministic?: boolean | undefined;
                local?: boolean | undefined;
                subscription?: boolean | undefined;
                stateful?: number | undefined;
                hangingApi?: boolean | undefined;
            } | undefined;
            parsing?: {
                functionTag?: string | undefined;
                functionTemplate?: string | undefined;
                resultParsing?: {
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } | undefined;
            } | undefined;
            internalPath?: string | undefined;
        }[]>]: never; }) | undefined;
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
        } & { [K_16 in Exclude<keyof I["averageBlockTime"], keyof Long>]: never; }) | undefined;
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
        } & { [K_17 in Exclude<keyof I["allowedBlockLagForQosSync"], keyof Long>]: never; }) | undefined;
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
        } & { [K_18 in Exclude<keyof I["blockLastUpdated"], keyof Long>]: never; }) | undefined;
        minStakeProvider?: ({
            denom?: string | undefined;
            amount?: string | undefined;
        } & {
            denom?: string | undefined;
            amount?: string | undefined;
        } & { [K_19 in Exclude<keyof I["minStakeProvider"], keyof Coin>]: never; }) | undefined;
        minStakeClient?: ({
            denom?: string | undefined;
            amount?: string | undefined;
        } & {
            denom?: string | undefined;
            amount?: string | undefined;
        } & { [K_20 in Exclude<keyof I["minStakeClient"], keyof Coin>]: never; }) | undefined;
        providersTypes?: Spec_ProvidersTypes | undefined;
    } & { [K_21 in Exclude<keyof I, keyof Spec>]: never; }>(base?: I | undefined): Spec;
    fromPartial<I_1 extends {
        index?: string | undefined;
        name?: string | undefined;
        imports?: string[] | undefined;
        apis?: {
            name?: string | undefined;
            blockParsing?: {
                parserArg?: string[] | undefined;
                parserFunc?: import("./service_api").parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } | undefined;
            computeUnits?: string | number | Long | undefined;
            enabled?: boolean | undefined;
            apiInterfaces?: {
                interface?: string | undefined;
                type?: string | undefined;
                extraComputeUnits?: string | number | Long | undefined;
                category?: {
                    deterministic?: boolean | undefined;
                    local?: boolean | undefined;
                    subscription?: boolean | undefined;
                    stateful?: number | undefined;
                    hangingApi?: boolean | undefined;
                } | undefined;
                overwriteBlockParsing?: {
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } | undefined;
            }[] | undefined;
            reserved?: {
                deterministic?: boolean | undefined;
                local?: boolean | undefined;
                subscription?: boolean | undefined;
                stateful?: number | undefined;
                hangingApi?: boolean | undefined;
            } | undefined;
            parsing?: {
                functionTag?: string | undefined;
                functionTemplate?: string | undefined;
                resultParsing?: {
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } | undefined;
            } | undefined;
            internalPath?: string | undefined;
        }[] | undefined;
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
        providersTypes?: Spec_ProvidersTypes | undefined;
    } & {
        index?: string | undefined;
        name?: string | undefined;
        imports?: (string[] & string[] & { [K_22 in Exclude<keyof I_1["imports"], keyof string[]>]: never; }) | undefined;
        apis?: ({
            name?: string | undefined;
            blockParsing?: {
                parserArg?: string[] | undefined;
                parserFunc?: import("./service_api").parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } | undefined;
            computeUnits?: string | number | Long | undefined;
            enabled?: boolean | undefined;
            apiInterfaces?: {
                interface?: string | undefined;
                type?: string | undefined;
                extraComputeUnits?: string | number | Long | undefined;
                category?: {
                    deterministic?: boolean | undefined;
                    local?: boolean | undefined;
                    subscription?: boolean | undefined;
                    stateful?: number | undefined;
                    hangingApi?: boolean | undefined;
                } | undefined;
                overwriteBlockParsing?: {
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } | undefined;
            }[] | undefined;
            reserved?: {
                deterministic?: boolean | undefined;
                local?: boolean | undefined;
                subscription?: boolean | undefined;
                stateful?: number | undefined;
                hangingApi?: boolean | undefined;
            } | undefined;
            parsing?: {
                functionTag?: string | undefined;
                functionTemplate?: string | undefined;
                resultParsing?: {
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } | undefined;
            } | undefined;
            internalPath?: string | undefined;
        }[] & ({
            name?: string | undefined;
            blockParsing?: {
                parserArg?: string[] | undefined;
                parserFunc?: import("./service_api").parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } | undefined;
            computeUnits?: string | number | Long | undefined;
            enabled?: boolean | undefined;
            apiInterfaces?: {
                interface?: string | undefined;
                type?: string | undefined;
                extraComputeUnits?: string | number | Long | undefined;
                category?: {
                    deterministic?: boolean | undefined;
                    local?: boolean | undefined;
                    subscription?: boolean | undefined;
                    stateful?: number | undefined;
                    hangingApi?: boolean | undefined;
                } | undefined;
                overwriteBlockParsing?: {
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } | undefined;
            }[] | undefined;
            reserved?: {
                deterministic?: boolean | undefined;
                local?: boolean | undefined;
                subscription?: boolean | undefined;
                stateful?: number | undefined;
                hangingApi?: boolean | undefined;
            } | undefined;
            parsing?: {
                functionTag?: string | undefined;
                functionTemplate?: string | undefined;
                resultParsing?: {
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } | undefined;
            } | undefined;
            internalPath?: string | undefined;
        } & {
            name?: string | undefined;
            blockParsing?: ({
                parserArg?: string[] | undefined;
                parserFunc?: import("./service_api").parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } & {
                parserArg?: (string[] & string[] & { [K_23 in Exclude<keyof I_1["apis"][number]["blockParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
                parserFunc?: import("./service_api").parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } & { [K_24 in Exclude<keyof I_1["apis"][number]["blockParsing"], keyof import("./service_api").BlockParser>]: never; }) | undefined;
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
            } & { [K_25 in Exclude<keyof I_1["apis"][number]["computeUnits"], keyof Long>]: never; }) | undefined;
            enabled?: boolean | undefined;
            apiInterfaces?: ({
                interface?: string | undefined;
                type?: string | undefined;
                extraComputeUnits?: string | number | Long | undefined;
                category?: {
                    deterministic?: boolean | undefined;
                    local?: boolean | undefined;
                    subscription?: boolean | undefined;
                    stateful?: number | undefined;
                    hangingApi?: boolean | undefined;
                } | undefined;
                overwriteBlockParsing?: {
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } | undefined;
            }[] & ({
                interface?: string | undefined;
                type?: string | undefined;
                extraComputeUnits?: string | number | Long | undefined;
                category?: {
                    deterministic?: boolean | undefined;
                    local?: boolean | undefined;
                    subscription?: boolean | undefined;
                    stateful?: number | undefined;
                    hangingApi?: boolean | undefined;
                } | undefined;
                overwriteBlockParsing?: {
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } | undefined;
            } & {
                interface?: string | undefined;
                type?: string | undefined;
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
                } & { [K_26 in Exclude<keyof I_1["apis"][number]["apiInterfaces"][number]["extraComputeUnits"], keyof Long>]: never; }) | undefined;
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
                } & { [K_27 in Exclude<keyof I_1["apis"][number]["apiInterfaces"][number]["category"], keyof import("./service_api").SpecCategory>]: never; }) | undefined;
                overwriteBlockParsing?: ({
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } & {
                    parserArg?: (string[] & string[] & { [K_28 in Exclude<keyof I_1["apis"][number]["apiInterfaces"][number]["overwriteBlockParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } & { [K_29 in Exclude<keyof I_1["apis"][number]["apiInterfaces"][number]["overwriteBlockParsing"], keyof import("./service_api").BlockParser>]: never; }) | undefined;
            } & { [K_30 in Exclude<keyof I_1["apis"][number]["apiInterfaces"][number], keyof import("./service_api").ApiInterface>]: never; })[] & { [K_31 in Exclude<keyof I_1["apis"][number]["apiInterfaces"], keyof {
                interface?: string | undefined;
                type?: string | undefined;
                extraComputeUnits?: string | number | Long | undefined;
                category?: {
                    deterministic?: boolean | undefined;
                    local?: boolean | undefined;
                    subscription?: boolean | undefined;
                    stateful?: number | undefined;
                    hangingApi?: boolean | undefined;
                } | undefined;
                overwriteBlockParsing?: {
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } | undefined;
            }[]>]: never; }) | undefined;
            reserved?: ({
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
            } & { [K_32 in Exclude<keyof I_1["apis"][number]["reserved"], keyof import("./service_api").SpecCategory>]: never; }) | undefined;
            parsing?: ({
                functionTag?: string | undefined;
                functionTemplate?: string | undefined;
                resultParsing?: {
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } | undefined;
            } & {
                functionTag?: string | undefined;
                functionTemplate?: string | undefined;
                resultParsing?: ({
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } & {
                    parserArg?: (string[] & string[] & { [K_33 in Exclude<keyof I_1["apis"][number]["parsing"]["resultParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } & { [K_34 in Exclude<keyof I_1["apis"][number]["parsing"]["resultParsing"], keyof import("./service_api").BlockParser>]: never; }) | undefined;
            } & { [K_35 in Exclude<keyof I_1["apis"][number]["parsing"], keyof import("./service_api").Parsing>]: never; }) | undefined;
            internalPath?: string | undefined;
        } & { [K_36 in Exclude<keyof I_1["apis"][number], keyof ServiceApi>]: never; })[] & { [K_37 in Exclude<keyof I_1["apis"], keyof {
            name?: string | undefined;
            blockParsing?: {
                parserArg?: string[] | undefined;
                parserFunc?: import("./service_api").parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } | undefined;
            computeUnits?: string | number | Long | undefined;
            enabled?: boolean | undefined;
            apiInterfaces?: {
                interface?: string | undefined;
                type?: string | undefined;
                extraComputeUnits?: string | number | Long | undefined;
                category?: {
                    deterministic?: boolean | undefined;
                    local?: boolean | undefined;
                    subscription?: boolean | undefined;
                    stateful?: number | undefined;
                    hangingApi?: boolean | undefined;
                } | undefined;
                overwriteBlockParsing?: {
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } | undefined;
            }[] | undefined;
            reserved?: {
                deterministic?: boolean | undefined;
                local?: boolean | undefined;
                subscription?: boolean | undefined;
                stateful?: number | undefined;
                hangingApi?: boolean | undefined;
            } | undefined;
            parsing?: {
                functionTag?: string | undefined;
                functionTemplate?: string | undefined;
                resultParsing?: {
                    parserArg?: string[] | undefined;
                    parserFunc?: import("./service_api").parserFunc | undefined;
                    defaultValue?: string | undefined;
                    encoding?: string | undefined;
                } | undefined;
            } | undefined;
            internalPath?: string | undefined;
        }[]>]: never; }) | undefined;
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
        } & { [K_38 in Exclude<keyof I_1["averageBlockTime"], keyof Long>]: never; }) | undefined;
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
        } & { [K_39 in Exclude<keyof I_1["allowedBlockLagForQosSync"], keyof Long>]: never; }) | undefined;
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
        } & { [K_40 in Exclude<keyof I_1["blockLastUpdated"], keyof Long>]: never; }) | undefined;
        minStakeProvider?: ({
            denom?: string | undefined;
            amount?: string | undefined;
        } & {
            denom?: string | undefined;
            amount?: string | undefined;
        } & { [K_41 in Exclude<keyof I_1["minStakeProvider"], keyof Coin>]: never; }) | undefined;
        minStakeClient?: ({
            denom?: string | undefined;
            amount?: string | undefined;
        } & {
            denom?: string | undefined;
            amount?: string | undefined;
        } & { [K_42 in Exclude<keyof I_1["minStakeClient"], keyof Coin>]: never; }) | undefined;
        providersTypes?: Spec_ProvidersTypes | undefined;
    } & { [K_43 in Exclude<keyof I_1, keyof Spec>]: never; }>(object: I_1): Spec;
};
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

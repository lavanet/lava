import Long from "long";
import _m0 from "protobufjs/minimal";
export declare const protobufPackage = "lavanet.lava.spec";
export declare enum parserFunc {
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
    UNRECOGNIZED = -1
}
export declare function parserFuncFromJSON(object: any): parserFunc;
export declare function parserFuncToJSON(object: parserFunc): string;
export interface ServiceApi {
    name: string;
    blockParsing?: BlockParser;
    computeUnits: Long;
    enabled: boolean;
    apiInterfaces: ApiInterface[];
    reserved?: SpecCategory;
    parsing?: Parsing;
    internalPath: string;
}
export interface Parsing {
    functionTag: string;
    functionTemplate: string;
    resultParsing?: BlockParser;
}
export interface ApiInterface {
    interface: string;
    type: string;
    extraComputeUnits: Long;
    category?: SpecCategory;
    overwriteBlockParsing?: BlockParser;
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
export declare const ServiceApi: {
    encode(message: ServiceApi, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): ServiceApi;
    fromJSON(object: any): ServiceApi;
    toJSON(message: ServiceApi): unknown;
    create<I extends {
        name?: string | undefined;
        blockParsing?: {
            parserArg?: string[] | undefined;
            parserFunc?: parserFunc | undefined;
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
                parserFunc?: parserFunc | undefined;
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
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } | undefined;
        } | undefined;
        internalPath?: string | undefined;
    } & {
        name?: string | undefined;
        blockParsing?: ({
            parserArg?: string[] | undefined;
            parserFunc?: parserFunc | undefined;
            defaultValue?: string | undefined;
            encoding?: string | undefined;
        } & {
            parserArg?: (string[] & string[] & { [K in Exclude<keyof I["blockParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
            parserFunc?: parserFunc | undefined;
            defaultValue?: string | undefined;
            encoding?: string | undefined;
        } & { [K_1 in Exclude<keyof I["blockParsing"], keyof BlockParser>]: never; }) | undefined;
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
        } & { [K_2 in Exclude<keyof I["computeUnits"], keyof Long>]: never; }) | undefined;
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
                parserFunc?: parserFunc | undefined;
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
                parserFunc?: parserFunc | undefined;
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
            } & { [K_3 in Exclude<keyof I["apiInterfaces"][number]["extraComputeUnits"], keyof Long>]: never; }) | undefined;
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
            } & { [K_4 in Exclude<keyof I["apiInterfaces"][number]["category"], keyof SpecCategory>]: never; }) | undefined;
            overwriteBlockParsing?: ({
                parserArg?: string[] | undefined;
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } & {
                parserArg?: (string[] & string[] & { [K_5 in Exclude<keyof I["apiInterfaces"][number]["overwriteBlockParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } & { [K_6 in Exclude<keyof I["apiInterfaces"][number]["overwriteBlockParsing"], keyof BlockParser>]: never; }) | undefined;
        } & { [K_7 in Exclude<keyof I["apiInterfaces"][number], keyof ApiInterface>]: never; })[] & { [K_8 in Exclude<keyof I["apiInterfaces"], keyof {
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
                parserFunc?: parserFunc | undefined;
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
        } & { [K_9 in Exclude<keyof I["reserved"], keyof SpecCategory>]: never; }) | undefined;
        parsing?: ({
            functionTag?: string | undefined;
            functionTemplate?: string | undefined;
            resultParsing?: {
                parserArg?: string[] | undefined;
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } | undefined;
        } & {
            functionTag?: string | undefined;
            functionTemplate?: string | undefined;
            resultParsing?: ({
                parserArg?: string[] | undefined;
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } & {
                parserArg?: (string[] & string[] & { [K_10 in Exclude<keyof I["parsing"]["resultParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } & { [K_11 in Exclude<keyof I["parsing"]["resultParsing"], keyof BlockParser>]: never; }) | undefined;
        } & { [K_12 in Exclude<keyof I["parsing"], keyof Parsing>]: never; }) | undefined;
        internalPath?: string | undefined;
    } & { [K_13 in Exclude<keyof I, keyof ServiceApi>]: never; }>(base?: I | undefined): ServiceApi;
    fromPartial<I_1 extends {
        name?: string | undefined;
        blockParsing?: {
            parserArg?: string[] | undefined;
            parserFunc?: parserFunc | undefined;
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
                parserFunc?: parserFunc | undefined;
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
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } | undefined;
        } | undefined;
        internalPath?: string | undefined;
    } & {
        name?: string | undefined;
        blockParsing?: ({
            parserArg?: string[] | undefined;
            parserFunc?: parserFunc | undefined;
            defaultValue?: string | undefined;
            encoding?: string | undefined;
        } & {
            parserArg?: (string[] & string[] & { [K_14 in Exclude<keyof I_1["blockParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
            parserFunc?: parserFunc | undefined;
            defaultValue?: string | undefined;
            encoding?: string | undefined;
        } & { [K_15 in Exclude<keyof I_1["blockParsing"], keyof BlockParser>]: never; }) | undefined;
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
        } & { [K_16 in Exclude<keyof I_1["computeUnits"], keyof Long>]: never; }) | undefined;
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
                parserFunc?: parserFunc | undefined;
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
                parserFunc?: parserFunc | undefined;
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
            } & { [K_17 in Exclude<keyof I_1["apiInterfaces"][number]["extraComputeUnits"], keyof Long>]: never; }) | undefined;
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
            } & { [K_18 in Exclude<keyof I_1["apiInterfaces"][number]["category"], keyof SpecCategory>]: never; }) | undefined;
            overwriteBlockParsing?: ({
                parserArg?: string[] | undefined;
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } & {
                parserArg?: (string[] & string[] & { [K_19 in Exclude<keyof I_1["apiInterfaces"][number]["overwriteBlockParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } & { [K_20 in Exclude<keyof I_1["apiInterfaces"][number]["overwriteBlockParsing"], keyof BlockParser>]: never; }) | undefined;
        } & { [K_21 in Exclude<keyof I_1["apiInterfaces"][number], keyof ApiInterface>]: never; })[] & { [K_22 in Exclude<keyof I_1["apiInterfaces"], keyof {
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
                parserFunc?: parserFunc | undefined;
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
        } & { [K_23 in Exclude<keyof I_1["reserved"], keyof SpecCategory>]: never; }) | undefined;
        parsing?: ({
            functionTag?: string | undefined;
            functionTemplate?: string | undefined;
            resultParsing?: {
                parserArg?: string[] | undefined;
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } | undefined;
        } & {
            functionTag?: string | undefined;
            functionTemplate?: string | undefined;
            resultParsing?: ({
                parserArg?: string[] | undefined;
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } & {
                parserArg?: (string[] & string[] & { [K_24 in Exclude<keyof I_1["parsing"]["resultParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } & { [K_25 in Exclude<keyof I_1["parsing"]["resultParsing"], keyof BlockParser>]: never; }) | undefined;
        } & { [K_26 in Exclude<keyof I_1["parsing"], keyof Parsing>]: never; }) | undefined;
        internalPath?: string | undefined;
    } & { [K_27 in Exclude<keyof I_1, keyof ServiceApi>]: never; }>(object: I_1): ServiceApi;
};
export declare const Parsing: {
    encode(message: Parsing, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): Parsing;
    fromJSON(object: any): Parsing;
    toJSON(message: Parsing): unknown;
    create<I extends {
        functionTag?: string | undefined;
        functionTemplate?: string | undefined;
        resultParsing?: {
            parserArg?: string[] | undefined;
            parserFunc?: parserFunc | undefined;
            defaultValue?: string | undefined;
            encoding?: string | undefined;
        } | undefined;
    } & {
        functionTag?: string | undefined;
        functionTemplate?: string | undefined;
        resultParsing?: ({
            parserArg?: string[] | undefined;
            parserFunc?: parserFunc | undefined;
            defaultValue?: string | undefined;
            encoding?: string | undefined;
        } & {
            parserArg?: (string[] & string[] & { [K in Exclude<keyof I["resultParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
            parserFunc?: parserFunc | undefined;
            defaultValue?: string | undefined;
            encoding?: string | undefined;
        } & { [K_1 in Exclude<keyof I["resultParsing"], keyof BlockParser>]: never; }) | undefined;
    } & { [K_2 in Exclude<keyof I, keyof Parsing>]: never; }>(base?: I | undefined): Parsing;
    fromPartial<I_1 extends {
        functionTag?: string | undefined;
        functionTemplate?: string | undefined;
        resultParsing?: {
            parserArg?: string[] | undefined;
            parserFunc?: parserFunc | undefined;
            defaultValue?: string | undefined;
            encoding?: string | undefined;
        } | undefined;
    } & {
        functionTag?: string | undefined;
        functionTemplate?: string | undefined;
        resultParsing?: ({
            parserArg?: string[] | undefined;
            parserFunc?: parserFunc | undefined;
            defaultValue?: string | undefined;
            encoding?: string | undefined;
        } & {
            parserArg?: (string[] & string[] & { [K_3 in Exclude<keyof I_1["resultParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
            parserFunc?: parserFunc | undefined;
            defaultValue?: string | undefined;
            encoding?: string | undefined;
        } & { [K_4 in Exclude<keyof I_1["resultParsing"], keyof BlockParser>]: never; }) | undefined;
    } & { [K_5 in Exclude<keyof I_1, keyof Parsing>]: never; }>(object: I_1): Parsing;
};
export declare const ApiInterface: {
    encode(message: ApiInterface, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): ApiInterface;
    fromJSON(object: any): ApiInterface;
    toJSON(message: ApiInterface): unknown;
    create<I extends {
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
            parserFunc?: parserFunc | undefined;
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
        } & { [K in Exclude<keyof I["extraComputeUnits"], keyof Long>]: never; }) | undefined;
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
        } & { [K_1 in Exclude<keyof I["category"], keyof SpecCategory>]: never; }) | undefined;
        overwriteBlockParsing?: ({
            parserArg?: string[] | undefined;
            parserFunc?: parserFunc | undefined;
            defaultValue?: string | undefined;
            encoding?: string | undefined;
        } & {
            parserArg?: (string[] & string[] & { [K_2 in Exclude<keyof I["overwriteBlockParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
            parserFunc?: parserFunc | undefined;
            defaultValue?: string | undefined;
            encoding?: string | undefined;
        } & { [K_3 in Exclude<keyof I["overwriteBlockParsing"], keyof BlockParser>]: never; }) | undefined;
    } & { [K_4 in Exclude<keyof I, keyof ApiInterface>]: never; }>(base?: I | undefined): ApiInterface;
    fromPartial<I_1 extends {
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
            parserFunc?: parserFunc | undefined;
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
        } & { [K_5 in Exclude<keyof I_1["extraComputeUnits"], keyof Long>]: never; }) | undefined;
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
        } & { [K_6 in Exclude<keyof I_1["category"], keyof SpecCategory>]: never; }) | undefined;
        overwriteBlockParsing?: ({
            parserArg?: string[] | undefined;
            parserFunc?: parserFunc | undefined;
            defaultValue?: string | undefined;
            encoding?: string | undefined;
        } & {
            parserArg?: (string[] & string[] & { [K_7 in Exclude<keyof I_1["overwriteBlockParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
            parserFunc?: parserFunc | undefined;
            defaultValue?: string | undefined;
            encoding?: string | undefined;
        } & { [K_8 in Exclude<keyof I_1["overwriteBlockParsing"], keyof BlockParser>]: never; }) | undefined;
    } & { [K_9 in Exclude<keyof I_1, keyof ApiInterface>]: never; }>(object: I_1): ApiInterface;
};
export declare const BlockParser: {
    encode(message: BlockParser, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): BlockParser;
    fromJSON(object: any): BlockParser;
    toJSON(message: BlockParser): unknown;
    create<I extends {
        parserArg?: string[] | undefined;
        parserFunc?: parserFunc | undefined;
        defaultValue?: string | undefined;
        encoding?: string | undefined;
    } & {
        parserArg?: (string[] & string[] & { [K in Exclude<keyof I["parserArg"], keyof string[]>]: never; }) | undefined;
        parserFunc?: parserFunc | undefined;
        defaultValue?: string | undefined;
        encoding?: string | undefined;
    } & { [K_1 in Exclude<keyof I, keyof BlockParser>]: never; }>(base?: I | undefined): BlockParser;
    fromPartial<I_1 extends {
        parserArg?: string[] | undefined;
        parserFunc?: parserFunc | undefined;
        defaultValue?: string | undefined;
        encoding?: string | undefined;
    } & {
        parserArg?: (string[] & string[] & { [K_2 in Exclude<keyof I_1["parserArg"], keyof string[]>]: never; }) | undefined;
        parserFunc?: parserFunc | undefined;
        defaultValue?: string | undefined;
        encoding?: string | undefined;
    } & { [K_3 in Exclude<keyof I_1, keyof BlockParser>]: never; }>(object: I_1): BlockParser;
};
export declare const SpecCategory: {
    encode(message: SpecCategory, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): SpecCategory;
    fromJSON(object: any): SpecCategory;
    toJSON(message: SpecCategory): unknown;
    create<I extends {
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
    } & { [K in Exclude<keyof I, keyof SpecCategory>]: never; }>(base?: I | undefined): SpecCategory;
    fromPartial<I_1 extends {
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
    } & { [K_1 in Exclude<keyof I_1, keyof SpecCategory>]: never; }>(object: I_1): SpecCategory;
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

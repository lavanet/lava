import Long from "long";
import _m0 from "protobufjs/minimal";
export declare const protobufPackage = "lavanet.lava.spec";
export declare enum functionTag {
    DISABLED = 0,
    GET_BLOCKNUM = 1,
    GET_BLOCK_BY_NUM = 2,
    SET_LATEST_IN_METADATA = 3,
    SET_LATEST_IN_BODY = 4,
    VERIFICATION = 5,
    UNRECOGNIZED = -1
}
export declare function functionTagFromJSON(object: any): functionTag;
export declare function functionTagToJSON(object: functionTag): string;
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
export interface ApiCollection {
    enabled: boolean;
    collectionData?: CollectionData;
    apis: Api[];
    headers: Header[];
    /** by collectionKey */
    inheritanceApis: CollectionData[];
    parseDirectives: ParseDirective[];
}
export interface CollectionData {
    apiInterface: string;
    internalPath: string;
    type: string;
    addOn: string;
}
export interface Header {
    name: string;
    kind: Header_HeaderType;
    functionTag: functionTag;
}
export declare enum Header_HeaderType {
    pass_send = 0,
    pass_reply = 1,
    pass_both = 2,
    /** pass_ignore - allows it to pass around but is not signed */
    pass_ignore = 3,
    UNRECOGNIZED = -1
}
export declare function header_HeaderTypeFromJSON(object: any): Header_HeaderType;
export declare function header_HeaderTypeToJSON(object: Header_HeaderType): string;
export interface Api {
    enabled: boolean;
    name: string;
    computeUnits: Long;
    extraComputeUnits: Long;
    category?: SpecCategory;
    blockParsing?: BlockParser;
}
export interface ParseDirective {
    functionTag: functionTag;
    functionTemplate: string;
    resultParsing?: BlockParser;
    apiName: string;
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
export declare const ApiCollection: {
    encode(message: ApiCollection, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): ApiCollection;
    fromJSON(object: any): ApiCollection;
    toJSON(message: ApiCollection): unknown;
    create<I extends {
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
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } | undefined;
        }[] | undefined;
        headers?: {
            name?: string | undefined;
            kind?: Header_HeaderType | undefined;
            functionTag?: functionTag | undefined;
        }[] | undefined;
        inheritanceApis?: {
            apiInterface?: string | undefined;
            internalPath?: string | undefined;
            type?: string | undefined;
            addOn?: string | undefined;
        }[] | undefined;
        parseDirectives?: {
            functionTag?: functionTag | undefined;
            functionTemplate?: string | undefined;
            resultParsing?: {
                parserArg?: string[] | undefined;
                parserFunc?: parserFunc | undefined;
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
        } & { [K in Exclude<keyof I["collectionData"], keyof CollectionData>]: never; }) | undefined;
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
                parserFunc?: parserFunc | undefined;
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
                parserFunc?: parserFunc | undefined;
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
            } & { [K_1 in Exclude<keyof I["apis"][number]["computeUnits"], keyof Long>]: never; }) | undefined;
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
            } & { [K_2 in Exclude<keyof I["apis"][number]["extraComputeUnits"], keyof Long>]: never; }) | undefined;
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
            } & { [K_3 in Exclude<keyof I["apis"][number]["category"], keyof SpecCategory>]: never; }) | undefined;
            blockParsing?: ({
                parserArg?: string[] | undefined;
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } & {
                parserArg?: (string[] & string[] & { [K_4 in Exclude<keyof I["apis"][number]["blockParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } & { [K_5 in Exclude<keyof I["apis"][number]["blockParsing"], keyof BlockParser>]: never; }) | undefined;
        } & { [K_6 in Exclude<keyof I["apis"][number], keyof Api>]: never; })[] & { [K_7 in Exclude<keyof I["apis"], keyof {
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
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } | undefined;
        }[]>]: never; }) | undefined;
        headers?: ({
            name?: string | undefined;
            kind?: Header_HeaderType | undefined;
            functionTag?: functionTag | undefined;
        }[] & ({
            name?: string | undefined;
            kind?: Header_HeaderType | undefined;
            functionTag?: functionTag | undefined;
        } & {
            name?: string | undefined;
            kind?: Header_HeaderType | undefined;
            functionTag?: functionTag | undefined;
        } & { [K_8 in Exclude<keyof I["headers"][number], keyof Header>]: never; })[] & { [K_9 in Exclude<keyof I["headers"], keyof {
            name?: string | undefined;
            kind?: Header_HeaderType | undefined;
            functionTag?: functionTag | undefined;
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
        } & { [K_10 in Exclude<keyof I["inheritanceApis"][number], keyof CollectionData>]: never; })[] & { [K_11 in Exclude<keyof I["inheritanceApis"], keyof {
            apiInterface?: string | undefined;
            internalPath?: string | undefined;
            type?: string | undefined;
            addOn?: string | undefined;
        }[]>]: never; }) | undefined;
        parseDirectives?: ({
            functionTag?: functionTag | undefined;
            functionTemplate?: string | undefined;
            resultParsing?: {
                parserArg?: string[] | undefined;
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } | undefined;
            apiName?: string | undefined;
        }[] & ({
            functionTag?: functionTag | undefined;
            functionTemplate?: string | undefined;
            resultParsing?: {
                parserArg?: string[] | undefined;
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } | undefined;
            apiName?: string | undefined;
        } & {
            functionTag?: functionTag | undefined;
            functionTemplate?: string | undefined;
            resultParsing?: ({
                parserArg?: string[] | undefined;
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } & {
                parserArg?: (string[] & string[] & { [K_12 in Exclude<keyof I["parseDirectives"][number]["resultParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } & { [K_13 in Exclude<keyof I["parseDirectives"][number]["resultParsing"], keyof BlockParser>]: never; }) | undefined;
            apiName?: string | undefined;
        } & { [K_14 in Exclude<keyof I["parseDirectives"][number], keyof ParseDirective>]: never; })[] & { [K_15 in Exclude<keyof I["parseDirectives"], keyof {
            functionTag?: functionTag | undefined;
            functionTemplate?: string | undefined;
            resultParsing?: {
                parserArg?: string[] | undefined;
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } | undefined;
            apiName?: string | undefined;
        }[]>]: never; }) | undefined;
    } & { [K_16 in Exclude<keyof I, keyof ApiCollection>]: never; }>(base?: I | undefined): ApiCollection;
    fromPartial<I_1 extends {
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
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } | undefined;
        }[] | undefined;
        headers?: {
            name?: string | undefined;
            kind?: Header_HeaderType | undefined;
            functionTag?: functionTag | undefined;
        }[] | undefined;
        inheritanceApis?: {
            apiInterface?: string | undefined;
            internalPath?: string | undefined;
            type?: string | undefined;
            addOn?: string | undefined;
        }[] | undefined;
        parseDirectives?: {
            functionTag?: functionTag | undefined;
            functionTemplate?: string | undefined;
            resultParsing?: {
                parserArg?: string[] | undefined;
                parserFunc?: parserFunc | undefined;
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
        } & { [K_17 in Exclude<keyof I_1["collectionData"], keyof CollectionData>]: never; }) | undefined;
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
                parserFunc?: parserFunc | undefined;
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
                parserFunc?: parserFunc | undefined;
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
            } & { [K_18 in Exclude<keyof I_1["apis"][number]["computeUnits"], keyof Long>]: never; }) | undefined;
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
            } & { [K_19 in Exclude<keyof I_1["apis"][number]["extraComputeUnits"], keyof Long>]: never; }) | undefined;
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
            } & { [K_20 in Exclude<keyof I_1["apis"][number]["category"], keyof SpecCategory>]: never; }) | undefined;
            blockParsing?: ({
                parserArg?: string[] | undefined;
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } & {
                parserArg?: (string[] & string[] & { [K_21 in Exclude<keyof I_1["apis"][number]["blockParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } & { [K_22 in Exclude<keyof I_1["apis"][number]["blockParsing"], keyof BlockParser>]: never; }) | undefined;
        } & { [K_23 in Exclude<keyof I_1["apis"][number], keyof Api>]: never; })[] & { [K_24 in Exclude<keyof I_1["apis"], keyof {
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
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } | undefined;
        }[]>]: never; }) | undefined;
        headers?: ({
            name?: string | undefined;
            kind?: Header_HeaderType | undefined;
            functionTag?: functionTag | undefined;
        }[] & ({
            name?: string | undefined;
            kind?: Header_HeaderType | undefined;
            functionTag?: functionTag | undefined;
        } & {
            name?: string | undefined;
            kind?: Header_HeaderType | undefined;
            functionTag?: functionTag | undefined;
        } & { [K_25 in Exclude<keyof I_1["headers"][number], keyof Header>]: never; })[] & { [K_26 in Exclude<keyof I_1["headers"], keyof {
            name?: string | undefined;
            kind?: Header_HeaderType | undefined;
            functionTag?: functionTag | undefined;
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
        } & { [K_27 in Exclude<keyof I_1["inheritanceApis"][number], keyof CollectionData>]: never; })[] & { [K_28 in Exclude<keyof I_1["inheritanceApis"], keyof {
            apiInterface?: string | undefined;
            internalPath?: string | undefined;
            type?: string | undefined;
            addOn?: string | undefined;
        }[]>]: never; }) | undefined;
        parseDirectives?: ({
            functionTag?: functionTag | undefined;
            functionTemplate?: string | undefined;
            resultParsing?: {
                parserArg?: string[] | undefined;
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } | undefined;
            apiName?: string | undefined;
        }[] & ({
            functionTag?: functionTag | undefined;
            functionTemplate?: string | undefined;
            resultParsing?: {
                parserArg?: string[] | undefined;
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } | undefined;
            apiName?: string | undefined;
        } & {
            functionTag?: functionTag | undefined;
            functionTemplate?: string | undefined;
            resultParsing?: ({
                parserArg?: string[] | undefined;
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } & {
                parserArg?: (string[] & string[] & { [K_29 in Exclude<keyof I_1["parseDirectives"][number]["resultParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } & { [K_30 in Exclude<keyof I_1["parseDirectives"][number]["resultParsing"], keyof BlockParser>]: never; }) | undefined;
            apiName?: string | undefined;
        } & { [K_31 in Exclude<keyof I_1["parseDirectives"][number], keyof ParseDirective>]: never; })[] & { [K_32 in Exclude<keyof I_1["parseDirectives"], keyof {
            functionTag?: functionTag | undefined;
            functionTemplate?: string | undefined;
            resultParsing?: {
                parserArg?: string[] | undefined;
                parserFunc?: parserFunc | undefined;
                defaultValue?: string | undefined;
                encoding?: string | undefined;
            } | undefined;
            apiName?: string | undefined;
        }[]>]: never; }) | undefined;
    } & { [K_33 in Exclude<keyof I_1, keyof ApiCollection>]: never; }>(object: I_1): ApiCollection;
};
export declare const CollectionData: {
    encode(message: CollectionData, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): CollectionData;
    fromJSON(object: any): CollectionData;
    toJSON(message: CollectionData): unknown;
    create<I extends {
        apiInterface?: string | undefined;
        internalPath?: string | undefined;
        type?: string | undefined;
        addOn?: string | undefined;
    } & {
        apiInterface?: string | undefined;
        internalPath?: string | undefined;
        type?: string | undefined;
        addOn?: string | undefined;
    } & { [K in Exclude<keyof I, keyof CollectionData>]: never; }>(base?: I | undefined): CollectionData;
    fromPartial<I_1 extends {
        apiInterface?: string | undefined;
        internalPath?: string | undefined;
        type?: string | undefined;
        addOn?: string | undefined;
    } & {
        apiInterface?: string | undefined;
        internalPath?: string | undefined;
        type?: string | undefined;
        addOn?: string | undefined;
    } & { [K_1 in Exclude<keyof I_1, keyof CollectionData>]: never; }>(object: I_1): CollectionData;
};
export declare const Header: {
    encode(message: Header, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): Header;
    fromJSON(object: any): Header;
    toJSON(message: Header): unknown;
    create<I extends {
        name?: string | undefined;
        kind?: Header_HeaderType | undefined;
        functionTag?: functionTag | undefined;
    } & {
        name?: string | undefined;
        kind?: Header_HeaderType | undefined;
        functionTag?: functionTag | undefined;
    } & { [K in Exclude<keyof I, keyof Header>]: never; }>(base?: I | undefined): Header;
    fromPartial<I_1 extends {
        name?: string | undefined;
        kind?: Header_HeaderType | undefined;
        functionTag?: functionTag | undefined;
    } & {
        name?: string | undefined;
        kind?: Header_HeaderType | undefined;
        functionTag?: functionTag | undefined;
    } & { [K_1 in Exclude<keyof I_1, keyof Header>]: never; }>(object: I_1): Header;
};
export declare const Api: {
    encode(message: Api, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): Api;
    fromJSON(object: any): Api;
    toJSON(message: Api): unknown;
    create<I extends {
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
            parserFunc?: parserFunc | undefined;
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
        } & { [K in Exclude<keyof I["computeUnits"], keyof Long>]: never; }) | undefined;
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
        } & { [K_1 in Exclude<keyof I["extraComputeUnits"], keyof Long>]: never; }) | undefined;
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
        } & { [K_2 in Exclude<keyof I["category"], keyof SpecCategory>]: never; }) | undefined;
        blockParsing?: ({
            parserArg?: string[] | undefined;
            parserFunc?: parserFunc | undefined;
            defaultValue?: string | undefined;
            encoding?: string | undefined;
        } & {
            parserArg?: (string[] & string[] & { [K_3 in Exclude<keyof I["blockParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
            parserFunc?: parserFunc | undefined;
            defaultValue?: string | undefined;
            encoding?: string | undefined;
        } & { [K_4 in Exclude<keyof I["blockParsing"], keyof BlockParser>]: never; }) | undefined;
    } & { [K_5 in Exclude<keyof I, keyof Api>]: never; }>(base?: I | undefined): Api;
    fromPartial<I_1 extends {
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
            parserFunc?: parserFunc | undefined;
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
        } & { [K_6 in Exclude<keyof I_1["computeUnits"], keyof Long>]: never; }) | undefined;
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
        } & { [K_7 in Exclude<keyof I_1["extraComputeUnits"], keyof Long>]: never; }) | undefined;
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
        } & { [K_8 in Exclude<keyof I_1["category"], keyof SpecCategory>]: never; }) | undefined;
        blockParsing?: ({
            parserArg?: string[] | undefined;
            parserFunc?: parserFunc | undefined;
            defaultValue?: string | undefined;
            encoding?: string | undefined;
        } & {
            parserArg?: (string[] & string[] & { [K_9 in Exclude<keyof I_1["blockParsing"]["parserArg"], keyof string[]>]: never; }) | undefined;
            parserFunc?: parserFunc | undefined;
            defaultValue?: string | undefined;
            encoding?: string | undefined;
        } & { [K_10 in Exclude<keyof I_1["blockParsing"], keyof BlockParser>]: never; }) | undefined;
    } & { [K_11 in Exclude<keyof I_1, keyof Api>]: never; }>(object: I_1): Api;
};
export declare const ParseDirective: {
    encode(message: ParseDirective, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): ParseDirective;
    fromJSON(object: any): ParseDirective;
    toJSON(message: ParseDirective): unknown;
    create<I extends {
        functionTag?: functionTag | undefined;
        functionTemplate?: string | undefined;
        resultParsing?: {
            parserArg?: string[] | undefined;
            parserFunc?: parserFunc | undefined;
            defaultValue?: string | undefined;
            encoding?: string | undefined;
        } | undefined;
        apiName?: string | undefined;
    } & {
        functionTag?: functionTag | undefined;
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
        apiName?: string | undefined;
    } & { [K_2 in Exclude<keyof I, keyof ParseDirective>]: never; }>(base?: I | undefined): ParseDirective;
    fromPartial<I_1 extends {
        functionTag?: functionTag | undefined;
        functionTemplate?: string | undefined;
        resultParsing?: {
            parserArg?: string[] | undefined;
            parserFunc?: parserFunc | undefined;
            defaultValue?: string | undefined;
            encoding?: string | undefined;
        } | undefined;
        apiName?: string | undefined;
    } & {
        functionTag?: functionTag | undefined;
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
        apiName?: string | undefined;
    } & { [K_5 in Exclude<keyof I_1, keyof ParseDirective>]: never; }>(object: I_1): ParseDirective;
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

import Long from "long";
import _m0 from "protobufjs/minimal";
export declare const protobufPackage = "lavanet.lava.pairing";
export interface EpochPayments {
    index: string;
    providerPaymentStorageKeys: string[];
}
export declare const EpochPayments: {
    encode(message: EpochPayments, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): EpochPayments;
    fromJSON(object: any): EpochPayments;
    toJSON(message: EpochPayments): unknown;
    create<I extends {
        index?: string | undefined;
        providerPaymentStorageKeys?: string[] | undefined;
    } & {
        index?: string | undefined;
        providerPaymentStorageKeys?: (string[] & string[] & { [K in Exclude<keyof I["providerPaymentStorageKeys"], keyof string[]>]: never; }) | undefined;
    } & { [K_1 in Exclude<keyof I, keyof EpochPayments>]: never; }>(base?: I | undefined): EpochPayments;
    fromPartial<I_1 extends {
        index?: string | undefined;
        providerPaymentStorageKeys?: string[] | undefined;
    } & {
        index?: string | undefined;
        providerPaymentStorageKeys?: (string[] & string[] & { [K_2 in Exclude<keyof I_1["providerPaymentStorageKeys"], keyof string[]>]: never; }) | undefined;
    } & { [K_3 in Exclude<keyof I_1, keyof EpochPayments>]: never; }>(object: I_1): EpochPayments;
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

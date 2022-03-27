import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "lavanet.lava.servicer";
export interface UniquePaymentStorageUserServicer {
    index: string;
    block: number;
}
export declare const UniquePaymentStorageUserServicer: {
    encode(message: UniquePaymentStorageUserServicer, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): UniquePaymentStorageUserServicer;
    fromJSON(object: any): UniquePaymentStorageUserServicer;
    toJSON(message: UniquePaymentStorageUserServicer): unknown;
    fromPartial(object: DeepPartial<UniquePaymentStorageUserServicer>): UniquePaymentStorageUserServicer;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};

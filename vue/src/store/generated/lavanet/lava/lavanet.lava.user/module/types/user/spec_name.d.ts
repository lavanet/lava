import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "lavanet.lava.user";
export interface SpecName {
    name: string;
}
export declare const SpecName: {
    encode(message: SpecName, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): SpecName;
    fromJSON(object: any): SpecName;
    toJSON(message: SpecName): unknown;
    fromPartial(object: DeepPartial<SpecName>): SpecName;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};

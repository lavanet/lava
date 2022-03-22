import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "lavanet.lava.user";
export interface BlockNum {
    num: number;
}
export declare const BlockNum: {
    encode(message: BlockNum, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): BlockNum;
    fromJSON(object: any): BlockNum;
    toJSON(message: BlockNum): unknown;
    fromPartial(object: DeepPartial<BlockNum>): BlockNum;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};

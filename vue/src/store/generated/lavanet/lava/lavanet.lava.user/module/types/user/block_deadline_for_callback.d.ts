import { BlockNum } from "../user/block_num";
import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "lavanet.lava.user";
export interface BlockDeadlineForCallback {
    deadline: BlockNum | undefined;
}
export declare const BlockDeadlineForCallback: {
    encode(message: BlockDeadlineForCallback, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): BlockDeadlineForCallback;
    fromJSON(object: any): BlockDeadlineForCallback;
    toJSON(message: BlockDeadlineForCallback): unknown;
    fromPartial(object: DeepPartial<BlockDeadlineForCallback>): BlockDeadlineForCallback;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};

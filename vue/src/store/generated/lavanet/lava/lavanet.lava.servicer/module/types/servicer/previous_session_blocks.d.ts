import { Writer, Reader } from "protobufjs/minimal";
import { BlockNum } from "../servicer/block_num";
export declare const protobufPackage = "lavanet.lava.servicer";
export interface PreviousSessionBlocks {
    blocksNum: number;
    changeBlock: BlockNum | undefined;
    overlapBlocks: number;
}
export declare const PreviousSessionBlocks: {
    encode(message: PreviousSessionBlocks, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): PreviousSessionBlocks;
    fromJSON(object: any): PreviousSessionBlocks;
    toJSON(message: PreviousSessionBlocks): unknown;
    fromPartial(object: DeepPartial<PreviousSessionBlocks>): PreviousSessionBlocks;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};

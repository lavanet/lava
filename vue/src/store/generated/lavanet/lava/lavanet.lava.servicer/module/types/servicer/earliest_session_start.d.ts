import { BlockNum } from "../servicer/block_num";
import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "lavanet.lava.servicer";
export interface EarliestSessionStart {
    block: BlockNum | undefined;
}
export declare const EarliestSessionStart: {
    encode(message: EarliestSessionStart, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): EarliestSessionStart;
    fromJSON(object: any): EarliestSessionStart;
    toJSON(message: EarliestSessionStart): unknown;
    fromPartial(object: DeepPartial<EarliestSessionStart>): EarliestSessionStart;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};

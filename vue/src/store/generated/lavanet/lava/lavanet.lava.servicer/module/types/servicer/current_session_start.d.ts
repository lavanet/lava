import { BlockNum } from "../servicer/block_num";
import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "lavanet.lava.servicer";
export interface CurrentSessionStart {
    block: BlockNum | undefined;
}
export declare const CurrentSessionStart: {
    encode(message: CurrentSessionStart, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): CurrentSessionStart;
    fromJSON(object: any): CurrentSessionStart;
    toJSON(message: CurrentSessionStart): unknown;
    fromPartial(object: DeepPartial<CurrentSessionStart>): CurrentSessionStart;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};

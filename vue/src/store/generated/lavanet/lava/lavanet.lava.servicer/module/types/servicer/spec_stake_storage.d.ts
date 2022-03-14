import { StakeStorage } from "../servicer/stake_storage";
import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "lavanet.lava.servicer";
export interface SpecStakeStorage {
    index: string;
    stakeStorage: StakeStorage | undefined;
}
export declare const SpecStakeStorage: {
    encode(message: SpecStakeStorage, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): SpecStakeStorage;
    fromJSON(object: any): SpecStakeStorage;
    toJSON(message: SpecStakeStorage): unknown;
    fromPartial(object: DeepPartial<SpecStakeStorage>): SpecStakeStorage;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};

import { StakeStorage } from "../servicer/stake_storage";
import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "lavanet.lava.servicer";
export interface SessionStorageForSpec {
    index: string;
    stakeStorage: StakeStorage | undefined;
}
export declare const SessionStorageForSpec: {
    encode(message: SessionStorageForSpec, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): SessionStorageForSpec;
    fromJSON(object: any): SessionStorageForSpec;
    toJSON(message: SessionStorageForSpec): unknown;
    fromPartial(object: DeepPartial<SessionStorageForSpec>): SessionStorageForSpec;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};

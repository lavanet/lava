import { Writer, Reader } from "protobufjs/minimal";
import { StakeMap } from "../servicer/stake_map";
import { SpecStakeStorage } from "../servicer/spec_stake_storage";
export declare const protobufPackage = "lavanet.lava.servicer";
export interface UnstakingServicersAllSpecs {
    id: number;
    unstaking: StakeMap | undefined;
    specStakeStorage: SpecStakeStorage | undefined;
}
export declare const UnstakingServicersAllSpecs: {
    encode(message: UnstakingServicersAllSpecs, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): UnstakingServicersAllSpecs;
    fromJSON(object: any): UnstakingServicersAllSpecs;
    toJSON(message: UnstakingServicersAllSpecs): unknown;
    fromPartial(object: DeepPartial<UnstakingServicersAllSpecs>): UnstakingServicersAllSpecs;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};

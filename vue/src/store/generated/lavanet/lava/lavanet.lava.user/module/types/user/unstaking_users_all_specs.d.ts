import { Writer, Reader } from "protobufjs/minimal";
import { UserStake } from "../user/user_stake";
export declare const protobufPackage = "lavanet.lava.user";
export interface UnstakingUsersAllSpecs {
    id: number;
    unstaking: UserStake | undefined;
}
export declare const UnstakingUsersAllSpecs: {
    encode(message: UnstakingUsersAllSpecs, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): UnstakingUsersAllSpecs;
    fromJSON(object: any): UnstakingUsersAllSpecs;
    toJSON(message: UnstakingUsersAllSpecs): unknown;
    fromPartial(object: DeepPartial<UnstakingUsersAllSpecs>): UnstakingUsersAllSpecs;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};

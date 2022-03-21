import { UserStake } from "../user/user_stake";
import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "lavanet.lava.user";
export interface StakeStorage {
    stakedUsers: UserStake | undefined;
}
export declare const StakeStorage: {
    encode(message: StakeStorage, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): StakeStorage;
    fromJSON(object: any): StakeStorage;
    toJSON(message: StakeStorage): unknown;
    fromPartial(object: DeepPartial<StakeStorage>): StakeStorage;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};

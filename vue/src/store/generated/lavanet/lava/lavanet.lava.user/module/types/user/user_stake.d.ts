import { Coin } from "../cosmos/base/v1beta1/coin";
import { BlockNum } from "../user/block_num";
import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "lavanet.lava.user";
export interface UserStake {
    index: string;
    stake: Coin | undefined;
    deadline: BlockNum | undefined;
}
export declare const UserStake: {
    encode(message: UserStake, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): UserStake;
    fromJSON(object: any): UserStake;
    toJSON(message: UserStake): unknown;
    fromPartial(object: DeepPartial<UserStake>): UserStake;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};

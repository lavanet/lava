import { Writer, Reader } from "protobufjs/minimal";
import { Params } from "../user/params";
import { UserStake } from "../user/user_stake";
import { SpecStakeStorage } from "../user/spec_stake_storage";
import { BlockDeadlineForCallback } from "../user/block_deadline_for_callback";
import { UnstakingUsersAllSpecs } from "../user/unstaking_users_all_specs";
export declare const protobufPackage = "lavanet.lava.user";
/** GenesisState defines the user module's genesis state. */
export interface GenesisState {
    params: Params | undefined;
    userStakeList: UserStake[];
    specStakeStorageList: SpecStakeStorage[];
    blockDeadlineForCallback: BlockDeadlineForCallback | undefined;
    unstakingUsersAllSpecsList: UnstakingUsersAllSpecs[];
    /** this line is used by starport scaffolding # genesis/proto/state */
    unstakingUsersAllSpecsCount: number;
}
export declare const GenesisState: {
    encode(message: GenesisState, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): GenesisState;
    fromJSON(object: any): GenesisState;
    toJSON(message: GenesisState): unknown;
    fromPartial(object: DeepPartial<GenesisState>): GenesisState;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};

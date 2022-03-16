import { Writer, Reader } from "protobufjs/minimal";
import { Params } from "../servicer/params";
import { StakeMap } from "../servicer/stake_map";
import { SpecStakeStorage } from "../servicer/spec_stake_storage";
import { BlockDeadlineForCallback } from "../servicer/block_deadline_for_callback";
import { UnstakingServicersAllSpecs } from "../servicer/unstaking_servicers_all_specs";
export declare const protobufPackage = "lavanet.lava.servicer";
/** GenesisState defines the servicer module's genesis state. */
export interface GenesisState {
    params: Params | undefined;
    stakeMapList: StakeMap[];
    specStakeStorageList: SpecStakeStorage[];
    blockDeadlineForCallback: BlockDeadlineForCallback | undefined;
    unstakingServicersAllSpecsList: UnstakingServicersAllSpecs[];
    /** this line is used by starport scaffolding # genesis/proto/state */
    unstakingServicersAllSpecsCount: number;
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

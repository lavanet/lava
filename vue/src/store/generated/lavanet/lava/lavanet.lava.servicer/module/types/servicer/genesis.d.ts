import { Writer, Reader } from "protobufjs/minimal";
import { Params } from "../servicer/params";
import { StakeMap } from "../servicer/stake_map";
import { SpecStakeStorage } from "../servicer/spec_stake_storage";
import { BlockDeadlineForCallback } from "../servicer/block_deadline_for_callback";
import { UnstakingServicersAllSpecs } from "../servicer/unstaking_servicers_all_specs";
import { CurrentSessionStart } from "../servicer/current_session_start";
import { PreviousSessionBlocks } from "../servicer/previous_session_blocks";
import { SessionStorageForSpec } from "../servicer/session_storage_for_spec";
import { EarliestSessionStart } from "../servicer/earliest_session_start";
import { UniquePaymentStorageUserServicer } from "../servicer/unique_payment_storage_user_servicer";
import { UserPaymentStorage } from "../servicer/user_payment_storage";
import { SessionPayments } from "../servicer/session_payments";
export declare const protobufPackage = "lavanet.lava.servicer";
/** GenesisState defines the servicer module's genesis state. */
export interface GenesisState {
    params: Params | undefined;
    stakeMapList: StakeMap[];
    specStakeStorageList: SpecStakeStorage[];
    blockDeadlineForCallback: BlockDeadlineForCallback | undefined;
    unstakingServicersAllSpecsList: UnstakingServicersAllSpecs[];
    unstakingServicersAllSpecsCount: number;
    currentSessionStart: CurrentSessionStart | undefined;
    previousSessionBlocks: PreviousSessionBlocks | undefined;
    sessionStorageForSpecList: SessionStorageForSpec[];
    earliestSessionStart: EarliestSessionStart | undefined;
    uniquePaymentStorageUserServicerList: UniquePaymentStorageUserServicer[];
    userPaymentStorageList: UserPaymentStorage[];
    /** this line is used by starport scaffolding # genesis/proto/state */
    sessionPaymentsList: SessionPayments[];
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
